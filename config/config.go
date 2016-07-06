package config

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
type Config struct {
    Consul string
    Name string
    Cmd string
    CmdReady string
    RegisteredIp string
    RegisteredPort int
    StartupCheckPeriod int
    CheckPeriod int
    ApplicationStop bool
    LogDirectory string
    StartupLogSize int
    RotateLogSize int
    LogFileFormat string
    Dependencies []DependencyConfig
    NetInterface string
}

type DependencyConfig struct {
    Name string
    OnlyAtStartup bool
}

var conf Config

func GetConfig() *Config {
    return &conf
}

//Load Json conffile and instanciate new Config
func (config *Config) LoadConfig() {
    config.setDefault()
    conffile := os.Getenv("AMPPILOT_CONFFILE") 
    if conffile != "" {
        fmt.Println("Pilot conffile "+conffile+":")
        data, err := ioutil.ReadFile(conffile)
        if err != nil {
            msg := fmt.Sprintf("Conffile read error: %v", err)
            //TODO: save in default startup log file
            fmt.Println(msg)
        } else {
            if err := json.Unmarshal(data, config); err != nil {
                mgs := fmt.Sprintf("Conffile json parsing error: %v", err)
                //TODO: save in default startup log file
                fmt.Println(mgs)
                os.Exit(1)
            }
        }
    }
    config.loadConfigUsingEnvVariable()
    config.RegisteredIp = getServiceIp(config.NetInterface)
    config.controlConfig()
}

//Set default value of configuration
func (config *Config) setDefault() {
    config.Consul = "consul:8500"
    config.Name = "unknown"
    config.CmdReady = ""
    config.RegisteredPort = 80
    config.StartupCheckPeriod = 1
    config.CheckPeriod = 10
    config.ApplicationStop = false
    config.LogDirectory = "."
    config.StartupLogSize = 0
    config.RotateLogSize = 0
    config.LogFileFormat = "2006-01-02 15:04:05.000"
    config.Dependencies = []DependencyConfig{}
    config.NetInterface="docker0"
}

//Update config with env variables
func (config *Config) loadConfigUsingEnvVariable() {
    config.Consul = getStringParameter("consul", config.Consul)
    config.Name = getStringParameter("SERVICE_NAME", config.Name)
    config.Cmd = getStringParameter("AMPPILOT_LAUNCH_CMD", config.Cmd)
    config.CmdReady = getStringParameter("AMPPILOT_READY_CMD", config.CmdReady)
    config.NetInterface = getStringParameter("AMPPILOT_NETINTERFACE", config.NetInterface)
    config.RegisteredPort = getIntParameter("AMPPILOT_REGISTEREDPORT", config.RegisteredPort)
    config.StartupCheckPeriod = getIntParameter("AMPPILOT_STARTUPCHECKPERIOD", config.StartupCheckPeriod)
    config.CheckPeriod = getIntParameter("AMPPILOT_CHECKPERIOD", config.CheckPeriod)
    config.ApplicationStop = getBoolParameter("AMPPILOT_APPLICATIONSTOP", config.ApplicationStop)
    config.LogDirectory = getStringParameter("AMPPILOT_LOGDIRECTORY", config.LogDirectory)
    config.StartupLogSize = getIntParameter("AMPPILOT_STARTUPLOGSIZE", config.StartupLogSize)
    config.RotateLogSize = getIntParameter("AMPPILOT_ROTATELOGSIZE", config.RotateLogSize)
    config.LogFileFormat = getStringParameter("AMPPILOT_LOGFILEFORMAT", config.LogFileFormat)
    config.Dependencies = getDependencyArrayParameter("DEPENDENCIES", config.Dependencies)
}

//Control configutation values, update or exit if critical issue
func (config *Config) controlConfig() {
    if config.Cmd == "" {
        //TODO: save in default startup log file
        fmt.Println("Config error: Cmd is mandatory")
        os.Exit(1) 
    }
    if !strings.HasSuffix(config.LogDirectory, "/") {
        config.LogDirectory+="/"
    }
    _, err := os.Stat(config.LogDirectory) 
    if os.IsNotExist(err) {
        errd := os.MkdirAll(config.LogDirectory, 0755)
        if errd == nil {
            fmt.Printf("Log directory %v didn't exist. It has been created\n", config.LogDirectory)
        } else {
            fmt.Printf("Log directory %v doesn't exist: Error creating it: %v'\n", config.LogDirectory, errd)
            os.Exit(1)
        }
    }
    if conf.StartupCheckPeriod > conf.CheckPeriod {
        conf.StartupCheckPeriod = conf.CheckPeriod
    }
}

//return env variable value if empty return default value
func getStringParameter(envVariableName string, def string) string {
    value := os.Getenv(envVariableName)
    if value == "" {
        return def
    }
    return value;
}

//return env variable value convert to bool if empty return default value
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

//return env variable value convert to int if empty return default value
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

//return env variable value convert to string array if empty return default value
func getDependencyArrayParameter(envVariableName string, def []DependencyConfig) []DependencyConfig {
    value := os.Getenv(envVariableName)
    if value == "" {
        return def
    } 
    if value == "" {
        return make([]DependencyConfig, 0)
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

