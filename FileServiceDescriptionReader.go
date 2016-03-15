package warden

import (
    // "io"
    // "fmt"
    "encoding/json"
    "io/ioutil"
    "log"
//    "time"
//    "strconv"
)

type FileServiceDescriptionReader struct {
    config string
}

func NewFileServiceDescriptionReader(config string) *FileServiceDescriptionReader {
    return &FileServiceDescriptionReader{config: config}
} // NewFileServiceDescriptionReader

func (fileServiceDescriptionReader *FileServiceDescriptionReader) Read() []*ServiceDescription {
    data, err := ioutil.ReadFile(fileServiceDescriptionReader.config)
    if err != nil {
        log.Fatal(err)
    }
    
    //log.Printf("data read: %s\n", data)
    
    var services []*ServiceDescription
    if err:= json.Unmarshal(data, &services); err != nil {
        return nil
    }
        
    return services
} // Read