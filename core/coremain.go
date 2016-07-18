package core

import (
    "runtime"
    "fmt"
    "time"
    "os"    
)

//launch main loop
func Run(version string) {
    applog.init()
    conf.load()
    if conf.Consul == "" {
        fmt.Println("Consul address is not defined: application is launched drrectly. amp-pilot is desactivated")
        mate.executeApp(false)
        os.Exit(0)    
    }
    mate.init(version)
    mate.trapSignal()
    runtime.GOMAXPROCS(4)
    applog.Log("waiting for dependencies...");
    mate.startPeriodicChecking()
    for {
        if mate.dependenciesReady && mate.appReady {
            mate.currentPeriod = conf.CheckPeriod
            mate.executeApp(true)
            mate.dependenciesReady = mate.checkDependencies(false)
            mate.appReady = false
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