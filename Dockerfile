FROM golang:1.14

# Set default shell to /bin/bash
SHELL ["/bin/bash", "-cu"]


ADD ca-certificates.crt /etc/ssl/certs/
RUN mkdir /tmp/forwardhook
ADD forwardhook.go /tmp/forwardhook

RUN go get github.com/Jeffail/gabs
RUN cd /tmp/forwardhook && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o forwardhook .
RUN mv /tmp/forwardhook/forwardhook /
CMD ["/forwardhook"]