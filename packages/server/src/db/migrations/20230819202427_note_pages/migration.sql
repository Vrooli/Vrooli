/*
 Warnings:
 
 - You are about to drop the column `participantsCount` on the `chat` table. All the data in the column will be lost.
 - You are about to drop the column `text` on the `note_version_translation` table. All the data in the column will be lost.
 
 */
-- Ensure UUID generation is enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- AlterTable
ALTER TABLE
    "chat" DROP COLUMN "participantsCount";

-- CreateTable
CREATE TABLE "note_page" (
    "id" UUID NOT NULL,
    "embedding" vector(768),
    "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
    "pageIndex" INTEGER NOT NULL DEFAULT 0,
    "text" VARCHAR(65536) NOT NULL,
    "translationId" UUID NOT NULL,
    CONSTRAINT "note_page_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "note_page_translationId_pageIndex_key" ON "note_page"("translationId", "pageIndex");

-- AddForeignKey
ALTER TABLE
    "note_page"
ADD
    CONSTRAINT "note_page_translationId_fkey" FOREIGN KEY ("translationId") REFERENCES "note_version_translation"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- Migrate existing data
INSERT INTO
    "note_page" ("id", "text", "translationId")
SELECT
    uuid_generate_v4(),
    "text",
    "id"
FROM
    "note_version_translation";

-- AlterTable
ALTER TABLE
    "note_version_translation" DROP COLUMN "text";