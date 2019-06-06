# notes for demo structure

# Lab Notes for first experience getting started with knative
Following this hands on walk through https://medium.com/google-cloud/hands-on-knative-part-1-f2d5ce89944e

Decided to use their code exactly at first, then going to upgrade to a decide roll app that can create d*N dice to back any type of gambling you could possibly want :P 

Would be powerful to do a demo with Java, Python, Golang, and Php

Deciding to do this with an http demo then going to refactor into gRPC
Created a hello-world.go file with the simiplest web server
```golang
package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", HelloServer)
	http.ListenAndServe(":8080", nil)
}

// HelloServer creates a server that will response with "Hello, world!".
func HelloServer(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, world!")
}
```


Created a dockerfile
```dockerfile
# Pre build step build binary
FROM golang:alpine AS build-env
ADD /src /src
RUN cd /src && go build -o goapp

# Mount binary into alpine container
FROM alpine
WORKDIR /app
COPY --from=build-env /src/goapp /app/
ENTRYPOINT ./goapp
```


Annoyingly there is no good up to date tutorial for knative install on GCP. This might be because Google Cloud run doesn't work with it. 

Trying to follow knative install steps and there's problems with the fact that it expects it to be setup in a single zone and not a region. This seems reasonable. Tryign to decide if I should try to figure out how it'll work in a regional context... Probably better to stick to a single zone in a region

On MC's basic project setup defined for our platform I ran the command and got some warnings
```bash
❯ gcloud beta container clusters create $CLUSTER_NAME \
  --addons=HorizontalPodAutoscaling,HttpLoadBalancing,Istio \
  --machine-type=n1-standard-4 \
  --cluster-version=latest --zone=$CLUSTER_ZONE \
  --enable-stackdriver-kubernetes --enable-ip-alias \
  --enable-autoscaling --min-nodes=1 --max-nodes=10 \
  --enable-autorepair \
  --scopes cloud-platform
WARNING: Starting in 1.12, new clusters will have basic authentication disabled by default. Basic authentication can be enabled (or disabled) manually using the `--[no-]enable-basic-auth` flag.
WARNING: Starting in 1.12, new clusters will not have a client certificate issued. You can manually enable (or disable) the issuance of the client certificate using the `--[no-]issue-client-certificate` flag.
WARNING: Newly created clusters and node-pools will have node auto-upgrade enabled by default. This can be disabled using the `--no-enable-autoupgrade` flag.
WARNING: Starting in 1.12, default node pools in new clusters will have their legacy Compute Engine instance metadata endpoints disabled by default. To create a cluster with legacy instance metadata endpoints disabled in the default node pool, run `clusters create` with the flag `--metadata disable-legacy-endpoints=true`.
WARNING: The Pod address range limits the maximum size of the cluster. Please refer to https://cloud.google.com/kubernetes-engine/docs/how-to/flexible-pod-cidr to learn how to optimize IP address allocation.
This will enable the autorepair feature for nodes. Please see https://cloud.google.com/kubernetes-engine/docs/node-auto-repair for more information on node autorepairs.
```

This failed due to having too many api-aliased clusters up at once int he same zone. This makes sense due to limits on vpc CIDR blocks

Next though it wants me to give the cluster admin my owner GCP permissions and this seems terrifying to me, there must be another way? This will eventually be assigned to a service account instead though with the right permissions assumedly and not my personal account. this is apparenlty needed for RBAC

Grant cluster-admin permissions to the current user:

   kubectl create clusterrolebinding cluster-admin-binding \
     --clusterrole=cluster-admin \
     --user=$(gcloud config get-value core/account)


Attempting to install CRDS

```bash
kubectl apply --selector knative.dev/crd-install=true \
   --filename https://github.com/knative/serving/releases/download/v0.6.0/serving.yaml \
   --filename https://github.com/knative/build/releases/download/v0.6.0/build.yaml \
   --filename https://github.com/knative/eventing/releases/download/v0.6.0/release.yaml \
   --filename https://github.com/knative/eventing-sources/releases/download/v0.6.0/eventing-sources.yaml \
   --filename https://github.com/knative/serving/releases/download/v0.6.0/monitoring.yaml \
   --filename https://raw.githubusercontent.com/knative/serving/v0.6.0/third_party/config/build/clusterrole.yaml```
```
https://github.com/knative/eventing-sources/releases/download/v0.6.0/eventing-sources.yaml
https://github.com/knative/eventing/releases/download/v0.6.0/release.yaml
https://github.com/knative/eventing/releases/download/v0.6.0/release.yaml



Created with the default knative service configuration:
```yaml
        - image: gcr.io/knative-samples/helloworld-go # The URL to the image of the app
          env:
            - name: TARGET # The environment variable printed out by the sample app
              value: "Go Sample v1"
