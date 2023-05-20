# Developing on a Remote Server
*Note:* A more detailed guide (minus step 1) can be [found here](https://www.digitalocean.com/community/tutorials/how-to-use-visual-studio-code-for-remote-development-via-the-remote-ssh-plugin).

Not able to develop locally? Want to test deployment on a remote server? Follow these steps:
1. Follow the [deployment steps](TODO) to learn how to host a VPS and set it up correctly  
2. Set up a pair of SSH keys. Make note of the file location that the keys are stored in
3. In VSCode, download the [Remote Development extension](https://code.visualstudio.com/docs/remote/remote-overview)  
4. Enter `CTRL+SHIFT+P` to open the Command Palette  
5. Search and select `Remote-SSH: Open Configuration File...`  
6. Edit configuration file to contain an entry with this format:  
    ```
    Host <any_name_for_remote_server>
        HostName <your_server_ip_or_hostname>
        User <your_username>
        IdentityFile <ssh_keys_location>
    ```
    **Note 1**: You can now reference your host by the name you chose, instead of its IP address  
    **Note 2**: `User` is likely `root`  
7. Open Command Palette again and select `Remote-SSH: Connect to Host...`   
8. A new VSCode terminal should open. Answer the questions (e.g. server type, server password), and you should be connected!  
9. Open the `Extensions` page in VSCode, and download the extensions you want to use  
10. After following the [environment variables setup](https://github.com/Vrooli/Vrooli#6-set-environment-variables), open the Command Palette and select `Ports: Focus on Ports View`  
11. Enter the port numbers of every port in the `.env` file. When running the project, you can now use localhost to access it