.PHONY: build push

# Build Docker image
build:
	docker buildx build --platform linux/amd64 \
		-t slav123/prom:latest \
		--load .

# Push to Docker Hub
push:
	docker push slav123/prom:latest