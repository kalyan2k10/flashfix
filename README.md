# FlashFix Operations Guide

This document contains the consolidated commands for the FlashFix microservices across local development and GKE production environments.


---

## 💻 Local Development (Docker)
Use these for rapid iteration and testing on your local machine.

### Spin up Environment
```bash
# Build and start all services locally
docker-compose up --build

# Query User DB
docker exec -it $(docker ps -qf "name=user-db") mysql -u root -padmin -e "USE userdb; SELECT * FROM users;"

# Query Evaluation DB
docker exec -it $(docker ps -qf "name=evaluation-db") mysql -u root -padmin -e "USE evaluationdb; SELECT * FROM evaluations;"

### Inspect Kafka Topics
```bash
# Read all messages from the user-registrations topic
docker-compose exec kafka /opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server localhost:9092 --topic user-registrations --from-beginning
```

```
build and push
docker build --platform linux/amd64 -t us-central1-docker.pkg.dev/gifted-airway-477703-v1/flashfix/user-service:v1 ./user-service
docker push us-central1-docker.pkg.dev/gifted-airway-477703-v1/flashfix/user-service:v1

# Build and Push Evaluation Service
docker build --platform linux/amd64 -t us-central1-docker.pkg.dev/gifted-airway-477703-v1/flashfix/evaluation-service:v1 ./evaluation-service
docker push us-central1-docker.pkg.dev/gifted-airway-477703-v1/flashfix/evaluation-service:v1
kubectl rollout restart deployment user-service evaluation-service
```

### Deployment & Operations
```bash
# Apply YAML configuration
kubectl apply -f flashfix-deployment.yaml

# Update image version without re-applying YAML
kubectl set image deployment/user-service user-service=us-central1-docker.pkg.dev/gifted-airway-477703-v1/flashfix/user-service:v5

# Force a restart (fresh pull)
kubectl rollout restart deployment user-service

# Check update progress
kubectl rollout status deployment user-service

# Check running image versions
kubectl get pods -o custom-columns=NAME:.metadata.name,STATUS:.status.phase,IMAGE:.spec.containers[*].image

# -------------------------------------------
# ✅ Verification
# -------------------------------------------
# 1. Get LoadBalancer IP
kubectl get svc user-service

# 2. Test Workflow (Replace <IP>)
curl -X POST http://<IP>/user -d '{"username": "cloud-user"}'
curl "http://<IP>/status?username=cloud-user"

# Stream logs by service label
kubectl logs -l app=user-service -f
kubectl logs -l app=evaluation-service -f

# Check external IPs
kubectl get svc user-service

# Query GKE Evaluation DB
kubectl exec -it $(kubectl get pods -l app=evaluation-db -o name) -- mysql -u root -padmin -e "USE evaluationdb; SELECT * FROM evaluations;"

# Delete deployments (The Nuclear Option)
kubectl delete deployment user-service evaluation-service

kube commands : 
 kubectl exec $(kubectl get pod -l app=kafka -o jsonpath="{.items[0].metadata.name}") -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:9092 --delete --topic user-registrations\n
kubectl exec $(kubectl get pod -l app=kafka -o jsonpath="{.items[0].metadata.name}") -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:9092 --create --topic user-registrations --partitions 3 --replication-factor 1
kubectl exec $(kubectl get pod -l app=kafka -o jsonpath="{.items[0].metadata.name}") -- /opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server kafka:9092 --topic user-registrations --from-beginning
 kubectl exec $(kubectl get pod -l app=user-db -o jsonpath="{.items[0].metadata.name}") -- mysql -u root -padmin -D userdb -e "select * from users;"

 kubectl exec $(kubectl get pod -l app=kafka -o jsonpath="{.items[0].metadata.name}") -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:9092 --list


  kubectl exec $(kubectl get pod -l app=kafka -o jsonpath="{.items[0].metadata.name}") -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:9092 --delete --topic evaluation-results
  kubectl exec $(kubectl get pod -l app=kafka -o jsonpath="{.items[0].metadata.name}") -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:9092 --create --topic evaluation-results --partitions 3 --replication-factor 1


gcloud container clusters delete flashfix-cluster --region us-central1

gcloud container clusters create-auto flashfix-cluster \
    --region us-central1 \
    --project gifted-airway-477703-v1


gcloud container clusters get-credentials flashfix-cluster --region us-central1
gcloud container clusters resize flashfix-cluster --num-nodes=1 --region us-central1
# Apply your existing deployment file
kubectl apply -f flashfix-deployment.yaml

kubectl get svc user-service -w

# Scale all your deployments to zero replicas
kubectl scale deployment user-service --replicas=0
kubectl scale deployment evaluation-service --replicas=0
kubectl scale deployment user-db --replicas=0
kubectl scale deployment evaluation-db --replicas=0
kubectl scale deployment kafka --replicas=0
kubectl patch svc user-service -p '{"spec": {"type": "ClusterIP"}}'

# Scale everything back to 1 (or your desired count)
kubectl scale deployment user-service --replicas=1
kubectl scale deployment evaluation-service --replicas=1
kubectl scale deployment user-db --replicas=1
kubectl scale deployment evaluation-db --replicas=1
kubectl scale deployment kafka --replicas=1
kubectl patch svc user-service -p '{"spec": {"type": "LoadBalancer"}}'

kubectl delete -f flashfix-deployment.yaml

# 2. Delete the Cluster (Stops Management and Node costs)
gcloud container clusters delete flashfix-cluster --region us-central1 --quiet

# 3. Delete the Artifact Registry images (Optional, but saves storage pennies)
gcloud artifacts repositories delete flashfix --location=us-central1 --quiet

# Build and Push for the Cloud (AMD64)
docker build --platform linux/amd64 -t us-central1-docker.pkg.dev/gifted-airway-477703-v1/flashfix/user-service:v1 ./user-service --push
docker build --platform linux/amd64 -t us-central1-docker.pkg.dev/gifted-airway-477703-v1/flashfix/evaluation-service:v1 ./evaluation-service --push

# Force the pull
kubectl rollout restart deployment user-service
kubectl rollout restart deployment evaluation-service

kubectl get pods