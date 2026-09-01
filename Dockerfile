# Build stage
FROM golang:1.25-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

# Install required system packages
RUN apk --no-cache add ca-certificates tzdata git

# Set working directory
WORKDIR /src

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies (this layer will be cached)
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build \
    -ldflags="-s -w -X main.minVersion=$(date -u +%Y%m%d.%H%M)" \
    -o /go/bin/prom

# Final stage
FROM scratch

# Copy certificates and timezone data from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /go/bin/prom /go/bin/prom

# Expose the application port
EXPOSE 9999

# Set the entrypoint
ENTRYPOINT ["/go/bin/prom"]
