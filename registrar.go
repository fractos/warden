package warden

import (
    "fmt"
    "time"
    "strings"
    "regexp"
//    "strconv"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/elb"
    "os/exec"
)

func (warden *Warden) startRegistrar(logger func(s string)) {
    logger("starting...")
    for {

        // read list of service descriptions
        for _, service := range warden.services {
            
            // for each service description:
            logger(fmt.Sprintf("service: %s.", service.Name))
            
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
                        
            // get list of containers for this image name
            containers := warden.getMatchingContainers(logger, service.ContainerName)

            if containers == nil {
                logger("problem while inspecting containers detected")
                break
            }

            //   ensure service has a frontend defined in redis
            warden.ensureServiceHasFrontend(logger, service)

            backendAddressesForService := warden.getServiceBackends(logger, service)
            
            if backendAddressesForService == nil {
                logger("problem while inspecting backends detected")
                break
            }
            
            logger("matching backends to list of active containers")
            
            for _, backendAddress := range backendAddressesForService {
                logger(fmt.Sprintf("matching backend with address %s", backendAddress))

                var containerToRemove = -1
                
                for containerIndex, container := range containers {
                    containerAddress := fmt.Sprintf("%s:%d", container.IPAddress, service.Port)
                    if containerAddress == backendAddress {
                        logger(fmt.Sprintf("found container for backend, id:%s, ipaddress:%s", container.Id, container.IPAddress))
                        containerToRemove = containerIndex
                        break
                    }    
                }
                
                if containerToRemove > -1 {
                    logger(fmt.Sprintf("removing container at position %d", containerToRemove))
                    containers = append(containers[:containerToRemove], containers[containerToRemove+1])
                    logger(fmt.Sprintf("container slice now length %d", len(containers)))
                    
                    warden.removeBackend(logger, service, backendAddress)
                }
            }

            if len(containers) == 0 {
                logger("there are 0 active containers")
                if len(backendAddressesForService) == 0 {
                    logger("and there are 0 backends registered")
                    logger("this host will now be deregistered from the load balancer for this service")
                    warden.deregisterServiceFromLoadBalancer(logger, service)
                } else {
                    logger("at least one backend registered")
                    logger("this host will now be (re-)registered to the load balancer for this service")
                    warden.registerServiceWithLoadBalancer(logger, service)
                }
                
                // done with this service
                break                
            }

            logger("at least one active container")

            addedServer := false

            for _, container := range containers {
                logger(fmt.Sprintf("matching container with id:%s ipaddress:%s", container.Id, container.IPAddress))
                
                containerAddress := fmt.Sprintf("%s:%d", container.IPAddress, service.Port)
                
                backendFound := false
                
                for _, backendAddress := range backendAddressesForService {
                    if containerAddress == backendAddress {
                        backendFound = true
                        break
                    }
                }
                
                if !backendFound {
                    logger(fmt.Sprintf("need backend for container with id:%s ipaddress:%s", container.Id, container.IPAddress))
                    warden.addBackend(logger, service, containerAddress)
                    addedServer = true
                }
            }
            
            if addedServer {
                logger("at least one server was added so ensuring registered to the load balancer for this service")
                warden.registerServiceWithLoadBalancer(logger, service)
            }
        }       
        
        time.Sleep(1 * time.Second)
        
        pong, err := warden.redisLocal.Ping().Result()
        if err != nil {
            panic(err)
        } else {
            logger(pong)
        }
    }
} // startRegistrar

func (warden *Warden) getServiceBackends(logger func(s string), service *ServiceDescription) []string {
    logger(fmt.Sprintf("getServiceBackends('%s')", service.Name))
    
    backendName := fmt.Sprintf("backend:%s", service.BackendName)
    
    result, err := warden.redisLocal.HGetAllMap(backendName).Result()

    if err != nil {
        logger(fmt.Sprintf("problem while getting backend hash from redis: %s", err))
        return nil
    }
    
    // pre-allocate size of slice    
    keys := make([]string, len(result))
    i := 0
    for k := range result {
        keys[i] = k
        i++
    }
    
    return keys
} // getServiceBackends

func (warden *Warden) ensureServiceHasFrontend(logger func(s string), service *ServiceDescription) {
    logger(fmt.Sprintf("ensureServiceHasFrontend('%s')", service.Name))
    
    // use warden.redisLocal client to check if this service has a frontend in nginx
    // check redis for 'frontend:' + service.Site
    
    frontendName := fmt.Sprintf("frontend:%s%s", service.LoadBalancerUrl, service.Site)
    frontendExists, err := warden.redisLocal.Exists(frontendName).Result()
    if err != nil {
        logger(fmt.Sprintf("problem while checking redis: %s", err))
        return
    }
    
    if frontendExists {
        logger(fmt.Sprintf("found frontend for %s", service.Site))
        // frontendContents, err := warden.redisLocal.Get(frontendName).Result()
        // if frontendContents != service.BackendName {
        //     logger(fmt.Sprintf("frontend for %s doesn't match backend name %s (actual: %s)", service.Site, service.BackendName, frontendContents))
        // }
    } else {
        logger(fmt.Sprintf("no frontend found for %s", service.Site))
        
        err := warden.redisLocal.Set(frontendName, service.BackendName, 0).Err()
        
        if err != nil {
            logger(fmt.Sprintf("problem while trying to create frontend with name %s: %s", frontendName, err))
            return
        }
    }
} // ensureServiceHasFrontend

