-- CreateExtension
CREATE EXTENSION IF NOT EXISTS "vector";

-- AlterTable
ALTER TABLE "api_version_translation" ADD COLUMN     "embedding" vector(768),
ADD COLUMN     "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- AlterTable
ALTER TABLE "chat_translation" ADD COLUMN     "embedding" vector(768),
ADD COLUMN     "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- AlterTable
ALTER TABLE "issue_translation" ADD COLUMN     "embedding" vector(768),
ADD COLUMN     "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- AlterTable
ALTER TABLE "meeting_translation" ADD COLUMN     "embedding" vector(768),
ADD COLUMN     "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- AlterTable
ALTER TABLE "note_version_translation" ADD COLUMN     "embedding" vector(768),
ADD COLUMN     "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- AlterTable
ALTER TABLE "organization_translation" ADD COLUMN     "embedding" vector(768),
ADD COLUMN     "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- AlterTable
ALTER TABLE "post_translation" ADD COLUMN     "embedding" vector(768),
ADD COLUMN     "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- AlterTable
ALTER TABLE "project_version_translation" ADD COLUMN     "embedding" vector(768),
ADD COLUMN     "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- AlterTable
ALTER TABLE "question_translation" ADD COLUMN     "embedding" vector(768),
ADD COLUMN     "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
ALTER COLUMN "description" SET DATA TYPE VARCHAR(16384);

-- AlterTable
ALTER TABLE "quiz_translation" ADD COLUMN     "embedding" vector(768),
ADD COLUMN     "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- AlterTable
ALTER TABLE "reminder" ADD COLUMN     "embedding" vector(768),
ADD COLUMN     "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true;

-- AlterTable
ALTER TABLE "routine_version_translation" ADD COLUMN     "embedding" vector(768),
ADD COLUMN     "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- AlterTable
ALTER TABLE "run_project" ADD COLUMN     "embedding" vector(768),
ADD COLUMN     "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true;

-- AlterTable
ALTER TABLE "run_routine" ADD COLUMN     "embedding" vector(768),
ADD COLUMN     "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true;

-- AlterTable
ALTER TABLE "smart_contract_version_translation" ADD COLUMN     "embedding" vector(768),
ADD COLUMN     "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- AlterTable
ALTER TABLE "standard_version_translation" ADD COLUMN     "embedding" vector(768),
ADD COLUMN     "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- AlterTable
ALTER TABLE "tag_translation" ADD COLUMN     "embedding" vector(768),
ADD COLUMN     "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- AlterTable
ALTER TABLE "user_translation" ADD COLUMN     "embedding" vector(768),
ADD COLUMN     "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP;
