# Single Server Deployment
Deploying on a single server is an important step in pre-production testing, as it allows you to test the entire stack in a production-like environment. It is also the easiest way to deploy the app, as it does not require any additional configuration. However, it is not recommended for production use, as it does not allow for horizontal scaling. If you are deploying to production, you should use the multi-server deployment guide.

Single server deployment consists of 3 main steps:  
1. Buying and setting up the server itself
2. Connecting the server to a domain name
3. Deploying the app to the server

## Buying and Setting Up the Server
[Here](https://www.digitalocean.com/community/tutorials/how-to-set-up-an-ubuntu-20-04-server-on-a-digitalocean-droplet) is an example of how to set up a server with DigitalOcean.

## Connecting the Server to a Domain Name
A Domain Name System (DNS) is a service that translates a domain name (e.g. vrooli.com) to an IP address (e.g. 192.81.123.456). This is not necessary for development, as you can simply use the IP address to access the server. However, it is still a good idea to set it up, since it is required for production.

Any DNS service will work, but [Cloudflare](https://www.cloudflare.com/) seems to currently be the cheapest. Once you buy a domain, you must set up the correct DNS records. This can be done through the site that you bought the domain from, or the site that you bought the VPS from. [Here](https://www.youtube.com/watch?v=wYDDYahCg60) is a good example. 

**Note**: DNS changes may take several hours to take effect.

## Deploying the App to the Server
Deploying the app to the server is quite simple, as the repo comes with scripts that do most of the work. All you have to do is follow these steps:

1. `cd ~` *(Start in home directory)*
2. `git clone --depth 1 --branch ${DESIRED_BRANCH} ${PROJECT_URL}` *(Clone last commit of desired branch)*
3. `cd ${PROJECT_NAME}` *(Enter project directory)*
4. `chmod +x ./scripts/*` *(Make scripts executable)*
5. `./scripts/deploy.sh` *(Deploy the app)*

If you are instead starting a development build, you may also need to complete these steps:
1. [Set up the reverse proxy](https://github.com/MattHalloran/NginxSSLReverseProxy#getting-started).
2. Edit .env variables. *(These are sent from the CI/CD pipeline in production builds)*
3. Make sure that the urls in `packages/ui/public/index.html` point to the correct website.
4. Instead of `./scripts/deploy.sh`, run `./scripts/development.sh` to start the app.