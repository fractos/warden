package main

import (
    "fmt"
    "time"
    "strconv"
    "gopkg.in/redis.v3"
//    "github.com/aws/aws-sdk-go"
)

type serviceManager struct {
    id string // instance ID
    heartbeat string // timestamp of last heartbeat
    subnet string // subnet that the instance belongs to
}

type serviceDescription struct {
    name string
    backendName string
    containerName string
    loadBalancerName string
    loadBalancerUrl string
    port int
    site string
    taskName string
    taskVersion int
}

var availabilityZone string
var instanceId string
var redisServiceManagement *redis.Client
var redisLocal *redis.Client

func main() {
    fmt.Println("Warden starting...")
    
    // fetch some runtime constants:    
    //   get instance ID
    //   get current availability zone
    availabilityZone = "eu-west-1a"
    instanceId = "def1"

    serviceManagementRedisAddress := "localhost:6379"
    var serviceManagementRedisDatabaseNumber int64 = 11
    nginxRedisAddress := "localhost:6379"
    var nginxRedisDatabaseNumber int64 = 15

    redisServiceManagement = redis.NewClient(&redis.Options{
        Addr: serviceManagementRedisAddress,
        Password: "",
        DB: serviceManagementRedisDatabaseNumber,
    })
    
    redisLocal = redis.NewClient(&redis.Options{
        Addr: nginxRedisAddress,
        Password: "",
        DB: nginxRedisDatabaseNumber,
    })
    
    // start registrar coroutine
    go registrar(func(s string) { fmt.Printf("registrar: %s\n", s) })
    // start manager coroutine
    manager(func(s string) { fmt.Printf("manager: %s\n", s) })
    
    fmt.Println("Warden finishing...")    
} // main


func registrar(logger func(s string)) {
    logger("starting...")
    for {
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

func manager(logger func(s string)) {
    
    logger("starting...")
    
    attemptNormalBehaviour := true
    
    // begin loop
    for {       
        // if our current availability zone is not listed as active:
        //   log
        //   attemptNormalBehaviour = false
        if !ourAvailabilityZoneIsActive(logger) {
            logger("our availability zone is not listed as active.")
            attemptNormalBehaviour = false
        }
        
        if attemptNormalBehaviour {
            oustServiceManager := !isServiceManagerHealthy(logger)
            if oustServiceManager {
                logger("we should try and become the leader.")
                weAreTheLeader := becomeLeader(logger)
                if weAreTheLeader {
                    managerLifecycle(logger)
                }
            }
        }   
        
        logger("sleeping...")
        time.Sleep(5 * time.Second)
    }
    
} // manager

func becomeLeader(logger func(s string)) bool {
    logger("becomeLeader")
    
    return true
} // becomeLeader

func managerLifecycle(logger func(s string)) {
    logger("managerLifecycle")
    
    for {      
       
        // if our current availability zone is not listed as active:
        //   log
        //   break from lifecycle loop (die gracefully?)
        
        if !ourAvailabilityZoneIsActive(logger) {
            logger("our availability zone is not listed as active.")
            break
        }
               
        // if kill switch found:
        //   log
        //   remove kill switch
        //   break from lifecycle loop

        // record heartbeat
        recordHeartbeat(logger)
        
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

func recordHeartbeat(logger func(s string)) {
    logger("recordHeartbeat")
} // recordHeartbeat

func ourAvailabilityZoneIsActive(logger func(s string)) bool {
    logger("ourAvailabilityZoneIsActive")
    
    // get list of active availability zones
    activeZones := getActiveAvailabilityZones(logger)
    
    for _, v := range activeZones {
        if v == availabilityZone {
            return true
        }
    }
    
    return false
} // ourAvailabilityZoneIsActive

func getActiveAvailabilityZones(logger func(s string)) []string {
    logger("getActiveAvailabilityZones")
    
    var zones = []string { "eu-west-1a", "eu-west-1b", "eu-west-1c" }
    return zones
} // getActiveAvailabilityZones

func isServiceManagerHealthy(logger func(s string)) bool {
    logger("isServiceManagerHealthy")
    
    logger("evaluating current service manager...")
    
    // get list of active availability zones
    //activeZones := getActiveAvailabilityZones(logger)
    
    // get current service manager (will return empty struct if none found)
    currentServiceManager := getCurrentServiceManager(logger)
    
    // check if struct is empty
    if currentServiceManager.id == "" {
        logger("... no current service manager. We can challenge.")
        return false
    }
    
    // check if our instance is listed as the current service manager
    if currentServiceManager.id == instanceId {
        logger("... that's us. We can challenge.")
        return false
    }
    
    var heartbeatAge = getTimeDifference(logger, currentServiceManager.heartbeat, getTimestamp())   
    
    logger(strconv.Itoa(heartbeatAge))
    
    //var shutdownCurrentServiceManager = false
    
    

    return false
} // isServiceManagerHealthy

func getTimestamp() string {
    now := time.Now()
    return now.Format(time.RFC3339)
} // getTimestamp

func getTimeDifference(logger func(s string), t1 string, t2 string) int {
    logger("getTimeDifference")
    logger(t1)
    logger(t2)
    
    time1, _ := time.Parse(time.RFC3339, t1)
    time2, _ := time.Parse(time.RFC3339, t2)
    
    duration := time2.Sub(time1)
        
    return int(duration.Seconds())
} // getTimeDifference

func getCurrentServiceManager(logger func(s string)) serviceManager {
    logger("getCurrentServiceManager")
    
    vals, err := redisServiceManagement.HGetAllMap("service-manager").Result()
    if err != nil {
        panic(err)
    } else {
        return serviceManager {
            id: vals["id"],
            heartbeat: vals["heartbeat"],
            subnet: vals["subnet"],
        }
    }
    
    return serviceManager { id: "", heartbeat: "", subnet: "" }
} // getCurrentServiceManager