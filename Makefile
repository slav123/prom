.PHONY: build push

# Build Docker image
build:
	podman build --platform linux/amd64 \
		-t slav123/prom:latest \
		--load .

# Push to Docker Hub
push:
	podman push slav123/prom:latest