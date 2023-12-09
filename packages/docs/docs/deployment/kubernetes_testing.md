# Kubernetes Configuration Testing
Vrooli can be run at varying levels of scale, from a single server to a full Kubernetes cluster (allows for horizontal scaling). Since we start off with Docker Compose for local development, we have the script `./scripts/dockerToKubernetes.sh` to convert the Docker Compose file a Kubernetes configuration.

However, this conversion process is not perfect. The configuration will most likely not work on the first try, and will require some debugging. This guide will walk you through the process of testing the Kubernetes configuration locally.

## Before Getting Started
1. Make sure you have run `./scripts/setup.sh` at some point, so you have the necessary services installed (e.g. Docker, Minikube).
2. Stop any running instances of Vrooli so you don't kill your computer.
3. Docker images should be on Docker Hub. If you have made changes to the Docker images, you will need to rebuild them and push them to Docker Hub. You can do this by running `./scripts/build.sh -u y`. If you want to make sure the images are safe to put on Docker Hub, see the next section. 

### Inspecting Docker Images
`docker-compose` services that were built with a Dockerfile cannot be pulled by Minikube unless they are on Docker Hub, so adding your images to a registry (typically Docker Hub) is a necessary step. However, it is crucial that no sensitive information is accidentally included in the image, as anyone could find it. If you are unsure if this is the case, you can inspect each image before pushing it to the registry. Here's how:

1. Check the corresponding Dockerfile. If there are any `ADD` or `COPY` commands, make sure they are not copying any sensitive files, such as `.env` files or `.git` files.
2. Run `./scripts/build.sh` without the `-u` flag. This will build the images, but not push them to the registry. 
3. Run `docker images` to find the image ID of the image you want to inspect.
4. Run `docker run -v /var/run/docker.sock:/var/run/docker.sock aquasec/trivy image <image_name>` to scan the image for vulnerabilities. This should identify all vulnerabilities. But if you still want to inspect the image manually, continue to the next step.
5. Run `docker run -it <image_id> /bin/sh` to open a shell inside the image.
6. Run `history` to see if any commands were run that might have included sensitive information. Also check `/root/.ash_history` or `/root/.bash_history` for the same reason.
7. Run `env` to see the environment variables that are set. Make sure there are no sensitive variables, such as passwords or API keys.
8. Make sure `/run/secrets` doesn't exist. If it does there are no sensitive files in it.
9. Go to the root of the project and run `ls -a` to see all the files in the image. make sure there are no sensitive files, such as `.env` files, `.git` files, `.pem` files, or database backups.
10. Check if any logs exist with sensitive information in `/var/log/`.
11. Make sure no private keys or certificates exist in `~/.ssh/`.

