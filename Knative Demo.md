# knative demo

## Creating a GKE cluster
export CLUSTER_NAME=knative-demo
export CLUSTER_ZONE=us-west1-b

gcloud beta container clusters create $CLUSTER_NAME \
  --addons=HorizontalPodAutoscaling,HttpLoadBalancing,Istio \
  --machine-type=n1-standard-4 \
  --cluster-version=latest --zone=$CLUSTER_ZONE \
  --enable-stackdriver-kubernetes --enable-ip-alias \
  --enable-autoscaling --min-nodes=1 --max-nodes=10 \
  --enable-autorepair \
  --scopes cloud-platform

kubectl create clusterrolebinding cluster-admin-binding \
     --clusterrole=cluster-admin \
     --user=$(gcloud config get-value core/account)

## Installing knative serving

We first apply the CRDs that define knative with the -l knative.dev/crd-install=true flag so that we prevent race conditions.

kubectl apply --selector knative.dev/crd-install=true \
   --filename https://github.com/knative/serving/releases/download/v0.6.0/serving.yaml \
   --filename https://github.com/knative/build/releases/download/v0.6.0/build.yaml \
   --filename https://github.com/knative/eventing/releases/download/v0.6.0/release.yaml \
   --filename https://github.com/knative/eventing-sources/releases/download/v0.6.0/eventing-sources.yaml \
   --filename https://github.com/knative/serving/releases/download/v0.6.0/monitoring.yaml \
   --filename https://raw.githubusercontent.com/knative/serving/v0.6.0/third_party/config/build/clusterrole.yaml

We run it again with the the specific selector flag to complete the install of the dependencies.
kubectl apply --filename https://github.com/knative/serving/releases/download/v0.6.0/serving.yaml --selector networking.knative.dev/certificate-provider!=cert-manager \
   --filename https://github.com/knative/build/releases/download/v0.6.0/build.yaml \
   --filename https://github.com/knative/eventing/releases/download/v0.6.0/release.yaml \
   --filename https://github.com/knative/eventing-sources/releases/download/v0.6.0/eventing-sources.yaml \
   --filename https://github.com/knative/serving/releases/download/v0.6.0/monitoring.yaml \
   --filename https://raw.githubusercontent.com/knative/serving/v0.6.0/third_party/config/build/clusterrole.yaml

So now that we have knative installed we can make sure that all the CRDs are running appropriately. I didn't rune down the extra eventing and build resources yet because it was unclear to me what side effects I might get for this demo and I wanted to avoid that.
```
   kubectl get pods --namespace knative-serving
   kubectl get pods --namespace knative-build
   kubectl get pods --namespace knative-eventing
   kubectl get pods --namespace knative-sources
   kubectl get pods --namespace knative-monitoring
```

## What is a knative service
Now that we've verified the install and all the CRDs are defined we can do deploy our first concept of a service. 

The question now is what is a knative service.

Knative services are the primary interface to interact with deployed code hosted on knative.
There a couple of components to a Service. A configuration, and route. Every time a configuration is update, patched or created an immutable Revision is created that describe the an underlying kubernetes deployments, and associated pods.

Keep this in mind when working with knative serving. the main things you'll be concerning yourself with are Services, Configurations, Revisions, and Deployments. Deployments are the only vanilla kubernetes resource and ideally you should never have to touch it. There are situations in which you will want to peak at them though to understand what is going on under the hood. In particular pod logs. 

The route will define the subdomain in which the service is accessed on.

----- Going to try to deploy a new service here and then start creating configurations and deal with blue green deployment -----

To start I'm going to deploy a simple hello world gRPC app which is a slight modification of the gRPC demo provided by knative.

```sh
vim grpc-ping.go
```

It basically echos a ping with value set in an environment variable. This will allow us to update configurations without having to rebuild docker images. When we build that actual platform the container image will be the value we change in the Configuration.

I've build and pushed up this image before hand to docker hub which is a public repository. I could have just as easily done this with gcr.io.The image is located at `docker.io/kwyn/grpc-ping-go` We'll need that for our new service defintiion

```
vim new-service.yaml
```

In the new service yaml you can see the meta data has the name of the service and the spec contains a template for the revision of the service we're creating. and also a spec of the container which is based right from k8s core container specification. 
I've also specified the http2 ports for the gRPC service to exist on.

I've also defined an environment variable for use to change so we have a simple way of identifying different revisions by the response we get from gRPC ping.

Before I deploy this I want to ensure I have a clean slate so I'm going to query for all the kubernetes resources.

```
kubectl get all
```
We can see we only have our eventing resources which I plan on not adding on our actual future install and the control plane. This is perfect. Let's apply our service

```
kubectl apply -f new-service.yaml
```
We can now see what resource this service created for us.  `k get all`

We can see that we've got many running resource now with many pods with different IPs, we can see a metrics pod associated with our service as well as the primary service pod. We also see a deployment has been created for us as well as a kuberenetes replicate set. We also have the knative special sauce autoscaler which we'll go into later. 

We also have an image cache that is baked into knative to allow rollbacks without having to pull the image across the network again. 
We also have cluster ingress which is the underlying resource that is controlled by the routing definition generated by our service.

There's a lot going on here, this is part of the power of knative. We would have had to write all of this complex logic our selves to underpin our platform, but instead we get this for free more or less. 

More interestingly we want to inspect the three resource I had described before to learn about them a bit more.

