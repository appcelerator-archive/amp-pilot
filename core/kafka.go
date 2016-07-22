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
}

type logMessage struct {
    Timestamp time.Time         `json:"timestamp"`
    Time_id  string             `json:"time_id"`
    Service_uuid string         `json:"service_uuid"` //obsolet to be removed
    Service_id string           `json:"service_id"`
    Service_name string         `json:"service_name"`
    Stack_id string              `json:"task_id"`
    Stack_name string            `json:"task_name"`
    Message string              `json:"message"`
    IsError bool                `json:"is_error"`
    Container_shortid string    `json:"container_id"`
    Container_id string         `json:"container_id"`
    Node_id string              `json:"node_id"`
    Host_ip string              `json:"host_ip"`
}

var kafka Kafka


//init Kafka struct
func (self *Kafka) init() {
    fmt.Println("kafka init")
    self.kafkaReady = false
    self.messageBuffer = make([]logMessage, maxBuffer)
    self.messageBufferIndex = 0
    self.startPeriodicKafkaChecking()
}

//launch the periodical Kafka checking and create Producer if ready
func (self *Kafka) startPeriodicKafkaChecking() {
    fmt.Println("start Kafka checking")
    go func() {
        for {
            ready := consul.IsDependencyReady("kafka")
            if ready && !self.kafkaReady {
                applog.Log("Kafka is ready")
                if self.producer == nil {
                    config := samara.NewConfig()
                    client, err := samara.NewClient(strings.Split(conf.Kafka,","), config)
                    if (err != nil) {
                        applog.LogError("Error on kafka client: ", err)
                    } else {
                        prod, err := samara.NewAsyncProducerFromClient(client)
                        if err != nil {
                            applog.LogError("Error on kafka producer: ", err)
                        } else {
                            applog.Log("Kafka producer created on topic: amp-logs")
                            self.producer = prod
                        }
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
            if (mate.appStarted) {
                time.Sleep(time.Duration(30) * time.Second)
            } else {
                time.Sleep(time.Duration(3) * time.Second)
            }
        }
    }()
}

//create the message struct save it in buffer if Kafka not reday, send to Kafka if ready,
func (self *Kafka) sendMessage(message string, isError bool) {
    var mes logMessage
    mes.Service_uuid = conf.Name
    mes.Service_name = conf.Name
    mes.Service_id = loadInfo.serviceId
    mes.Host_ip = conf.RegisteredIp
    mes.Node_id = loadInfo.nodeId
    mes.Container_id = loadInfo.containerId
    mes.Container_shortid = loadInfo.containerShortId
    mes.Stack_id = loadInfo.stackId
    mes.Stack_name = loadInfo.stackName
    mes.Message = message
    mes.IsError = isError
    mes.Timestamp = time.Now()
    mes.Time_id = fmt.Sprintf("%v", time.Now().UnixNano())
    //fmt.Println("send message: ", mes)    
    if !self.kafkaReady {
        self.saveMessageOnBuffer(mes)
    } else {
        self.sendToKafka(mes)
    }
}

//Marshal the message and send it to Kafka
func (self *Kafka) sendToKafka(mes logMessage) {
    var data string
    if (len(mes.Message)>0 && mes.Message[0:1] == "{") {
        mesMap := make(map[string]string)
        var objmap map[string]*json.RawMessage
        err := json.Unmarshal([]byte(mes.Message), &objmap)
        if (err == nil) {
            for key, value := range objmap {
                if (key != "timestamp") {
                    data, err2 := value.MarshalJSON()
                    if (err2 == nil) {
                        mesMap[key] = strings.Trim(string(data),"\"")
                    }
                }
            }
            mesMap["message"] = mesMap["msg"]
            mesMap["service_uuid"] = mes.Service_uuid
            mesMap["service_id"] = mes.Service_id
            mesMap["service_name"] = mes.Service_name
            mesMap["stack_id"] = mes.Stack_id
            mesMap["stack_name"] = mes.Stack_name
            mesMap["node_id"] = mes.Node_id            
            mesMap["host_ip"] = mes.Host_ip
            mesMap["container_id"] = mes.Container_id
            mesMap["container_shortid"] = mes.Container_shortid
            if (mes.IsError) {
                mesMap["is_error"] = "true"
            } else {
                mesMap["is_error"] = "false"
            }
            mesMap["time_id"] = mes.Time_id
            dat, _ := json.Marshal(mesMap)
            data = string(dat)
            data = fmt.Sprintf("{\"timestamp\": %v, %s",mes.Timestamp.Unix(), data[1:])
        } else {
            dat, _ := json.Marshal(mes)  
            data = string(dat) 
        }
    } else {
        dat, _ := json.Marshal(mes)
        data = string(dat)
    }
    select {
        case self.producer.Input() <- &samara.ProducerMessage{Topic: conf.KafkaTopic, Key: nil, Value: samara.StringEncoder(data)}:
            //fmt.Println("sent")
            break
        case err := <-self.producer.Errors():
            fmt.Println("Error sending message to kafka: ", err)
            break
    }
}

//Save the message struct in Buffer
func (self *Kafka) saveMessageOnBuffer(msg logMessage) {
    if self.messageBufferIndex < maxBuffer {
        self.messageBuffer[self.messageBufferIndex] = msg
        self.messageBufferIndex++
    }
}

//Send all message in buffer to Kafka
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

//Close Kafka producer
func (self *Kafka) close() error {
    if self.producer == nil {
        return nil
    }
    if err := self.producer.Close(); err != nil {
        return err
    }
    return nil
}