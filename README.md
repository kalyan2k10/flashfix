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

## ☁️ GKE Production (Kubernetes)

### Build & Release
```bash
# Build & Push User Service
docker buildx build --platform linux/amd64 \
  -t us-central1-docker.pkg.dev/gifted-airway-477703-v1/flashfix/user-service:v5 ./user-service --push

# Build & Push Evaluation Service
docker buildx build --platform linux/amd64 \
  -t us-central1-docker.pkg.dev/gifted-airway-477703-v1/flashfix/evaluation-service:v1 ./evaluation-service --push

# Check Artifact Registry tags
gcloud artifacts docker tags list us-central1-docker.pkg.dev/gifted-airway-477703-v1/flashfix/user-service --sort-by=~timestamp
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
kubectl exec $(kubectl get pod -l app=kafka -o jsonpath="{.items[0].metadata.name}") -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:9092 --create --topic user-registrations --partitions 3 --replication-factor 1\n
kubectl exec $(kubectl get pod -l app=kafka -o jsonpath="{.items[0].metadata.name}") -- /opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server kafka:9092 --topic user-registrations --from-beginning\n
 kubectl exec $(kubectl get pod -l app=user-db -o jsonpath="{.items[0].metadata.name}") -- mysql -u root -padmin -D userdb -e "select * from users;"

 exec $(kubectl get pod -l app=kafka -o jsonpath="{.items[0].metadata.name}") -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:9092 --list


  kubectl exec $(kubectl get pod -l app=kafka -o jsonpath="{.items[0].metadata.name}") -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:9092 --delete --topic evaluation-results
  kubectl exec $(kubectl get pod -l app=kafka -o jsonpath="{.items[0].metadata.name}") -- /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:9092 --create --topic evaluation-results --partitions 3 --replication-factor 1