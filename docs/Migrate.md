# Database Migrations
Database migrations require an interactive terminal, which means they cannot be a part of the docker-compose setup process. 

## Initial Migration
Before you even think about migrating your schema, make sure you have already created an initial migration. To do this:  
1. Make sure schema.prisma matches the current database schema.  
2. Start the project with `docker-compose up -d`, and wait for the server container to finish starting.
3. `docker exec -it server sh`   
4. `cd packages/server`  
5. Migrate without applying schema - `prisma migrate dev --name init --create-only --schema src/db/schema.prisma`. This will generate a migration file in src/db/migrations.
6. Open migration file, and add extension create lines for every database extension used.
    e.g. `CREATE EXTENSION IF NOT EXISTS citext;`
7. `prisma migrate dev --name init --schema src/db/schema.prisma`  
8. Type `exit` to exit the shell.  


## Non-Initial Migrations
1. Make sure schema.prisma matches the current database schema.  
2. Start the project with `docker-compose up -d`, and wait for the server container to finish starting.  
3. Edit schema.prisma to how you'd like it to look.  
3. `docker exec -it server sh`   
4. `cd packages/server`  
5. `prisma migrate dev --name <ENTER_NAME_FOR_MIGRATTION> --schema src/db/schema.prisma`  
4. Type `exit` to exit the shell.  