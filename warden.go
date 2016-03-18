package warden

import (
    "fmt"
    "time"
//    "strconv"
    "gopkg.in/redis.v3"
//    "github.com/aws/aws-sdk-go"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/aws/ec2metadata"
    // "os/exec"
)

// Warden holds basic instance data plus redis clients for both global service management and intra-host load-balancing.
type Warden struct {
    region string
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
    warden.region = warden.getRegion(func(s string) { fmt.Printf("warden: %s\n", s) })
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
    //warden.ec2Service = ec2.New(session.New(), &aws.Config{Region: aws.String(warden.region)})
    
    return "def1"
} // getInstanceId

func (warden *Warden) getRegion(logger func(s string)) string {
    logger("getting region")
    m := ec2metadata.New(session.New())
    region, err := m.Region()
    if err != nil {
        panic(err)
    }
    return region 
} // getRegion
