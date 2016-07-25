package core

import (
    "fmt"
    "os"
    "strings"
    "errors"
    "github.com/docker/engine-api/client"
    "github.com/docker/engine-api/types"
    "golang.org/x/net/context"
)

type LoadInfo struct {
    containerShortId string
    containerId string
    serviceName string
    serviceId string
    stackId string
    stackName string
    nodeId string
    imageId string
    cmd string
    entryPoint string
    archi string
    os string
    registeredPort int
    consul string
    kafka string
}

var loadInfo LoadInfo

//set default parameter values when the loader is not used (but still used as conf default values)
func InitLoader() {
    loadInfo.serviceName = "unknown"
    loadInfo.kafka = ""
    loadInfo.consul = ""
    loadInfo.cmd = ""
    loadInfo.entryPoint = ""
    loadInfo.registeredPort = 0
}

//set all needed amp-pilot variable searching in the container itself if needed
func AutoLoad(cmd []string) error {
    loadInfo.initForLoading()
    loadInfo.getContainerInformation()
    if (len(cmd) > 0) {
        loadInfo.cmd = strings.Trim(loadInfo.entryPoint + " " + loadInfo.cmdToString(cmd), " ")
        fmt.Println("cmd replaced by args=", loadInfo.cmd)
    }
    return nil
}

//Set default parametrer when the loader is used
func (self *LoadInfo) initForLoading() {
    self.containerShortId=os.Getenv("HOSTNAME")
    self.consul = "consul:8500"
    self.kafka = "kafka:9092"
    os.Setenv("CONSUL", "") //Desactivate the embebed amp-pilot if exist
}

//set the default value of the variable if empty
func (self *LoadInfo) setEnvVariable(name string, defaultValue string) {
    var val=os.Getenv(name)
    if (val =="") {
        os.Setenv(name, defaultValue)
    }
}

//get container and image information with Dccker API
func (self *LoadInfo) getContainerInformation() error {
    fmt.Println("ShortContainerId="+self.containerShortId)

    defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
    cli, err := client.NewClient("unix:///var/run/docker.sock", "v1.24", nil, defaultHeaders)
    if err != nil {
        return  err
    }

    inspect, err := cli.ContainerInspect(context.Background(), self.containerShortId)
    if err != nil {
        return err
    }
    self.containerId = inspect.ID
    fmt.Println("ContainerId", self.containerId)
    options := types.ContainerListOptions{All: true}
    containers, err := cli.ContainerList(context.Background(), options)
    if err != nil {
        return err
    }
    var container types.Container
    var found = false
    for _, cont := range containers {
        if (cont.ID == self.containerId) {
            container = cont
            found = true
        }
    }
    if (!found) {
        return errors.New("Impossible to find local container information")
    }
    self.imageId = container.ImageID
    fmt.Println("ImageId="+self.imageId)
    labels := container.Labels
    for key, value := range labels {
        if (key == "com.docker.swarm.service.name") {
            self.serviceName = value
        } else if (key == "com.docker.swarm.service.id") {
            self.serviceId = value
        } else if (key == "com.docker.swarm.task.name") {
            self.stackName = value
        } else if (key == "com.docker.swarm.task.id") {
            self.stackId = value
        } else if (key == "com.docker.swarm.node.id") {
            self.nodeId = value            
        }
    }
    fmt.Println("ServiceName=", self.serviceName)
    ports := container.Ports
    fmt.Println("ports: ", ports)
    if len(ports)>0 {
        self.registeredPort = ports[0].PublicPort
    } 
    fmt.Println("Registered port=", self.registeredPort)
    image, _, err := cli.ImageInspectWithRaw(context.Background(), self.imageId, false)
    if (err != nil) {
        return err
    }
    self.cmd = self.cmdToString(image.Config.Cmd)
    fmt.Println("cmd=", self.cmd)
    self.entryPoint = self.cmdToString(image.Config.Entrypoint)
    fmt.Println("entryPoint=", self.entryPoint)    
    self.cmd = strings.Trim(self.entryPoint + " "+ self.cmd, " ")
    fmt.Println("Cmd result: "+self.cmd)
    self.archi = image.Architecture
    self.os = image.Os
    return nil
}

//create a string usinf a string array
func (self *LoadInfo) cmdToString(cmd []string) string {
    var ret string = ""
    for _, arg := range cmd {
        ret+=arg+" "
    }
    return strings.Trim(ret, " ")
}

