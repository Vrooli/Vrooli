# Repo Setup
If you've followed the [prerequisites](prerequisites.md), you should be ready to download and set up the repo. Here's how to do it:

## 1. Download this repository
In the directory of your choice, enter `git clone <REPO_URL>` (or `git clone --depth 1 --branch main <REPO_URL-url>` on the production server, since you only need the latest commit). On Windows, make sure this is done from an Ubuntu terminal in Windows Terminal. If the code is stored on the Windows file system, then docker will be **extremely** slow - and likely unusable.  

**Note:** If you plan to run the project directly on Windows (i.e. without using WSL), you should run the following commands in Git Bash instead of the normal terminal or PowerShell, and use the `bash` command whenever calling a script.

## 2. Install packages
1. `cd <PROJECT_NAME>`  
2. `vrooli setup` (Note: This will install dependencies and configure the development environment)
4. Restart code editor  

**Note:** If `vrooli setup` installs any global packages, this is because they are either used by a `Dockerfile` or `package.json`. If you want to make sure the dependency versions are correct, you should check those files.

## 3. Set environment variables  
1. Edit environment variables in [.env-example](https://github.com/Vrooli/Vrooli/blob/master/.env-example)
2. Rename the file to .env-dev or .env-prod, depending on the environment you're setting up.

## 4. Set up linting (Python repos only)  
If you're working on a Python repo, the linter will not work correctly unless you open up VSCode in the Docker container. To do this, follow [this guide](/setup/getting_started/working_with_docker.html)