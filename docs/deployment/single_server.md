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
- **OS**: Ubuntu 20.04 LTS or newer

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
3. Select Ubuntu 20.04 or newer as the operating system
4. Choose a plan that meets the requirements above
5. Choose a datacenter region (ideally close to your target audience)
6. Add your SSH key or set a password
7. Create the Droplet

[Here](https://www.digitalocean.com/community/tutorials/how-to-set-up-an-ubuntu-20-04-server-on-a-digitalocean-droplet) is a detailed guide on setting up a server with DigitalOcean.

### Initial Server Security

Once your server is running, you should secure it:

```bash
# Update packages
sudo apt update && sudo apt upgrade -y

# Install firewall
sudo apt install ufw

# Allow SSH, HTTP, and HTTPS
sudo ufw allow ssh
sudo ufw allow http
sudo ufw allow https

# Enable firewall
sudo ufw enable

# Create a non-root user with sudo privileges (if not done during setup)
sudo adduser deployer
sudo usermod -aG sudo deployer
```

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

### Setting Up SSL with Let's Encrypt

Once your domain is pointing to your server, you should set up SSL:

```bash
# Install Certbot
sudo apt install certbot python3-certbot-nginx

# Obtain certificate
sudo certbot --nginx -d yourdomain.com -d www.yourdomain.com -d api.yourdomain.com
```

## Deploying the App to the Server

Deploying the app to the server is quite simple, as the repo comes with scripts that do most of the work.

### Production Deployment

Follow these steps for a production deployment:

1. Connect to your server:
   ```bash
   ssh username@your_server_ip
   ```

2. Clone the repository:
   ```bash
   cd ~
   git clone --depth 1 --branch main https://github.com/Vrooli/Vrooli.git
   cd Vrooli
   ```

3. Make scripts executable:
   ```bash
   chmod +x ./scripts/*
   ```

4. Run the deployment script:
   ```bash
   ./scripts/deploy.sh
   ```

5. Follow the prompts to complete the deployment.

### Development Deployment

For a development environment, follow these additional steps:

1. Set up the reverse proxy:
   ```bash
   git clone https://github.com/MattHalloran/NginxSSLReverseProxy
   cd NginxSSLReverseProxy
   ./setup.sh
   ```

2. Configure environment variables:
   ```bash
   cd ~/Vrooli
   cp .env-example .env-dev
   nano .env-dev  # Edit variables as needed
   ```

3. Start the development environment:
   ```bash
   ./scripts/develop.sh
   ```

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

## Troubleshooting

### Common Issues

1. **Docker containers not starting**:
   ```bash
   # Check container status
   docker ps -a
   
   # View detailed logs
   docker logs container_name
   ```

2. **Web server not accessible**:
   ```bash
   # Check if NGINX is running
   sudo systemctl status nginx
   
   # Check firewall rules
   sudo ufw status
   ```

3. **Database connection issues**:
   ```bash
   # Check if PostgreSQL container is running
   docker ps | grep db
   
   # Check database logs
   docker logs db
   ```

### Using VSCode Peacock Extension

When working with multiple environments, it's easy to confuse production and development servers. The VSCode Peacock extension can help by color-coding your VSCode windows.

1. Install the Peacock extension from the VSCode marketplace
2. Set different colors for different environments:
   - Production: Red (#ff0000)
   - Staging: Orange (#ff9900)
   - Development: Green (#00ff00)

This visual cue helps prevent accidentally making changes to the wrong environment.

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