PROVISIONER_IMAGE?=kubeboost/hostpath-multihost-provisioner
MANAGER_IMAGE?=kubeboost/hostpath-multihost-manager

PROVISIONER_TAG_GIT=$(PROVISIONER_IMAGE):$(shell git rev-parse HEAD)
PROVISIONER_TAG_LATEST=$(PROVISIONER_IMAGE):latest
MANAGER_TAG_GIT=$(MANAGER_IMAGE):$(shell git rev-parse HEAD)
MANAGER_TAG_LATEST=$(MANAGER_IMAGE):latest

all: provisioner_image manager_image provisioner_push manager_push

provisioner_image:
	docker build -t $(PROVISIONER_TAG_GIT) . -f Dockerfile.provisioner
	docker tag $(PROVISIONER_TAG_GIT) $(PROVISIONER_TAG_LATEST)

manager_image:
	docker build -t $(MANAGER_TAG_GIT) . -f Dockerfile.manager
	docker tag $(MANAGER_TAG_GIT) $(MANAGER_TAG_LATEST)

provisioner_push:
	docker push $(PROVISIONER_TAG_GIT)
	docker push $(PROVISIONER_TAG_LATEST)

manager_push:
	docker push $(MANAGER_TAG_GIT)
	docker push $(MANAGER_TAG_LATEST)