Let's take a look at the service we just created
```
kubectl get service
```

We see we have multiple service. The one we're interested in is the one that matches our service name exactly

```
kubectl get service grpc-ping
```

where we're getting the service resource and then describing it via yaml instead of the pretty printed output kubectl gives us that is omitting some details.

Once create the service you need to query for the cluster ingress IP created for you by the knative internals
```sh
export SERVICE_IP=`kubectl get svc istio-ingressgateway --namespace istio-system --output jsonpath="{.status.loadBalancer.ingress[*].ip}"
echo $SERVICE_IP
```

This accessed the istio ingress resource that is acting as our load balancer in the cluster. This is what knative will use instead of googles load balancers since istio supports http2 load balancing.  Istio ontop of load balancing acts as a reverse proxy the will route the request to the appropriate pod based on the request domain header.

Also the reason we didn't see this loadbalancer in the large list of `kubectl get all` resources is that it lives in it's own name space. 

As an aside you can view what name spaces are in a cluster by running `kubectl get namespaces` this is useful when you're diving into an unfamiliar k8s deployment. 

Another thing that you might be curious is to get at the pods for the given service so let's see what we have...
```
kubectl get pods
```

Wait, we have no pods, why is that? Well knative will scale down to zero. Since our service has no traffic our pods have scaled down to zero to save resources.

So, let's make some traffic so we can see how this works. 

We'll run our client in a docker container for simplicity since I build the server and client together in the same docker file. 
You'll see I've references the env variable with the service ip and the public port 80. The server host overide is given as a parameter as we aren't curling directly for a url but istio uses the host to route to the correct service. We also include the insecure flag as I haven't set up TLS certs yet.

```
docker run -ti --entrypoint=/client docker.io/kwyn/grpc-ping-go \
  -server_addr="${SERVICE_IP}:80" \
  -server_host_override="grpc-ping.default.example.com" \
  -insecure
```

And now when we run `kubectl get pods` we see that we have one pod running with two containers! (the istio side car and the "user" container")

And that's pretty much all there is to that for knative serving straight up and down. 

Any question before I dive into how one does blue green deployments?


## Bluegreen demo
The idea behind blue green deployments is you can deploy a change without routing traffic to it right away and slowly pipe traffic to it. To do this in knative we need to update the configuration of a service to generate a new revision and get the revision name.  

We can find the current revision names by running `k get revision`
Then we can add this to a new route definition.

By default service are set up to use the latest revision. Now that we have a revision we can explictily define the revision in the `traffic` parameter in the knative deployment along with how much traffic to route. This will edit the underlying knative route.

```
kubectl edit service.serving.knative.dev/grpc-ping
```
There's a differnece between the knative service and the internal kubernetes idea of a service. You can tell which one you're dealing with by the `API . Version` seciton if it says `serving.knative.dev/v1alpha1` then it's knative if it's just a version number it's k8s. 

Then we'll change the traffic to the latest revision we got from k get revisions and save the file which will update our configuration and route.

Now we want to prove to ourselves this is all working still. So we'll hit the service again to just be sure.

```
docker run -ti --entrypoint=/client docker.io/kwyn/grpc-ping-go \
  -server_addr="${SERVICE_IP}:80" \
  -server_host_override="grpc-ping.default.example.com" \
  -insecure
```

Okay so that looks good. 

Let's go ahead and update our service environment variable to generate a new revision.

we can do this through kubectl edit again
```
kubectl edit service.serving.knative.dev/grpc-ping
```
so we've now update our environment variable lets see if our new change is live (it shouldn't be)
```
docker run -ti --entrypoint=/client docker.io/kwyn/grpc-ping-go \
  -server_addr="${SERVICE_IP}:80" \
  -server_host_override="grpc-ping.default.example.com" \
  -insecure
```

So we're not certain if the new revision is even okay, we want to be able to test it directly but we have no way of accessing directly yet. We can do that by adding a tag which will create a new subdomain just for that revision.

```
kubectl edit service.serving.knative.dev/grpc-ping
```
so I've added a tag but now I have no idea what the new URL is, if I hit the old url I'll get the same thing. What's the new url? To find our we'll have to query the knative route description again

```
kubectl describe route grpc-ping
```

There is a convention for this, it will just append it to the subdomain with a - but for robust ness we will probably always want to query the API. 

we can see that it's now at grpc-ping-v2

Let's test that out and see if we get the response we want.
```
docker run -ti --entrypoint=/client docker.io/kwyn/grpc-ping-go \
  -server_addr="${SERVICE_IP}:80" \
  -server_host_override="grpc-ping-v2.default.example.com" \
  -insecure
```
Huzzah! we can now test out our change without it being live to any users and it's in a production cluster! How exciting.

now, we're happy with a few request we've manually sent through and nothings exploded. Let's go ahead and start rolling out the change to say 50% of traffic, I say this just so that I don't have to query too many times to see the alternation.

```
kubectl edit service.serving.knative.dev/grpc-ping 
```

Cool no we can monitor the behavior of our service with real traffic and see if anything goes extremely poorly. 

If we're happy we simply roll the change all the way forward by editing the last one to 100%. If you don't want the old revision remove it entirely and knative will eventually grabage collect it. If you are worried you'll need to roll back you can keep it at zero and the reference to it will force knative to keep it around.

## Creating a new revision without routing traffic to it.

## Improvements
- Add stack driver monitoring plugin 
- Enable custom domain
- Enable certs
