package warden

import (
    "fmt"
    "time"
    "strings"
    "regexp"
//    "strconv"
//    "github.com/aws/aws-sdk-go"
    "os/exec"
)

func (warden *Warden) startRegistrar(logger func(s string)) {
    logger("starting...")
    for {

        // read list of service descriptions
        for _, service := range warden.services {
            
            // for each service description:
            logger(fmt.Sprintf("service: %s.", service.Name))
            
            // get list of containers for this image name
            containers := warden.getMatchingContainers(logger, service.Name)
                        
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
        
        pong, err := warden.redisLocal.Ping().Result()
        if err != nil {
            panic(err)
        } else {
            logger(pong)
        }
    }
} // startRegistrar

func (warden *Warden) ensureServiceHasFrontend(logger func(s string), service *ServiceDescription) {
    logger(fmt.Sprintf("ensureServiceHasFrontend('%s')", service.Name))
    
    // use warden.redisLocal client to check if this service has a frontend in nginx
    // check redis for 'frontend:' + service.Site
    
    frontendName := fmt.Sprintf("frontend:%s", service.Site)
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

func (warden *Warden) getMatchingContainers(logger func(s string), containerName string) []Container {
    logger(fmt.Sprintf("getMatchingContainers('%s')...", containerName))
    cmdName := "docker"
    cmdArgs := []string{"ps", "-a", "--filter=status=running"}
    out, err := exec.Command(cmdName, cmdArgs...).Output()
    if err != nil {
        logger(fmt.Sprintf("getMatchingContainers: problem: %s\n", err))
        panic(err)
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