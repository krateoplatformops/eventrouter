

kubectl apply -f manifests/ns.yaml
kubectl apply -f manifests/sa.yaml
kubectl apply -f manifests/rbac.yaml
kubectl apply -f manifests/rbac-bind.yaml
kubectl apply -f manifests/deployment.yaml

kubectl apply -f crds/
