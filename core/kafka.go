package core

import (
    "time"
    "fmt"
    "strings"
    "encoding/json"
    samara "github.com/Shopify/sarama"
)

const maxBuffer = 10000

type Kafka struct {
    producer samara.AsyncProducer
    kafkaReady bool 
    messageBuffer []logMessage
    messageBufferIndex int
    timeIndex int
}

type logMessage struct {
    Timestamp time.Time     `json:"timestamp"`
    Time_id time.Time       `json:"time_id"`
    Service_uuid string     `json:"service_uuid"`
    Message string          `json:"message"`
    IsError bool            `json:"is_error"`
    Container_id string     `json:"container_id"`
    Host_ip string          `json:"host_id"`
}

var kafka Kafka


func (self *Kafka) init() {
    fmt.Println("kafka init")
    self.kafkaReady = false
    self.messageBuffer = make([]logMessage, maxBuffer)
    self.messageBufferIndex = 0
    self.timeIndex = 0 
    self.startPeriodicKafkaChecking()
}

func (self *Kafka) startPeriodicKafkaChecking() {
    fmt.Println("start Kafka checking")
    go func() {
        for {
            ready := consul.IsDependencyReady("kafka")
            if ready && !self.kafkaReady {
                applog.Log("Kafka is ready")
                if self.producer == nil {
                    config := samara.NewConfig()
                    client, err := samara.NewClient(strings.Split("kafka:9092",","), config)
                    prod, err := samara.NewAsyncProducerFromClient(client)
                    if err != nil {
                        applog.LogError("Error on kafka producer: ", err)
                    } else {
                        applog.Log("Kafka producer created on topic: amp-logs")
                        self.producer = prod
                    }
                }
                if (self.producer != nil) {
                    self.kafkaReady = true
                    self.sendMessageBuffer()
                }
            } else if !ready && self.kafkaReady {
                self.kafkaReady = false
                applog.Log("Kafka is not ready yet")
            }
            time.Sleep(time.Duration(30) * time.Second)
        }
    }()
}

func (self *Kafka) sendMessage(message string, isError bool) {
    var mes logMessage
    mes.Service_uuid = conf.Name
    mes.Host_ip = conf.RegisteredIp
    mes.Container_id = conf.ContainerId
    mes.Message = message
    mes.IsError = isError
    mes.Timestamp = time.Now()
    self.timeIndex++
    mes.Time_id = time.Now()
    //fmt.Println("send message: ", mes)    
    if !self.kafkaReady {
        self.saveMessageOnBuffer(mes)
    } else {
        self.sendToKafka(mes)
    }
}

func (self *Kafka) sendToKafka(mes logMessage) {
    data, _ := json.Marshal(mes)
    select {
        case self.producer.Input() <- &samara.ProducerMessage{Topic: conf.KafkaTopic, Key: nil, Value: samara.StringEncoder(string(data))}:
            //fmt.Println("sent")
            break
        case err := <-self.producer.Errors():
            fmt.Println("Error sending message to kafka: ", err)
            break
    }
}

func (self *Kafka) saveMessageOnBuffer(msg logMessage) {
    if self.messageBufferIndex < maxBuffer {
        self.messageBuffer[self.messageBufferIndex] = msg
        self.messageBufferIndex++
    }
}

func (self *Kafka) sendMessageBuffer() {
    applog.Log("Write message buffer to Kafka (%v)", self.messageBufferIndex)
    if self.messageBufferIndex >0 {
        for  ii := 0; ii < self.messageBufferIndex; ii++ {
            self.sendToKafka(self.messageBuffer[ii])
        }
    }
    self.messageBufferIndex = 0
    applog.Log("Write message buffer done")
}

func (self *Kafka) close() error {
    if self.producer == nil {
        return nil
    }
    if err := self.producer.Close(); err != nil {
        return err
    }
    return nil
}