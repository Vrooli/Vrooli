-- AlterTable
ALTER TABLE "api" ADD COLUMN     "completedAt" TIMESTAMPTZ(6);

-- AlterTable
ALTER TABLE "api_version" ADD COLUMN     "completedAt" TIMESTAMPTZ(6);

-- AlterTable
ALTER TABLE "note_version" ADD COLUMN     "completedAt" TIMESTAMPTZ(6);
