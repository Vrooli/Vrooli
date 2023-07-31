# Working With Docker
All Vrooli repos are set up to use Docker for both development and deployment. Here's what you need to know to work with Docker:

## Common Docker Commands
- Start: `docker-compose up -d`
- Stop: `docker-compose down`
- Force stop all containers: `docker kill $(docker ps -q)`
- Delete all containers: `docker system prune --all`
- Delete all containers and volumes: `docker system prune --all --volumes`
- Full deployment test (except for Nginx, as that's handled by a different container): `docker-compose down && docker-compose up --build --force-recreate`
- Rebuild with fresh database: `docker-compose down && rm -rf "${PROJECT_DIR}/data/postgres" && docker-compose up --build --force-recreate`
- Check logs for a docker container: `docker logs <container-name>`

## Docker Setup
By default, the docker containers rely on an external network. This network is used for the server's nginx docker container (used during deployment). During development, there is no need to run an nginx container. Instead, you can enter: `docker network create nginx-proxy` and the network will be created.

## Starting Docker (Development)
Once the docker network is set up, you can start the entire application by entering in the root directory: `docker-compose up --build --force-recreate -d`. Let's break down what this command does:
- `docker-compose up` starts the docker containers
- `--build` rebuilds the containers
- `--force-recreate` forces the containers to be recreated
- `-d` runs the containers in the background

The only flag that is required is `-d`. The other flags are optional, but recommended if you 
already had Vrooli running before and want to make sure you have the latest changes. Generally, this is when you need each flag:
- `--build` when you update dependencies, scripts, or `package.json` files (i.e. anything that changes the behavior of the container)
- `--force-recreate` when you update a `Dockerfile` or `docker-compose.yml`
- `-d` when you want to run the containers in the background, which is typically always

## Starting Docker (Production)
If the repo you're working on has both a `docker-compose.yml` and `docker-compose-prod.yml`, this means that there is a separate configuration for testing/deploying production code. To start the production containers, enter: `docker-compose -f docker-compose-prod.yml up -d`. This command is the same as the development command, except it uses the `docker-compose-prod.yml` file instead of the `docker-compose.yml` file. See the previous section to learn more about the flags you can use.

## Stopping Docker
To stop the docker containers, enter: `docker-compose down` (or `docker-compose -f docker-compose-prod.yml down` for production). This will stop the containers, but will not remove them. To remove the containers, enter: `docker-compose down --rmi all`. This will stop the containers and remove them. If you want to remove the containers and the volumes, enter: `docker-compose down --rmi all -v`. This will stop the containers, remove them, and remove the volumes. **Note:** If you remove the volumes, you will lose all data stored in the database.

## Logging  
- To view the logs for a container, enter: `docker logs <container-name>`. 
- To follow the logs in real time, enter: `docker logs -f <container-name>`.
- To debug an unhealthy container, enter `docker inspect <container-name>`.

## Entering a Container
Sometimes you'll need to view the contents of a container. To do this, enter: `docker exec -it <container-name> sh`. This will open a bash terminal in the container. To exit the container, enter: `exit`.

Sometimes it's more helpful to enter the container using a different command. For example, the database container is often accessed using `psql` instead. To do this, enter: `docker exec -it <container-name> psql -U <POSTGRES_USER>`, where `<POSTGRES_USER>` is found in the `docker-compose.yml` file.

## Developing Inside a Container
Using the [Dev Containers VSCode extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers), you can develop inside of a container. When working with Python, this is required for modules linting and autocomplete to work correctly. To set this up, follow these steps:  
1. Install the [Dev Containers VSCode extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
2. Make sure the docker container you want to develop in is running
3. Open the Command Palette (Ctrl+Shift+P) and select: `Dev Containers: Attach to Running Container...`
4. Select the container you want to develop in, and VSCode will open a new window with the container attached
5. Inside the new VSCode window, go to the Extensions tab and make sure the [Python extension](https://marketplace.visualstudio.com/items?itemName=ms-python.python) (and other recommended extensions) are installed