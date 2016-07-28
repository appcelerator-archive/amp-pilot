package core

import (
    "runtime"
    "fmt"
    "time"
    "os"    
)

var ampPilotVersion string

//launch main loop
func Run(version string) {
    ampPilotVersion = version
    applog.init()
    conf.load()
    if conf.Consul == "" {
        fmt.Println("Consul address is not defined: application is launched drrectly. amp-pilot is desactivated")
        mate.executeApp(false)
        os.Exit(0)    
    }
    mate.init()
    mate.trapSignal()
    runtime.GOMAXPROCS(6)
    applog.Log("waiting for dependencies...");
    mate.startPeriodicChecking()
    for {
        if mate.dependenciesReady {
            mate.currentPeriod = conf.CheckPeriod
            mate.executeApp(true)
            mate.dependenciesReady = mate.checkDependencies(false)
            mate.currentPeriod = conf.StartupCheckPeriod
            if mate.stopApp {
                consul.DeregisterApp(mate.serviceId)
                applog.Log("App mate has stopped")
                os.Exit(0)
            }
            mate.stopApp = conf.ApplicationStop
        } 
        time.Sleep(1 * time.Second)
    }
}