```

# Background
I attempted to use our cluster creation at first to test autoscaling. So to attempt to test autoscaling and eliminating any misconfiguration on our part I'm going to try starting from a gke cluster with knative on it to figure out if I can test out the load on grpc

created a cluster
```
gcloud container clusters create $CLUSTER_NAME \
     --zone=$CLUSTER_ZONE \
     --cluster-version=latest \
     --machine-type=n1-standard-4 \
     --enable-autoscaling --min-nodes=1 --max-nodes=10 \
     --enable-autorepair \
     --scopes=service-control,service-management,compute-rw,storage-ro,cloud-platform,logging-write,monitoring-write,pubsub,datastore \
     --num-nodes=3
```

output: 
```
WARNING: In June 2019, node auto-upgrade will be enabled by default for newly created clusters and node pools. To disable it, use the `--no-enable-autoupgrade` flag.
WARNING: Starting in 1.12, new clusters will have basic authentication disabled by default. Basic authentication can be enabled (or disabled) manually using the `--[no-]enable-basic-auth` flag.
WARNING: Starting in 1.12, new clusters will not have a client certificate issued. You can manually enable (or disable) the issuance of the client certificate using the `--[no-]issue-client-certificate` flag.
WARNING: Currently VPC-native is not the default mode during cluster creation. In the future, this will become the default mode and can be disabled using `--no-enable-ip-alias` flag. Use `--[no-]enable-ip-alias` flag to suppress this warning.
WARNING: Starting in 1.12, default node pools in new clusters will have their legacy Compute Engine instance metadata endpoints disabled by default. To create a cluster with legacy instance metadata endpoints disabled in the default node pool, run `clusters create` with the flag `--metadata disable-legacy-endpoints=true`.
WARNING: Your Pod address range (`--cluster-ipv4-cidr`) can accommodate at most 1008 node(s). 
This will enable the autorepair feature for nodes. Please see https://cloud.google.com/kubernetes-engine/docs/node-auto-repair for more information on node autorepairs.
Creating cluster knative-clean-test in us-west1-b... Cluster is being health-checked (master is healthy)...done.                                                                                                                                 
Created [https://container.googleapis.com/v1/projects/kwyn-testbed/zones/us-west1-b/clusters/knative-clean-test].
To inspect the contents of your cluster, go to: https://console.cloud.google.com/kubernetes/workload_/gcloud/us-west1-b/knative-clean-test?project=kwyn-testbed
kubeconfig entry generated for knative-clean-test.
NAME                LOCATION    MASTER_VERSION  MASTER_IP       MACHINE_TYPE   NODE_VERSION  NUM_NODES  STATUS
knative-clean-test  us-west1-b  1.13.6-gke.5    35.203.132.116  n1-standard-4  1.13.6-gke.5  3          RUNNING
```

Granted myself permissions to the cluster
```
   kubectl create clusterrolebinding cluster-admin-binding \
   --clusterrole=cluster-admin \
   --user=$(gcloud config get-value core/account)
```
output:
```
clusterrolebinding.rbac.authorization.k8s.io "cluster-admin-binding" created
```

Installed istio
```
kubectl apply --filename https://github.com/knative/serving/releases/download/v0.4.0/istio-crds.yaml && \
kubectl apply --filename https://github.com/knative/serving/releases/download/v0.4.0/istio.yaml
```

```
customresourcedefinition.apiextensions.k8s.io "virtualservices.networking.istio.io" created
customresourcedefinition.apiextensions.k8s.io "destinationrules.networking.istio.io" created
customresourcedefinition.apiextensions.k8s.io "serviceentries.networking.istio.io" created
customresourcedefinition.apiextensions.k8s.io "gateways.networking.istio.io" created
customresourcedefinition.apiextensions.k8s.io "envoyfilters.networking.istio.io" created
customresourcedefinition.apiextensions.k8s.io "policies.authentication.istio.io" created
customresourcedefinition.apiextensions.k8s.io "meshpolicies.authentication.istio.io" created
customresourcedefinition.apiextensions.k8s.io "httpapispecbindings.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "httpapispecs.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "quotaspecbindings.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "quotaspecs.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "rules.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "attributemanifests.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "bypasses.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "circonuses.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "deniers.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "fluentds.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "kubernetesenvs.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "listcheckers.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "memquotas.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "noops.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "opas.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "prometheuses.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "rbacs.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "redisquotas.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "servicecontrols.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "signalfxs.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "solarwindses.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "stackdrivers.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "statsds.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "stdios.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "apikeys.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "authorizations.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "checknothings.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "kuberneteses.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "listentries.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "logentries.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "edges.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "metrics.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "quotas.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "reportnothings.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "servicecontrolreports.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "tracespans.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "rbacconfigs.rbac.istio.io" created
customresourcedefinition.apiextensions.k8s.io "serviceroles.rbac.istio.io" created
customresourcedefinition.apiextensions.k8s.io "servicerolebindings.rbac.istio.io" created
customresourcedefinition.apiextensions.k8s.io "adapters.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "instances.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "templates.config.istio.io" created
customresourcedefinition.apiextensions.k8s.io "handlers.config.istio.io" created
namespace "istio-system" created
configmap "istio-galley-configuration" created
configmap "istio-statsd-prom-bridge" created
configmap "istio-security-custom-resources" created
configmap "istio" created
configmap "istio-sidecar-injector" created
serviceaccount "istio-galley-service-account" created
serviceaccount "istio-egressgateway-service-account" created
serviceaccount "istio-ingressgateway-service-account" created
serviceaccount "istio-mixer-service-account" created
serviceaccount "istio-pilot-service-account" created
serviceaccount "istio-cleanup-secrets-service-account" created
clusterrole.rbac.authorization.k8s.io "istio-cleanup-secrets-istio-system" created
clusterrolebinding.rbac.authorization.k8s.io "istio-cleanup-secrets-istio-system" created
job.batch "istio-cleanup-secrets" created
serviceaccount "istio-security-post-install-account" created
clusterrole.rbac.authorization.k8s.io "istio-security-post-install-istio-system" created
clusterrolebinding.rbac.authorization.k8s.io "istio-security-post-install-role-binding-istio-system" created
job.batch "istio-security-post-install" created
serviceaccount "istio-citadel-service-account" created
serviceaccount "istio-sidecar-injector-service-account" created
customresourcedefinition.apiextensions.k8s.io "virtualservices.networking.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "destinationrules.networking.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "serviceentries.networking.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "gateways.networking.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "envoyfilters.networking.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "httpapispecbindings.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "httpapispecs.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "quotaspecbindings.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "quotaspecs.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "rules.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "attributemanifests.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "bypasses.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "circonuses.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "deniers.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "fluentds.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "kubernetesenvs.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "listcheckers.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "memquotas.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "noops.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "opas.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "prometheuses.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "rbacs.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "redisquotas.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "servicecontrols.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "signalfxs.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "solarwindses.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "stackdrivers.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "statsds.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "stdios.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "apikeys.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "authorizations.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "checknothings.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "kuberneteses.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "listentries.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "logentries.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "edges.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "metrics.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "quotas.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "reportnothings.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "servicecontrolreports.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "tracespans.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "rbacconfigs.rbac.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "serviceroles.rbac.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "servicerolebindings.rbac.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "adapters.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "instances.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "templates.config.istio.io" configured
customresourcedefinition.apiextensions.k8s.io "handlers.config.istio.io" configured
clusterrole.rbac.authorization.k8s.io "istio-galley-istio-system" created
clusterrole.rbac.authorization.k8s.io "istio-egressgateway-istio-system" created
clusterrole.rbac.authorization.k8s.io "istio-ingressgateway-istio-system" created
clusterrole.rbac.authorization.k8s.io "istio-mixer-istio-system" created
clusterrole.rbac.authorization.k8s.io "istio-pilot-istio-system" created
clusterrole.rbac.authorization.k8s.io "istio-citadel-istio-system" created
clusterrole.rbac.authorization.k8s.io "istio-sidecar-injector-istio-system" created
clusterrolebinding.rbac.authorization.k8s.io "istio-galley-admin-role-binding-istio-system" created
clusterrolebinding.rbac.authorization.k8s.io "istio-egressgateway-istio-system" created
clusterrolebinding.rbac.authorization.k8s.io "istio-ingressgateway-istio-system" created
clusterrolebinding.rbac.authorization.k8s.io "istio-mixer-admin-role-binding-istio-system" created
clusterrolebinding.rbac.authorization.k8s.io "istio-pilot-istio-system" created
clusterrolebinding.rbac.authorization.k8s.io "istio-citadel-istio-system" created
clusterrolebinding.rbac.authorization.k8s.io "istio-sidecar-injector-admin-role-binding-istio-system" created
service "istio-galley" created
service "istio-egressgateway" created
service "istio-ingressgateway" created
service "istio-policy" created
service "istio-telemetry" created
service "istio-pilot" created
service "istio-citadel" created
service "istio-sidecar-injector" created
deployment.extensions "istio-galley" created
deployment.extensions "istio-egressgateway" created
deployment.extensions "istio-ingressgateway" created
deployment.extensions "istio-policy" created
deployment.extensions "istio-telemetry" created
deployment.extensions "istio-pilot" created
deployment.extensions "istio-citadel" created
deployment.extensions "istio-sidecar-injector" created
gateway.networking.istio.io "istio-autogenerated-k8s-ingress" created
horizontalpodautoscaler.autoscaling "istio-egressgateway" created
horizontalpodautoscaler.autoscaling "istio-ingressgateway" created
horizontalpodautoscaler.autoscaling "istio-policy" created
horizontalpodautoscaler.autoscaling "istio-telemetry" created
horizontalpodautoscaler.autoscaling "istio-pilot" created
mutatingwebhookconfiguration.admissionregistration.k8s.io "istio-sidecar-injector" created
attributemanifest.config.istio.io "istioproxy" created
attributemanifest.config.istio.io "kubernetes" created
stdio.config.istio.io "handler" created
logentry.config.istio.io "accesslog" created
logentry.config.istio.io "tcpaccesslog" created
rule.config.istio.io "stdio" created
rule.config.istio.io "stdiotcp" created
metric.config.istio.io "requestcount" created
metric.config.istio.io "requestduration" created
metric.config.istio.io "requestsize" created
metric.config.istio.io "responsesize" created
metric.config.istio.io "tcpbytesent" created
metric.config.istio.io "tcpbytereceived" created
prometheus.config.istio.io "handler" created
rule.config.istio.io "promhttp" created
rule.config.istio.io "promtcp" created
kubernetesenv.config.istio.io "handler" created
rule.config.istio.io "kubeattrgenrulerule" created
rule.config.istio.io "tcpkubeattrgenrulerule" created
kubernetes.config.istio.io "attributes" created
destinationrule.networking.istio.io "istio-policy" created
destinationrule.networking.istio.io "istio-telemetry" created
serviceaccount "cluster-local-gateway-service-account" created
clusterrole.rbac.authorization.k8s.io "cluster-local-gateway-istio-system" created
clusterrolebinding.rbac.authorization.k8s.io "cluster-local-gateway-istio-system" created
service "cluster-local-gateway" created
deployment.extensions "cluster-local-gateway" created
horizontalpodautoscaler.autoscaling "cluster-local-gateway" created
```
```
$ kubectl get pods --namespace istio-system
NAME                                      READY     STATUS      RESTARTS   AGE
cluster-local-gateway-5b57c887bc-szld9    1/1       Running     0          1m
istio-citadel-74b8694796-qlj6k            1/1       Running     0          2m
istio-cleanup-secrets-cd6cz               0/1       Completed   0          2m
istio-egressgateway-78f4ff7cd7-ldl8j      1/1       Running     0          2m
istio-galley-864c774fcf-txvvk             1/1       Running     0          2m
istio-ingressgateway-58bd55686-blc6t      1/1       Running     0          2m
istio-pilot-7fb97c7484-2f7ll              2/2       Running     0          1m
istio-pilot-7fb97c7484-h7rkf              2/2       Running     0          1m
istio-pilot-7fb97c7484-sb4xs              2/2       Running     0          2m
istio-policy-5c46d6f859-8lrwx             2/2       Running     0          2m
istio-security-post-install-r5flz         0/1       Completed   0          2m
istio-sidecar-injector-67bd7b75b7-bvbv8   1/1       Running     0          2m
istio-telemetry-7c44bc885f-gzg89          2/2       Running     0          2m
```

Install knative components into the cluster:
```
❯  kubectl apply --filename https://github.com/knative/serving/releases/download/v0.4.0/serving.yaml \
   --filename https://github.com/knative/build/releases/download/v0.4.0/build.yaml \
   --filename https://github.com/knative/eventing/releases/download/v0.4.0/release.yaml \
   --filename https://github.com/knative/eventing-sources/releases/download/v0.4.0/release.yaml \
   --filename https://github.com/knative/serving/releases/download/v0.4.0/monitoring.yaml \
   --filename https://raw.githubusercontent.com/knative/serving/v0.4.0/third_party/config/build/clusterrole.yaml
namespace "knative-serving" created
clusterrole.rbac.authorization.k8s.io "knative-serving-admin" created
clusterrole.rbac.authorization.k8s.io "knative-serving-core" created
serviceaccount "controller" created
clusterrolebinding.rbac.authorization.k8s.io "knative-serving-controller-admin" created
gateway.networking.istio.io "knative-ingress-gateway" created
gateway.networking.istio.io "cluster-local-gateway" created
customresourcedefinition.apiextensions.k8s.io "clusteringresses.networking.internal.knative.dev" created
customresourcedefinition.apiextensions.k8s.io "configurations.serving.knative.dev" created
customresourcedefinition.apiextensions.k8s.io "images.caching.internal.knative.dev" created
customresourcedefinition.apiextensions.k8s.io "podautoscalers.autoscaling.internal.knative.dev" created
customresourcedefinition.apiextensions.k8s.io "revisions.serving.knative.dev" created
customresourcedefinition.apiextensions.k8s.io "routes.serving.knative.dev" created
customresourcedefinition.apiextensions.k8s.io "services.serving.knative.dev" created
service "activator-service" created
service "controller" created
service "webhook" created
image.caching.internal.knative.dev "queue-proxy" created
deployment.apps "activator" created
service "autoscaler" created
deployment.apps "autoscaler" created
configmap "config-autoscaler" created
configmap "config-controller" created
configmap "config-domain" created
configmap "config-gc" created
configmap "config-istio" created
configmap "config-logging" created
configmap "config-network" created
configmap "config-observability" created
deployment.apps "controller" created
deployment.apps "webhook" created
namespace "knative-build" created
podsecuritypolicy.policy "knative-build" created
clusterrole.rbac.authorization.k8s.io "knative-build-admin" created
serviceaccount "build-controller" created
clusterrolebinding.rbac.authorization.k8s.io "build-controller-admin" created
customresourcedefinition.apiextensions.k8s.io "builds.build.knative.dev" created
customresourcedefinition.apiextensions.k8s.io "buildtemplates.build.knative.dev" created
customresourcedefinition.apiextensions.k8s.io "clusterbuildtemplates.build.knative.dev" created
customresourcedefinition.apiextensions.k8s.io "images.caching.internal.knative.dev" configured
service "build-controller" created
service "build-webhook" created
image.caching.internal.knative.dev "creds-init" created
image.caching.internal.knative.dev "git-init" created
image.caching.internal.knative.dev "gcs-fetcher" created
image.caching.internal.knative.dev "nop" created
configmap "config-logging" created
deployment.apps "build-controller" created
deployment.apps "build-webhook" created
namespace "knative-eventing" created
serviceaccount "eventing-controller" created
clusterrolebinding.rbac.authorization.k8s.io "eventing-controller-admin" created
customresourcedefinition.apiextensions.k8s.io "channels.eventing.knative.dev" created
customresourcedefinition.apiextensions.k8s.io "clusterchannelprovisioners.eventing.knative.dev" created
customresourcedefinition.apiextensions.k8s.io "subscriptions.eventing.knative.dev" created
configmap "default-channel-webhook" created
service "webhook" created
deployment.apps "eventing-controller" created
deployment.apps "webhook" created
configmap "config-logging" created
serviceaccount "in-memory-channel-controller" created
clusterrole.rbac.authorization.k8s.io "in-memory-channel-controller" created
clusterrolebinding.rbac.authorization.k8s.io "in-memory-channel-controller" created
deployment.apps "in-memory-channel-controller" created
serviceaccount "in-memory-channel-dispatcher" created
clusterrole.rbac.authorization.k8s.io "in-memory-channel-dispatcher" created
clusterrolebinding.rbac.authorization.k8s.io "in-memory-channel-dispatcher" created
deployment.apps "in-memory-channel-dispatcher" created
namespace "knative-sources" created
serviceaccount "controller-manager" created
clusterrole.rbac.authorization.k8s.io "eventing-sources-controller" created
clusterrolebinding.rbac.authorization.k8s.io "eventing-sources-controller" created
customresourcedefinition.apiextensions.k8s.io "awssqssources.sources.eventing.knative.dev" created
customresourcedefinition.apiextensions.k8s.io "containersources.sources.eventing.knative.dev" created
customresourcedefinition.apiextensions.k8s.io "cronjobsources.sources.eventing.knative.dev" created
customresourcedefinition.apiextensions.k8s.io "githubsources.sources.eventing.knative.dev" created
customresourcedefinition.apiextensions.k8s.io "kuberneteseventsources.sources.eventing.knative.dev" created
service "controller" created
statefulset.apps "controller-manager" created
namespace "knative-monitoring" created
service "elasticsearch-logging" created
serviceaccount "elasticsearch-logging" created
clusterrole.rbac.authorization.k8s.io "elasticsearch-logging" created
clusterrolebinding.rbac.authorization.k8s.io "elasticsearch-logging" created
statefulset.apps "elasticsearch-logging" created
service "kibana-logging" created
deployment.apps "kibana-logging" created
configmap "fluentd-ds-config" created
logentry.config.istio.io "requestlog" created
fluentd.config.istio.io "requestloghandler" created
rule.config.istio.io "requestlogtofluentd" created
serviceaccount "fluentd-ds" created
clusterrole.rbac.authorization.k8s.io "fluentd-ds" created
clusterrolebinding.rbac.authorization.k8s.io "fluentd-ds" created
service "fluentd-ds" created
daemonset.apps "fluentd-ds" created
configmap "grafana-dashboard-definition-istio" created
configmap "grafana-dashboard-definition-mixer" created
configmap "grafana-dashboard-definition-pilot" created
serviceaccount "kube-state-metrics" created
role.rbac.authorization.k8s.io "kube-state-metrics-resizer" created
rolebinding.rbac.authorization.k8s.io "kube-state-metrics" created
clusterrole.rbac.authorization.k8s.io "kube-state-metrics" created
clusterrolebinding.rbac.authorization.k8s.io "kube-state-metrics" created
deployment.extensions "kube-state-metrics" created
service "kube-state-metrics" created
configmap "grafana-dashboard-definition-kubernetes-deployment" created
configmap "grafana-dashboard-definition-kubernetes-capacity-planning" created
configmap "grafana-dashboard-definition-kubernetes-cluster-health" created
configmap "grafana-dashboard-definition-kubernetes-cluster-status" created
configmap "grafana-dashboard-definition-kubernetes-control-plane-status" created
configmap "grafana-dashboard-definition-kubernetes-resource-requests" created
configmap "grafana-dashboard-definition-kubernetes-nodes" created
configmap "grafana-dashboard-definition-kubernetes-pods" created
configmap "grafana-dashboard-definition-kubernetes-statefulset" created
serviceaccount "node-exporter" created
clusterrole.rbac.authorization.k8s.io "node-exporter" created
clusterrolebinding.rbac.authorization.k8s.io "node-exporter" created
daemonset.extensions "node-exporter" created
service "node-exporter" created
configmap "grafana-dashboard-definition-knative-efficiency" created
configmap "grafana-dashboard-definition-knative-reconciler" created
configmap "scaling-config" created
configmap "grafana-dashboard-definition-knative" created
configmap "grafana-datasources" created
configmap "grafana-dashboards" created
service "grafana" created
deployment.apps "grafana" created
metric.config.istio.io "revisionrequestcount" created
metric.config.istio.io "revisionrequestduration" created
metric.config.istio.io "revisionrequestsize" created
metric.config.istio.io "revisionresponsesize" created
prometheus.config.istio.io "revisionpromhandler" created
rule.config.istio.io "revisionpromhttp" created
configmap "prometheus-scrape-config" created
service "kube-controller-manager" created
service "prometheus-system-discovery" created
serviceaccount "prometheus-system" created
role.rbac.authorization.k8s.io "prometheus-system" created
role.rbac.authorization.k8s.io "prometheus-system" created
role.rbac.authorization.k8s.io "prometheus-system" created
role.rbac.authorization.k8s.io "prometheus-system" created
clusterrole.rbac.authorization.k8s.io "prometheus-system" created
rolebinding.rbac.authorization.k8s.io "prometheus-system" created
rolebinding.rbac.authorization.k8s.io "prometheus-system" created
rolebinding.rbac.authorization.k8s.io "prometheus-system" created
rolebinding.rbac.authorization.k8s.io "prometheus-system" created
clusterrolebinding.rbac.authorization.k8s.io "prometheus-system" created
service "prometheus-system-np" created
statefulset.apps "prometheus-system" created
service "zipkin" created
deployment.apps "zipkin" created
clusterrole.rbac.authorization.k8s.io "knative-serving-build" created
unable to recognize "https://github.com/knative/eventing/releases/download/v0.4.0/release.yaml": no matches for kind "ClusterChannelProvisioner" in version "eventing.knative.dev/v1alpha1"
unable to recognize "https://github.com/knative/eventing/releases/download/v0.4.0/release.yaml": no matches for kind "ClusterChannelProvisioner" in version "eventing.knative.dev/v1alpha1"
```
Annoying the last two lines failed but there's a note to re-run
Second run appeared to be okay
```
configmap "config-logging" unchanged
clusterchannelprovisioner.eventing.knative.dev "in-memory" created
clusterchannelprovisioner.eventing.knative.dev "in-memory-channel" created
serviceaccount "in-memory-channel-controller" unchanged
clusterrole.rbac.authorization.k8s.io "in-memory-channel-controller" configured
clusterrolebinding.rbac.authorization.k8s.io "in-memory-channel-controller" configured
deployment.apps "in-memory-channel-controller" unchanged
serviceaccount "in-memory-channel-dispatcher" unchanged
clusterrole.rbac.authorization.k8s.io "in-memory-channel-dispatcher" configured
clusterrolebinding.rbac.authorization.k8s.io "in-memory-channel-dispatcher" configured
deployment.apps "in-memory-channel-dispatcher" unchanged
namespace "knative-sources" configured
serviceaccount "controller-manager" unchanged
clusterrole.rbac.authorization.k8s.io "eventing-sources-controller" configured
clusterrolebinding.rbac.authorization.k8s.io "eventing-sources-controller" configured
customresourcedefinition.apiextensions.k8s.io "awssqssources.sources.eventing.knative.dev" configured
customresourcedefinition.apiextensions.k8s.io "containersources.sources.eventing.knative.dev" configured
customresourcedefinition.apiextensions.k8s.io "cronjobsources.sources.eventing.knative.dev" configured
customresourcedefinition.apiextensions.k8s.io "githubsources.sources.eventing.knative.dev" configured
customresourcedefinition.apiextensions.k8s.io "kuberneteseventsources.sources.eventing.knative.dev" configured
service "controller" unchanged
statefulset.apps "controller-manager" unchanged
namespace "knative-monitoring" configured
service "elasticsearch-logging" unchanged
serviceaccount "elasticsearch-logging" unchanged
clusterrole.rbac.authorization.k8s.io "elasticsearch-logging" configured
clusterrolebinding.rbac.authorization.k8s.io "elasticsearch-logging" configured
statefulset.apps "elasticsearch-logging" configured
service "kibana-logging" unchanged
deployment.apps "kibana-logging" configured
configmap "fluentd-ds-config" unchanged
logentry.config.istio.io "requestlog" configured
fluentd.config.istio.io "requestloghandler" configured
rule.config.istio.io "requestlogtofluentd" configured
serviceaccount "fluentd-ds" unchanged
clusterrole.rbac.authorization.k8s.io "fluentd-ds" configured
clusterrolebinding.rbac.authorization.k8s.io "fluentd-ds" configured
service "fluentd-ds" unchanged
daemonset.apps "fluentd-ds" unchanged
configmap "grafana-dashboard-definition-istio" unchanged
configmap "grafana-dashboard-definition-mixer" unchanged
configmap "grafana-dashboard-definition-pilot" unchanged
serviceaccount "kube-state-metrics" unchanged
role.rbac.authorization.k8s.io "kube-state-metrics-resizer" unchanged
rolebinding.rbac.authorization.k8s.io "kube-state-metrics" unchanged
clusterrole.rbac.authorization.k8s.io "kube-state-metrics" configured
clusterrolebinding.rbac.authorization.k8s.io "kube-state-metrics" configured
deployment.extensions "kube-state-metrics" unchanged
service "kube-state-metrics" unchanged
configmap "grafana-dashboard-definition-kubernetes-deployment" unchanged
configmap "grafana-dashboard-definition-kubernetes-capacity-planning" unchanged
configmap "grafana-dashboard-definition-kubernetes-cluster-health" unchanged
configmap "grafana-dashboard-definition-kubernetes-cluster-status" unchanged
configmap "grafana-dashboard-definition-kubernetes-control-plane-status" unchanged
configmap "grafana-dashboard-definition-kubernetes-resource-requests" unchanged
configmap "grafana-dashboard-definition-kubernetes-nodes" unchanged
configmap "grafana-dashboard-definition-kubernetes-pods" unchanged
configmap "grafana-dashboard-definition-kubernetes-statefulset" unchanged
serviceaccount "node-exporter" unchanged
clusterrole.rbac.authorization.k8s.io "node-exporter" configured
clusterrolebinding.rbac.authorization.k8s.io "node-exporter" configured
daemonset.extensions "node-exporter" unchanged
service "node-exporter" unchanged
configmap "grafana-dashboard-definition-knative-efficiency" unchanged
configmap "grafana-dashboard-definition-knative-reconciler" unchanged
configmap "scaling-config" unchanged
configmap "grafana-dashboard-definition-knative" unchanged
configmap "grafana-datasources" unchanged
configmap "grafana-dashboards" unchanged
service "grafana" unchanged
deployment.apps "grafana" unchanged
metric.config.istio.io "revisionrequestcount" configured
metric.config.istio.io "revisionrequestduration" configured
metric.config.istio.io "revisionrequestsize" configured
metric.config.istio.io "revisionresponsesize" configured
prometheus.config.istio.io "revisionpromhandler" configured
rule.config.istio.io "revisionpromhttp" configured
configmap "prometheus-scrape-config" unchanged
service "kube-controller-manager" unchanged
service "prometheus-system-discovery" unchanged
serviceaccount "prometheus-system" unchanged
role.rbac.authorization.k8s.io "prometheus-system" unchanged
role.rbac.authorization.k8s.io "prometheus-system" unchanged
role.rbac.authorization.k8s.io "prometheus-system" unchanged
role.rbac.authorization.k8s.io "prometheus-system" unchanged
clusterrole.rbac.authorization.k8s.io "prometheus-system" configured
rolebinding.rbac.authorization.k8s.io "prometheus-system" unchanged
rolebinding.rbac.authorization.k8s.io "prometheus-system" unchanged
rolebinding.rbac.authorization.k8s.io "prometheus-system" unchanged
rolebinding.rbac.authorization.k8s.io "prometheus-system" unchanged
clusterrolebinding.rbac.authorization.k8s.io "prometheus-system" configured
service "prometheus-system-np" unchanged
statefulset.apps "prometheus-system" unchanged
service "zipkin" unchanged
deployment.apps "zipkin" unchanged
clusterrole.rbac.authorization.k8s.io "knative-serving-build" configured
```

check on running pods: 
```
❯    kubectl get pods --namespace knative-serving
   kubectl get pods --namespace knative-build
   kubectl get pods --namespace knative-eventing
   kubectl get pods --namespace knative-sources
   kubectl get pods --namespace knative-monitoring
NAME                          READY     STATUS    RESTARTS   AGE
activator-7c8b59d78-px4wj     2/2       Running   1          2m
autoscaler-666c9bfcc6-479st   2/2       Running   1          2m
controller-799cd5c6dc-46sv2   1/1       Running   0          2m
webhook-5b66fdf6b9-vbgqq      1/1       Running   0          2m
NAME                                READY     STATUS    RESTARTS   AGE
build-controller-7b8987d675-nd2cm   1/1       Running   0          2m
build-webhook-74795c8696-x5bwh      1/1       Running   0          2m
NAME                                            READY     STATUS    RESTARTS   AGE
eventing-controller-864657d8d4-6gksm            1/1       Running   0          2m
in-memory-channel-controller-f794cc9d8-9mv5g    1/1       Running   0          2m
in-memory-channel-dispatcher-8595c7f8d7-k6btk   2/2       Running   1          2m
webhook-5d76776d55-5lkmc                        1/1       Running   0          2m
NAME                   READY     STATUS    RESTARTS   AGE
controller-manager-0   1/1       Running   0          2m
NAME                                  READY     STATUS    RESTARTS   AGE
elasticsearch-logging-0               1/1       Running   0          2m
elasticsearch-logging-1               1/1       Running   0          1m
fluentd-ds-js44x                      1/1       Running   0          2m
fluentd-ds-mk8t5                      1/1       Running   0          2m
fluentd-ds-trjgr                      1/1       Running   0          2m
grafana-568674f4f9-8ktzj              1/1       Running   0          2m
kibana-logging-7698db4f94-fxtgw       1/1       Running   0          2m
kube-state-metrics-5c46b58f6b-8dqcp   4/4       Running   0          1m
node-exporter-29zsm                   2/2       Running   0          2m
node-exporter-8lkmk                   2/2       Running   0          2m
node-exporter-hwkvz                   2/2       Running   0          2m
prometheus-system-0                   1/1       Running   0          1m
prometheus-system-1                   1/1       Running   0          1m
```

All appears to be up as far as I can tell

Next is to create a grpc-ping hello world. I'm going to steal the grpc ping demo more or less but port it to my own code. 

Though to prove that this is working I'm going to use the demo for autoscaling straight up and down. 


the Serving API is still broken:
```
kubectl apply --filename docs/serving/samples/autoscale-go/service.yaml
Error from server (InternalError): error when creating "docs/serving/samples/autoscale-go/service.yaml": Internal error occurred: admission webhook "webhook.serving.knative.dev" denied the request: mutation failed: expected exactly one, got neither: spec.manual, spec.pinned, spec.release, spec.runLatest
```

Fixed this by restructuring the yaml

There was a mismatch in documentation between 0.4 listed in the install isntructions on the installing on gke. and 0.6 being the latest documentation. 

Asked in slack if I should move to the 0.4 docs to test autoscaling or if I should try to upgrade the isntall instructions. Was recommended that I switch back to 0.4 docs


SOME NOTES ABOUT AUTOSCALING: The panic algorithm kicks in after a large influx of request and stabalizes after 60 seconds. We want to make sure that we test the long tail load that a service can handle by testing each level of load for at least 4 minutes each. To make this easier we can rely on the script that I created