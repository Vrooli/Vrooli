# Getting Started
This guide will walk you through setting up the any Vrooli repo for development. If you want to deploy the project, follow the [deployment guide](TODO).

## 1. Prerequisites  
Before developing a website from this template, make sure you have the following installed:   
1. *Windows only*: [Windows Terminal](https://www.microsoft.com/store/productId/9N0DX20HK701)
2. *Windows only*: [Ubuntu](https://www.microsoft.com/store/productId/9PNKSF5ZN4SW)
3. [Docker](https://www.docker.com/). On Windows, Docker will guide you through enabling Windows Subsystem for Linux (WSL)
4. [VSCode](https://code.visualstudio.com/) *(also look into enabling Settings Sync)*    

**Note 1**: On Windows, NPM and Yarn should be installed from Ubuntu directly.  
**Note 2**: If you want to develop on a remote server (e.g. computer too slow, don't have admin privileges for Docker), you can follow the [instructions for setting this up](https://github.com/Vrooli/Vrooli/blob/remote-development/README.md#4-set-up-remote-development-if-you-cant-develop-locally). If this is the case, the only prerequisite you need is VSCode.

## 2. Configure for WSL (Windows only) 
VSCode and Docker require additional setup on Windows, since you must use the Windows Subsystem for Linux (WSL). First, make sure the WSL status indicator can appear in VSCode:   
1. Open VSCode  
2. Right click the status bar, and make sure "Remote Host" is checked

Next, make sure Docker's WSL integation uses the Ubuntu distro downloaded in step 1:  
1. Open Docker Desktop
2. Go to Resources -> WSL Integration  
3. Select the Ubuntu distro
4. Apply and restart 
    
## 3. Make sure files are using `LF` line endings (Windows only)
1. Open a terminal in the WSL distro you're using for the project (either from Windows Terminal or the built-in terminal in VSCode)  
2. `git config --global core.autocrlf false`  
3. `git config --global core.eol lf`  
4. To apply this retroactively:    
    4.1. `git rm --cached -rf .`  
    4.2. `git diff --cached --name-only -z | xargs -n 50 -0 git add -f`  
    4.3. `git ls-files -z | xargs -0 rm`  
    4.4. `git checkout .`  
    
## 4. Set up Remote Development (if you can't develop locally)  
A more detailed guide (minus step 1) can be [found here](https://www.digitalocean.com/community/tutorials/how-to-use-visual-studio-code-for-remote-development-via-the-remote-ssh-plugin).
1. Follow the [deployment steps](https://github.com/Vrooli/Vrooli#deploying-project) to learn how to host a VPS and set it up correctly  
2. Set up a pair of SSH keys. Make note of the file location that the keys are stored in
1. In VSCode, download the [Remote Development extension](https://code.visualstudio.com/docs/remote/remote-overview)  
2. Enter `CTRL+SHIFT+P` to open the Command Palette  
3. Search and select `Remote-SSH: Open Configuration File...`  
4. Edit configuration file to contain an entry with this format:  
    ```
    Host <any_name_for_remote_server>
        HostName <your_server_ip_or_hostname>
        User <your_username>
        IdentityFile <ssh_keys_location>
    ```
    **Note 1**: You can now reference your host by the name you chose, instead of its IP address  
    **Note 2**: `User` is likely `root`  
5. Open Command Palette again and select `Remote-SSH: Connect to Host...`   
6. A new VSCode terminal should open. Answer the questions (e.g. server type, server password), and you should be connected!  
7. Open the `Extensions` page in VSCode, and download the extensions you want to use  
8. After following the [environment variables setup](https://github.com/Vrooli/Vrooli#6-set-environment-variables), open the Command Palette and select `Ports: Focus on Ports View`  
9. Enter the port numbers of every port in the `.env` file. When running the project, you can now use localhost to access it

## 5. Download this repository
In the directory of your choice, enter `git clone <REPO_URL>`. On Windows, make sure this is done from an Ubuntu terminal in Windows Terminal. If the code is stored on the Windows file system, then docker will be **extremely** slow - and likely unusable.  

To open the project from the command line, enter `code <PROJECT_NAME>` from the directory you cloned in, or `code .` from the project's directory.

## 6. Install packages
1. `cd <PROJECT_NAME>`  
2. `chmod +x ./scripts/* && ./scripts/setup.sh`
4. Restart code editor  

**Note:** If `setup.sh` installs any global packages, this is because they are either used by a `Dockerfile` or `package.json`. If you want to make sure the dependency versions are correct, you should check those files.

## 7. Set environment variables  
1. Edit environment variables in [.env-example](https://github.com/Vrooli/Vrooli/blob/master/.env-example)
2. Rename the file to .env

## 8. Docker
By default, the docker containers rely on an external network. This network is used for the server's nginx docker container. During development, there is no need to run an nginx container. Instead, you can enter: `docker network create nginx-proxy`   
Once the docker network is set up, you can start the entire application by entering in the root directory: `docker-compose up --build --force-recreate -d`