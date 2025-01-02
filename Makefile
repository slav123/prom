build-amd:
	docker build -t slav123/prom .

deploy:
	docker push slav123/prom
	kubectl apply -f nodeport.yaml

build:
	docker buildx build --platform linux/amd64 -t slav123/prom --load .
	docker push slav123/prom