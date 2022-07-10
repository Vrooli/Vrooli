/*
  Warnings:

  - A unique constraint covering the columns `[taggedId,tagTag]` on the table `organization_tags` will be added. If there are existing duplicate values, this will fail.
  - A unique constraint covering the columns `[taggedId,tagTag]` on the table `project_tags` will be added. If there are existing duplicate values, this will fail.
  - A unique constraint covering the columns `[taggedId,tagTag]` on the table `routine_tags` will be added. If there are existing duplicate values, this will fail.
  - A unique constraint covering the columns `[taggedId,tagTag]` on the table `standard_tags` will be added. If there are existing duplicate values, this will fail.

*/
-- CreateIndex
CREATE UNIQUE INDEX "organization_tags_taggedId_tagTag_key" ON "organization_tags"("taggedId", "tagTag");

-- CreateIndex
CREATE UNIQUE INDEX "project_tags_taggedId_tagTag_key" ON "project_tags"("taggedId", "tagTag");

-- CreateIndex
CREATE UNIQUE INDEX "routine_tags_taggedId_tagTag_key" ON "routine_tags"("taggedId", "tagTag");

-- CreateIndex
CREATE UNIQUE INDEX "standard_tags_taggedId_tagTag_key" ON "standard_tags"("taggedId", "tagTag");
