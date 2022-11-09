# EventRouter

This service serves as an active watcher of event resource in the kubernetes system, which takes those events and pushes them to a user specified _HTTP hook_ (called _Registration_).

this service will POST to the specified _endpoint_ a JSON containing the event info.

Here a sample event info:


```json
{
    "type":"Warning",
    "reason":"CannotCreateExternalResource",
    "deploymentId":"XXXXXXAAA1212121",
    "time":1660835529,
    "message":"cannot create EKS node group: ResourceInUseException: Cluster: test-1 is not in a valid state",
    "source":"managed/nodegroup",
    "involvedObject":{
       "apiVersion":"eks.aws.crossplane.io/v1alpha1",
       "kind":"NodeGroup",
       "name":"test-1-ng",
       "uid":"6365c158-8ee1-4d36-a33a-ba3cc0958ee0"
    },
    "metadata":{
       "creationTimestamp":"2022-10-26T17:25:12+02:00",
       "name":"test-1-ng.170c791ccd13d0cd",
       "namespace":"default",
       "uid":"6aa0a50b-1b5b-46e0-b5ec-a1118286f0c4"
    }
}
```

## How to install

```sh
$ helm repo add krateo https://charts.krateo.io
$ helm repo update krateo
$ helm install eventrouter krateo/eventrouter --namespace krateo-system --create-namespace
```

## Configuring a _Registration_

To configure your _hook_ in order to receive events info, create and apply a manifest with _Registration_ resource:

```sh
cat <<EOF | kubectl apply -f -
apiVersion: eventrouter.krateo.io/v1alpha1
kind: Registration
metadata:
  name: httpecho-registration
spec:
  # Your custom hook service name
  serviceName: HTTP Echo
  # Your custom hook service endpoint
  endpoint: http://127.0.0.1:9090/handle
EOF
```

In the above example the endpoint service at _ http://127.0.0.1:9090/handle_ will receive the eventhandler data.