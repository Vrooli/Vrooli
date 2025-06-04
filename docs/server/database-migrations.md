# Database Migrations
This guide will walk you through migrating the database. If you want to read more general information about the data, you can visit [the Prisma docs](https://www.prisma.io/docs/concepts/components/prisma-migrate).

Database migrations require an interactive terminal, which means they cannot be a part of the docker-compose setup process. 

## Getting DB_URL
To use Prisma, you'll need to make sure that the `DB_URL` environment variable is set. Since we calculate this in `scripts.sh` instead of directly as a docker-compose environment variable, it won't be available automatically. Instead, we'll need to export it manually whenever we're in the container. To do this, enter `export DB_URL=postgresql://${DB_USER}:${DB_PASSWORD}@postgres:5432;`, where `DB_USER` and `DB_PASSWORD` are the values you set in your `.env` file.

For a development environment where you haven't changed the environment variables, you can use `export DB_URL=postgresql://site:databasepassword@postgres:5432;`.

## Initial Migration
Before you even think about migrating your schema, make sure you have already created an initial migration. To do this:  
1. Make sure schema.prisma matches the current database schema.  
2. Start the project with `docker-compose up -d`, and wait for the server container to finish starting. If the server cannot run due to an error, it's okay to comment out everything in the `server.sh` script to get it to start. To keep the server running, you can add `tail -f /dev/null` to the end of the script.  
3. `docker exec -it server sh`   
4. `./scripts/migrateInit.sh`  
5. Type `exit` to exit the shell.  


## Non-Initial Migrations
1. Make sure schema.prisma matches the current database schema.  
2. Create a backup of the database (`data/postgres`) and name it something like `postgres_backup_<DATE>`. Hopefully you won't need it, but it's better to be safe than sorry.
3. Start the project with `docker-compose up -d`, and wait for the server container to finish starting.  
4. `docker exec -it server sh`  
5. Follow the steps a few sections above to set the `DB_URL` environment variable. 
6. `cd packages/server`  
7. Check the migration status: `yarn prisma migrate status`. If you get the message "Database schema is up to date!", then you should be good to continue. If not, you may need to mark migrations as applied (assuming they are already applied).
8. Edit schema.prisma **in the `dist` directory** to how you'd like it to look, and save the file  
9. Enter `yarn prisma migrate dev --name <ENTER_NAME_FOR_MIGRATTION>`. Include `--create-only` if you want to create the migration without applying it (e.g. need to move data around)
10. Type `exit` to exit the shell.  
11. Move the new migration folder in `packages/server/dist/db/migrations` to `packages/server/src/db/migrations`.
12. Restart the server with `docker-compose restart server` and make sure that it starts successfully. This will also apply the migration if you used `-create-only` in step 8.
13. `cd packages/server`
14. Enter `yarn prisma generate` to update the Prisma types.


## Resolving Migration Issues
When resolving issues, the first thing to do is to enter this command: `yarn prisma migrate status`, after following the same steps above for accessing the server.

### Dumping and Reimporting the Database
If that doesn't work, you might need to resort to more drastic measures. One such measure is by exporting the database, deleting it, and then reimporting it. Follow these steps:

1. **VERY IMPORTANT:** Make a copy of the database (`data/postgres`) and put it somewhere safe. This is your backup in case something goes wrong.
2. Start the app with `docker-compose up`.
3. Wait for the `postgres` container to be `healthy` (see `docker ps -a`).
4. Enter the container using `docker exec -it postgres sh`.
5. If you want to export the whole database, run `pg_dump -U <DB_USER> postgres > /tmp/dump.sql`. If you want to export specific tables, use `pg_dump -U <DB_USER> -t table1 -t table2 postgres > /tmp/dump.sql`. Replace `<DB_USER>` with the actual database user name defined in your `.env` file.

At this point, you have a backup of your database (or selected tables) saved inside the Docker container. Next, you'll delete and recreate the database.

6. Log into the PostgreSQL database with `psql -U <DB_USER> postgres`.
7. Drop the database using the `DROP DATABASE postgres;` command.
8. Recreate the database using the `CREATE DATABASE postgres;` command.
9. Exit the PostgreSQL interface using `\q`.

Now, you'll import the data you previously exported.

10. Import the database dump using `psql -U <DB_USER> -d postgres -f /tmp/dump.sql`.
11. Once the process completes, you can exit the container using `exit`.
12. Finally, restart your Docker app with `docker-compose restart`.

And that's it! You've now deleted and reimported your database. If there were any issues related to the database schema or migration files, they should be resolved.

#### Transferring the Dump to Another Container
In the above steps, we exported, deleted, and reimported the database using the same Docker container. However, there might be scenarios where you need to export the database from one container and import it into another. In that case, you would need to transfer the database dump to your local machine first. Here's how you do that:

1. After exporting the database (Step 5 above), exit the container using `exit`.
2. Move the dump file to your local machine using `docker cp postgres:/tmp/dump.sql .`.

To import the dump into a different Docker container:

1. Copy the dump file from your local machine to the new Docker container using `docker cp ./dump.sql postgres:/tmp/`.
2. Enter the new container using `docker exec -it postgres sh`.
3. Import the database dump using `psql -U <DB_USER> -d postgres -f /tmp/dump.sql`.
4. Once the process completes, you can exit the container using `exit`.
