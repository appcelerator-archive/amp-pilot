package main

import (
    "fmt"
    "os"
    "github.com/appcelerator/amp-pilot/core"
)

const version string = "1.1.1-5"

func main() {
    args := os.Args[1:]
    fmt.Println("amp-pilot started with argument: ", args)
    core.InitLoader()
    if (len(args)>0 && args[0] == "autotest") {
        err := core.AutoLoad(args)
        if (err!=nil) {
            fmt.Println("start error:",err)
            os.Exit(1)
        }
        os.Exit(0)
    }
    if (len(args)>0 && args[0] == "initBinaries") {
        core.InitBinaries(args)
    } else {
        loadingMode := os.Getenv("AMPPILOT_STANDALONE")
        if (loadingMode == "") {
            err := core.AutoLoad(args)
            if (err!=nil) {
                fmt.Println("start error:",err)
            }
        }
        core.Run(version)
    }
}