package main

import (
    "fmt"
    "os"
    "github.com/appcelerator/amp-pilot/core"
)

const version string = "1.1.1-4"

func main() {
    args := os.Args[1:]
    fmt.Println("amp-pilot started with argument: ", args)
    core.InitLoader()
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