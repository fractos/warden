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
        
        pong, err := warden.redisLocal.Ping().Result()
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
    
    lines := strings.Split(string(out), "\n")
    
    var containers []string
     
    for _, line := range lines {
        if strings.Contains(line, containerName) {
            re1, _ := regexp.Compile(`^(.*?)\s.*$`)
            containerId := re1.FindStringSubmatch(line)[1]
            containers = append(containers, containerId)
        }
    }
    
    logger(fmt.Sprintf("found containers: %s", strings.Join(containers, ", ")))
    
    return containers
} // getMatchingContainers