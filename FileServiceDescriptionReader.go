package warden

import (
    "encoding/json"
    "io/ioutil"
    "log"
)

type FileServiceDescriptionReader struct {
    config string
}

// New creates a new instance of the service description reader for this passed filename
func NewFileServiceDescriptionReader(config string) *FileServiceDescriptionReader {
    return &FileServiceDescriptionReader{config: config}
} // NewFileServiceDescriptionReader

func (fileServiceDescriptionReader *FileServiceDescriptionReader) Read() []*ServiceDescription {
    data, err := ioutil.ReadFile(fileServiceDescriptionReader.config)
    if err != nil {
        log.Fatal(err)
    }
    
    var services []*ServiceDescription
    if err:= json.Unmarshal(data, &services); err != nil {
        return nil
    }
        
    return services
} // Read