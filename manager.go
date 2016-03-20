package warden

import (
    "fmt"
    "time"
    "math/rand"
//    "strconv"
    "gopkg.in/redis.v3"
//    "github.com/aws/aws-sdk-go"
)

func (warden *Warden) startManager(logger func(s string), configuration *Configuration) {
    
    logger("starting...")
    
    // begin loop
    for {       
        // if our current availability zone is not listed as active:
        if !warden.ourAvailabilityZoneIsActive(logger) {
            logger("our availability zone is not listed as active.")

        } else {
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
    logger("becomeLeader()")
    
    logger("...announcing ourselves as a leadership candidate")
    
    diceValue := warden.rollDice(logger)
    
    // record instanceId on an expiring key
    candidateKey := warden.recordCandidateData(logger)

    // record that key against the dice value in a sorted set
    warden.recordLeadershipChallenge(logger, diceValue, candidateKey)
    
    // wait for guard time
    logger(fmt.Sprintf("... waiting for guard time (%d seconds)", GuardTime))
    time.Sleep(time.Duration(GuardTime) * time.Second)
    
    // winner is the one with the highest score with a key that has not expired
    winnerInstanceId := warden.getLeadershipChallengeWinner(logger)
    
    if winnerInstanceId != warden.instanceId {
        logger(fmt.Sprintf("... we did not win the leadership competition (%s did)", winnerInstanceId))
        return false
    }
    
    logger("... we won the leadership challenge")
    
    return true
} // becomeLeader

func (warden *Warden) rollDice(logger func(s string)) int {
    logger("... rollDice()")
    
    s1 := rand.NewSource(time.Now().UnixNano())
    r1 := rand.New(s1)
    roll := r1.Intn(100)
    
    logger(fmt.Sprintf("... roll was %d", roll))
    return roll
} // rollDice

func (warden *Warden) recordCandidateData(logger func(s string)) string {
    logger("... recordCandidateData()")
    
    key := warden.getTimestamp()
    
    logger(fmt.Sprintf("... recording candidate data for %s with key %s", warden.instanceId, key))
    logger(fmt.Sprintf("... (this will expire in %d seconds)", CandidateExpiry))
    
    warden.redisServiceManagement.Set(key, warden.instanceId, time.Duration(CandidateExpiry) * time.Second)
    
    return key    
} // recordCandidateData

func (warden *Warden) recordLeadershipChallenge(logger func(s string), diceValue int, candidateKey string) {
    logger(fmt.Sprintf("... recordLeadershipChallenge(%d, '%s')", diceValue, candidateKey))
    
    logger(fmt.Sprintf("... recording leadership challenge for key %s with score %d", candidateKey, diceValue))

    err := warden.redisServiceManagement.ZAdd("leadership-challenge", redis.Z{float64(diceValue), candidateKey}).Err()
    
    if err != nil {
        logger(fmt.Sprintf("problem while adding to sorted set: %s", err))
    }
} // recordLeadershipChallenge

func (warden *Warden) getLeadershipChallengeWinner(logger func(s string)) string {
    logger("... getLeadershipChallengeWinner()")
    
        
} // getLeadershipChallengeWinner

func (warden *Warden) managerLifecycle(logger func(s string)) {
    logger("managerLifecycle()")
    
    for {
        // if our current availability zone is not listed as active:
        //   leave lifecycle
        if !warden.ourAvailabilityZoneIsActive(logger) {
            logger("... our availability zone is no longer listed as active.")
            break
        }

        logger("... our availability zone is active.")

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

        for _, service := range warden.services {
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
        }        
        
        logger("sleeping...")
        
        time.Sleep(5 * time.Second)
    }
} // managerLifecycle

func (warden *Warden) killSwitchPresent(logger func(s string), id string) bool {
    k := fmt.Sprintf("%s-kill", id)
    val, err := warden.redisServiceManagement.Exists(k).Result()
    if err != nil {
        logger(fmt.Sprintf("... problem while checking key: %s", err))
        return false
    }
    return val
} // killSwitchPresent

func (warden *Warden) removeKillSwitch(logger func(s string), id string) {
    logger("... removeKillSwitch")
    k := fmt.Sprintf("%s-kill", id)
    warden.redisServiceManagement.Del(k)
} // removeKillSwitch

func (warden *Warden) recordHeartbeat(logger func(s string)) {
    timestamp := warden.getTimestamp()
    logger(fmt.Sprintf("... recording heartbeat at %s.", timestamp))
    warden.redisServiceManagement.HMSet("service-manager", "id", warden.instanceId, "heartbeat", timestamp, "subnet", warden.availabilityZone)
    warden.redisServiceManagement.Expire("service-manager", time.Duration(MaxHeartbeatAge * time.Second))
} // recordHeartbeat

func (warden *Warden) isServiceManagerHealthy(logger func(s string)) bool {
    logger("... evaluating current service manager...")   
    
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

func (warden *Warden) signalServiceManagerToShutdown(logger func(s string), currentServiceManager *ServiceManager) {
    logger("... signalling current service manager to shutdown.")
    
    var k = fmt.Sprintf("%s-kill", currentServiceManager.id);
    warden.redisServiceManagement.Set(k, "1", 0)
} // signalServiceManagerToShutdown

func (warden *Warden) getTimestamp() string {
    now := time.Now()
    return now.Format(time.RFC3339)
} // getTimestamp

func (warden *Warden) getCurrentServiceManager(logger func(s string)) *ServiceManager {
    vals, err := warden.redisServiceManagement.HGetAllMap("service-manager").Result()
    
    if err != nil {
        logger(fmt.Sprintf("problem while getting current service manager: %s", err))
        return nil
    }
    
    return &ServiceManager {
        id: vals["id"],
        heartbeat: vals["heartbeat"],
        subnet: vals["subnet"],
    }
} // getCurrentServiceManager