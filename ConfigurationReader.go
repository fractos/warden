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

type ConfigurationReader struct {
    configFilename string
}

// New creates a new instance of the configuration reader for this passed filename
func NewConfigurationReader(filename string) *ConfigurationReader {
    return &ConfigurationReader{configFilename: filename}
} // NewConfigurationReader

func (configurationReader *ConfigurationReader) Read() *Configuration {
    data, err := ioutil.ReadFile(configurationReader.configFilename)
    if err != nil {
        log.Fatal(err)
    }
    
    var config *Configuration
    if err:= json.Unmarshal(data, &config); err != nil {
        return nil
    }
        
    return config
} // Read