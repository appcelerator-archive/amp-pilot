FROM ubuntu:16.04


RUN mkdir -p /usr/src/app
WORKDIR /usr/src/app

COPY ./amp-pilot /bin/amp-pilot
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
