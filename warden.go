package warden

import (
    "fmt"
    "time"
//    "strconv"
    "gopkg.in/redis.v3"
//    "github.com/aws/aws-sdk-go"
)

type Warden struct {
    availabilityZone string
    instanceId string
    redisServiceManagement *redis.Client
    redisLocal *redis.Client    
}

const MaxHeartbeatAge = 300
const GuardTime = 15
const ActivityPause = 300
const ServiceManagerPause = 30
const CandidateExpiry = 30

func (self *Warden) Start() {
    fmt.Println("Warden starting...")
    
    // fetch some runtime constants:    
    //   get instance ID
    //   get current availability zone
    self.availabilityZone = "eu-west-1a"
    self.instanceId = "def1"

    serviceManagementRedisAddress := "localhost:6379"
    var serviceManagementRedisDatabaseNumber int64 = 11
    nginxRedisAddress := "localhost:6379"
    var nginxRedisDatabaseNumber int64 = 15

    var serviceDescriptionReaderConfig string = "services.json"

    self.redisServiceManagement = redis.NewClient(&redis.Options{
        Addr: serviceManagementRedisAddress,
        Password: "",
        DB: serviceManagementRedisDatabaseNumber,
    })
    
    self.redisLocal = redis.NewClient(&redis.Options{
        Addr: nginxRedisAddress,
        Password: "",
        DB: nginxRedisDatabaseNumber,
    })
    
    var serviceDescriptionReader ServiceDescriptionReader
    serviceDescriptionReader = NewFileServiceDescriptionReader(serviceDescriptionReaderConfig)

    // start registrar coroutine
    go self.registrar(
        func(s string) { fmt.Printf("registrar: %s\n", s) },
        serviceDescriptionReader)
    // start manager coroutine
    self.manager(func(s string) { fmt.Printf("manager: %s\n", s) })
    
    fmt.Println("Warden finishing...")    
} // main

func (self *Warden) registrar(logger func(s string), serviceDescriptionReader ServiceDescriptionReader) {
    logger("starting...")
    for {

        var services []*ServiceDescription = (serviceDescriptionReader).Read()
        for _, item := range services {
            logger(fmt.Sprintf("item: %s.", item.Name))
        }
        // read list of service descriptions
        
        // for each service description:
        //   ensure service has a frontend defined in redis
        //   synchronise backends with redis, i.e.:
        //     get list of backends from redis
        //     check each backend to see if it has a matching running container.
        //     if yes then remove from working list of containers
        //     if no running container found, then remove from redis and remove from working list of containers
        //     if no containers in list:
        //       if no backends exist:  
        //         deregister from the load balancer
        //       else:
        //         ensure registered with the load balancer
        //     else:  
        //       for each container left over in the working list:
        //         if there is no backend for this container:
        //           add backend details to redis
        //       if added any backend details to redis:
        //         ensure registered with the load balancer
        
        time.Sleep(1 * time.Second)
        
        client := redis.NewClient(&redis.Options{
            Addr:     "localhost:6379",
            Password: "", // no password set
            DB:       0,  // use default DB
        })

        pong, err := client.Ping().Result()
        if err != nil {
            panic(err)
        } else {
            logger(pong)
        }
    }
    
    logger("finishing...")
} // registrar

func (self *Warden) manager(logger func(s string)) {
    
    logger("starting...")
    
    attemptNormalBehaviour := true
    
    // begin loop
    for {       
        // if our current availability zone is not listed as active:
        //   log
        //   attemptNormalBehaviour = false
        if !self.ourAvailabilityZoneIsActive(logger) {
            logger("our availability zone is not listed as active.")
            attemptNormalBehaviour = false
        }
        
        if attemptNormalBehaviour {
            oustServiceManager := !self.isServiceManagerHealthy(logger)
            if oustServiceManager {
                logger("we should try and become the leader.")
                weAreTheLeader := self.becomeLeader(logger)
                if weAreTheLeader {
                    self.managerLifecycle(logger)
                }
            }
        }   
        
        logger("sleeping...")
        time.Sleep(5 * time.Second)
    }
    
} // manager

func (self *Warden) becomeLeader(logger func(s string)) bool {
    logger("becomeLeader")
    
    return true
} // becomeLeader

