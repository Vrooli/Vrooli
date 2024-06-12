-- AlterTable
ALTER TABLE "api_version" ADD COLUMN     "isLatestPublic" BOOLEAN NOT NULL DEFAULT false;

-- AlterTable
ALTER TABLE "code_version" ADD COLUMN     "isLatestPublic" BOOLEAN NOT NULL DEFAULT false;

-- AlterTable
ALTER TABLE "note_version" ADD COLUMN     "isLatestPublic" BOOLEAN NOT NULL DEFAULT false;

-- AlterTable
ALTER TABLE "project_version" ADD COLUMN     "isLatestPublic" BOOLEAN NOT NULL DEFAULT false;

-- AlterTable
ALTER TABLE "routine_version" ADD COLUMN     "isLatestPublic" BOOLEAN NOT NULL DEFAULT false;

-- AlterTable
ALTER TABLE "standard_version" ADD COLUMN     "isLatestPublic" BOOLEAN NOT NULL DEFAULT false;
