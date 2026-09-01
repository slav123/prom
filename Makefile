.PHONY: build push login

# Build Docker image
build:
	podman build --platform linux/amd64 \
		-t slav123/prom:latest \
		--load .

# Login to Docker Hub
login:
	podman login -u $(DOCKER_USERNAME) -p $(DOCKER_PASSWORD) registry-1.docker.io

# Push to Docker Hub
push:
	podman push slav123/prom:latest