package main

import (
    "fmt"
    "os"
    "github.com/appcelerator/amp-pilot/core"
)

const version string = "1.1.1-3"

func main() {
    args := os.Args[1:]
    core.InitLoader()
    loadingMode := os.Getenv("AMPPILOT_STANDALONE")
    if (loadingMode == "") {
        err := core.AutoLoad(args)
        if (err!=nil) {
            fmt.Println("start error:",err)
        }
    }
    core.Run(version)
}