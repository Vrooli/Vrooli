# True Remote Production Deployment Guide

## Overview
This guide walks through deploying Vrooli to a real cloud environment with external access, SSL certificates, and production-grade infrastructure.

## Quick Deployment (Pre-configured Environment)

If you already have everything set up (Kubernetes cluster, operators installed, DNS configured), use these commands for rapid deployment:

```bash
# Complete production deployment sequence
cd /root/Vrooli

# 1. Set production kubeconfig
export KUBECONFIG=/root/Vrooli/k8s/kubeconfig-vrooli-prod.yaml

# 2. Verify cluster connection
kubectl config current-context
kubectl cluster-info

# 3. Check prerequisites (optional but recommended)
./scripts/helpers/deploy/k8s-prerequisites.sh --check-only

# 4. Build production artifacts
vrooli build
# Note: Production settings are configured in .vrooli/service.json

# 5. Deploy to production
vrooli deploy
# Note: Deployment configuration is in .vrooli/service.json
```

**Notes:**
- Build process can take 20+ minutes
- Ensure version number matches your release
- Always verify correct cluster connection before deploying
- The deploy script uses the current kubectl context

## Prerequisites Checklist

### ✅ Already Available
- [x] Docker images built and pushed to Docker Hub
- [x] Domain configured: vrooli.com  
- [x] Production environment variables in `.env-prod`
- [x] AWS credentials for S3 storage
- [x] Existing DigitalOcean infrastructure

### 🔧 Required Setup
- [ ] Cloud Kubernetes cluster
- [ ] SSL certificates for HTTPS
- [ ] External load balancer
- [ ] Production database (managed)
- [ ] DNS configuration

## Deployment Options

### Option 1: DigitalOcean Kubernetes (DOKS) - Recommended
*Recommended since your existing server is on DigitalOcean*

#### Step 1: Install doctl (DigitalOcean CLI)
```bash
# Install doctl
curl -sL https://github.com/digitalocean/doctl/releases/download/v1.104.0/doctl-1.104.0-linux-amd64.tar.gz | tar -xzv
sudo mv doctl /usr/local/bin

# Authenticate (requires DigitalOcean API token)
doctl auth init
```

#### Step 2: Create DOKS Cluster
```bash
# Create production Kubernetes cluster
doctl kubernetes cluster create vrooli-prod \
  --region nyc1 \
  --size s-2vcpu-4gb \
  --count 3 \
  --auto-upgrade=true \
  --maintenance-window="sunday=02:00" \
  --surge-upgrade=true \
  --ha=true

# Connect kubectl to cluster
doctl kubernetes cluster kubeconfig save vrooli-prod
```

#### Step 3: Install Required Operators
```bash
# Install cert-manager for SSL
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.2/cert-manager.yaml

# Install nginx-ingress for load balancing
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/do/deploy.yaml

# Wait for load balancer IP
kubectl get svc -n ingress-nginx
```

#### Step 4: Configure DNS
```bash
# Get external load balancer IP
EXTERNAL_IP=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

echo "Configure DNS A records:"
echo "vrooli.com -> $EXTERNAL_IP"
echo "www.vrooli.com -> $EXTERNAL_IP"
echo "app.vrooli.com -> $EXTERNAL_IP"
```

#### Step 5: Deploy Application
```bash
# Apply production values
helm install vrooli ./k8s/chart \
  --namespace vrooli \
  --create-namespace \
  --values k8s/chart/values-prod.yaml \
  --set image.tag=2.0.0 \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set productionDomain=vrooli.com
```

### Option 2: AWS EKS
*If you prefer AWS since you have credentials*

#### Step 1: Install AWS CLI & eksctl
```bash
# Install AWS CLI
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install

# Configure AWS (replace with your own credentials)
aws configure set aws_access_key_id $AWS_ACCESS_KEY_ID 
aws configure set aws_secret_access_key $AWS_SECRET_ACCESS_KEY  
aws configure set region us-east-1

# Install eksctl
curl --silent --location "https://github.com/weaveworks/eksctl/releases/latest/download/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp
sudo mv /tmp/eksctl /usr/local/bin
```

#### Step 2: Create EKS Cluster
```bash
# Create production EKS cluster
eksctl create cluster \
  --name vrooli-prod \
  --region us-east-1 \
  --nodegroup-name standard-workers \
  --node-type m5.large \
  --nodes 3 \
  --nodes-min 1 \
  --nodes-max 6 \
  --managed

# Install AWS Load Balancer Controller
kubectl apply -k "github.com/aws/eks-charts/stable/aws-load-balancer-controller//crds?ref=master"

helm repo add eks https://aws.github.io/eks-charts
helm install aws-load-balancer-controller eks/aws-load-balancer-controller \
  -n kube-system \
  --set clusterName=vrooli-prod \
  --set serviceAccount.create=false \
  --set serviceAccount.name=aws-load-balancer-controller
```

