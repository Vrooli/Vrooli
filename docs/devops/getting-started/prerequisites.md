# Prerequisites & System Requirements

This document provides consolidated setup requirements and tool installation guides for Vrooli development and deployment.

## System Requirements

### Minimum Requirements (Development)
- **CPU**: 4 cores
- **RAM**: 8GB
- **Storage**: 50GB SSD
- **OS**: Linux (Ubuntu 22.04 LTS), macOS (12+), or Windows (WSL2)
- **Network**: Stable internet connection

### Recommended Requirements (Production)
- **CPU**: 8+ cores  
- **RAM**: 16GB+
- **Storage**: 100GB+ SSD
- **OS**: Ubuntu 22.04 LTS
- **Network**: 1Gbps+ connection

## Required Tools Installation

### Core Tools

#### Git
```bash
# Ubuntu/Debian
sudo apt update && sudo apt install git

# macOS
brew install git

# Verify installation
git --version
```

#### Node.js (via Package Manager)
```bash
# Ubuntu/Debian - install via NodeSource
curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -
sudo apt-get install -y nodejs

# macOS
brew install node

# Verify installation
node --version
npm --version
```

#### PNPM (Package Manager)
```bash
# Install via corepack (recommended)
corepack enable
corepack prepare pnpm@latest --activate

# Or install globally
npm install -g pnpm

# Or via shell script
curl -fsSL https://get.pnpm.io/install.sh | sh -

# Verify installation (should be 8.x or later)
pnpm --version
```

### Development Tools

#### Docker & Docker Compose
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install ca-certificates curl gnupg lsb-release

# Add Docker's official GPG key
sudo mkdir -m 0755 -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

# Set up repository
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Install Docker Engine
sudo apt update
sudo apt install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# macOS
brew install --cask docker

# Add user to docker group (Linux)
sudo usermod -aG docker $USER
newgrp docker

# Verify installation
docker --version
docker compose version
```

#### PostgreSQL Client Tools
```bash
# Ubuntu/Debian
sudo apt install postgresql-client

# macOS
brew install postgresql

# Verify installation
psql --version
```

#### Redis Client Tools
```bash
# Ubuntu/Debian
sudo apt install redis-tools

# macOS
brew install redis

# Verify installation
redis-cli --version
```

### Build Tools

#### Build Essentials
```bash
# Ubuntu/Debian
sudo apt install build-essential python3-dev

# macOS (Xcode Command Line Tools)
xcode-select --install

# Verify installation
gcc --version
python3 --version
```

#### TypeScript & ESLint (Global)
```bash
# Install globally for editor support
pnpm add -g typescript eslint

# Verify installation
tsc --version
eslint --version
```

### Optional Development Tools

#### Visual Studio Code
```bash
# Ubuntu/Debian (via snap)
sudo snap install --classic code

# macOS
brew install --cask visual-studio-code

# Or download from https://code.visualstudio.com/
```

#### Database Administration
```bash
# pgAdmin (PostgreSQL)
# Download from https://www.pgadmin.org/download/

# Redis Desktop Manager alternatives
# RedisInsight: https://redis.com/redis-enterprise/redis-insight/
```

## Kubernetes Tools (For K8s Development)

### kubectl
```bash
# Linux
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# macOS
brew install kubectl

# Verify installation
kubectl version --client
```

### Helm
```bash
# Linux
curl https://get.helm.sh/helm-v3.12.0-linux-amd64.tar.gz | tar xz
sudo mv linux-amd64/helm /usr/local/bin/

# macOS
brew install helm

# Verify installation
helm version
```

### Minikube (For Local K8s Development)
```bash
# Linux
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube

# macOS
brew install minikube

# Verify installation
minikube version
```

## Cloud Provider Tools (Optional)

### AWS CLI
```bash
# Linux/macOS
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install

# macOS (via Homebrew)
brew install awscli

# Verify installation
aws --version
```

### Google Cloud CLI
```bash
# Install instructions: https://cloud.google.com/sdk/docs/install

# Verify installation
gcloud version
```

## Testing Tools

### BATS (Bash Automated Testing System)
```bash
# Installed automatically by setup scripts
# Manual installation:
git clone https://github.com/bats-core/bats-core.git
cd bats-core
sudo ./install.sh /usr/local

# Verify installation
bats --version
```

### ShellCheck (Shell Script Linting)
```bash
# Ubuntu/Debian
sudo apt install shellcheck

# macOS
brew install shellcheck

# Verify installation
shellcheck --version
```

## Environment Verification

After installing all prerequisites, run this verification script:

```bash
#!/bin/bash
echo "=== Vrooli Prerequisites Verification ==="

# Core tools
echo "Git: $(git --version)"
echo "Node.js: $(node --version)"
echo "PNPM: $(pnpm --version)"
echo "Docker: $(docker --version)"
echo "Docker Compose: $(docker compose version)"

# Database tools
echo "PostgreSQL Client: $(psql --version)"
echo "Redis CLI: $(redis-cli --version)"

# Build tools
echo "GCC: $(gcc --version | head -1)"
echo "Python3: $(python3 --version)"

# Optional tools
if command -v kubectl &> /dev/null; then
    echo "kubectl: $(kubectl version --client --short)"
fi

if command -v helm &> /dev/null; then
    echo "Helm: $(helm version --short)"
fi

if command -v minikube &> /dev/null; then
    echo "Minikube: $(minikube version --short)"
fi

echo "=== Verification Complete ==="
```

## Troubleshooting Common Installation Issues

### Permission Issues
```bash
# If you encounter permission errors with Docker
sudo usermod -aG docker $USER
newgrp docker

# If you encounter npm permission errors
mkdir ~/.npm-global
npm config set prefix '~/.npm-global'
echo 'export PATH=~/.npm-global/bin:$PATH' >> ~/.bashrc
source ~/.bashrc
```

### Path Issues
```bash
# If commands are not found, check PATH
echo $PATH

# Add to PATH if needed (add to ~/.bashrc or ~/.zshrc)
export PATH="/usr/local/bin:$PATH"
export PATH="$HOME/.local/bin:$PATH"
```

### PNPM Issues
```bash
# If PNPM is not found after installation
echo 'export PATH="$HOME/.local/share/pnpm:$PATH"' >> ~/.bashrc
source ~/.bashrc

# Clear PNPM cache if needed
pnpm store prune
```

## Next Steps

After completing prerequisites installation:

1. **Development Setup**: See [Development Environment Setup](../development-environment.md)
2. **Server Deployment**: See [Server Deployment Guide](../server-deployment.md)
3. **Build System**: See [Build System Guide](../build-system.md)
4. **Troubleshooting**: See [Troubleshooting Guide](../troubleshooting.md)

## Quick Installation Script

For automated installation on Ubuntu/Debian systems:

```bash
#!/bin/bash
# Quick prerequisites installation script for Ubuntu/Debian

set -e

echo "Installing Vrooli prerequisites..."

# Update system
sudo apt update && sudo apt upgrade -y

# Install core packages
sudo apt install -y git curl wget build-essential python3-dev ca-certificates gnupg lsb-release

# Install Node.js
curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -
sudo apt-get install -y nodejs

# Install PNPM
corepack enable
corepack prepare pnpm@latest --activate

# Install Docker
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Add user to docker group
sudo usermod -aG docker $USER

# Install database tools
sudo apt install -y postgresql-client redis-tools

# Install optional tools
sudo apt install -y shellcheck

echo "Prerequisites installation complete!"
echo "Please log out and log back in for Docker group membership to take effect."
```

Save this script and run with `bash install-prerequisites.sh` for automated setup. 