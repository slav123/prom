build:
	docker build -t slav123/prom .

deploy:
	docker push slav123/prom
	kubectl apply -f nodeport.yaml
