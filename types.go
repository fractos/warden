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