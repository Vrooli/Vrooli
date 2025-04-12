# Developing on a Remote Server
*Note:* A more detailed guide (minus step 1) can be [found here](https://www.digitalocean.com/community/tutorials/how-to-use-visual-studio-code-for-remote-development-via-the-remote-ssh-plugin).

Not able to develop locally? Want to test deployment on a remote server? Follow these steps:
1. Follow the [deployment steps](/docs/deployment/README.md) to learn how to host a VPS and set it up correctly  
2. Use `./scripts/keylessSsh.sh <server_ip>` to generate a pair of SSH keys for the server, if you don't already have them. This will prevent you from having to enter a password when connecting to the server.
3. In VSCode, download the [Remote Development extension](https://code.visualstudio.com/docs/remote/remote-overview) if you don't already have it.
4. Enter `CTRL+SHIFT+P` to open the Command Palette  
5. Search and select `Remote-SSH: Open Configuration File...`  
6. Add new connection by entering `ssh root@<server_ip>`  
7. Open Command Palette again and select `Remote-SSH: Connect to Host...` 
8. A new VSCode terminal should open. Answer the questions (e.g. server type, server password), and you should be connected!  
9. Open the `Extensions` page in VSCode, and download the extensions you want to use  
10. After following the [environment variables setup](repo_setup.md#3-set-environment-variables), open the Command Palette and select `Ports: Focus on Ports View`  
11. Enter the port numbers of every port in the `.env` file. When running the project, you can now use localhost to access it