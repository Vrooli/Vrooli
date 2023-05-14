# Repo Setup
If you've followed the [prerequisites](TODO), you should be ready to download and set up the repo. Here's how to do it:

## 1. Download this repository
In the directory of your choice, enter `git clone <REPO_URL>` (or `git clone --depth 1 --branch main <REPO_URL-url>` on the production server, since you only need the latest commit). On Windows, make sure this is done from an Ubuntu terminal in Windows Terminal. If the code is stored on the Windows file system, then docker will be **extremely** slow - and likely unusable.  

To open the project from the command line, enter `code <PROJECT_NAME>` from the directory you cloned in, or `code .` from the project's directory.

## 2. Install packages
1. `cd <PROJECT_NAME>`  
2. `chmod +x ./scripts/* && ./scripts/setup.sh`
4. Restart code editor  

**Note:** If `setup.sh` installs any global packages, this is because they are either used by a `Dockerfile` or `package.json`. If you want to make sure the dependency versions are correct, you should check those files.

## 3. Set environment variables  
1. Edit environment variables in [.env-example](https://github.com/Vrooli/Vrooli/blob/master/.env-example)
2. Rename the file to .env