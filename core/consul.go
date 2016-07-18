package core

import (
    "fmt"
    "encoding/json"
    "net/http"
    "io/ioutil"
    "bytes"
)

//Json format of GET response of consul service health check
type consulHealth struct {
    ServiceName string
    Status string
}

//Json format of the POST data to register a service on Consul
type consulRegisterService struct {
    ID string
    Name string
    Address string
    Port int
  //ServiceCheck consulCheck
}

type consulRegisterCheck struct {
    ID string
    Name string
    Notes string
    ServiceID string
    Status string
    TTL string
}

//Json format of Check item in the POST data to register a service on Consul
type consulCheck struct {
    TTL string
}

type Consul struct {
    serviceRegistered bool 
}

var consul Consul

//Check if one dependency is ready using Consul
func (self *Consul) IsDependencyReady(name string) bool {
    data, err := self.getJson("http://"+conf.Consul+"/v1/health/checks/"+name)
    if err != nil {
        return false
    }
    var consulHealthAnswer []consulHealth
    err = json.Unmarshal(data, &consulHealthAnswer)
    if err == nil {
        if len(consulHealthAnswer) == 0 {
            return false;
        }
        if consulHealthAnswer[0].Status != "passing" {
            return false
        }
        return true
    }
    return false
}

//Register app mate onto Consul and/or heard-beat
func (self *Consul) RegisterApp(serviceId string, name string, currentPoll int) {
    if !self.serviceRegistered {
        //applog.Log("app mate registered")
        registerDataService := consulRegisterService {
            ID: serviceId,
            Name: name,
            Address: conf.RegisteredIp,
            Port: conf.RegisteredPort,
            //ServiceCheck: consulCheck {
            //    TTL: fmt.Sprintf("%ds", currentPoll * 2),
            //},
        }
        payloadServ, _ := json.Marshal(registerDataService)
        _, err := self.putJson("http://"+conf.Consul+"/v1/agent/service/register", payloadServ)
        if err == nil {
            self.serviceRegistered=true
        }
    }
    registerDataCheck := consulRegisterCheck {
        ID:  serviceId,
        Name: serviceId,
        Notes:  fmt.Sprintf("TTL for %s set by amp-pilot", name),
        ServiceID: serviceId,
        Status: "passing",
        TTL: fmt.Sprintf("%ds", currentPoll * 2),
    }
    payloadCheck, _ := json.Marshal(registerDataCheck)
    self.putJson("http://"+conf.Consul+"/v1/agent/check/register", payloadCheck)   
}

//De-register app mate onto Consul
func (self *Consul) DeregisterApp(serviceId string) {
    self.serviceRegistered = false
    self.getJson("http://"+conf.Consul+"/v1/agent/service/deregister/"+serviceId)
    self.getJson("http://"+conf.Consul+"/v1/agent/check/deregister/"+serviceId)
}

//execute HTTP GET
func (self *Consul) getJson(url string) ([]byte, error) {
    r, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer r.Body.Close()
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        return nil, err
    }
    return body, err
}

//Execute HTTP PUT
func (self *Consul) putJson(url string, payload []byte) ([]byte, error) {
    var jsonStr = []byte(payload)
    req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonStr))
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        //msg := fmt.Sprintf("%s", err, "string")
        //applog.LogError(msg)
        return nil, err
    }
    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)
    return body, err
}