## Basic Steps
1. Create the Kubernetes configuration (if you haven't already) using `./scripts/dockerToKubernetes.sh`.
2. Start Minikube in a Docker container, using `minikube start --driver=docker --force`. *Note: You can stop minikube using `minikube stop`.*
3. Point your terminal to use the Docker daemon inside Minikube, using `eval $(minikube docker-env)`. This sets a few Docker-related environment variables, so that Docker commands point to the Docker daemon inside Minikube, rather than the one on your local machine. *Note: You can undo this using `eval $(minikube docker-env -u)`.*
4. Add the following lines to your hosts file (`/etc/hosts` for Linux and Mac, `C:\Windows\System32\Drivers\etc\hosts` for Windows):
    ```
    127.0.0.1 docs.vrooli.com
    127.0.0.1 vrooli.com
    ```
    This will allow you to access the services using the same URLs as in production.
    *NOTE:* We are NOT using Minikube's IP address here, since it doesn't work for some reason. This may be because we're using Docker for the driver, instead of something like hyperv.
    **WARNING:** Remember to remove these lines from `/etc/hosts` when you are done testing. Otherwise, you will not be able to access the real Vrooli website.
5. Enable ingress on Minikube, using `minikube addons enable ingress`, then `minikube addone enable ingress-dns`. *Note: You can disable ingress using `minikube addons disable ingress`.*
6. If you want to start Kubernetes from scratch, run `kubectl delete all --all`. This will delete all Kubernetes objects, including pods, deployments, services, etc. *Note: You can delete specific objects using `kubectl delete <object-type> <object-name>`.*
7. Apply the configuration to Minikube, using `kubectl apply -f <generated-file>.yaml`. *Note: You can delete the configuration using `kubectl delete -f <generated-file>.yaml`.*
8. Use kubectl commands to monitor the status of your pods, deployments, and services. Reapply the configuration when changes are made. *Note: If you delete sections from the configuration, you may need to run `kubectl delete -f <generated-file>.yaml` before reapplying the configuration.*

## Required Changes
The auto-generated Kubernetes configuration will not work out of the box. You will need to debug it and make some changes. Here are some common steps you will likely need to take:

1. Change every deployment strategy to `RollingUpdate` instead of `Recreate`. This will allow pods to be updated without downtime.
2. Remove `metadata.annotations` from every object. This is not necessary, and has no effect on the configuration.
3. Change the `image` field for every container which uses a Dockerfile (i.e. is not an existing public image) to the format `<docker_username>/vrooli_<service_name>:<dev_or_prod>-<version>`. For example, `server:prod` becomes `matthalloran8/vrooli_server:prod-1.9.7`. This will allow Minikube to pull the correct images from Docker Hub.
4. Replace every non-sensitive `secretKeyRef` (e.g. ports) with a hard-coded value. For example, you would replace this:
    ```yaml
    env:
      - name: VIRTUAL_HOST
        valueFrom:
          secretKeyRef:
            name: db-claim3
            key: VIRTUAL_HOST_DOCS
    ```
    with this:
    ```yaml
    env:
      - name: VIRTUAL_HOST
        value: "docs.vrooli.com,www.docs.vrooli.com"
    ```
5. Keep every sensitive `secretKey`, but replace `secretKeyRef.name` with the same value as `secretKeyRef.key`. For example, you would replace this:
    ```yaml
    env:
      - name: DB_PASSWORD
        valueFrom:
          secretKeyRef:
            name: db-claim3
            key: DB_PASSWORD
    ```
    with this:
    ```yaml
    env:
      - name: DB_PASSWORD
        valueFrom:
          secretKeyRef:
            name: DB_PASSWORD
            key: DB_PASSWORD
    ```
6. Every *Deployment* uses `/src/app/scripts`, which means we can use one *Persistent Volume Claim (PVC)* for all of them. Find every `mountPath` that references `/src/app/scripts` and make sure they all use the same *PVC*. You should rename this *PVC* to `scripts-claim`, and change the `accessModes` to `ReadOnlyMany`. Make sure you have removed the other *PVCs* that aren't being used, and changed all corresponding `claimName` fields to `scripts-claim`.
7. Do the same as number 5, but for `/src/app/packages/ui/dist`.
8. Any PVC which references code stored in version control needs a corresponding `initContainers` section in the Kubernetes configuration. This is because the code is not stored in the Docker image, so it needs to be pulled from version control when the pod starts. For example, the `docs` service has this PVC mount:
    ```yaml
    volumeMounts:
        - mountPath: /docs
            name: docs-claim0
    ```
    So we need to add this `initContainers` section:
    ```yaml
    initContainers:
        - name: init-docs-claim0
          image: alpine/git
          command: ["/bin/sh", "-c"]
          args:
            - |
              rm -rf /docs/* &&
              git clone -n --depth=1 -b master --filter=tree:0 https://github.com/Vrooli/Vrooli /tmp/init-docs-claim0 &&
              cd /tmp/init-docs-claim0 &&
              git sparse-checkout set --no-cone packages/docs &&
              git checkout &&
              mv /tmp/init-docs-claim0/packages/docs/* /docs &&
              rm -rf /tmp/init-docs-claim0
          volumeMounts:
          - name: docs-claim0
            mountPath: /docs
    ```
    **NOTE:** There should only be one init section per PVC. If multiple deployments are using the same PVC, then only one of them should have an init section.
9. Do something similar to step 7 for the ui dist folder. This is stored in S3 instead of git, so the init section should look something like this:
    ```yml
    initContainers:
        - name: init-ui
          image: alpine
          command: ["/bin/sh", "-c"]
          args:
            - >
              apk --no-cache add aws-cli tar &&
              aws s3 cp s3://vrooli-bucket/builds/v1.9.7/build.tar.gz /tmp/build.tar.gz &&
              tar -xzvf /tmp/build.tar.gz -C /path/to/mounted/volume &&
              rm /tmp/build.tar.gz
          env:
            - name: AWS_ACCESS_KEY_ID
              valueFrom:
                secretKeyRef:
                  name: AWS_ACCESS_KEY_ID
                  key: AWS_ACCESS_KEY_ID
            - name: AWS_SECRET_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: AWS_SECRET_ACCESS_KEY
                  key: AWS_SECRET_ACCESS_KEY
          volumeMounts:
            - name: ui-dist-claim
              mountPath: /srv/app/packages/ui/dist
    ```
10. Expose the ports for `server`, `ui`, and `docs`. This is accomplished by adding an *Ingress* resource. It should look something like this:
    ```yaml
    apiVersion: networking.k8s.io/v1
    kind: Ingress
    metadata:
    name: vrooli-ingress
    spec:
    rules:
        - host: "docs.vrooli.com"
        http:
            paths:
            - path: /
                pathType: Prefix
                backend:
                service:
                    name: docs
                    port:
                    number: 4000
        - host: "vrooli.com"
        http:
            paths:
            - path: /api
                pathType: Prefix
                backend:
                service:
                    name: server
                    port:
                    number: 5329
            - path: /
                pathType: Prefix
                backend:
                service:
                    name: ui
                    port:
                    number: 3000
    ```

## Kubernetes Configuration Overview
Kubernetes employs a variety of object types to facilitate orchestration and scaling of containerized applications:

- **Pods**: Basic deployable units that represent one or more containers in your application.
- **Services**: Expose an application running on a set of Pods as a network service. This is useful for managing network access and load balancing.
- **Deployments**: Manage the desired state for your application, ensuring the specified number of pod replicas are running.
- **ReplicaSets**: Ensures a specified number of pod replicas are running at any given time. Managed by Deployments.
- **ConfigMaps & Secrets**: Allow environment-specific configuration separation from application images.
- **PersistentVolumeClaims**: Request storage resources for use with pods.
- **Ingress**: Manage external access to services, typically HTTP.
- **Cluster**: The overall Kubernetes environment, consisting of a master and one or more nodes.

For Vrooli's deployment, we utilize these objects to ensure a smooth and scalable operation of the application in a Kubernetes cluster.

### Docker Compose vs. Kubernetes
For those familiar with Docker Compose, transitioning to Kubernetes can be a paradigm shift. Both are orchestration tools, but Kubernetes is more extensive and built for larger, distributed environments. Here's how you can map Docker Compose concepts to Kubernetes:

1. **Services (Docker Compose) vs. Pods (Kubernetes)**:
    - **Docker Compose**: A service defines how a container runs, specifying things like the image to use, ports to bind, and replicas.
    - **Kubernetes**: A Pod is the smallest deployable unit and can run one or more containers. While you might run multiple replicas of a service in Docker Compose, in Kubernetes, you would have multiple pods.

2. **Networking (Docker Compose) vs. Services (Kubernetes)**:
    - **Docker Compose**: All containers are on a single network, and they can communicate with each other over this network.
    - **Kubernetes**: Services (not to be confused with the Docker concept by the same name) abstract away the Pod's ephemeral nature and provide a stable endpoint to communicate with Pods. A Kubernetes Service is like a persistent IP for your set of pods, and it manages load balancing and port mapping.

3. **Deploy (Docker Compose) vs. Deployments/ReplicaSets (Kubernetes)**:
    - **Docker Compose**: The deploy key can specify how many replicas of a service to run.
    - **Kubernetes**: Deployments manage the desired state of your application and control the ReplicaSet, which ensures the specified number of pod replicas are running. They handle updates and rollbacks too.

4. **Volumes (Docker Compose) vs. PersistentVolumes and PersistentVolumeClaims (Kubernetes)**:
    - **Docker Compose**: You define volumes for persistent storage.
    - **Kubernetes**: PersistentVolumes (PV) represent storage resources in a cluster, and PersistentVolumeClaims (PVC) are requests for those resources. Essentially, PVCs are to PVs what containers are to VMs—they provide abstraction and ensure efficient resource utilization.

5. **Environment Variables (Docker Compose) vs. ConfigMaps & Secrets (Kubernetes)**:
    - **Docker Compose**: You can define environment variables directly in the service definition.
    - **Kubernetes**: For better management and to separate configuration from application images, ConfigMaps and Secrets are used. ConfigMaps handle non-sensitive data, while Secrets are for sensitive data.

6. **Docker Compose Scaling vs. HorizontalPodAutoscaler (Kubernetes)**:
    - **Docker Compose**: You can scale services using the `--scale` flag with `docker-compose up`.
    - **Kubernetes**: The HorizontalPodAutoscaler automatically scales the number of pods based on CPU utilization or other select metrics.

7. **Ports (Docker Compose) vs. Ingress (Kubernetes)**:
    - **Docker Compose**: Exposing ports in services makes them accessible outside.
    - **Kubernetes**: While Services can expose ports, Ingress manages external access to the services in a cluster, providing HTTP and HTTPS routing, load balancing, and domain-based virtual hosting.

Remember, while Docker Compose is designed for defining and running multi-container Docker applications, Kubernetes goes a step further by providing tools for deploying, scaling, and managing containerized applications across clusters. This means a steeper learning curve, but much more power and flexibility, especially for large-scale applications.

### Interacting with HashiCorp Vault for Secrets Management
While Kubernetes offers its own Secrets object, for enhanced capabilities and security, we utilize HashiCorp Vault, a dedicated secret management tool. Notably, our Vault instance runs in a separate cluster, distinct from our main application's Kubernetes cluster. Here's how we'll be integrating and interacting with this external Vault setup:

1. **Separate Cluster Deployment**:
    - Running Vault in a separate cluster provides isolation, ensuring that potential vulnerabilities in one system don't directly compromise the other. This adds an extra layer of security to our secrets management.

2. **Authentication**:
    - **Kubernetes Auth Method**: Even though Vault is external, it provides an authentication method tailored for Kubernetes. Pods in our application cluster can authenticate with the external Vault using their service account tokens.
    - **Other Methods**: Vault supports a variety of auth methods, and depending on our security policies, we might leverage tokens, LDAP, or other mechanisms.

3. **Secrets Storage**:
    - Sensitive information is stored in Vault instead of Kubernetes Secrets. Vault's secret backends are utilized to store, generate, and manage different types of secrets, from static API keys to dynamic credentials.

4. **Dynamic Secrets**:
    - Vault can dynamically generate secrets on-demand. This reduces the lifetime of secrets, minimizing exposure and risk.

5. **Integration with Pods**:
    - **Init Containers**: To fetch secrets from the external Vault before the main application starts, we can use init containers. These containers will populate the secrets where the application expects them.
    - **Sidecar Containers**: Sidecar containers can dynamically fetch and refresh secrets from Vault, ensuring applications always have access to valid credentials.
    - **Vault Agent**: To simplify integration, the Vault Agent can be employed. This client-side tool can automatically fetch, renew, and cache secrets.

6. **Security**:
    - **Encryption**: Data in transit between our application cluster and the external Vault is encrypted. Additionally, Vault ensures secrets are encrypted at rest.
    - **Access Control**: Vault's policies control who accesses what, ensuring that only authorized applications or entities fetch particular secrets.
    - **Audit Logs**: All interactions are logged, creating an audit trail that shows when and how secrets were accessed.

7. **Lifecycle Management**:
    - We leverage Vault's capabilities for secrets rotation, expiration, and versioning. This ensures static secrets don't remain in the ecosystem for extended periods.

By integrating with an external HashiCorp Vault cluster, we ensure a robust, dynamic, and auditable secret management system that augments our Kubernetes application's security and efficiency.

## Kubectl Commands
`kubectl` is the command-line tool for interacting with a Kubernetes cluster. It provides functionalities to create, inspect, update, and delete Kubernetes resources.

### Status Checking
Pods in Kubernetes can have a variety of statuses, which can be indicative of their health and readiness. Here are some common statuses you may encounter, especially when debugging:

- **Pending**: The pod has been accepted by the system but is waiting for one or more of its containers to be created and scheduled.
  
- **Running**: The pod has been bound to a node, and all of its containers have been created. At least one container is still running, or is in the process of starting or restarting.
  
- **Succeeded**: All containers in the pod have terminated successfully, and will not be restarted.
  
- **Failed**: All containers in the pod have terminated. At least one container has terminated in failure (exited with a non-zero exit status).
  
- **CrashLoopBackOff**: A container in the pod is failing to start and is being restarted repeatedly by Kubernetes. This is often caused by configuration errors, such as invalid command-line arguments, or application errors that cause the container to exit immediately after start.
  
- **ImagePullBackOff**: Kubernetes is unable to pull the container image specified. This could be because the image does not exist, there are network issues, or there are credentials issues with the container registry.
  
- **ContainerCreating**: Kubernetes is in the process of creating the container for this pod.
  
- **Evicted**: The pod was evicted from its node due to resource constraints.
  
- **Terminating**: The pod is in the process of being terminated.
  
- **Unknown**: The state of the pod cannot be determined.

Use the following commands to monitor the status of your various Kubernetes objects:

```bash
kubectl get pods
kubectl get deployments
kubectl get services
kubectl get replicaSets
kubectl get configMaps
kubectl get secrets
kubectl get pvc
kubectl get ingress
```

When you encounter problematic statuses like `CrashLoopBackOff` or `ImagePullBackOff`, you can delve deeper by describing the pod for more detailed events and information:

```bash
kubectl describe pod <pod-name>
```

### Logs
If a pod isn't behaving as expected (such a having the wrong status), inspect its logs:

```bash
kubectl logs <pod-name>
```

Note that logs are specific to pods since they run the containers. Deployments, services, and other objects don't have logs. Instead, you'd inspect the logs of the pods that are managed by these objects.

### Describe Objects
Describing a Kubernetes object provides detailed information about the object, its configuration, and its current state. This is particularly helpful for troubleshooting.

```bash
kubectl describe pod <pod-name>
kubectl describe deployment <deployment-name>
kubectl describe service <service-name>
```

You can describe any Kubernetes object to fetch its detailed configuration and status. 

**Note:** When debugging, always start with the **Events** section, then inspect the status and conditions of each container within the pod. If the reasons aren't clear from the describe output, diving into the container logs is the next step.

#### Interpreting Describe Output
When you describe a Kubernetes object, you get a comprehensive report on its status and configuration. We'll show an example of a pod describe output and break down the important parts.

##### Example Output
<blockquote>
Name:             db-8497f5c586-jdhpj

Namespace:        default

Priority:         0

Service Account:  default

Node:             minikube/192.168.49.2

Start Time:       Wed, 18 Oct 2023 16:31:57 -0400

Labels:           io.kompose.network/vrooli-app=true

                  io.kompose.service=db

                  pod-template-hash=8497f5c586

Annotations:      kompose.cmd: kompose convert -f /root/Programming/Vrooli/scripts/../docker-compose-prod.yml.edit -o k8s-docker-compose-prod-env.yml

                  kompose.version: 1.28.0 (c4137012e)

Status:           Running

IP:               10.244.0.20

IPs:

  IP:           10.244.0.20

Controlled By:  ReplicaSet/db-8497f5c586

Containers:

  db:

    Container ID:  docker://f91ef130fa1608a884b3321e009b58d29f103b2aba1f8209b77fbc6bb8061035

    Image:         ankane/pgvector:v0.4.1

    Image ID:      docker-pullable://ankane/pgvector@sha256:f6eaf4a48794f70f616de15378172a22811c4f3c50a02ddf97ff68f3a242fed1
    
    Port:          5432/TCP
    
    Host Port:     0/TCP
    
    Args:
    
      /bin/sh
    
      -c
    
      /srv/app/scripts/getSecrets.sh production /tmp/secrets.$()$() DB_PASSWORD:POSTGRES_PASSWORD && . /tmp/secrets.$()$() && rm /tmp/secrets.$()$() && exec docker-entrypoint.sh postgres
    
    State:          Waiting
    
      Reason:       CrashLoopBackOff
    
    Last State:     Terminated
    
      Reason:       Error
    
      Exit Code:    127
    
      Started:      Wed, 18 Oct 2023 17:13:21 -0400
    
      Finished:     Wed, 18 Oct 2023 17:13:21 -0400
    
    Ready:          False
    
    Restart Count:  13
    
    Liveness:       exec [pg_isready -U site && psql -U site -d postgres -c 'SELECT 1'] delay=0s timeout=5s period=10s #success=1 #failure=5
    
    Environment:
    
      POSTGRES_USER:  site
    
      PROJECT_DIR:    /srv/app
    
    Mounts:
    
      /docker-entrypoint-initdb.d from db-claim1 (rw)
    
      /run/secrets/vrooli/prod from db-claim3 (rw)
    
      /srv/app/scripts from db-claim2 (rw)
    
      /var/lib/postgresql/data from db-claim0 (rw)
    
      /var/run/secrets/kubernetes.io/serviceaccount from kube-api-access-zhxp7 (ro)

Conditions:

  Type              Status

  Initialized       True 

  Ready             False 

  ContainersReady   False 

  PodScheduled      True 

Volumes:

  db-claim0:

    Type:       PersistentVolumeClaim (a reference to a PersistentVolumeClaim in the same namespace)

    ClaimName:  db-claim0

    ReadOnly:   false

  db-claim1:

    Type:       PersistentVolumeClaim (a reference to a PersistentVolumeClaim in the same namespace)

    ClaimName:  db-claim1

    ReadOnly:   false

  db-claim2:

    Type:       PersistentVolumeClaim (a reference to a PersistentVolumeClaim in the same namespace)

    ClaimName:  db-claim2

    ReadOnly:   false

  db-claim3:

    Type:       PersistentVolumeClaim (a reference to a PersistentVolumeClaim in the same namespace)

    ClaimName:  db-claim3

    ReadOnly:   false

  kube-api-access-zhxp7:

    Type:                    Projected (a volume that contains injected data from multiple sources)

    TokenExpirationSeconds:  3607

    ConfigMapName:           kube-root-ca.crt

    ConfigMapOptional:       <nil>

    DownwardAPI:             true

QoS Class:                   BestEffort

Node-Selectors:              <none>

Tolerations:                 node.kubernetes.io/not-ready:NoExecute op=Exists for 300s

                             node.kubernetes.io/unreachable:NoExecute op=Exists for 300s

Events:

  Type     Reason            Age                   From               Message

  ----     ------            ----                  ----               -------

  Warning  FailedScheduling  45m                   default-scheduler  0/1 nodes are available: persistentvolumeclaim "db-claim3" not found. preemption: 0/1 nodes are available: 1 No preemption victims found for incoming pod..

  Normal   Scheduled         45m                   default-scheduler  Successfully assigned default/db-8497f5c586-jdhpj to minikube

  Normal   Pulled            43m (x5 over 45m)     kubelet            Container image "ankane/pgvector:v0.4.1" already present on machine

  Normal   Created           43m (x5 over 45m)     kubelet            Created container db

  Normal   Started           43m (x5 over 45m)     kubelet            Started container db

  Warning  BackOff           5m1s (x192 over 45m)  kubelet            Back-off restarting failed container db in pod db-8497f5c586-jdhpj_default(f9cda42f-a130-45f0-8b1a-ded39107ea65)
</blockquote>

##### Output Breakdown
Let's break down the important parts of the output:

1. **General Information**: 
   - **Name**, **Namespace**, **Priority**, etc., provide metadata about the pod.
   - **Start Time**: Indicates when the pod was started.
   - **Labels** & **Annotations**: Useful for filtering and categorizing pods.

2. **Status**: Indicates the overall status of the pod. In the example, the status is `Running`, but the pod is not healthy (as we'll see further down).

3. **Containers**: 
   - Provides details of each container within the pod.
   - **Image**: The Docker image the container is running.
   - **State**: The current state of the container. Here, the container is `Waiting` due to `CrashLoopBackOff`, which means the container is repeatedly crashing and Kubernetes is trying to restart it.
   - **Last State**: Provides info about the previous state of the container. In this case, the container `Terminated` due to an `Error` with exit code `127` (commonly meaning a command inside the container didn't execute correctly).
   - **Restart Count**: Indicates how many times the container has been restarted, which can be a sign of instability.

4. **Conditions**: 
   - Shows conditions the pod needs to meet. For example, `Initialized` being `True` means initialization containers (if any) have successfully completed.
   - `Ready` being `False` is a sign that the pod isn't ready to handle traffic or execute its main function.

5. **Volumes**: 
   - Lists all the storage volumes attached to the pod. These can be mounts from the host node, PersistentVolumeClaims, or other types of storage resources.

6. **Events**: 
   - Critical for debugging. Lists the sequence of events related to the pod's lifecycle.
   - In the example, there are several warnings, like `FailedScheduling` (initially, it couldn't find a node to schedule due to a missing PVC) and `BackOff` (indicating repeated container crashes).

##### Output Interpretation
Here's what one might infer from the example's describe output:

- The `CrashLoopBackOff` status is often due to application errors. Check the container logs with `kubectl logs <pod-name>` for more details.
  
- `Exit Code: 127` typically indicates that the command in the container couldn't be found or executed. It might be due to a misconfigured entrypoint or command in the container image.
  
- Missing PVCs, like the `FailedScheduling` warning about `persistentvolumeclaim "db-claim3" not found`, can prevent pods from starting. Ensure all required PVCs are created and bound to a PersistentVolume.

- High `Restart Count` values indicate that the container is frequently crashing. It's usually coupled with other error messages that provide insight into why the crashes are occurring.

### Accessing Applications
In Kubernetes, applications are typically run inside pods. However, these pods are isolated and don't expose their services to the outside world by default. To make them accessible, we use something called a "Service".

Here's a simple breakdown:

1. **Pods**: Think of pods as your actual running applications. They do the work, but by themselves, they keep their work private and hidden.

2. **Services**: These are like gateways that can open up access to those private pods. There are different types of services based on how you want to expose your application:

   - **ClusterIP (default)**: Exposes the service on an internal IP within the cluster, making it reachable only from within the cluster.
   
   - **NodePort**: Exposes the service on each node's IP at a static port. You can then access the service through `<NodeIP>:<NodePort>`.
   
   - **LoadBalancer**: Exposes the service externally using a cloud provider's load balancer. This will assign an external IP to the service.
   
   - **ExternalName**: Maps the service to a specified external name.

With Minikube, things are made a bit simpler. Even if you've set up your service with a type like `LoadBalancer`, which usually requires a cloud provider's infrastructure, Minikube can still emulate this on your local machine.

To access services in Minikube, use the following command:

```bash
minikube service <service-name>
```

This command will open up your default web browser and take you to the service, making it super easy to test your applications locally!

### Executing Commands Inside a Pod
Interact directly with a running pod:

```bash
kubectl exec -it <pod-name> -- /bin/sh
```

This is useful for debugging, checking the internal state, or performing administrative tasks inside the container. It's like SSHing into a virtual machine. Note that running commands inside a pod is specific to pods, since they encapsulate the containers.

### 8. Clean Up
Once done with testing, you can clean up the deployed objects and stop Minikube using:

```bash
kubectl delete -f <generated-file>.yaml
minikube stop
```

## Troubleshooting
When things don't go as planned:

1. **Check Pod Status**: `kubectl describe pod <pod-name>` can give insights into why a pod might not be running.  
2. **View Logs**: `kubectl logs <pod-name>` to view the logs of a container in the pod.  
3. **Network Issues**: Check `Service` and `Ingress` configurations. Ensure network policies allow desired traffic.  
4. **Resource Constraints**: 
   - **Kubernetes Level**: Use `kubectl top` to get an overview of node and pod resource usage:
     ```bash
     kubectl top nodes
     kubectl top pods
     ```
   - **Node Level**:
     - **CPU Usage**: Use `top` or `htop` (needs to be installed separately) to monitor CPU utilization on the node. They provide a real-time view of the processes and their CPU usage.
     - **Memory Usage**: 
       - `free -h`: Gives a quick overview of the total, used, and available memory.
       - `vmstat`: Provides a snapshot of memory, processes, IO, and CPU activity.
     - **Storage Usage**: 
       - `df -h`: Shows the disk usage of all mounted filesystems.
       - `du -sh <directory>`: Summarizes disk usage of a particular directory.

   Ensure that your nodes have sufficient resources to run the pods. If a node is consistently running at high resource utilization, consider adding more nodes to your cluster or resizing existing ones to better specifications.

## Resource Monitoring
Kubernetes provides resource metrics that can be queried using `kubectl top`:

- `kubectl top nodes` to view nodes' resource utilization.
- `kubectl top pods` to view resource utilization by pods.

For detailed metrics and visualization, consider deploying monitoring solutions like Prometheus and Grafana to the cluster.
