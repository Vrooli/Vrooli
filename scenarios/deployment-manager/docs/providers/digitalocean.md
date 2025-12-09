# Provider Notes · DigitalOcean

> **Status:** Reference material only. Use this guide when planning Tier 4 (SaaS/Cloud) deployments or when deployment-manager needs provider-specific inputs. It does **not** represent a supported end-to-end deployment pipeline yet.

This document is the original DigitalOcean walkthrough (pricing, `doctl`, cluster creation). Keep it handy for infrastructure planning, but pair it with the [Tier 4](../tiers/tier-4-saas.md) plan and the future `scenario-to-cloud` automation.

## Cloudflare Tunnel Quick Reference

Tier 1 often runs on a DigitalOcean VPS that also hosts app-monitor. Instead of duplicating instructions here, follow the dedicated [Cloudflare tunnel guide](cloudflare-tunnel.md). It walks through installation, DNS routing, and service setup so your DigitalOcean box remains reachable without exposing ports.

---

## Manual DigitalOcean Kubernetes Setup Guide

This guide walks through setting up a production Kubernetes cluster on DigitalOcean step-by-step, with cost transparency and decision points.

## Step 1: Cost Analysis & Planning

### DigitalOcean Kubernetes Pricing (as of 2025)

**Control Plane:** FREE (DigitalOcean doesn't charge for the Kubernetes control plane)

**Worker Nodes (you pay for these):**
- **s-1vcpu-2gb**: $12/month each (~$36/month for 3 nodes)
- **s-2vcpu-4gb**: $24/month each (~$72/month for 3 nodes) ⭐ **Recommended**
- **s-4vcpu-8gb**: $48/month each (~$144/month for 3 nodes)

**Additional Services:**
- **Load Balancer**: $12/month (for external access)
- **Block Storage**: $0.10/GB/month (for persistent volumes)
- **Managed PostgreSQL**: $15/month (db-s-1vcpu-1gb)
- **Managed Redis**: $15/month (db-s-1vcpu-1gb)

**Total Monthly Cost Estimate:**
- **Minimal setup**: ~$75/month (3 small nodes + LB + databases)
- **Recommended setup**: ~$114/month (3 medium nodes + LB + databases)
- **High performance**: ~$189/month (3 large nodes + LB + databases)

## Step 2: DigitalOcean Account Setup

### 2.1 Create Account & Get API Token
1. Go to https://cloud.digitalocean.com/
2. Sign up or log in
3. Navigate to **API** → **Tokens**
4. Generate new token with **Read** and **Write** scopes
5. **Save the token securely** - you'll need it for CLI access

### 2.2 Install doctl CLI
```bash
# Download and install doctl
curl -sL https://github.com/digitalocean/doctl/releases/download/v1.104.0/doctl-1.104.0-linux-amd64.tar.gz | tar -xzv
sudo mv doctl /usr/local/bin

# Verify installation
doctl version

# Authenticate (you'll be prompted for your API token)
doctl auth init
```

## Step 3: Explore Available Options

### 3.1 Check Available Regions
```bash
# List all available regions
doctl kubernetes options regions

# Output will show regions like:
# Slug    Name               Available
# nyc1    New York 1         true
# nyc3    New York 3         true
# sfo3    San Francisco 3    true
# lon1    London 1           true
```

**Recommendation:** Choose `nyc1` (closest to your current server at 165.227.84.109)

### 3.2 Check Available Node Sizes
```bash
# List available Kubernetes node sizes
doctl kubernetes options sizes

# Output will show sizes like:
# Slug           Memory    VCPUs    Disk     Price Monthly
# s-1vcpu-2gb    2048      1        50       12.00
# s-2vcpu-4gb    4096      2        80       24.00
# s-4vcpu-8gb    8192      4        160      48.00
```

### 3.3 Check Available Kubernetes Versions
```bash
# List available Kubernetes versions
doctl kubernetes options versions

# Output will show versions like:
# Slug            Kubernetes Version
# 1.28.2-do.0     1.28.2
# 1.29.1-do.0     1.29.1
# 1.30.0-do.0     1.30.0
```

## Step 4: Decision Point - Cluster Configuration

Based on your application requirements, here are recommended configurations:

### Option A: Development/Testing ($36/month + services)
```bash
CLUSTER_CONFIG="development"
REGION="nyc1"
NODE_SIZE="s-1vcpu-2gb"
NODE_COUNT=3
K8S_VERSION="1.30.0-do.0"  # Use latest stable
```

### Option B: Production (Recommended) ($72/month + services)
```bash
CLUSTER_CONFIG="production"
REGION="nyc1"
NODE_SIZE="s-2vcpu-4gb"
NODE_COUNT=3
K8S_VERSION="1.30.0-do.0"
```

### Option C: High Performance ($144/month + services)
```bash
CLUSTER_CONFIG="high-performance"
REGION="nyc1" 
NODE_SIZE="s-4vcpu-8gb"
NODE_COUNT=3
K8S_VERSION="1.30.0-do.0"
```

## Step 5: Create the Cluster (Manual Commands)

### 5.1 Set Your Configuration
```bash
# Choose your configuration (edit these values based on your choice above)
CLUSTER_NAME="vrooli-prod"
REGION="nyc1"
NODE_SIZE="s-2vcpu-4gb"     # Change this based on your choice
NODE_COUNT=3
K8S_VERSION="1.30.0-do.0"   # Update to latest from step 3.3
```

### 5.2 Preview the Cluster Creation
```bash
# This shows what will be created WITHOUT actually creating it
echo "Cluster Configuration Preview:"
echo "============================"
echo "Name: $CLUSTER_NAME"
echo "Region: $REGION"
echo "Node Size: $NODE_SIZE"
echo "Node Count: $NODE_COUNT"
echo "Kubernetes Version: $K8S_VERSION"
echo ""
echo "Estimated Monthly Cost:"
case $NODE_SIZE in
  "s-1vcpu-2gb")
    echo "  Worker Nodes: \$12 × $NODE_COUNT = \$$(($NODE_COUNT * 12))/month"
    ;;
  "s-2vcpu-4gb")
    echo "  Worker Nodes: \$24 × $NODE_COUNT = \$$(($NODE_COUNT * 24))/month"
    ;;
  "s-4vcpu-8gb")
    echo "  Worker Nodes: \$48 × $NODE_COUNT = \$$(($NODE_COUNT * 48))/month"
    ;;
esac
echo "  Load Balancer: \$12/month (when created)"
echo "  Control Plane: \$0/month (free)"
```

### 5.3 Create the Cluster
```bash
# WARNING: This will start billing for the cluster
# Only run this when you're ready to proceed

echo "Creating cluster: $CLUSTER_NAME"
echo "This will take 5-10 minutes..."

doctl kubernetes cluster create $CLUSTER_NAME \
  --region $REGION \
  --size $NODE_SIZE \
  --count $NODE_COUNT \
  --version $K8S_VERSION \
  --auto-upgrade=true \
  --maintenance-window="sunday=02:00" \
  --surge-upgrade=true \
  --ha=false \
  --wait

echo "Cluster creation completed!"
```

### 5.4 Configure kubectl
```bash
# Download cluster configuration
doctl kubernetes cluster kubeconfig save $CLUSTER_NAME

# Verify connection
kubectl cluster-info
kubectl get nodes

# You should see 3 nodes in Ready state
```

## Step 6: Monitor Cluster Creation

### 6.1 Check Cluster Status
```bash
# Check cluster status
doctl kubernetes cluster get $CLUSTER_NAME

# Check nodes in DigitalOcean dashboard
# Go to: https://cloud.digitalocean.com/kubernetes/clusters
```

### 6.2 Verify Cluster Health
```bash
# Check node status
kubectl get nodes -o wide

# Check system pods
kubectl get pods -n kube-system

# Check cluster info
kubectl cluster-info
```

## Step 7: Install Essential Components

### 7.1 Install cert-manager (for SSL certificates)
```bash
echo "Installing cert-manager..."
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.2/cert-manager.yaml

# Wait for cert-manager to be ready
echo "Waiting for cert-manager to be ready..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/instance=cert-manager -n cert-manager --timeout=300s

echo "cert-manager installed successfully!"
```

### 7.2 Install nginx-ingress (for load balancing)
```bash
echo "Installing nginx-ingress controller..."
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/do/deploy.yaml

# This creates a DigitalOcean Load Balancer (starts billing $12/month)
echo "Waiting for load balancer to get external IP..."

# Monitor load balancer creation
kubectl get svc -n ingress-nginx -w

# This will show "pending" initially, then show an external IP when ready
# Press Ctrl+C when you see an external IP assigned
```

### 7.3 Get External IP Address
```bash
# Get the external IP of your load balancer
EXTERNAL_IP=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

if [[ -n "$EXTERNAL_IP" ]]; then
  echo "🌐 Load Balancer External IP: $EXTERNAL_IP"
  echo ""
  echo "📋 DNS Configuration Required:"
  echo "   Create A records pointing to $EXTERNAL_IP:"
  echo "   vrooli.com -> $EXTERNAL_IP"
  echo "   www.vrooli.com -> $EXTERNAL_IP"
  echo "   app.vrooli.com -> $EXTERNAL_IP"
else
  echo "⚠️  External IP not yet assigned. Check again in a few minutes:"
  echo "   kubectl get svc -n ingress-nginx ingress-nginx-controller"
fi
```

## Step 8: Create Managed Databases (Optional but Recommended)

### 8.1 Estimate Database Costs
```bash
echo "Database Pricing Options:"
echo "========================"
echo "PostgreSQL:"
echo "  db-s-1vcpu-1gb:  \$15/month (1 vCPU, 1GB RAM, 10GB storage)"
echo "  db-s-2vcpu-2gb:  \$30/month (2 vCPU, 2GB RAM, 25GB storage)"
echo "  db-s-4vcpu-8gb:  \$120/month (4 vCPU, 8GB RAM, 115GB storage)"
echo ""
echo "Redis:"
echo "  db-s-1vcpu-1gb:  \$15/month (1 vCPU, 1GB RAM)"
echo "  db-s-2vcpu-2gb:  \$30/month (2 vCPU, 2GB RAM)"
echo ""
echo "Recommendation: Start with db-s-1vcpu-1gb for both (\$30/month total)"
```

### 8.2 Create PostgreSQL Database
```bash
# Create managed PostgreSQL (this starts billing $15/month)
echo "Creating managed PostgreSQL database..."

DB_NAME="vrooli-postgres"
DB_SIZE="db-s-1vcpu-1gb"  # Change to db-s-2vcpu-2gb for better performance

doctl databases create $DB_NAME \
  --engine pg \
  --size $DB_SIZE \
  --region $REGION \
  --num-nodes 1

echo "PostgreSQL database creation initiated..."
echo "This will take 5-10 minutes to complete."
```

### 8.3 Create Redis Database
```bash
# Create managed Redis (this starts billing $15/month)
echo "Creating managed Redis database..."

REDIS_NAME="vrooli-redis"
REDIS_SIZE="db-s-1vcpu-1gb"  # Change to db-s-2vcpu-2gb for better performance

doctl databases create $REDIS_NAME \
  --engine redis \
  --size $REDIS_SIZE \
  --region $REGION \
  --num-nodes 1

echo "Redis database creation initiated..."
echo "This will take 5-10 minutes to complete."
```

### 8.4 Monitor Database Creation
```bash
# Check database status
echo "Checking database status..."
doctl databases list

# Wait for databases to show "online" status
echo "Waiting for databases to come online..."
while true; do
  PG_STATUS=$(doctl databases get $DB_NAME --format Status --no-header 2>/dev/null || echo "creating")
  REDIS_STATUS=$(doctl databases get $REDIS_NAME --format Status --no-header 2>/dev/null || echo "creating")
  
  echo "PostgreSQL: $PG_STATUS, Redis: $REDIS_STATUS"
  
  if [[ "$PG_STATUS" == "online" && "$REDIS_STATUS" == "online" ]]; then
    echo "✅ Both databases are online!"
    break
  fi
  
  sleep 30
done
```

## Step 9: Get Database Connection Information

### 9.1 PostgreSQL Connection Details
```bash
# Get PostgreSQL connection info
echo "PostgreSQL Connection Information:"
echo "================================="
PG_INFO=$(doctl databases connection $DB_NAME --format Host,Port,User,Password,Database --no-header)
echo "Connection String: $PG_INFO"

# Parse connection details
PG_HOST=$(echo $PG_INFO | awk '{print $1}')
PG_PORT=$(echo $PG_INFO | awk '{print $2}')
PG_USER=$(echo $PG_INFO | awk '{print $3}')
PG_PASS=$(echo $PG_INFO | awk '{print $4}')
PG_DB=$(echo $PG_INFO | awk '{print $5}')

echo "Host: $PG_HOST"
echo "Port: $PG_PORT"
echo "User: $PG_USER"
echo "Password: $PG_PASS"
echo "Database: $PG_DB"
```

### 9.2 Redis Connection Details
```bash
# Get Redis connection info
echo ""
echo "Redis Connection Information:"
echo "============================"
REDIS_INFO=$(doctl databases connection $REDIS_NAME --format Host,Port,Password --no-header)
echo "Connection String: $REDIS_INFO"

# Parse connection details
REDIS_HOST=$(echo $REDIS_INFO | awk '{print $1}')
REDIS_PORT=$(echo $REDIS_INFO | awk '{print $2}')
REDIS_PASS=$(echo $REDIS_INFO | awk '{print $3}')

echo "Host: $REDIS_HOST"
echo "Port: $REDIS_PORT"
echo "Password: $REDIS_PASS"
```

## Step 10: Current Status Summary

### 10.1 Infrastructure Created
```bash
echo "🎉 Infrastructure Summary:"
echo "=========================="
echo "✅ Kubernetes Cluster: $CLUSTER_NAME ($NODE_COUNT × $NODE_SIZE nodes)"
echo "✅ Load Balancer: nginx-ingress with external IP"
echo "✅ SSL Management: cert-manager installed"
echo "✅ PostgreSQL: $DB_NAME ($DB_SIZE)"
echo "✅ Redis: $REDIS_NAME ($REDIS_SIZE)"
echo ""
echo "💰 Monthly Costs:"
case $NODE_SIZE in
  "s-1vcpu-2gb") NODE_COST=$(($NODE_COUNT * 12)) ;;
  "s-2vcpu-4gb") NODE_COST=$(($NODE_COUNT * 24)) ;;
  "s-4vcpu-8gb") NODE_COST=$(($NODE_COUNT * 48)) ;;
esac
echo "   Kubernetes Nodes: \$$NODE_COST/month"
echo "   Load Balancer: \$12/month"
echo "   PostgreSQL: \$15/month"
echo "   Redis: \$15/month"
echo "   ===================="
TOTAL_COST=$((NODE_COST + 12 + 15 + 15))
echo "   TOTAL: \$$TOTAL_COST/month"
```

### 10.2 Next Steps
```bash
echo ""
echo "📋 Next Steps:"
echo "=============="
echo "1. Configure DNS A records (using your domain registrar):"
echo "   vrooli.com -> $EXTERNAL_IP"
echo "   www.vrooli.com -> $EXTERNAL_IP"
echo "   app.vrooli.com -> $EXTERNAL_IP"
echo ""
echo "2. Deploy the Vrooli application:"
echo "   ./scripts/deploy/deploy-to-existing-cluster.sh"
echo ""
echo "3. Monitor the deployment:"
echo "   kubectl get pods -n vrooli"
echo "   kubectl get ingress -n vrooli"
echo ""
echo "4. Test access after DNS propagation:"
echo "   curl -I https://vrooli.com"
```

## Cleanup Instructions (If Needed)

### To Delete Everything and Stop Billing:
```bash
# Delete the Kubernetes cluster
doctl kubernetes cluster delete $CLUSTER_NAME

# Delete the databases
doctl databases delete $DB_NAME
doctl databases delete $REDIS_NAME

# Verify deletion
doctl kubernetes cluster list
doctl databases list
```

## Monitoring Costs

### Check Current Usage:
1. Go to https://cloud.digitalocean.com/billing
2. View current month usage
3. Set up billing alerts if desired

### Cost Optimization Tips:
- Start with smaller node sizes and scale up if needed
- Use horizontal pod autoscaling to handle traffic spikes
- Consider DigitalOcean's reserved instances for 1-year commitments (20% discount)
- Monitor resource usage and right-size your nodes

## Security Considerations

### Immediate Security Tasks:
1. Enable DigitalOcean Firewall for cluster nodes
2. Set up network policies in Kubernetes
3. Enable audit logging
4. Configure RBAC (Role-Based Access Control)
5. Use secrets management for sensitive data

---

**You now have complete visibility into costs and can proceed step-by-step at your own pace!**
