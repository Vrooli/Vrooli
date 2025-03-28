# Logs
There are a two different types of logs you can utilize to debug the application.

## Console Logs
Console logs are the most basic type of log. They are used to print out information to the console. For UI code, these can be viewed in the browser's developer console. For the server and other containers, you must use `docker logs <container_name>` to view the logs.

## Winston Logs
The server uses Winston to log information to the console and to a file. When in production mode, they are only logged in a file. The logs can be found at `data/logs`.

To automatically backup and track logs for production, you can use Window's `Task Scheduling` feature.

### Task scheduling on Windows (with WSL)
1. Open `Task Scheduler` from the Windows start menu 
2. Select `Create Task`
3. Enter a task name and description
4. In the Triggers tab, select "New" and select "At startup" as the trigger.
5. In the Actions tab, select "New" and select "Start a program" as the action.
6. In the "Program/script" field, enter this command to open WSL: `C:\Windows\System32\wsl.exe`
7. In the "Add arguments" field, enter the path to the .sh script you created in step 3, using Linux-style paths. For example: `/root/Programming/ReactGraphQLTemplate/scripts/listen.sh`
8. Press `Ok` to save the task