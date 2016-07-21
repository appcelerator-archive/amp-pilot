###amp-pilot

amp-pilot is launched inside a container controling the start/stop of the real application (mate app) and registering it to consul
A set of applications using amp-pilot wait themself that their dependencies (other applications) are ready to start and so start in the right order avoiding any external scheduler.


### build

* in the git cloned directory, execute
    * git clone the project in $GOPATH/src (need to have go installed with the right GOPATH)
    * glide install the first time, glide update after (need to have glide installed)
    * go build

### Tags

* 1.1.0: last version including kafka feature

### Features

 * Start/restart app mate when all its dependencies are ready
 * Stop app mate if during execution a dependency become not ready (optionaly a dependency won't stop the app if it failed during app running)
 * Register app mate to consul just after startup (optionaly after executing a script to check if the app is really ready).
 * De-register when exit (SIGTERM recevied by amp-pilot is routed to app mate to make it stop)
 * heart-beat 'application ready' to consul on regular basis with variable speed concidering the application is starting or started and ready.
 * optionnally, send logs (amp-pilot and application mate) to kafka. Save logs in memory if  Kafka is not ready (with a limit)



### Configuration

amp-pilot is configurable using either conffile or environment variables or both.


If a connfile is used, the env. variable AMPPILOT_CONFFILE has to set with the full conffile path. The conffile is a json as this:

```
    {
        "Consul": "localhost:8500",
        "Name": "test",
        "Cmd": "./test.sh",
        "CmdReady": "./checkReady.sh",
        "RegisterPort" : "8080"
        "StartupCheckPeriod": 1,
        "CheckPeriod": 5,
        "ApplicationStop": false,
        "NetInterface": "eth0"
        "Dependencies" : [
            {
                "Name": "name1",
                "OnlyAtStartup": false
            },
            {
                "Name": "name2",
                "OnlyAtStartup": false                
            }
        ],
        "Kafka": "zookeeper:2181",
        "KafkaTopic": "amp-logs"
    }
```

Conffile is optional and can do not exist. In all cases, the following environment variables are prioritary for the configuration values:

 * CONSUL: consul addr, if this variable does'nt exist, the application is launched directly as amp-pilot do not exist
 * SERVICE_NAME: app mate name, mandatory
 * AMPPILOT_LAUNCH_CMD: app mate launch script, mandatory
 * AMPPILOT_READY_CMD: script to check if the app mate is ready, if does't exist then a started app is concidered ready.
 * AMPPILOT_NETINTERFACE: the network interface to get the local IP address which has been registered on Consul, default: eth0
 * AMPPILOT_REGISTEREDPORT: the port which has going to be register in Consul, default: 80
 * AMPPILOT_STARTUPCHECKPERIOD: period of dependencies checking and consul heartbeat at startup, default=1 second
 * AMPPILOT_CHECKPERIOD: period of dependencies checking and consul heartbeat after startup, default=10 seconds
 * AMPPILOT_STOPATMATESTOP: stop the container if the app mate stop by it-self, default=false
 * KAFKA: address and port of Kafka node, could be Zookeeper node (amp-pilot will find all the other Kafka nodes if exist), if doesn't exist, then no logs is sent to Kafka
 * KAFKA_TOPIC: the topic on which amp-pilot sends the logs, default: amp-logs
 * DEPENDENCIES: dependency names list, if not exist then app mate don't have dependency and starts immediatly
 * AMPPILOT_[Dependency Name]_ONLYATSTARTUP: to specify for the dependency [DependencyName] that it will be needed at startup, but should not stop the app mate if not ready during app mate running. [Dependency_name] should be uppercase and without '-' character as for instance: amp-log-worker -> AMPLOGWORKER



### install inside a container

to instal amp-pilot in a alpine container, add these in the Dockerfile:


ENV AMPPILOT=1.1.0
RUN curl -Lo /tmp/amp-pilot.alpine.tgz https://github.com/appcelerator/amp-pilot/releases/download/$AMPPILOT/amp-pilot.alpine-$AMPPILOT.tgz
RUN tar xvz -f /tmp/amp-pilot.alpine.tgz && mv ./amp-pilot.alpine /bin/


### load amp-pilot dynamically in a container

* install amp-pilot binary in /bin/amp-pilot with executable rights, it should have:
    * pilotLoader 
    * amp-pilot.alpine
    * amp-pilot.amd64  for the other linux 64 bits


create service this way:

docker service create --network [amp-swarm] --name [your service name] \
 -m type=bind,source=/bin/amp-pilot,target=/bin/amp-pilot \
 -m type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock \
 [your image] /bin/amp-pilot/pilotLoader


 the needed parameters are:
 * -m type=bind,source=/bin/amp-pilot,target=/bin/amp-pilot \
 * -m type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock \
 * /bin/amp-pilot/pilotLoader

 it's possible to add any AMPPILOT variable to modify the behavior, as for instance
 * -e DEPENDENCIES="service1, service2"

 the needed paramters are set automatically:
    * SERVICE_NAME
    * AMPPILOT_LAUNCH_CMD
    * CONSUL (default consul:8500)
    * KAFKA (default zookeeper:2181)
    * AMPPILOT_REGISTEREDPORT


#### future

Handle Docker 1.12 healthcheck 