func (self *Warden) managerLifecycle(logger func(s string)) {
    logger("managerLifecycle")
    
    for {
        // if our current availability zone is not listed as active:
        //   leave lifecycle
        if !self.ourAvailabilityZoneIsActive(logger) {
            logger("... our availability zone is not listed as active.")
            break
        }

        // if kill switch found:
        //   log
        //   remove kill switch
        //   break from lifecycle loop
        if self.killSwitchPresent(logger, self.instanceId) {
            logger("... kill switch found - removing and leaving lifecycle.")
            self.removeKillSwitch(logger, self.instanceId)
            break
        }

        // record heartbeat
        self.recordHeartbeat(logger)
        
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

func (self *Warden) killSwitchPresent(logger func(s string), id string) bool {
    k := fmt.Sprintf("%s-kill", id)
    val, err := self.redisServiceManagement.Exists(k).Result()
    if err != nil {
        panic(err)
    } else {
        return val
    }
} // killSwitchPresent

func (self *Warden) removeKillSwitch(logger func(s string), id string) {
    logger("removeKillSwitch")
    k := fmt.Sprintf("%s-kill", id)
    self.redisServiceManagement.Del(k)
} // removeKillSwitch

func (self *Warden) recordHeartbeat(logger func(s string)) {
    timestamp := self.getTimestamp()
    logger(fmt.Sprintf("recording heartbeat at %s.", timestamp))
    self.redisServiceManagement.HMSet("service-manager", "id", self.instanceId, "heartbeat", timestamp, "subnet", self.availabilityZone)
    self.redisServiceManagement.Expire("service-manager", time.Duration(MaxHeartbeatAge * time.Second))
} // recordHeartbeat

func (self *Warden) availabilityZoneIsActive(logger func(s string), z string) bool {
    // get list of active availability zones
    activeZones := self.getActiveAvailabilityZones(logger)
    
    for _, v := range activeZones {
        if v == z {
            return true
        }
    }
    
    return false
} // availabilityZoneIsActive

func (self *Warden) ourAvailabilityZoneIsActive(logger func(s string)) bool {
    return self.availabilityZoneIsActive(logger, self.availabilityZone)
} // ourAvailabilityZoneIsActive

// TODO
func (self *Warden) getActiveAvailabilityZones(logger func(s string)) []string {
    logger("getActiveAvailabilityZones")
    
    zones := []string { "eu-west-1a", "eu-west-1b", "eu-west-1c" }
    return zones
} // getActiveAvailabilityZones

func (self *Warden) isServiceManagerHealthy(logger func(s string)) bool {
    logger("evaluating current service manager...")   
    
    // get current service manager (will return empty struct if none found)
    currentServiceManager := self.getCurrentServiceManager(logger)
    
    // check if struct is empty
    if currentServiceManager.id == "" {
        logger("... no current service manager. We can challenge.")
        return false
    }
    
    // check if our instance is listed as the current service manager
    if currentServiceManager.id == self.instanceId {
        logger("... that's us. We can challenge.")
        return false
    }
    
    var heartbeatAge = self.getTimeDifference(logger, currentServiceManager.heartbeat, self.getTimestamp())   
        
    var shutdownCurrentServiceManager = false
    
    if heartbeatAge > MaxHeartbeatAge {
        logger(fmt.Sprintf("... current service manager's heartbeat is more than %d seconds in the past. We can challenge.", MaxHeartbeatAge))
        shutdownCurrentServiceManager = true
    } else if !self.availabilityZoneIsActive(logger, currentServiceManager.subnet) {
        logger(fmt.Sprintf("... current service manager's subnet %s is not listed as available. We can challenge.", currentServiceManager.subnet))
        shutdownCurrentServiceManager = true
    } else {
        logger("... current service manager seems healthy. No need to challenge.")
        return true
    }

    if shutdownCurrentServiceManager {
        self.signalServiceManagerToShutdown(logger, currentServiceManager)
    }

    return false
} // isServiceManagerHealthy

func (self *Warden) signalServiceManagerToShutdown(logger func(s string), currentServiceManager ServiceManager) {
    logger("signalling current service manager to shutdown.")
    
    var k = fmt.Sprintf("%s-kill", currentServiceManager.id);
    self.redisServiceManagement.Set(k, "1", 0)
} // signalServiceManagerToShutdown

func (self *Warden) getTimestamp() string {
    now := time.Now()
    return now.Format(time.RFC3339)
} // getTimestamp

func (self *Warden) getTimeDifference(logger func(s string), t1 string, t2 string) int {   
    time1, _ := time.Parse(time.RFC3339, t1)
    time2, _ := time.Parse(time.RFC3339, t2)
    
    duration := time2.Sub(time1)
        
    return int(duration.Seconds())
} // getTimeDifference

func (self *Warden) getCurrentServiceManager(logger func(s string)) ServiceManager {
    vals, err := self.redisServiceManagement.HGetAllMap("service-manager").Result()
    if err != nil {
        panic(err)
    } else {
        return ServiceManager {
            id: vals["id"],
            heartbeat: vals["heartbeat"],
            subnet: vals["subnet"],
        }
    }
    
    return ServiceManager { id: "", heartbeat: "", subnet: "" }
} // getCurrentServiceManager