package appmate

import (
    "fmt"
    "time"
    "os/exec"
    "runtime"
    "os"
    "math/rand"
    "os/signal"
    "syscall"
    "strings"
    //amp-pilot package
    "consul"
    "applog"
    "config"

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
    conf *config.Config = config.GetConfig()
)


//Main loop
func Run() {
    conf.LoadConfig()
    applog.InitLog()
    initMate()
    trapSignal()
    runtime.GOMAXPROCS(4)
    applog.Log("waiting for dependencies...");
    startPeriodicChecking()
    for {
        if mate.dependenciesReady && mate.appReady {
            mate.currentPeriod = conf.CheckPeriod
            executeApp()
            mate.dependenciesReady = checkDependencies(false)
            mate.appReady=false
            mate.currentPeriod = conf.StartupCheckPeriod
            if mate.stopApp {
                consul.DeregisterApp(mate.serviceId)
                applog.Log("App mate has stopped")
                os.Exit(0)
            }
            mate.stopApp = conf.StopAtMateStop
        } 
        time.Sleep(1 * time.Second)
    }
}

//Set app mate initial values
func initMate() {
    rd := rand.New(rand.NewSource(time.Now().UnixNano()))
    id := rd.Int()
    mate.serviceId = fmt.Sprintf("%v_%v",conf.Name, id)
    mate.dependenciesReady = false
    mate.currentPeriod = conf.StartupCheckPeriod
    mate.killTime = time.Now().Add(-KillSafeDuration)
    mate.stopApp = conf.StopAtMateStop
    mate.appReady = true
    displayConfig()
}

func displayConfig() {
    applog.Log("----------------------------------------------------------------------------")
    applog.Log("Configuration:")
    applog.Log("Consul addr: %v", conf.Consul)
    applog.Log("App mate name: %v", conf.Name)
    applog.Log("App mate script cmd: %v", conf.Cmd)
    applog.Log("Stop container at app mate stop: %v", conf.StopAtMateStop)
    applog.Log("Startup check period: %v sec.", conf.StartupCheckPeriod)
    applog.Log("Check period: %v sec.", conf.CheckPeriod)
    applog.Log("Log directory: %v", conf.LogDirectory)
    applog.Log("Startup log size: %v MB", conf.StartupLogSize)
    applog.Log("Rotate log size: %v MB", conf.RotateLogSize)
    applog.Log("Dependency names list: %v", conf.Dependencies)
    applog.Log("Service instance id: "+mate.serviceId)
    applog.Log("----------------------------------------------------------------------------")
}

//Launch a routine to catch SIGTERM Signal
func trapSignal() {
    ch := make(chan os.Signal, 1)
    signal.Notify(ch, os.Interrupt)
    signal.Notify(ch, syscall.SIGTERM)
    go func() {
        <-ch
        applog.Log("\namp-pilot received SIGTERM signal")
        if isAppLaunched() {    
            stopApp()
        }
        consul.DeregisterApp(mate.serviceId)
        applog.CloseFiles()
        os.Exit(1)
    }()
}

//Check if all dependencies are ready
func checkDependencies(appLaunched bool) bool {
    if len(conf.Dependencies) == 0 {
        return true
    }
    var slog string = "check dependencies: "
    if !mate.killTime.Add(KillSafeDuration).Before(time.Now()) {
        slog+=" not ready (kill safe period)"
        applog.Log(slog)
        return false    
    }
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
    applog.Log(slog)
    return ret;
}

//Verify is app mate is ready using script conf.CmdReady. if not exist app mate is concidered ready
func isAppReady() bool {
    if conf.CmdReady == "" {
        return true
    }
    applog.Log("execute: "+conf.CmdReady)
    cmdList := strings.Split(conf.CmdReady, " ")[:]
    cmd := exec.Command(cmdList[0], cmdList[1:]...)
    err := cmd.Run()
    if err != nil {
    fmt.Println("app mate not ready: "+conf.CmdReady+" throw error=", err)
     return false  
    }
    fmt.Println("app mate ready: "+conf.CmdReady+" return code 0")
    return true
}

//Launch the app mate usin conffile cmd command
func executeApp() {
    applog.Log("execute: "+conf.Cmd);
    cmdList := strings.Split(conf.Cmd, " ")[:]
    mate.app = exec.Command(cmdList[0], cmdList[1:]...)
    //mate.App.Stdout = os.Stdout
    //mate.App.Stderr = os.Stderr
    mate.app.Stderr = applog.GetPipeStderrWriter()
    mate.app.Stdout = applog.GetPipeStdoutWriter()
    mate.appStarted = true
    mate.app.Run()
    mate.appStarted = false
}

//Stop app mate
func stopApp() {
    applog.Log("Send SIGTERM signal to app: "+conf.Name)
    mate.killTime=time.Now()
    mate.stopApp = false
    if mate.app != nil {
        //TODO: SIGTERM then wait and kill if app mate not stopped
        mate.app.Process.Kill()
    }
}

//Verify is app mate is launched
func isAppLaunched() bool {
    return mate.appStarted
    /*
    //Don't work correctly:
    if mate.app == nil {
        return false
    }
    if mate.app.ProcessState == nil  {
        return true
    }
    return mate.app.ProcessState.Exited()
    */
}

//Check dependencies, register if app mate is started and ready, stop app if a dependency is not ready
func checkForDependenciesAndReadyness() {
    launched := isAppLaunched()
    if launched && mate.appReady {
        consul.RegisterApp(mate.serviceId, conf.Name, conf.CheckPeriod)
    }
    mate.dependenciesReady = checkDependencies(launched)
    if mate.dependenciesReady {
        if !launched {
            mate.appReady = isAppReady()
        }
    } else {
        if launched {
            stopApp()
        }
        applog.Log("waiting for dependencies");
    }
}

//laucnh routine to check (register/dependencies) on regular basis and be able to change its period dynamically
func startPeriodicChecking() {
    go func() {
        for {
            checkForDependenciesAndReadyness()
            time.Sleep(time.Duration(mate.currentPeriod) * time.Second)
        }
    }()
}