#### Step 3: Configure Route 53 DNS
```bash
# Create hosted zone (if not exists)
aws route53 create-hosted-zone --name vrooli.com --caller-reference $(date +%s)

# Get load balancer DNS
ALB_DNS=$(kubectl get ingress vrooli-ingress -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')

# Create DNS records pointing to ALB
aws route53 change-resource-record-sets --hosted-zone-id YOUR_ZONE_ID --change-batch '{
  "Changes": [{
    "Action": "CREATE",
    "ResourceRecordSet": {
      "Name": "vrooli.com",
      "Type": "CNAME",
      "TTL": 300,
      "ResourceRecords": [{"Value": "'$ALB_DNS'"}]
    }
  }]
}'
```

### Option 3: Google Cloud GKE
*Alternative cloud provider option*

#### Step 1: Install gcloud CLI
```bash
# Install gcloud CLI
curl https://sdk.cloud.google.com | bash
exec -l $SHELL
gcloud init
```

#### Step 2: Create GKE Cluster
```bash
# Create production GKE cluster
gcloud container clusters create vrooli-prod \
  --zone us-east1-b \
  --machine-type n1-standard-2 \
  --num-nodes 3 \
  --enable-autoscaling \
  --min-nodes 1 \
  --max-nodes 10 \
  --enable-autorepair \
  --enable-autoupgrade

# Get credentials
gcloud container clusters get-credentials vrooli-prod --zone us-east1-b
```

## Production Configuration Requirements

### 1. SSL Certificates
```yaml
# cert-manager ClusterIssuer for Let's Encrypt
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: matthalloran8@gmail.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
```

### 2. Production Database
```bash
# Option A: Managed PostgreSQL (DigitalOcean)
doctl databases create vrooli-db --engine pg --size db-s-1vcpu-1gb --region nyc1

# Option B: AWS RDS
aws rds create-db-instance \
  --db-instance-identifier vrooli-prod-db \
  --db-instance-class db.t3.micro \
  --engine postgres \
  --master-username site \
  --master-user-password YnrMnA2DTcN3ae \
  --allocated-storage 20
```

### 3. Production Redis
```bash
# Option A: Managed Redis (DigitalOcean)
doctl databases create vrooli-redis --engine redis --size db-s-1vcpu-1gb --region nyc1

# Option B: AWS ElastiCache
aws elasticache create-cache-cluster \
  --cache-cluster-id vrooli-prod-redis \
  --cache-node-type cache.t3.micro \
  --engine redis \
  --num-cache-nodes 1
```

### 4. External Secrets Management
```bash
# Install External Secrets Operator
helm repo add external-secrets https://charts.external-secrets.io
helm install external-secrets external-secrets/external-secrets -n external-secrets-system --create-namespace

# Configure AWS Secrets Manager or HashiCorp Vault
```

## Monitoring & Observability

### 1. Install Prometheus & Grafana
```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack
```

### 2. Install Logging (ELK Stack)
```bash
helm repo add elastic https://helm.elastic.co
helm install elasticsearch elastic/elasticsearch
helm install kibana elastic/kibana
helm install filebeat elastic/filebeat
```

## Security Hardening

### 1. Network Policies
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: vrooli-network-policy
spec:
  podSelector:
    matchLabels:
      app: vrooli
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: nginx-ingress
```

### 2. Pod Security Standards
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: vrooli
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

## Backup & Disaster Recovery

### 1. Database Backups
```bash
# Automated backups with Velero
helm repo add vmware-tanzu https://vmware-tanzu.github.io/helm-charts/
helm install velero vmware-tanzu/velero \
  --set-file credentials.secretContents.cloud=./credentials-velero
```

### 2. Application State Backup
```bash
# Backup application data
kubectl create job backup-job --from=cronjob/backup-cronjob
```

## Cost Optimization

### 1. Resource Limits
```yaml
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 512Mi
```

### 2. Horizontal Pod Autoscaling
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: vrooli-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: vrooli-server
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

## Post-Deployment Checklist

- [ ] SSL certificates valid and auto-renewing
- [ ] DNS properly configured and propagated
- [ ] Health checks passing
- [ ] Monitoring dashboards configured
- [ ] Backup systems operational
- [ ] Security scans completed
- [ ] Performance testing completed
- [ ] Documentation updated

## Troubleshooting

### Common Issues
1. **SSL Certificate Issues**: Check cert-manager logs
2. **DNS Propagation**: Use `dig vrooli.com` to verify
3. **Load Balancer Issues**: Check ingress controller logs
4. **Database Connectivity**: Verify security groups/firewalls

### Commands for Debugging
```bash
# Check pod logs
kubectl logs -f deployment/vrooli-server

# Check ingress status
kubectl get ingress -A

# Check certificates
kubectl get certificates

# Check external secrets
kubectl get externalsecrets
```