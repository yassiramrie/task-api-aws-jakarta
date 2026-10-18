#!/usr/bin/env bash
set -euo pipefail

docker build -t task-api:dev .
if ! kind get clusters | grep -qx task-demo; then
  kind create cluster --name task-demo
fi
kind load docker-image task-api:dev --name task-demo
kubectl apply -k k8s/overlays/local
kubectl -n task-demo rollout status deployment/task-api --timeout=120s
echo 'Run: kubectl -n task-demo port-forward service/task-api 8080:80'
