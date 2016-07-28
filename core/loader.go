package core

import (
    "fmt"
    "os"
    "strings"
    "errors"
    "github.com/docker/engine-api/client"
    "github.com/docker/engine-api/types"
    "golang.org/x/net/context"
    "time"
    "io"
)

type LoadInfo struct {
    containerShortId string
    containerId string
    serviceName string
    serviceId string
    nodeId string
    imageId string
    cmd string
    entryPoint string
    archi string
    os string
    registeredPort int
    consul string
    kafka string
    pilotLoader bool
}

const srcFolder = "/go/bin/"
const destFolder = "/bin/amppilot/"

var loadInfo LoadInfo

//set default parameter values when the loader is not used (but still used as conf default values)
func InitLoader() {
    loadInfo.serviceName = "unknown"
    loadInfo.kafka = ""
    loadInfo.consul = ""
    loadInfo.cmd = ""
    loadInfo.entryPoint = ""
    loadInfo.registeredPort = 0
    loadInfo.serviceId = ""
    loadInfo.pilotLoader = false
    loadInfo.kafka = os.Getenv("AMPPILOT_KAFKA")
    if (strings.ToLower(loadInfo.kafka) == "none") {
        loadInfo.kafka = ""
    }
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
    self.consul = os.Getenv("CONSUL")
    if self.consul == "" {
        self.consul = "consul:8500"
    }
    self.kafka = os.Getenv("AMPPILOT_KAFKA")
    if self.kafka == "" {
        self.kafka = "kafka:9092"
    }
    if self.kafka == "none" {
        self.kafka = ""
    }
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
            fmt.Println("ServiceName=", self.serviceName)
        } else if (key == "com.docker.swarm.service.id") {
            self.serviceId = value
            fmt.Println("ServiceId=", self.serviceId)
        } else if (key == "com.docker.swarm.node.id") {
            self.nodeId = value  
            fmt.Println("Noded=", self.nodeId)          
        }
    }
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

//initialize amp-pilot and create the binaries folder with all the needed binary files
func InitBinaries(cmd []string) {
    createLoader()
    AutoLoad(cmd)
    applog.init()
    loadInfo.cmd="."   
    conf.load()
    mate.init()
    mate.trapSignal()
    
    createBinary("amp-pilot.amd64")
    createBinary("amp-pilot.alpine")

    applog.Log("Binaries written")
    if (conf.Consul == "") {
        applog.LogError("CONSUL not set. amp-pilot terminated")
        os.Exit(0)
    } 
    applog.Log("Start registering to consul: %s", conf.Consul)
    for {
        consul.RegisterApp("amp-pilot", "amp-pilot", 15)
        time.Sleep(time.Duration(10) * time.Second)
    }
}

//copy a amp-pilot binary in the amp-pilot folder
func createBinary(name string) {
    exist := isExist(destFolder + name, false)
    if (exist) {
        err := os.Remove(destFolder + name)
        if (err != nil) {
            applog.LogError("Warning: erreur deleting binary files %s: %v", name, err)
        }    
    }
    applog.Log("writing binary file %s", name)
    err2 := copy(srcFolder + name, destFolder + name)
    if (err2 != nil)  {
        applog.LogError("Warning: erreur creating binary files %s: %v", name, err2)
        if !exist {
            applog.LogError("Error: The file %s doesn't exist", name)
        }
    }
}

//create the amp-pilot binaries directory if not exist and copy pilotLoader in it
func createLoader() {
    name := "pilotLoader"
    /*
    if (!isExist(destFolder, true)) {
        fmt.Printf("create binaries folder: %s\n", destFolder)
        err := os.MkdirAll(destFolder, 0755)
        if (err != nil) {
            fmt.Printf("Erreur creating binaries floder: %v\n", err)
        }
    } else {
        fmt.Printf("binaries folder already exist: %s\n", destFolder)
    }
    */
    fmt.Printf("writing loader %s\n", name)
    err2 := copy(srcFolder + name, destFolder + name)
    if (err2 != nil)  {
        fmt.Printf("Warning: erreur creating loader %s: %v\n", name, err2)
        if !isExist(destFolder + name, false) {
            fmt.Printf("Error: The file %s doesn't exist\n", name)
        }
    }
}

//binary copy a file in another
func copy(src string, dst string) error {
    in, err := os.Open(src)
    if err != nil { 
        applog.Log("err1")
        return err 
    }
    defer in.Close()
    out, err := os.Create(dst)
    if err != nil {
        applog.Log("err2") 
        return err 
    }
    defer out.Close()
    cherr := out.Chmod(0755)
    if cherr != nil {
        applog.Log("err3") 
        return cherr 
    }
    _, err = io.Copy(out, in)
    if err != nil { 
        applog.Log("err4")
        return err 
    }
    cerr := out.Close()
    if cerr != nil {
        applog.Log("err5") 
        return cerr 
    }
    return nil
}

//verify is a file exist and if it's a directory or not
func isExist(file string, shouldBeDir bool) bool {
    in, err := os.Stat(file)
    if err != nil {
        return false 
    }
    if (shouldBeDir && !in.IsDir()) {
        return false
    }
    return true
}

