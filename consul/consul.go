package consul

import (
    "fmt"
    "encoding/json"
    "config"
    "net/http"
    "io/ioutil"
    "bytes"
    "applog"
    "strconv"
)

//Json format of GET response of consul service health check
type consulHealth struct {
    ServiceName string
    Status string
}

//Json format of the POST data to register a service on Consul
type consulRegister struct {
  ID string
  Name string
  Address string
  Port int
  ServiceCheck consulCheck
}

//Json format of Check item in the POST data to register a service on Consul
type consulCheck struct {
    TTL string
}

var (
    conf *config.Config = config.GetConfig()
)

//Check if one dependency is ready using Consul
func IsDependencyReady(name string) bool {
    data, err := getJson("http://"+conf.Consul+"/v1/health/checks/"+name)
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

//Register app mate onto Consul
func RegisterApp(id string, name string, currentPoll int) {
    registerData := consulRegister {
        ID: id,
        Name: name,
        Address: "localhost",
        Port: 8080,
        ServiceCheck: consulCheck {
            TTL: strconv.Itoa(currentPoll * 2) + "s",
        },
    }
    payload, _ := json.Marshal(registerData)
    putJson("http://"+conf.Consul+"/v1/agent/service/register", payload)
}

//De-register app mate onto Consul
func DeregisterApp(id string) {
    getJson("http://"+conf.Consul+"/v1/agent/service/deregister/"+id)
    applog.Log("app de-registered: "+id)
}

//execute HTTP GET
func getJson(url string) ([]byte, error) {
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
func putJson(url string, payload []byte) ([]byte, error) {
    var jsonStr = []byte(payload)
    req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonStr))
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        msg := fmt.Sprintf("%s", err, "string")
        applog.LogError(msg)
        return nil, err
    }
    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)
    return body, err
}
