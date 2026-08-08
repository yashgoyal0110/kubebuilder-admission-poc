#!/usr/bin/env bash
# Stands the whole thing up on a throwaway kind cluster and proves admission is live.
#
# envtest already covers the logic. This covers the bit envtest cannot: that the image builds, the
# manifests apply in the right order, cert-manager issues a certificate the Deployment can mount,
# and the API server can actually reach the webhook through the Service.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

CLUSTER=${CLUSTER:-admission-poc}
IMAGE=${IMAGE:-controller:demo}

for tool in kind kubectl docker; do
  command -v "${tool}" >/dev/null || { echo "${tool} is required"; exit 1; }
done

echo "==> creating cluster ${CLUSTER}"
kind get clusters 2>/dev/null | grep -qx "${CLUSTER}" || kind create cluster --name "${CLUSTER}"

echo "==> installing cert-manager"
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.16.2/cert-manager.yaml
kubectl -n cert-manager wait --for=condition=Available --timeout=300s \
  deployment/cert-manager deployment/cert-manager-webhook deployment/cert-manager-cainjector

echo "==> building and loading ${IMAGE}"
docker build -t "${IMAGE}" .
kind load docker-image "${IMAGE}" --name "${CLUSTER}"

echo "==> applying manifests"
kubectl apply -k config/default
kubectl -n system set image deployment/controller-manager "manager=${IMAGE}"
kubectl -n system rollout status deployment/controller-manager --timeout=180s

echo
echo "==> a device with no protocol should come back defaulted to mqtt"
kubectl apply -f - <<'YAML'
apiVersion: devices.yashgoyal.dev/v1alpha1
kind: Device
metadata:
  name: demo-thermostat
spec:
  model: thermostat-v1
  endpoint: tcp://10.0.0.1:1883
  nodeName: edge-node-1
YAML
kubectl get device demo-thermostat -o jsonpath='{.spec.protocol}{"\n"}'

echo "==> an opcua device pointed at an mqtt endpoint should be refused"
if kubectl apply -f - <<'YAML' 2>/dev/null
apiVersion: devices.yashgoyal.dev/v1alpha1
kind: Device
metadata:
  name: demo-wrong-scheme
spec:
  model: sensor-v1
  protocol: opcua
  endpoint: tcp://10.0.0.2:1883
  nodeName: edge-node-1
YAML
then
  echo "FAIL: the webhook accepted a mismatched endpoint"
  exit 1
fi
echo "refused, as expected"

echo "==> changing the model should be refused"
if kubectl patch device demo-thermostat --type=merge -p '{"spec":{"model":"thermostat-v2"}}' 2>/dev/null; then
  echo "FAIL: the webhook accepted a model change"
  exit 1
fi
echo "refused, as expected"

echo
echo "done. tear down with: kind delete cluster --name ${CLUSTER}"
