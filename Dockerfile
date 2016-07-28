FROM appcelerator/alpine:3.4


RUN apk update
RUN apk add bash go bzr git mercurial subversion openssh-client ca-certificates && mkdir -p /go/src /go/bin && chmod -R 777 /go
ENV GOPATH /go
ENV PATH /go/bin:$PATH
RUN mkdir -p /go/src/github.com/appcelerator/amp-pilot /go/bin
WORKDIR /go/src/github.com/appcelerator/amp-pilot
COPY ./ ./
RUN rm -rf ./vendor
RUN go get -u github.com/Masterminds/glide/...
RUN glide install
RUN go build -o /go/bin/amp-pilot.alpine              

COPY ./amp-pilot /go/bin/amp-pilot.amd64
COPY ./pilotLoader /go/bin/pilotLoader
RUN chmod +x /go/bin/*

ENTRYPOINT ["/go/bin/amp-pilot.alpine"]
CMD ["initBinaries"]
