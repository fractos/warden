package warden

import (
	"gopkg.in/redis.v3"
)

// ServiceDescription holds the definition of a service as read from the services file.
type ServiceDescription struct {
	Name             string `json:"name"`
	BackendName      string `json:"backendName"`
	ContainerName    string `json:"containerName"`
	LoadBalancerName string `json:"loadBalancerName"`
	LoadBalancerUrl  string `json:"loadBalancerUrl"`
	Port             int    `json:"port"`
	Site             string `json:"site"`
	TaskName         string `json:"taskName"`
	TaskVersion      int    `json:"taskVersion"`
}

// ServiceDescriptionReader describes an interface for a Read() function that returns a list of ServiceDescription objects.
type ServiceDescriptionReader interface {
	Read() []*ServiceDescription
}

// Configuration holds the basic environment information for Warden including Redis details and the name of the service description file
type Configuration struct {
	NginxRedisAddress          string `json:"nginxRedisAddress"`
	NginxRedisDatabaseNumber   int64  `json:"nginxRedisDatabaseNumber"`
	ServiceDescriptionFilename string `json:"serviceDescriptionFilename"`
	RegistrarSleepTimeSeconds  int64  `json:"registrarSleepTimeSeconds"`
}

// Container holds some basic information about a Docker container.
type Container struct {
	Id        string
	IPAddress string
}

// Warden holds basic instance data plus a redis client for intra-host load-balancing.
type Warden struct {
	region           string
	availabilityZone string
	instanceId       string
	services         []*ServiceDescription
	redisLocal       *redis.Client
}
