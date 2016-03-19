package warden

type ServiceManager struct {
    id string // instance ID
    heartbeat string // timestamp of last heartbeat
    subnet string // subnet that the instance belongs to
}

type ServiceDescription struct {
    Name string `json:"name"`
    BackendName string `json:"backendName"`
    ContainerName string `json:"containerName"`
    LoadBalancerName string `json:"loadBalancerName"`
    LoadBalancerUrl string `json:"loadBalancerUrl"`
    Port int `json:"port"`
    Site string `json:"site"`
    TaskName string `json:"taskName"`
    TaskVersion int `json:"taskVersion"`
}

type ServiceDescriptionReader interface {
    Read() []*ServiceDescription
}

// Configuration holds the basic environment information for Warden including Redis details and the name of the service description file
type Configuration struct {
    ServiceManagementRedisAddress string `json:"serviceManagementRedisAddress"`
    ServiceManagementRedisDatabaseNumber int64 `json:"serviceManagementRedisDatabaseNumber"`
    NginxRedisAddress string `json:"nginxRedisAddress"`
    NginxRedisDatabaseNumber int64 `json:"nginxRedisDatabaseNumber"`
    ServiceDescriptionFilename string `json:"serviceDescriptionFilename"`
}

type Container struct {
    Id string
    IPAddress string
}