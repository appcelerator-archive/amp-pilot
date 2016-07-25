FROM alpine:3.4


RUN apk update
RUN apk add bash go bzr git mercurial subversion openssh-client ca-certificates
RUN mkdir -p /go/src /go/bin && chmod -R 777 /go
ENV GOPATH /go
ENV PATH /go/bin:$PATH
RUN mkdir -p /go/src/github.com/appcelerator/amp-pilot
WORKDIR /go/src/github.com/appcelerator/amp-pilot
COPY ./ ./
RUN go get -u github.com/Masterminds/glide/...
RUN glide install
RUN go build                   
COPY ./test.sh /bin/test.sh

ENV CONSUL=consul:8500
ENV KAFKA=zookeeper:2181
ENV SERVICE_NAME=amp-test
ENV AMPPILOT_LAUNCH_CMD=/bin/test.sh
ENV AMPPILOT_STARTUPCHECKPERIOD=1
ENV AMPPILOT_CHECKPERIOD=10
ENV AMPPILOT_STOPATMATESTOP=false
ENV DEPENDENCIES=""

CMD ["/bin/amp-pilot"]
