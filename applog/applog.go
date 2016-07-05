package applog

import (
    "fmt"
    "io"
    "os"
    "bufio"
    "config"
    "time"
)

const startupLogName string = "startup.log"
const currentRotateLogName string = "current.log"
const previousRotateLogName string = "previous.log"

//All Log related variables
type logData struct {
    stdout io.WriteCloser
    stderr io.WriteCloser
    pipeStdoutWriter *io.PipeWriter
    pipeStdoutReader *io.PipeReader
    pipeStderrWriter *io.PipeWriter
    pipeStderrReader *io.PipeReader  
    startupLogPath string
    previousRotateLogPath string  
    currentRotateLogPath string  
    startupFile *os.File
    rotateFile *os.File
    currentStartupSize int
    startupMaxSize int
    currentRotateSize int    
    rotateMaxSize int
}

var (
    appLog logData
    conf *config.Config = config.GetConfig()
)


//Send log message as app mate stdout
func Log(msg string, arg ...interface{}) {
    ret := fmt.Sprintf(msg, arg...) 
    appLog.pipeStdoutWriter.Write([]byte(ret+"\n"))
}

//Send log message as app mate stderr
func LogError(msg string, arg ...interface{}) {
    ret := fmt.Sprintf(msg, arg...) 
    appLog.pipeStderrWriter.Write([]byte(ret+"\n"))
}

//Close all log files
func CloseFiles() {
    if appLog.startupFile != nil {
        appLog.startupFile.Close()
    }
    if appLog.rotateFile != nil {
        appLog.rotateFile.Close()
    }
}

//Get the log stderr writer
func GetPipeStderrWriter() *io.PipeWriter {
    return appLog.pipeStderrWriter
}

//Get the log stdout writer
func GetPipeStdoutWriter() *io.PipeWriter {
    return appLog.pipeStdoutWriter
}

//Init log initial values
func InitLog() {
    //Init Writer and Reader to route all log messages (stdout by default)
    ro, wo := io.Pipe()
    appLog.pipeStdoutReader = ro
    appLog.pipeStdoutWriter = wo
    re, we := io.Pipe()
    appLog.pipeStderrReader = re
    appLog.pipeStderrWriter = we    
    appLog.stderr = NewStderrWriter()
    appLog.stdout = NewStdoutWriter() 

    //Create and open startup log file if set in config
    if conf.StartupLogSize > 0 {
        appLog.currentStartupSize = 0
        appLog.startupMaxSize = 1048576 * conf.StartupLogSize
        appLog.startupLogPath = conf.LogDirectory+startupLogName
        _, err  := os.Create(appLog.startupLogPath)
        if err != nil {
            fmt.Println("ERROR: Impossible to create log file: " + appLog.startupLogPath)
            os.Exit(1)
        }
        fs, err1 := os.OpenFile(appLog.startupLogPath, os.O_RDWR, 0644)
        if err1 != nil {
            fmt.Println("ERROR: Impossible to open log file: " + appLog.startupLogPath)
            os.Exit(1)
        }
        appLog.startupFile = fs
    }

    //Create and open rotate files if set in config
    if conf.RotateLogSize > 0 {
        appLog.currentRotateSize = 0
        appLog.rotateMaxSize = 1048576 * conf.RotateLogSize        
        appLog.previousRotateLogPath = conf.LogDirectory+previousRotateLogName
        appLog.currentRotateLogPath = conf.LogDirectory+currentRotateLogName
        _, err2  := os.Create(appLog.previousRotateLogPath)
        if err2 != nil {
            fmt.Println("ERROR: Impossible to create log file: " + appLog.previousRotateLogPath)
            os.Exit(1)
        }
        _, err3  := os.Create(appLog.currentRotateLogPath)
        if err3 != nil {
            fmt.Println("ERROR: Impossible to create log file: " + appLog.currentRotateLogPath)
            os.Exit(1)
        }
        if initRotateLog() != nil {
            os.Exit(1)
        }
    }
}

//Launch new routine to app mate read/write stdout
func NewStdoutWriter() io.WriteCloser {
    go func() {
        scanner := bufio.NewScanner(appLog.pipeStdoutReader)
        for scanner.Scan() {
            msg := scanner.Text()
            fmt.Println(msg)
            writeInLogFiles(msg)
        }
    }()
    return appLog.pipeStdoutWriter
}
//Launch new routine to app mate read/write stderr
func NewStderrWriter() io.WriteCloser {
    go func() {
        scanner := bufio.NewScanner(appLog.pipeStderrReader)
        for scanner.Scan() {
            msg := scanner.Text()            
            fmt.Println(msg)
            writeInLogFiles(msg)
        }
    }()
    return appLog.pipeStderrWriter
}

//Route log message to log files (startup or rotate concidering conffile)
func writeInLogFiles(msg string) {
    if appLog.startupFile != nil || appLog.rotateFile != nil {
        msg = fmt.Sprintf("%s %s\n", time.Now().Format(conf.LogFileFormat), msg)        
    }  
    if appLog.startupFile != nil {
        if appLog.currentStartupSize < appLog.startupMaxSize {
            appLog.currentStartupSize+=len(msg)
            appLog.startupFile.WriteString(msg)
        } else {
            appLog.startupFile.WriteString("Reatched maximum size of startup log file\n")
            appLog.startupFile.Close()
            appLog.startupFile = nil
        }
    }
    if appLog.rotateFile != nil {
        if appLog.currentRotateSize < appLog.rotateMaxSize {
            appLog.currentRotateSize+=len(msg)
            appLog.rotateFile.WriteString(msg)
        } else {
            appLog.rotateFile.Close()
            appLog.currentRotateSize = 0
            os.Remove(appLog.previousRotateLogPath)
            err :=  os.Rename(appLog.currentRotateLogPath, appLog.previousRotateLogPath)
            if err != nil {
                fmt.Println("Rotate log error: ", err)
            }
            initRotateLog()
            fmt.Println("log file rotated")
        }
    }
}

//Create and open current rotate log file
func initRotateLog() error {   
    if appLog.rotateMaxSize > 0 {
        _, err  := os.Create(appLog.currentRotateLogPath)
        if err != nil {
            fmt.Println("ERROR: Impossible to create log file: " + appLog.currentRotateLogPath)
            return err
        }
        fs, err := os.OpenFile(appLog.currentRotateLogPath, os.O_RDWR, 0644)
        if err != nil {
            fmt.Println("ERROR: Impossible to open log file: " + appLog.currentRotateLogPath)
            return err
        }        
        appLog.rotateFile = fs
    }
    return nil
}
