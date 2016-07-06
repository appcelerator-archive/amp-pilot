###amp-pilot

amp-pilot is launched inside a container controling the start/stop of the real application (mate app) and registering it to consul if needed.
It avoids an external 


### Features

 * Start/restart app mate when all its dependencies are ready
 * Stop app mate if during execution a dependency become not ready (optionaly a dependency won't stop the app if it failed during app running)
 * Register app mate to consul when executed, optionaly when ready using a script and de-register when exit
 * heart-beat application redeay to consul on regular basis with variable speed concidering the application is starting or started and ready.


### Configuration

amp-pilot is configurable using either conffile or environment variables or both.


If a connfile is used, the env. variable AMPPILOT_CONFFILE has to set with the full conffile path. The conffile is a json as this:

```
    {
        "Consul": "localhost:8500",
        "Name": "test",
        "Cmd": "./test.sh",
        "CmdReady": "./checkReady.sh",
        "StartupCheckPeriod": 1,
        "CheckPeriod": 5,
        "StopAtMateStop": false,
        "LogDirectory": "./log",
        "StartupLogSize": 0,
        "RotateLogSize": 0,
        "Dependencies" : [
            {
                "Name": "name1",
                "OnlyAtStartup": false
            },
            {
                "Name": "name2",
                "OnlyAtStartup": false                
            }
        ]
    }
```

Conffile is optional and can do not exist. In all cases, if exist, the following environment variables are prioritary for the configuration values:

 * consul: consul addr default=localhost:8500
 * SERVICE_NAME: app mate name, mandatory
 * AMPPILOT_LAUNCH_CMD: app mate launch script, mandatory
 * AMPPILOT_READY_CMD: script to check if the app mate is ready, if does't exist then a started app is concidered ready.
 * AMPPILOT_NETINTERFACE: the network interface to get the local IP address which has been registered on Consul, default: docker0
 * AMPPILOT_REGISTEREDPORT: the port which has going to be register in Consul, default: 80
 * AMPPILOT_STARTUPCHECKPERIOD: period of dependencies check and consul register at startup, default=1
 * AMPPILOT_CHECKPERIOD: period of dependecies check and consul register after startup, default=10
 * AMPPILOT_STOPATMATESTOP: stop the container if the app mate stop by it-self, default=false
 * AMPPILOT_LOGDIRECTORY: log directory, default='.'
 * AMPPILOT_STARTUPLOGSIZE: startup log size (MB), if 0 then no startup logs, default=1
 * AMPPILOT_ROTATELOGSIZE: rotate log size (MB), if 0 then no rotate logs, default=1
 * AMPPILOT_LOGFILEFORMAT: log format for logs written in local files, based on time.Format (package time), default "2006-01-02 15:04:05.000"
 * DEPENDENCIES: dependency names list, if not exist then app mate don't have dependency
 * AMPPILOT_[Dependency Name]_ONLYATSTARTUP: to specify for the dependency [DependencyName] that it will be needed at startup, but should not stop the app mate if not ready during app mate running. [Dependency_name] should be uppercase and without '-' character as for instance: amp-log-worker -> AMPLOGWORKER

 ### logs files

 
 Optionaly amp-pilot creates logs files locally in $AMPPILOT_LOGDIRECTORY in case of global log chain failure

 if $AMPPILOT_STARTUPLOGSIZE > 0, then amp-pilot creates a startup.log file containing both amp-pilot and app mate logs until the size of the file reachs $AMPPILOT_STARTUPLOGSIZE MB and then amp-pilot stops to add logs in this file.


if $AMPPILOT_ROTATELOGSIZE > 0, then amp-pilot create a rotatelogs in current.log and previous.log files containing both amp-pilot and app mate log. It feed first current.log. When the size of current.log reachs $AMPPILOT_ROTATELOGSIZE MB then it moves current.log to previous.log and set current.log empty in order to to continue to store logs

Then with both current.log and previous.log you have up the $AMPPILOT_ROTATELOGSIZE *2 MB of the last logs.




### install

to instal amp-pilot in a alpine container, add these in the Dockerfile:


ENV AMPPILOT=1.0.0
RUN curl -Lo /tmp/amp-pilot.alpine.tgz https://github.com/appcelerator/amp-pilot/releases/download/$AMPPILOT/amp-pilot.alpine-$AMPPILOT.tgz
RUN tar xvz -f /tmp/amp-pilot.alpine.tgz && mv ./amp-pilot.alpine /bin/


#### future

Handle Docker 1.12 healthcheck 

