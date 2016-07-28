package core

import (
    "os"
    "fmt"
    "io/ioutil"
    "encoding/json"
    "strconv"
    "strings"
    "net"
)

//Json format of conffile
type PilotConfig struct {
    Consul string
    Name string
    Cmd string
    CmdReady string
    RegisteredIp string
    RegisteredPort int
    StartupCheckPeriod int
    CheckPeriod int
    ApplicationStop bool
    LogFileFormat string
    Dependencies []DependencyConfig
    NetInterface string
    Kafka string
    KafkaTopic string
    //updated by amp-pilot
    Host string
}

type DependencyConfig struct {
    Name string
    OnlyAtStartup bool
}

var conf PilotConfig

//Load Json conffile and instanciate new Config
func (self *PilotConfig) load() {
    self.setDefault()
    conffile := os.Getenv("AMPPILOT_CONFFILE") 
    if conffile != "" {
        fmt.Println("Pilot conffile "+conffile+":")
        data, err := ioutil.ReadFile(conffile)
        if err != nil {
            applog.Log("Conffile read error: %v", err)
        } else {
            if err := json.Unmarshal(data, conf); err != nil {
                applog.Log("Conffile json parsing error: %v", err)
                os.Exit(1)
            }
        }
    }
    self.loadConfigUsingEnvVariable()
    self.RegisteredIp = getServiceIp(self.NetInterface)
    self.controlConfig()
}

//Set default value of configuration
func (self *PilotConfig) setDefault() {
    self.Consul = loadInfo.consul
    self.Name = loadInfo.serviceName
    self.Cmd = loadInfo.cmd
    self.CmdReady = ""
    self.RegisteredPort = loadInfo.registeredPort
    self.StartupCheckPeriod = 1
    self.CheckPeriod = 10
    self.ApplicationStop = false
    self.LogFileFormat = "2006-01-02'T'15.04.05.000"//Format to be accepted by elasticsearch
    self.Dependencies = make([]DependencyConfig, 0)
    self.NetInterface="eth0"
    self.Kafka= loadInfo.kafka
    self.KafkaTopic="amp-logs"
    host, err := os.Hostname()
    if err == nil {
        self.Host = host
    } else {
        self.Host = ""
    }
    //displayIp()
}

//Update config with env variables
func (self *PilotConfig) loadConfigUsingEnvVariable() {
    self.Consul = getStringParameter("CONSUL", self.Consul)
    self.Name = getStringParameter("SERVICE_NAME", self.Name)
    self.Cmd = getStringParameter("AMPPILOT_LAUNCH_CMD", self.Cmd)
    self.CmdReady = getStringParameter("AMPPILOT_READY_CMD", self.CmdReady)
    self.NetInterface = getStringParameter("AMPPILOT_NETINTERFACE", self.NetInterface)
    self.RegisteredPort = getIntParameter("AMPPILOT_REGISTEREDPORT", self.RegisteredPort)
    self.StartupCheckPeriod = getIntParameter("AMPPILOT_STARTUPCHECKPERIOD", self.StartupCheckPeriod)
    self.CheckPeriod = getIntParameter("AMPPILOT_CHECKPERIOD", self.CheckPeriod)
    self.ApplicationStop = getBoolParameter("AMPPILOT_APPLICATIONSTOP", self.ApplicationStop)
    //self.LogFileFormat = getStringParameter("AMPPILOT_LOGFILEFORMAT", self.LogFileFormat)
    self.Dependencies = getDependencyArrayParameter("DEPENDENCIES", self.Dependencies)
    self.Kafka = getStringParameter("AMPPILOT_KAFKA", self.Kafka)
    self.KafkaTopic = getStringParameter("KAFKA_TOPIC", self.KafkaTopic)
}

//Control configutation values, update or exit if critical issue
func (self *PilotConfig) controlConfig() {
    if self.Cmd == "" {
        applog.Log("Config error: Cmd is mandatory")
        os.Exit(1) 
    }
    if conf.StartupCheckPeriod > conf.CheckPeriod {
        conf.StartupCheckPeriod = conf.CheckPeriod
    }
}

//return env variable value, if empty return default value
func getStringParameter(envVariableName string, def string) string {
    value := os.Getenv(envVariableName)
    if value == "" {
        return def
    }
    return value;
}

//return env variable value convert to bool, if empty return default value
func getBoolParameter(envVariableName string, def bool) bool {
    value := os.Getenv(envVariableName)
    if value == "" {
        return def
    }
    if value == "true" {
        return  true
    }
    return false;
}

//return env variable value convert to int, if empty return default value
func getIntParameter(envVariableName string, def int) int {
    value := os.Getenv(envVariableName)
    if value != "" {
        ivalue, err := strconv.Atoi(value)
        if err != nil {
            return def
        }
        return ivalue
    } else {
        return def
    }
}

//return env variable value convert to string array, if empty return default value
func getDependencyArrayParameter(envVariableName string, def []DependencyConfig) []DependencyConfig {
    value := os.Getenv(envVariableName)
    if value == "" {
        return def
    } 
    list := strings.Split( strings.Replace(value," ","", -1), ",")
    ret := make([]DependencyConfig, len(list))
    for ii := 0; ii < len(list); ii++ {
        ret[ii].Name = list[ii]
        ret[ii].OnlyAtStartup = false
        varName := "AMPPILOT_"+strings.ToUpper(strings.Replace(ret[ii].Name,"-","", -1))+"_ONLYONSTARTUP"
        val := os.Getenv(varName)
        if val == "true" {
            ret[ii].OnlyAtStartup = true
        } 
    }
    return ret
}

//get the ip address corresponding to the configured net interface
func getServiceIp(netInterface string) string {
    list, err := net.Interfaces()
    if err != nil {
        fmt.Println("get net interfaces error: ",err)
        return "127.0.0.1"
    } 
    for _, iface := range list {
        if (iface.Name == netInterface) {
            addrs, err := iface.Addrs()
            if err != nil || len(addrs) == 0 {
                fmt.Printf("get ip for net interfaces %s error: %v\n", netInterface, err)
                return "127.0.0.1"
            }
            return strings.Split(addrs[0].String(), "/")[0]
        }
    }
    return "127.0.0.1"
}

//for debug: display the list of all interface with the first ip address
func displayIp() {
    applog.Log("Ip list: ")
    list, err := net.Interfaces()
    if err != nil {
        applog.Log("get net interfaces error: ",err)
    } else {
        for _, iface := range list {
            addrs, _ := iface.Addrs()
            applog.Log("interface: %s, ip: %s\n", iface.Name, strings.Split(addrs[0].String(), "/")[0])
        }
    }
    applog.Log("end list")
}

