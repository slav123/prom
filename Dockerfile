FROM golang:1.17-alpine as build-env
RUN apk --update add --no-cache ca-certificates openssl git tzdata
# All these steps will be cached
RUN mkdir /prom
WORKDIR /prom
COPY go.mod .
# <- COPY go.mod and go.sum files to the workspace
COPY go.sum .

# Get dependancies - will also be cached if we won't change mod/sum
RUN go mod download
# COPY the source code as the last step
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-s -w -X main.minVersion=`date -u +%Y%m%d.%H%M`" -o /go/bin/prom
FROM scratch
# <- Second step to build minimal image
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-env /go/bin/prom /go/bin/prom
EXPOSE 9999
ENTRYPOINT ["/go/bin/prom"]