func (warden *Warden) removeBackend(logger func(s string), service *ServiceDescription, backendAddress string) {
    logger(fmt.Sprintf("removeBackend('%s', '%s')", service.Site, backendAddress))
    
    backendName := fmt.Sprintf("backend:%s", service.BackendName)
    err := warden.redisLocal.HDel(backendName, backendAddress).Err()
    if err != nil {
        logger(fmt.Sprintf("problem while trying to remove backend: %s", err))
    }
} // removeBackend

func (warden *Warden) addBackend(logger func(s string), service *ServiceDescription, address string) {
    logger(fmt.Sprintf("addBackend('%s', '%s')", service.Name, address))
    
    backendName := fmt.Sprintf("backend:%s", service.BackendName)
    err := warden.redisLocal.HSet(backendName, address, "0").Err()
    if err != nil {
        logger(fmt.Sprintf("problem while trying to add backend: %s", err))
    }
} // addBackend

func (warden *Warden) getMatchingContainers(logger func(s string), containerName string) []Container {
    logger(fmt.Sprintf("getMatchingContainers('%s')...", containerName))
    cmdName := "docker"
    cmdArgs := []string{"ps", "-a", "--filter=status=running"}
    out, err := exec.Command(cmdName, cmdArgs...).Output()
    if err != nil {
        logger(fmt.Sprintf("getMatchingContainers: problem: %s\n", err))
        return nil
    }
    
    lines := strings.Split(string(out), "\n")
    
    var containers []Container
         
    for _, line := range lines {
        if strings.Contains(line, containerName) {
            re1, _ := regexp.Compile(`^(.*?)\s.*$`)
            containerId := re1.FindStringSubmatch(line)[1]
            container := Container { Id: containerId, IPAddress: warden.getContainerIPAddress(logger, containerId) }
            containers = append(containers, container)
            logger(fmt.Sprintf("found matching container id:%s ipaddress:%s", container.Id, container.IPAddress))
        }
    }
        
    return containers
} // getMatchingContainers

func (warden *Warden) getContainerIPAddress(logger func(s string), containerId string) string {
    cmdName := "docker"
    cmdArgs := []string{"inspect", containerId}
    out, err := exec.Command(cmdName, cmdArgs...).Output()
    
    if err != nil {
        logger(fmt.Sprintf("problem while inspect container with id %s: %s", containerId, err))
        return ""
    }
    
    lines := strings.Split(string(out), "\n")
    
    for _, line := range lines {
        re1, _ := regexp.Compile(fmt.Sprintf(`.*?\"IPAddress\"\: \"(.*?)\"\,`))
        if re1.MatchString(line) {
            return re1.FindStringSubmatch(line)[1]
        }
    }
    
    return ""
} // inspectContainer

func (warden *Warden) registerServiceWithLoadBalancer(logger func (s string), service *ServiceDescription) {
    logger(fmt.Sprintf("registerServiceWithLoadBalancer('%s')", service.Name))
    svc := elb.New(session.New(), &aws.Config{Region: aws.String(warden.region)})
    
    params := &elb.RegisterInstancesWithLoadBalancerInput{
        Instances: []*elb.Instance{
            {
                InstanceId: aws.String(warden.instanceId),
            },
        },
        LoadBalancerName: aws.String(service.LoadBalancerName),
    }
    
    _, err := svc.RegisterInstancesWithLoadBalancer(params)
    
    if err != nil {
        logger(fmt.Sprintf("problem while registering service with load balancer: %s", err))
    }
} // registerServiceWithLoadBalancer

func (warden *Warden) deregisterServiceFromLoadBalancer(logger func (s string), service *ServiceDescription) {
    logger(fmt.Sprintf("deregisterServiceFromLoadBalancer('%s')", service.Name))
    svc := elb.New(session.New(), &aws.Config{Region: aws.String(warden.region)})
    
    params := &elb.DeregisterInstancesFromLoadBalancerInput{
        Instances: []*elb.Instance{
            {
                InstanceId: aws.String(warden.instanceId),
            },
        },
        LoadBalancerName: aws.String(service.LoadBalancerName),
    }
    
    _, err := svc.DeregisterInstancesFromLoadBalancer(params)
    
    if err != nil {
        logger(fmt.Sprintf("problem while deregistering service from load balancer: %s", err))
    }
} // deregisterServiceFromLoadBalancer