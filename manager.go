package warden

import (
    "fmt"
    "time"
//    "strconv"
//    "gopkg.in/redis.v3"
//    "github.com/aws/aws-sdk-go"
)

func (warden *Warden) startManager(logger func(s string), configuration *Configuration) {
    
    logger("starting...")
    
    attemptNormalBehaviour := true
    
    // begin loop
    for {       
        // if our current availability zone is not listed as active:
        //   log
        //   attemptNormalBehaviour = false
        if !warden.ourAvailabilityZoneIsActive(logger) {
            logger("our availability zone is not listed as active.")
            attemptNormalBehaviour = false
        }
        
        if attemptNormalBehaviour {
            oustServiceManager := !warden.isServiceManagerHealthy(logger)
            if oustServiceManager {
                logger("we should try and become the leader.")
                weAreTheLeader := warden.becomeLeader(logger)
                if weAreTheLeader {
                    warden.managerLifecycle(logger)
                }
            }
        }   
        
        logger("sleeping...")
        time.Sleep(time.Duration(configuration.ManagerSleepTimeSeconds) * time.Second)
    }
    
} // manager

func (warden *Warden) becomeLeader(logger func(s string)) bool {
    logger("becomeLeader")
    
    return true
} // becomeLeader

func (warden *Warden) managerLifecycle(logger func(s string)) {
    logger("managerLifecycle")
    
    for {
        // if our current availability zone is not listed as active:
        //   leave lifecycle
        if !warden.ourAvailabilityZoneIsActive(logger) {
            logger("... our availability zone is not listed as active.")
            break
        }

        // if kill switch found:
        //   log
        //   remove kill switch
        //   break from lifecycle loop
        if warden.killSwitchPresent(logger, warden.instanceId) {
            logger("... kill switch found - removing and leaving lifecycle.")
            warden.removeKillSwitch(logger, warden.instanceId)
            break
        }

        // record heartbeat
        warden.recordHeartbeat(logger)
        
        // read list of service descriptions
        
        // for each service:
        //   get current traffic level
        //   calculate desired number of tasks
        //   get current number of tasks
        //   log
        //   if current < desired:
        //     increase number of tasks by difference
        //   elsif current > desired:
        //     decrease number of tasks by difference
        //   else:
        //     (current == desired) = no action
        
        logger("sleeping...")
        
        time.Sleep(5 * time.Second)
    }
} // managerLifecycle

func (warden *Warden) killSwitchPresent(logger func(s string), id string) bool {
    k := fmt.Sprintf("%s-kill", id)
    val, err := warden.redisServiceManagement.Exists(k).Result()
    if err != nil {
        panic(err)
    } else {
        return val
    }
} // killSwitchPresent

func (warden *Warden) removeKillSwitch(logger func(s string), id string) {
    logger("removeKillSwitch")
    k := fmt.Sprintf("%s-kill", id)
    warden.redisServiceManagement.Del(k)
} // removeKillSwitch

func (warden *Warden) recordHeartbeat(logger func(s string)) {
    timestamp := warden.getTimestamp()
    logger(fmt.Sprintf("recording heartbeat at %s.", timestamp))
    warden.redisServiceManagement.HMSet("service-manager", "id", warden.instanceId, "heartbeat", timestamp, "subnet", warden.availabilityZone)
    warden.redisServiceManagement.Expire("service-manager", time.Duration(MaxHeartbeatAge * time.Second))
} // recordHeartbeat

func (warden *Warden) isServiceManagerHealthy(logger func(s string)) bool {
    logger("evaluating current service manager...")   
    
    // get current service manager (will return empty struct if none found)
    currentServiceManager := warden.getCurrentServiceManager(logger)
    
    // check if struct is empty
    if currentServiceManager.id == "" {
        logger("... no current service manager. We can challenge.")
        return false
    }
    
    // check if our instance is listed as the current service manager
    if currentServiceManager.id == warden.instanceId {
        logger("... that's us. We can challenge.")
        return false
    }
    
    var heartbeatAge = warden.getTimeDifference(logger, currentServiceManager.heartbeat, warden.getTimestamp())   
        
    var shutdownCurrentServiceManager = false
    
    if heartbeatAge > MaxHeartbeatAge {
        logger(fmt.Sprintf("... current service manager's heartbeat is more than %d seconds in the past. We can challenge.", MaxHeartbeatAge))
        shutdownCurrentServiceManager = true
    } else if !warden.availabilityZoneIsActive(logger, currentServiceManager.subnet) {
        logger(fmt.Sprintf("... current service manager's subnet %s is not listed as available. We can challenge.", currentServiceManager.subnet))
        shutdownCurrentServiceManager = true
    } else {
        logger("... current service manager seems healthy. No need to challenge.")
        return true
    }

    if shutdownCurrentServiceManager {
        warden.signalServiceManagerToShutdown(logger, currentServiceManager)
    }

    return false
} // isServiceManagerHealthy

func (warden *Warden) signalServiceManagerToShutdown(logger func(s string), currentServiceManager ServiceManager) {
    logger("signalling current service manager to shutdown.")
    
    var k = fmt.Sprintf("%s-kill", currentServiceManager.id);
    warden.redisServiceManagement.Set(k, "1", 0)
} // signalServiceManagerToShutdown

func (warden *Warden) getTimestamp() string {
    now := time.Now()
    return now.Format(time.RFC3339)
} // getTimestamp

func (warden *Warden) getCurrentServiceManager(logger func(s string)) ServiceManager {
    vals, err := warden.redisServiceManagement.HGetAllMap("service-manager").Result()
    
    if err != nil {
        panic(err)
    }
    
    return ServiceManager {
        id: vals["id"],
        heartbeat: vals["heartbeat"],
        subnet: vals["subnet"],
    }
} // getCurrentServiceManager