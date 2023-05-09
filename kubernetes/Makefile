minikube:
	bash scripts/minikube.sh

setup: minikube
	helm upgrade --install csi secrets-store-csi-driver/secrets-store-csi-driver -f helm/csi.yaml --namespace kube-system
	kubectl --namespace kube-system rollout status daemonset/csi-secrets-store-csi-driver
	helm upgrade --install vault hashicorp/vault -f helm/vault.yaml --namespace vault --create-namespace
	kubectl --namespace vault rollout status deployment/vault-agent-injector
	kubectl apply -f applications/payments-database.yaml
	kubectl rollout status deployment/payments-database
	kubectl apply -f applications/payments-processor.yaml
	kubectl rollout status deployment/payments-processor
	bash scripts/setup.sh

debug:
	docker build -t payments-app ../payments-app
	kubectl apply -f applications/payments-app.yaml
	kubectl rollout status deployment/payments-app-deployment
	kubectl port-forward svc/payments-app 8081:8081
	

app:
	kubectl apply -f applications/payments-app.yaml
	# kubectl apply -f applications/payments-app-term.yaml
	kubectl rollout status deployment/payments-app-deployment

csi:
	kubectl apply -f applications/payments-processor.csi.yaml
	kubectl delete pods -l app=payments-app

clean-cluster:
	kubectl delete --ignore-not-found -f applications/
	helm del vault --namespace vault
	helm del csi --namespace kube-system
	kill -9 $(shell cat pidfile)
	rm -rf pidfile

clean-minikube:
	minikube delete

clean: clean-cluster clean-minikube
