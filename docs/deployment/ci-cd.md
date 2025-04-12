# CI/CD for Vrooli

This guide explains how to set up Continuous Integration/Continuous Deployment (CI/CD) for your Vrooli project.

## What is CI/CD?

CI/CD automates testing and deployment of your code when you push changes to your repository:
- **CI (Continuous Integration)**: Automatically runs tests when code is pushed
- **CD (Continuous Deployment)**: Automatically deploys your application when tests pass

## Prerequisites

1. A VPS (Virtual Private Server) set up according to the [Single Server Deployment](/docs/setup/single_server.md) guide
2. SSH access to your VPS (see [Remote Setup](/docs/setup/remote_setup.md) for details)
3. Access to your Vrooli GitHub repository

## Quick Setup

### Server Setup (On Your VPS)

```bash
# 1. Set up the Nginx reverse proxy (if not already done)
git clone https://github.com/Vrooli/NginxSSLReverseProxy.git
cd NginxSSLReverseProxy
sudo ./setup.sh

# 2. Set up CI/CD deployment tools
git clone https://github.com/Vrooli/Vrooli.git
cd Vrooli
sudo ./scripts/setup.sh --ci-cd y
```

### GitHub Configuration

1. Generate an SSH key (if you don't have one)
   ```bash
   ssh-keygen -t ed25519 -C "your_email@example.com"
   ssh-copy-id username@your-vps-ip
   ```

2. Add these secrets to your GitHub repository (Settings > Secrets > Actions):
   - `VPS_HOST`: Your VPS IP or hostname
   - `VPS_USERNAME`: SSH username for your VPS
   - `SSH_PRIVATE_KEY`: Content of your SSH private key file
   - `DEPLOY_PATH`: Full path to your Vrooli deployment directory (e.g., `/root/Vrooli`)

3. The repository already contains workflow files for both development and production:
   - `.github/workflows/deploy-prod.yml` - Handles deployment to production (main branch)
   - `.github/workflows/deploy-dev.yml` - Handles deployment to development environment

   These workflow files automatically run tests and deploy your code when you push to the respective branches.

## Managing Your Deployment

### Environment Configuration

Set up your production environment variables:
```bash
# On your VPS
sudo vim /path/to/your/Vrooli/.env-prod
```

Required variables:
```
NODE_ENV=production
VIRTUAL_HOST=your-domain.com
LETSENCRYPT_HOST=your-domain.com
LETSENCRYPT_EMAIL=your-email@example.com
```

### Triggering Deployments

Deployments are triggered automatically when you push to the respective branches:
- Push to `main` branch for production deployment
- Push to `development` branch for development deployment

To trigger a manual deployment:
1. Go to the Actions tab in your GitHub repository
2. Select the workflow you want to run ("VPS Deployment" or "Development VPS Deployment")
3. Click "Run workflow"

Or deploy directly from your VPS:
```bash
cd /path/to/your/Vrooli
./deploy.sh
```

### Monitoring

Monitor your deployment with:
```bash
# Check container status
cd /path/to/your/Vrooli
docker-compose -f docker-compose-prod.yml ps

# View logs
docker-compose -f docker-compose-prod.yml logs -f

# Check application health
curl https://testsite.com/healthcheck
curl http://localhost:{SERVER_URL:5329}/healthcheck
```

### Rolling Back Changes

If a deployment causes issues:
```bash
cd /path/to/your/Vrooli
git checkout <previous-commit-hash>
./deploy.sh
```

## Troubleshooting

- **SSH Issues**: Verify SSH key configuration and permissions
- **Deployment Failures**: Check Docker logs with `docker-compose -f docker-compose-prod.yml logs`
- **Application Not Working**: Verify container health with `docker ps` and check proxy logs with `docker logs nginx-proxy` 