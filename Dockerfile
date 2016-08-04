FROM appcelerator/alpine:3.4

ENV GOPATH /go
ENV PATH /go/bin:$PATH
RUN mkdir -p /go/src /go/bin && chmod -R 777 /go
RUN mkdir -p /go/src/github.com/appcelerator/amp-pilot /go/bin
WORKDIR /go/src/github.com/appcelerator/amp-pilot
COPY ./ ./
RUN apk update && \
    apk --virtual build-deps add go git && \
    go get -u github.com/Masterminds/glide/... && \
    glide install && \
    go build -o /go/bin/amp-pilot.alpine && \
    apk del build-deps && cd / && rm -rf $GOPATH/src /var/cache/apk/*
    
#linux:amd64 can't be cross builded on alpine, should be build externally and copied
#ENV GOOS=linux
#ENV GOARCH=amd64
#RUN go build -o /go/bin/amp-pilot.amd64
COPY ./amp-pilot /go/bin/amp-pilot.amd64

COPY ./pilotLoader /go/bin/pilotLoader
RUN chmod +x /go/bin/*

HEALTHCHECK --interval=3s --timeout=10s --retries=6 CMD pidof amp-pilot.alpine

ENTRYPOINT ["/go/bin/amp-pilot.alpine"]
CMD ["initBinaries"]
