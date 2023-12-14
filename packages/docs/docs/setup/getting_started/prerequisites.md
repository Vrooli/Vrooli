# Prerequisites  
Make sure you have the following installed:   
1. *Windows only*: [Windows Terminal](https://www.microsoft.com/store/productId/9N0DX20HK701)
2. *Windows only*: [Ubuntu (version 20 or higher)](https://apps.microsoft.com/detail/9MTTCL66CPXJ?hl=en-us&gl=US)
3. [Docker](https://www.docker.com/). On Windows, Docker will guide you through enabling Windows Subsystem for Linux (WSL)
4. [VSCode](https://code.visualstudio.com/) *(also look into enabling Settings Sync)*    

**Note 1**: On Windows, NPM and Yarn should be installed from Ubuntu directly.  
**Note 2**: If you want to develop on a remote server (e.g. computer too slow, don't have admin privileges for Docker), you can follow the [instructions for setting this up](https://github.com/Vrooli/Vrooli/blob/remote-development/README.md#4-set-up-remote-development-if-you-cant-develop-locally). If this is the case, the only prerequisite you need is VSCode.

## Configure for WSL (Windows only) 
VSCode and Docker require additional setup on Windows, since you must use the Windows Subsystem for Linux (WSL). First, make sure the WSL status indicator can appear in VSCode:   
1. Open VSCode  
2. Right click the status bar, and make sure "Remote Host" is checked

Next, make sure Docker's WSL integation uses the Ubuntu distro downloaded in step 1:  
1. Open Docker Desktop
2. Go to Resources -> WSL Integration  
3. Select the Ubuntu distro
4. Apply and restart 

## Make sure files are using `LF` line endings (Windows only)
1. Open a terminal in the WSL distro you're using for the project (either from Windows Terminal or the built-in terminal in VSCode)  
2. `git config --global core.autocrlf false`  
3. `git config --global core.eol lf`  
4. To apply this retroactively:    
    4.1. `git rm --cached -rf .`  
    4.2. `git diff --cached --name-only -z | xargs -n 50 -0 git add -f`  
    4.3. `git ls-files -z | xargs -0 rm`  
    4.4. `git checkout .`  
