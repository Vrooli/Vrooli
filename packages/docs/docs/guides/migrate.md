# Database Migrations
This guide will walk you through migrating the database. If you want to read more general information about the data, you can visit [the Prisma docs](https://www.prisma.io/docs/concepts/components/prisma-migrate).

Database migrations require an interactive terminal, which means they cannot be a part of the docker-compose setup process. 

## Getting DB_URL
To use Prisma, you'll need to make sure that the `DB_URL` environment variable is set. Since we calculate this in `scripts.sh` instead of directly as a docker-compose environment variable, it won't be available automatically. Instead, we'll need to export it manually whenever we're in the container. To do this, enter `export DB_PASSWORD=$(cat /run/secrets/vrooli/dev/DB_PASSWORD); export DB_URL=postgresql://${DB_USER}:${DB_PASSWORD}@db:5432;`

## Initial Migration
Before you even think about migrating your schema, make sure you have already created an initial migration. To do this:  
1. Make sure schema.prisma matches the current database schema.  
2. Start the project with `docker-compose up -d`, and wait for the server container to finish starting. If the server cannot run due to an error, it's okay to comment out everything in the `server.sh` script to get it to start. To keep the server running, you can add `tail -f /dev/null` to the end of the script.  
3. `docker exec -it server sh`   
4. `./scripts/migrateInit.sh`  
5. Type `exit` to exit the shell.  


## Non-Initial Migrations
1. Make sure schema.prisma matches the current database schema.  
2. Start the project with `docker-compose up -d`, and wait for the server container to finish starting.  
3. `docker exec -it server sh`   
4. `cd packages/server`  
5. Check the migration status: `prisma migrate status`. If you get the message "Database schema is up to date!", then you should be good to continue. If not, you may need to mark migrations as applied (assuming they are already applied).
6. Edit schema.prisma to how you'd like it to look, and save the file  
7. `prisma migrate dev --name <ENTER_NAME_FOR_MIGRATTION>`  
8. Type `exit` to exit the shell.  
9. Move the new migration folder in `packages/server/dist/db/migrations` to `packages/server/src/db/migrations`.
10. Restart the server with `docker-compose restart server` and make sure that it starts successfully.
11. `cd packages/server`
12. Enter `yarn prisma generate` to update the Prisma types.


## Resolving Migration Issues
The first thing to do when trying to resolve issues is to enter this command: `prisma migrate status`, after following the same steps above for accessing the server.