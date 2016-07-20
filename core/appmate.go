package core

import (
    "fmt"
    "time"
    "os/exec"
    "os"
    "math/rand"
    "os/signal"
    "syscall"
    "strings"
)

const KillSafeDuration time.Duration = 30 * time.Second //min of time between two kill


//All app mate related variables
type appMate struct {
    serviceId string
    currentPeriod int
    dependenciesReady bool
    appReady bool
    appStarted bool
    killTime time.Time
    app *exec.Cmd
    startupLogActivated bool
    rotateLogActiveted bool
    stopApp bool
}

var (
    mate appMate
)


//Set app mate initial values
func (self * appMate) init() {
    rd := rand.New(rand.NewSource(time.Now().UnixNano()))
    id := rd.Int()
    self.serviceId = fmt.Sprintf("%v_%v",conf.Name, id)
    self.dependenciesReady = false
    self.currentPeriod = conf.StartupCheckPeriod
    self.killTime = time.Now().Add(-KillSafeDuration)
    self.stopApp = conf.ApplicationStop
    self.appReady = false
    self.displayConfig()
}

//display amp-pilot configuration
func (self * appMate) displayConfig() {
    applog.Log("amp-pilot version: %v", ampPilotVersion)
    applog.Log("----------------------------------------------------------------------------")
    applog.Log("Configuration:")
    applog.Log("Consul addr: %v", conf.Consul)
    applog.Log("Kafka addr: %v", conf.Kafka)
    applog.Log("Kafka topic: %v", conf.KafkaTopic)
    applog.Log("App mate name: %v", conf.Name)
    applog.Log("App mate script cmd: %v", conf.Cmd)
    applog.Log("App mate script ready cmd: %v", conf.CmdReady)
    applog.Log("Stop container if app mate stop by itself: %v", conf.ApplicationStop)
    applog.Log("Startup check period: %v sec.", conf.StartupCheckPeriod)
    applog.Log("Check period: %v sec.", conf.CheckPeriod)
    applog.Log("Dependency list {name, onlyAtStartup}: %v", conf.Dependencies)
    applog.Log("Service instance id: "+self.serviceId)
    applog.Log("Service registered IP: %s (on interface: %s)", conf.RegisteredIp, conf.NetInterface)
    if (conf.RegisteredPort == 0) {
        applog.Log("Service registered Port: no registered port")
    } else {
        applog.Log("Service registered Port: %v",conf.RegisteredPort)
    }
    applog.Log("----------------------------------------------------------------------------")
}

//Launch a routine to catch SIGTERM Signal
func (self * appMate) trapSignal() {
    ch := make(chan os.Signal, 1)
    signal.Notify(ch, os.Interrupt)
    signal.Notify(ch, syscall.SIGTERM)
    go func() {
        <-ch
        applog.Log("\namp-pilot received SIGTERM signal")
        if self.isAppLaunched() {    
            self.stopAppMate()
        }
        consul.DeregisterApp(self.serviceId)
        kafka.close()
        os.Exit(1)
    }()
}

//Check if all dependencies are ready
func (self * appMate) checkDependencies(appLaunched bool) bool {
    //no dependency case
    if len(conf.Dependencies) == 0 {
        return true
    }
    var slog string = "check dependencies: "
    //after an application kill, there is a safe period during which the application shouldn't be restarted (even if all its dependencies are ready)
    if !self.killTime.Add(KillSafeDuration).Before(time.Now()) {
        slog+=" not ready (kill safe period)"
        applog.Log(slog)
        return false    
    }
    //Check dependencies
    var ret bool = true
    for ii := 0; ii < len(conf.Dependencies); ii++ {
        dep := conf.Dependencies[ii]
        if !consul.IsDependencyReady(dep.Name) {
            if (dep.OnlyAtStartup && appLaunched) {
                slog+=dep.Name+"=not ready (but not mandatory) "
            } else {
                slog+=dep.Name+"=not ready "
                ret=false
            }
        } else {
            slog+=dep.Name+"=ready "
        } 
    }  
    if (!ret || !self.appStarted) { //to do not be too much verbose, don't log if app is started, excepted if there is a dependency failure
        applog.Log(slog)
    }
    return ret;
}

//Verify is app mate is ready using script conf.CmdReady. if not exist app mate is concidered ready
func (self * appMate) isAppReady() bool {
    if conf.CmdReady == "" {
        return true
    }
    applog.Log("execute: "+conf.CmdReady)
    cmdList := strings.Split(conf.CmdReady, " ")[:]
    cmd := exec.Command(cmdList[0], cmdList[1:]...)
    err := cmd.Run()
    if err != nil {
        applog.Log("app mate not ready: "+conf.CmdReady+" throw error=", err)
        return false  
    }
    applog.Log("app mate ready: "+conf.CmdReady+" return code 0")
    return true
}

//Launch the app mate usin conffile cmd command
func (self * appMate) executeApp(attachLog bool) {
    applog.Log("execute: "+conf.Cmd);
    cmdList := strings.Split(conf.Cmd, " ")[:]
    self.app = exec.Command(cmdList[0], cmdList[1:]...)
    if attachLog {
        self.app.Stderr = applog.getPipeStderrWriter()
        self.app.Stdout = applog.getPipeStdoutWriter()
    }
    self.appStarted = true
    self.app.Run()
    self.appStarted = false
}

//Stop app mate
func (self * appMate) stopAppMate() {
    applog.Log("Send SIGTERM signal to app: "+conf.Name)
    self.killTime=time.Now()
    self.stopApp = false
    if self.app != nil {
        //TODO: SIGTERM then wait and kill if app mate not stopped
        self.app.Process.Kill()
    }
}

//Verify is app mate is launched
func (self * appMate) isAppLaunched() bool {
    return self.appStarted
    /*
    //Don't work correctly and actually not needed.TODO: supress function isAppLaunched, self.addStarted if enough
    if self.app == nil {
        return false
    }
    if self.app.ProcessState == nil  {
        return true
    }
    return self.app.ProcessState.Exited()
    */
}

//Check dependencies and register if app mate is started and ready, stop app if a dependency is not ready
func (self * appMate) checkForDependenciesAndReadyness() {
    launched := self.isAppLaunched()
    if launched && self.appReady {
        consul.RegisterApp(self.serviceId, conf.Name, conf.CheckPeriod)
    }
    self.dependenciesReady = self.checkDependencies(launched)
    if self.dependenciesReady {
        if !launched {
            self.appReady = self.isAppReady()
        }
    } else {
        if launched {
            self.stopAppMate()
        }
        applog.Log("waiting for dependencies");
    }
}

//laucnh routine to check dependencies and register on regular basis and be able to change its period dynamically
func (self * appMate) startPeriodicChecking() {
    go func() {
        for {
            self.checkForDependenciesAndReadyness()
            time.Sleep(time.Duration(self.currentPeriod) * time.Second)
        }
    }()
}


