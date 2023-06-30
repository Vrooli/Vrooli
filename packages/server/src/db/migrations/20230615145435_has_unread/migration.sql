-- AlterTable
ALTER TABLE "api_version" ADD COLUMN     "schemaText" VARCHAR(16384);

-- AlterTable
ALTER TABLE "chat_participants" ADD COLUMN     "hasUnread" BOOLEAN NOT NULL DEFAULT true;
