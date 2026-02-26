# ==========================================
# 1. BUILD AND PUSH (THE PIPELINE)
# ==========================================
# Always use linux/amd64 for GKE compatibility
docker buildx build --platform linux/amd64 \
  -t us-central1-docker.pkg.dev/gifted-airway-477703-v1/flashfix/user-service:v5 \
  . --push

# ==========================================
# 2. DEPLOYMENT AND UPDATES
# ==========================================
# Apply the YAML configuration
kubectl apply -f flashfix-deployment.yaml

# If you only want to update the image version without re-applying YAML
kubectl set image deployment/user-service user-service=us-central1-docker.pkg.dev/gifted-airway-477703-v1/flashfix/user-service:v5

# Force a restart (useful to clear buffers or force a fresh pull)
kubectl rollout restart deployment user-service

# Check update progress
kubectl rollout status deployment user-service

# ==========================================
# 3. MONITORING & LOGS (THE TRUTH)
# ==========================================
# See what image version is actually running in each pod
kubectl get pods -o custom-columns=NAME:.metadata.name,STATUS:.status.phase,IMAGE:.spec.containers[*].image

# Stream logs for User Service (using labels)
kubectl logs -l app=user-service -f

# Stream logs for Evaluation Service
kubectl logs -l app=evaluation-service -f

# Check external IP and port mapping
kubectl get svc user-service

# ==========================================
# 4. ARTIFACT REGISTRY & CLEANUP
# ==========================================
# Check if your tag actually exists in the cloud
gcloud artifacts docker tags list us-central1-docker.pkg.dev/gifted-airway-477703-v1/flashfix/user-service --sort-by=~timestamp

# "The Nuclear Option" - Clean up old deployments to avoid label conflicts
kubectl delete deployment user-service evaluation-service

docker-compose up --build