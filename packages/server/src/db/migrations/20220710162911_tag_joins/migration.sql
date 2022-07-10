/**
* This migration changes join tables between tags and tagged entities. 
* Instead of connecting to tags via ID, it connects to tags via tag name 
* (since this is already unique). 
* To avoid data loss, we must make sure that the join tables are updated 
* to use the new tag name, before dropping the old tag ID column. 
* The process is as follows:
* 1. Add a new column to the join tables, called `tagTag`
* 2. Populate the new column with the correct tag name, using the old tag ID
* 3. Update new column to have not null constraint
* 4. Apply unique constraint to the new column
* 5. Apply foreign key constraint to the new column
* 6. Drop the old foreign key constraint
* 7. Drop the old unique constraint
* 8. Drop the old column
*/

-- Create new column on every join table
ALTER TABLE "organization_tags" ADD COLUMN "tagTag" VARCHAR(128);
ALTER TABLE "project_tags" ADD COLUMN "tagTag" VARCHAR(128);
ALTER TABLE "routine_tags" ADD COLUMN "tagTag" VARCHAR(128);
ALTER TABLE "standard_tags" ADD COLUMN "tagTag" VARCHAR(128);

-- Populate new column with tag name, using old tag ID
UPDATE "organization_tags" SET "tagTag" = (SELECT "tag" FROM "tag" WHERE "tag"."id" = "organization_tags"."tagId");
UPDATE "project_tags" SET "tagTag" = (SELECT "tag" FROM "tag" WHERE "tag"."id" = "project_tags"."tagId");
UPDATE "routine_tags" SET "tagTag" = (SELECT "tag" FROM "tag" WHERE "tag"."id" = "routine_tags"."tagId");
UPDATE "standard_tags" SET "tagTag" = (SELECT "tag" FROM "tag" WHERE "tag"."id" = "standard_tags"."tagId");

-- Update new column to have not null constraint
ALTER TABLE "organization_tags" ALTER COLUMN "tagTag" SET NOT NULL;
ALTER TABLE "project_tags" ALTER COLUMN "tagTag" SET NOT NULL;
ALTER TABLE "routine_tags" ALTER COLUMN "tagTag" SET NOT NULL;
ALTER TABLE "standard_tags" ALTER COLUMN "tagTag" SET NOT NULL;

-- Apply unique constraint to new column
CREATE UNIQUE INDEX "organization_tags_tagTag_key" ON "organization_tags"("tagTag");
CREATE UNIQUE INDEX "project_tags_tagTag_key" ON "project_tags"("tagTag");
CREATE UNIQUE INDEX "routine_tags_tagTag_key" ON "routine_tags"("tagTag");
CREATE UNIQUE INDEX "standard_tags_tagTag_key" ON "standard_tags"("tagTag");

-- Apply foreign key constraint to new column
ALTER TABLE "organization_tags" ADD CONSTRAINT "organization_tags_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "project_tags" ADD CONSTRAINT "project_tags_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "routine_tags" ADD CONSTRAINT "routine_tags_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "standard_tags" ADD CONSTRAINT "standard_tags_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;

-- Drop old foreign key constraint
ALTER TABLE "organization_tags" DROP CONSTRAINT "organization_tags_tagId_fkey";
ALTER TABLE "project_tags" DROP CONSTRAINT "project_tags_tagId_fkey";
ALTER TABLE "routine_tags" DROP CONSTRAINT "routine_tags_tagId_fkey";
ALTER TABLE "standard_tags" DROP CONSTRAINT "standard_tags_tagId_fkey";

-- Drop old unique constraint
DROP INDEX "organization_tags_taggedId_tagId_key";
DROP INDEX "project_tags_taggedId_tagId_key";
DROP INDEX "routine_tags_taggedId_tagId_key";
DROP INDEX "standard_tags_taggedId_tagId_key";

-- Drop old column
ALTER TABLE "organization_tags" DROP COLUMN "tagId";
ALTER TABLE "project_tags" DROP COLUMN "tagId";
ALTER TABLE "routine_tags" DROP COLUMN "tagId";
ALTER TABLE "standard_tags" DROP COLUMN "tagId";
