package warden

import (
    "fmt"
    "time"
//    "strconv"
    "gopkg.in/redis.v3"
//    "github.com/aws/aws-sdk-go"
    "os/exec"
)

// Warden holds basic instance data plus redis clients for both global service management and intra-host load-balancing.
type Warden struct {
    availabilityZone string
    instanceId string
    services []*ServiceDescription
    redisServiceManagement *redis.Client
    redisLocal *redis.Client    
}

const MaxHeartbeatAge = 300
const GuardTime = 15
const ActivityPause = 300
const ServiceManagerPause = 30
const CandidateExpiry = 30

// Start will run both the registrar and manager co-routines.
func (warden *Warden) Start() {
    fmt.Println("Warden starting...")
    
    // fetch some runtime constants:    
    //   get instance ID
    //   get current availability zone
    warden.availabilityZone = warden.getAvailabilityZone(func(s string) { fmt.Printf("warden: %s\n", s) })
    warden.instanceId = warden.getInstanceId(func(s string) { fmt.Printf("warden: %s\n", s) })

    serviceManagementRedisAddress := "localhost:6379"
    var serviceManagementRedisDatabaseNumber int64 = 11
    nginxRedisAddress := "localhost:6379"
    var nginxRedisDatabaseNumber int64 = 15

    var serviceDescriptionReaderConfig = "services.json"

    var serviceDescriptionReader ServiceDescriptionReader
    serviceDescriptionReader = NewFileServiceDescriptionReader(serviceDescriptionReaderConfig)

    warden.services = (serviceDescriptionReader).Read()

    warden.redisServiceManagement = redis.NewClient(&redis.Options{
        Addr: serviceManagementRedisAddress,
        Password: "",
        DB: serviceManagementRedisDatabaseNumber,
    })
    
    warden.redisLocal = redis.NewClient(&redis.Options{
        Addr: nginxRedisAddress,
        Password: "",
        DB: nginxRedisDatabaseNumber,
    })
    
    // start registrar coroutine
    go warden.startRegistrar(func(s string) { fmt.Printf("registrar: %s\n", s) })
    // start manager coroutine
    warden.startManager(func(s string) { fmt.Printf("manager: %s\n", s) })
    
    fmt.Println("Warden finishing...")    
} // main

func (warden *Warden) startRegistrar(logger func(s string)) {
    logger("starting...")
    for {

        // read list of service descriptions
        for _, service := range warden.services {
            
            // for each service description:
            logger(fmt.Sprintf("service: %s.", service.Name))
            
            // get list of containers for this image name
            var containers = warden.getMatchingContainers(logger, service.Name)
            
            for _, container := range containers {
                logger(fmt.Sprintf("container: %s.", container))
            }
            
            //   ensure service has a frontend defined in redis
            warden.ensureServiceHasFrontend(logger, service)

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
        }       
        
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
} // startRegistrar

func (warden *Warden) ensureServiceHasFrontend(logger func(s string), service *ServiceDescription) {

} // ensureServiceHasFrontend

func (warden *Warden) getMatchingContainers(logger func(s string), containerName string) []string {
    logger(fmt.Sprintf("getMatchingContainers('%s')...", containerName))
    cmdName := "docker"
    cmdArgs := []string{"ps", "-a"}
    out, err := exec.Command(cmdName, cmdArgs...).Output()
    if err != nil {
        logger(fmt.Sprintf("getMatchingContainers: problem: %s\n", err))
        panic(err)
    }
    return []string { fmt.Sprintf("%s", out) }
} // getMatchingContainers

func (warden *Warden) availabilityZoneIsActive(logger func(s string), z string) bool {
    // get list of active availability zones
    activeZones := warden.getActiveAvailabilityZones(logger)
    
    for _, v := range activeZones {
        if v == z {
            return true
        }
    }
    
    return false
} // availabilityZoneIsActive

func (warden *Warden) ourAvailabilityZoneIsActive(logger func(s string)) bool {
    return warden.availabilityZoneIsActive(logger, warden.availabilityZone)
} // ourAvailabilityZoneIsActive

// TODO
func (warden *Warden) getActiveAvailabilityZones(logger func(s string)) []string {
    logger("getActiveAvailabilityZones")
    
    zones := []string { "eu-west-1a", "eu-west-1b", "eu-west-1c" }
    return zones
} // getActiveAvailabilityZones

func (warden *Warden) getTimeDifference(logger func(s string), t1 string, t2 string) int {   
    time1, _ := time.Parse(time.RFC3339, t1)
    time2, _ := time.Parse(time.RFC3339, t2)
    
    duration := time2.Sub(time1)
        
    return int(duration.Seconds())
} // getTimeDifference

func (warden *Warden) getAvailabilityZone(logger func(s string)) string {
    logger("getting availability zone")
    return "eu-west-1a"
} // getAvailabilityZone

func (warden *Warden) getInstanceId(logger func(s string)) string {
    logger("getting instance Id")
    return "def1"
} // getInstanceId
