package core

import (
    "fmt"
    "io"
    "os"
    "bufio"
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

var applog logData


//Send log message as if was an app mate stdout
func (self *logData) Log(msg string, arg ...interface{}) {
    ret := fmt.Sprintf(msg, arg...) 
    self.pipeStdoutWriter.Write([]byte(ret+"\n"))
}

//Send log message as it was an app mate stderr
func (self *logData) LogError(msg string, arg ...interface{}) {
    ret := fmt.Sprintf(msg, arg...) 
    self.pipeStderrWriter.Write([]byte(ret+"\n"))
}

//Get the log stderr writer
func (self *logData) getPipeStderrWriter() *io.PipeWriter {
    return self.pipeStderrWriter
}

//Get the log stdout writer
func (self *logData) getPipeStdoutWriter() *io.PipeWriter {
    return self.pipeStdoutWriter
}

//Init log initial values
func (self *logData) init() {
    //Init Writer and Reader to route all log messages (stdout by default)
    ro, wo := io.Pipe()
    self.pipeStdoutReader = ro
    self.pipeStdoutWriter = wo
    re, we := io.Pipe()
    self.pipeStderrReader = re
    self.pipeStderrWriter = we    
    self.stderr = self.newStderrWriter()
    self.stdout = self.newStdoutWriter() 
    kafka.init()
}

//Launch new routine to app mate read/write stdout
func (self *logData) newStdoutWriter() io.WriteCloser {
    go func() {
        scanner := bufio.NewScanner(self.pipeStdoutReader)
        for scanner.Scan() {
            msg := scanner.Text()
            fmt.Println(msg)
            if conf.Kafka != "" {
                kafka.sendMessage(msg, false)
            }
        }
    }()
    return self.pipeStdoutWriter
}

//Launch new routine to app mate read/write stderr
func (self *logData) newStderrWriter() io.WriteCloser {
    go func() {
        scanner := bufio.NewScanner(self.pipeStderrReader)
        for scanner.Scan() {
            msg := scanner.Text()            
            fmt.Println(msg)
            if conf.Kafka != "" {
                kafka.sendMessage(msg, true)
            }
        }
    }()
    return self.pipeStderrWriter
}


