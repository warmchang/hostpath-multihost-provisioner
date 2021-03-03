# Common environment variable config.
FROM golang:1.15 AS base
WORKDIR /app
COPY . /app

# Compile the manager.
FROM base AS manager-builder
RUN go build ./cmd/manager/main.go

# Compile the provisioner.
FROM base AS provisioner-builder
RUN go build ./cmd/provisioner/main.go

# Define the provisioner container.
FROM scratch AS hostpath-multihost-provisioner
COPY --from=provisioner-builder /app/main /hostpath-multihost-provisioner
CMD ["/hostpath-multihost-provisioner"]

# Define the manager container.
FROM scratch AS hostpath-multihost-manager
COPY --from=manager-builder /app/main /hostpath-multihost-manager
CMD ["/hostpath-multihost-manager"]
