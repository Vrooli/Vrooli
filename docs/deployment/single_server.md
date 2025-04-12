# Single Server Deployment

Deploying on a single server is an important step in pre-production testing, as it allows you to test the entire stack in a production-like environment. It is also the easiest way to deploy the app, as it does not require any additional configuration. However, it is not recommended for high-traffic production use, as it does not allow for horizontal scaling.

Single server deployment consists of 3 main steps:  
1. Buying and setting up the server itself
2. Connecting the server to a domain name
3. Deploying the app to the server

## Buying and Setting Up the Server

### Server Requirements

For a comfortable Vrooli deployment, your server should meet these minimum requirements:

- **CPU**: 2+ CPU cores
- **RAM**: 4GB+ RAM
- **Storage**: 20GB+ SSD storage
- **OS**: Ubuntu 22.04 LTS

For development or testing environments, these specs can be reduced.

### Server Provider Options

You can use any VPS provider, but here are some popular options:

- [DigitalOcean](https://www.digitalocean.com/) - Simple UI and good documentation
- [Linode](https://www.linode.com/) - Good performance to cost ratio
- [AWS EC2](https://aws.amazon.com/ec2/) - More advanced with many additional services
- [Google Cloud](https://cloud.google.com/compute) - Similar to AWS with different pricing
- [Vultr](https://www.vultr.com/) - Competitive pricing and global availability

### Setting Up the Server

Here's a basic setup process using DigitalOcean as an example:

1. Create an account on DigitalOcean
2. Create a new Droplet (DigitalOcean's term for a VPS)
3. Select Ubuntu 22.04 as the operating system
4. Choose a plan that meets the requirements above
5. Choose a datacenter region (ideally close to your target audience)
6. For SSH authentication, you have two options:
   - **Server Setup Mode**: Run `./scripts/keylessSsh.sh -s` to generate an SSH key pair. Copy the displayed public key into DigitalOcean's "Add SSH Key" field during Droplet creation. After creating the droplet, enter its IP when prompted to finalize the setup.
   - **Password Auth**: Set a root password during Droplet creation, then run `./scripts/keylessSsh.sh <server_ip>` after the server is ready to set up key-based authentication.
7. Complete the Droplet creation process

[Here](https://www.digitalocean.com/community/tutorials/how-to-set-up-an-ubuntu-20-04-server-on-a-digitalocean-droplet) is a detailed guide on setting up a server with DigitalOcean.

### Connecting to the Server

Once the server is created and SSH is set up, you can connect to it in several ways:

- Using IP directly: `./scripts/connectToServer.sh <server_ip>`
- Using IP from .env file: `./scripts/connectToServer.sh` 
- Using a custom .env file: `./scripts/connectToServer.sh -e /path/to/.env`

## Connecting the Server to a Domain Name

A Domain Name System (DNS) is a service that translates a domain name (e.g., vrooli.com) to an IP address (e.g., 192.81.123.456). While not strictly necessary for development, it's essential for production and provides a more professional experience.

### Purchasing a Domain

You can purchase domains from many providers:

- [Namecheap](https://www.namecheap.com/)
- [Google Domains](https://domains.google/)
- [GoDaddy](https://www.godaddy.com/)

### Setting Up DNS Records

After purchasing a domain, you'll need to set up DNS records:

1. **A Record**: Points your domain to your server's IP address
2. **CNAME Record**: Points subdomains to the main domain
3. **MX Records**: For email services (if needed)

You can use your domain registrar's DNS service, but [Cloudflare](https://www.cloudflare.com/) is a popular option that provides additional security and performance benefits.

#### Example DNS Configuration

| Type | Name | Value | TTL |
|------|------|-------|-----|
| A | @ | your_server_ip | Auto |
| CNAME | www | @ | Auto |
| CNAME | api | @ | Auto |
| CNAME | dev | @ | Auto |

**Note**: DNS changes may take several hours to propagate globally.

## Deploying the App to the Server

Deploying the app to the server is quite simple, as the repo comes with scripts that do most of the work.

### Deployment Process

The deployment process is the same for both production and development environments, with only the final script differing:

1. Connect to your server using one of these methods:
   ```bash
   # Using our helper script (recommended)
   ./scripts/connectToServer.sh <server_ip>
   
   # Or using standard SSH
   ssh root@<server_ip>
   ```

2. Set up the Vrooli repository:
   ```bash
   # Main branch
   cd ~ && \
   git clone https://github.com/Vrooli/Vrooli.git --depth 1 --branch main && \
   cd Vrooli && \
   chmod +x ./scripts/*
   ```

   ```bash
   # Development branch
   cd ~ && \
   git clone https://github.com/Vrooli/Vrooli.git --depth 1 --branch dev && \
   cd Vrooli && \
   chmod +x ./scripts/*
   ```

3. Configure environment variables:
   ```bash
   cp .env-example .env-prod  # For production
   # OR
   cp .env-example .env-dev   # For development
   
   # Edit variables as needed
   vim .env-prod  # Or .env-dev for development
   ```

4. Run setup script:
   ```bash
   # Production
   # -r y: Run on remote server
   # -p y: Use production environment
   # -k n: Do not use Kubernetes (use Docker Compose instead)
   ./scripts/setup.sh -r y -p y -k n
   ```

   ```bash
   # Development
   # -r y: Run on remote server
   # -p n: Use development environment
   # -k n: Do not use Kubernetes (use Docker Compose instead)
   ./scripts/setup.sh -r y -p n -k n
   ```

5. Run the appropriate deployment script:
   For production:
   ```bash
   # -c: Clean up old builds (prevents build directory accumulation)
   ./scripts/deploy.sh -c
   ```
   
   For development:
   ```bash
   ./scripts/develop.sh
   ```

6. Follow the prompts to complete the deployment.

## Post-Deployment Steps

After successful deployment, perform these checks:

1. Verify that the application is running:
   ```bash
   docker ps
   ```

2. Check the logs for any errors:
   ```bash
   docker logs server
   docker logs ui
   ```

3. Test the application by navigating to your domain in a browser

4. Set up a monitoring solution (optional):
   ```bash
   # Install simple monitoring with Glances
   sudo apt install glances
   ```

## Maintenance

### Regular Updates

Keep your server and application updated:

```bash
# System updates
sudo apt update && sudo apt upgrade -y

# Application updates
cd ~/Vrooli
git pull
./scripts/deploy.sh
```

### Backup Strategy

Set up regular backups of your database:

```bash
# Run the backup script
cd ~/Vrooli
./scripts/backup.sh
```

You can also set up a cron job for automatic backups:

```bash
# Edit crontab
crontab -e

# Add a daily backup at 2 AM
0 2 * * * /home/username/Vrooli/scripts/backup.sh
``` 