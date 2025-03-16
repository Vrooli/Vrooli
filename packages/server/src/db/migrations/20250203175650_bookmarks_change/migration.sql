-- 1. Create the new enum type if it does not exist.
CREATE TYPE "BookmarkFor" AS ENUM (
  'Api', 
  'Code', 
  'Comment', 
  'Issue', 
  'Note', 
  'Post', 
  'Project', 
  'Question', 
  'QuestionAnswer', 
  'Quiz', 
  'Routine', 
  'Standard', 
  'Tag', 
  'Team', 
  'User'
);

-- 2. Add the new columns as nullable.
ALTER TABLE "bookmark" 
  ADD COLUMN "targetId" UUID,
  ADD COLUMN "targetType" "BookmarkFor";

-- 3. Populate the new columns using the existing data.
--    (Assumes that for each row only one of the old foreign key columns is non-null.)
UPDATE "bookmark"
SET 
  "targetId" = COALESCE(
    "apiId", 
    "codeId", 
    "commentId", 
    "issueId", 
    "noteId", 
    "postId", 
    "projectId", 
    "questionAnswerId", 
    "questionId", 
    "quizId", 
    "routineId", 
    "standardId", 
    "tagId", 
    "teamId", 
    "userId"
  ),
  "targetType" = CASE 
      WHEN "apiId" IS NOT NULL THEN 'Api'::"BookmarkFor"
      WHEN "codeId" IS NOT NULL THEN 'Code'::"BookmarkFor"
      WHEN "commentId" IS NOT NULL THEN 'Comment'::"BookmarkFor"
      WHEN "issueId" IS NOT NULL THEN 'Issue'::"BookmarkFor"
      WHEN "noteId" IS NOT NULL THEN 'Note'::"BookmarkFor"
      WHEN "postId" IS NOT NULL THEN 'Post'::"BookmarkFor"
      WHEN "projectId" IS NOT NULL THEN 'Project'::"BookmarkFor"
      WHEN "questionAnswerId" IS NOT NULL THEN 'QuestionAnswer'::"BookmarkFor"
      WHEN "questionId" IS NOT NULL THEN 'Question'::"BookmarkFor"
      WHEN "quizId" IS NOT NULL THEN 'Quiz'::"BookmarkFor"
      WHEN "routineId" IS NOT NULL THEN 'Routine'::"BookmarkFor"
      WHEN "standardId" IS NOT NULL THEN 'Standard'::"BookmarkFor"
      WHEN "tagId" IS NOT NULL THEN 'Tag'::"BookmarkFor"
      WHEN "teamId" IS NOT NULL THEN 'Team'::"BookmarkFor"
      WHEN "userId" IS NOT NULL THEN 'User'::"BookmarkFor"
      ELSE NULL
  END;

-- (Optional: Verify that every row now has non-null values in the new columns)
-- SELECT COUNT(*) FROM "bookmark" WHERE "targetId" IS NULL OR "targetType" IS NULL;

-- 4. Alter the new columns to be NOT NULL.
ALTER TABLE "bookmark"
  ALTER COLUMN "targetId" SET NOT NULL,
  ALTER COLUMN "targetType" SET NOT NULL;

-- 5. Drop the old foreign key constraints.
ALTER TABLE "bookmark" DROP CONSTRAINT "bookmark_apiId_fkey";
ALTER TABLE "bookmark" DROP CONSTRAINT "bookmark_codeId_fkey";
ALTER TABLE "bookmark" DROP CONSTRAINT "bookmark_commentId_fkey";
ALTER TABLE "bookmark" DROP CONSTRAINT "bookmark_issueId_fkey";
ALTER TABLE "bookmark" DROP CONSTRAINT "bookmark_noteId_fkey";
ALTER TABLE "bookmark" DROP CONSTRAINT "bookmark_postId_fkey";
ALTER TABLE "bookmark" DROP CONSTRAINT "bookmark_projectId_fkey";
ALTER TABLE "bookmark" DROP CONSTRAINT "bookmark_questionAnswerId_fkey";
ALTER TABLE "bookmark" DROP CONSTRAINT "bookmark_questionId_fkey";
ALTER TABLE "bookmark" DROP CONSTRAINT "bookmark_quizId_fkey";
ALTER TABLE "bookmark" DROP CONSTRAINT "bookmark_routineId_fkey";
ALTER TABLE "bookmark" DROP CONSTRAINT "bookmark_standardId_fkey";
ALTER TABLE "bookmark" DROP CONSTRAINT "bookmark_tagId_fkey";
ALTER TABLE "bookmark" DROP CONSTRAINT "bookmark_teamId_fkey";
ALTER TABLE "bookmark" DROP CONSTRAINT "bookmark_userId_fkey";

-- 6. Drop the old columns.
ALTER TABLE "bookmark" 
  DROP COLUMN "apiId",
  DROP COLUMN "codeId",
  DROP COLUMN "commentId",
  DROP COLUMN "issueId",
  DROP COLUMN "noteId",
  DROP COLUMN "postId",
  DROP COLUMN "projectId",
  DROP COLUMN "questionAnswerId",
  DROP COLUMN "questionId",
  DROP COLUMN "quizId",
  DROP COLUMN "routineId",
  DROP COLUMN "standardId",
  DROP COLUMN "tagId",
  DROP COLUMN "teamId",
  DROP COLUMN "userId";

-- 7. Create an index on the new polymorphic columns.
CREATE INDEX "bookmark_targetType_targetId_idx" ON "bookmark"("targetType", "targetId");

-- (Optional: Any additional changes, for example:)
ALTER TABLE "run_project" ADD COLUMN "data" VARCHAR(16384);
