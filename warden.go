package warden

import (
    "fmt"
    "time"
    "strings"
//    "strconv"
    "gopkg.in/redis.v3"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/service/ec2"
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

func wardenLog(s string) {
    fmt.Printf("warden: %s\n", s)
} // WardenLog

// Start will run both the registrar and manager co-routines.
func (warden *Warden) Start() {
    wardenLog("starting...")
    
    // fetch some runtime constants:    
    //   get instance ID
    //   get current availability zone
    warden.region, warden.availabilityZone, warden.instanceId = warden.getInstanceIdentity(wardenLog)

    wardenLog(fmt.Sprintf("region is %s, availability zone is %s, instance ID is %s", warden.region, warden.availabilityZone, warden.instanceId))

    configReader := NewConfigurationReader("config.json")
    configuration := configReader.Read()
    
    var serviceDescriptionReader ServiceDescriptionReader
    serviceDescriptionReader = NewFileServiceDescriptionReader(configuration.ServiceDescriptionFilename)

    warden.services = (serviceDescriptionReader).Read()

    if warden.ourAvailabilityZoneIsActive(wardenLog) {
        wardenLog("our availability zone is active")
    } else {
        wardenLog("our availability zone is not active!")
    }

    warden.redisServiceManagement = redis.NewClient(&redis.Options{
        Addr: configuration.ServiceManagementRedisAddress,
        Password: "",
        DB: configuration.ServiceManagementRedisDatabaseNumber,
    })
    
    warden.redisLocal = redis.NewClient(&redis.Options{
        Addr: configuration.NginxRedisAddress,
        Password: "",
        DB: configuration.NginxRedisDatabaseNumber,
    })
    
    // start registrar coroutine
    warden.startRegistrar(func(s string) { fmt.Printf("registrar: %s\n", s) })
    
        panic("gulp")

    // start manager coroutine
    warden.startManager(func(s string) { fmt.Printf("manager: %s\n", s) })
    
    fmt.Println("Warden finishing...")    
} // main

func (warden *Warden) ourAvailabilityZoneIsActive(logger func(s string)) bool {
    return warden.availabilityZoneIsActive(logger, warden.availabilityZone)
} // ourAvailabilityZoneIsActive

func (warden *Warden) getActiveAvailabilityZones(logger func(s string)) []string {
    logger("getActiveAvailabilityZones")
    
    svc := ec2.New(session.New(), &aws.Config{Region: aws.String(warden.region)})
    
    resp, err := svc.DescribeAvailabilityZones(nil)
    
    if err != nil {
        logger(fmt.Sprintf("error: %s", err))
        return nil
    }
    
    var zones []string
    
    for _, az := range resp.AvailabilityZones {
        if *az.State == "available" {
            zones = append(zones, *az.ZoneName)
        }
    }
    
    logger(fmt.Sprintf("active availability zones: %s", strings.Join(zones, ", ")))
    
    return zones
} // getActiveAvailabilityZones

func (warden *Warden) getTimeDifference(logger func(s string), t1 string, t2 string) int {   
    time1, _ := time.Parse(time.RFC3339, t1)
    time2, _ := time.Parse(time.RFC3339, t2)
    
    duration := time2.Sub(time1)
        
    return int(duration.Seconds())
} // getTimeDifference

func (warden *Warden) getInstanceIdentity(logger func(s string)) (string, string, string) {
    logger("getting instance identity")
    m := ec2metadata.New(session.New())
    doc, err := m.GetInstanceIdentityDocument()
    if err != nil {
        panic(err)
    }
    return doc.Region, doc.AvailabilityZone, doc.InstanceID
} // getInstanceIdentity

func (warden *Warden) availabilityZoneIsActive(logger func(s string), z string) bool {
    // get list of active availability zones
    activeZones := warden.getActiveAvailabilityZones(logger)
    
    if activeZones != nil {
        for _, v := range activeZones {
            if v == z {
                return true
            }
        }
    }
    
    return false
} // availabilityZoneIsActive