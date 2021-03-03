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
FROM scratch AS hostpath-provisioner
COPY --from=provisioner-builder /app/main /hostpath-provisioner
CMD ["/hostpath-provisioner"]

# Define the manager container.
FROM scratch AS distributed-hostpath-storage-manager
COPY --from=manager-builder /app/main /distributed-hostpath-storage-manager
CMD ["/distributed-hostpath-storage-manager"]
