/*
  NOTE: Tried polymorphism for certain tables, but it doesn't align with our business logic. We need 
  to be able to query relations in a single select for things like permissions and labels

  Warnings:

  - You are about to drop the column `targetId` on the `bookmark` table. All the data in the column will be lost.
  - You are about to drop the column `targetType` on the `bookmark` table. All the data in the column will be lost.

*/
-- DropIndex
DROP INDEX "bookmark_targetType_targetId_idx";

-- AlterTable
ALTER TABLE "bookmark" DROP COLUMN "targetId",
DROP COLUMN "targetType",
ADD COLUMN     "apiId" UUID,
ADD COLUMN     "codeId" UUID,
ADD COLUMN     "commentId" UUID,
ADD COLUMN     "issueId" UUID,
ADD COLUMN     "noteId" UUID,
ADD COLUMN     "postId" UUID,
ADD COLUMN     "projectId" UUID,
ADD COLUMN     "questionAnswerId" UUID,
ADD COLUMN     "questionId" UUID,
ADD COLUMN     "quizId" UUID,
ADD COLUMN     "routineId" UUID,
ADD COLUMN     "standardId" UUID,
ADD COLUMN     "tagId" UUID,
ADD COLUMN     "teamId" UUID,
ADD COLUMN     "userId" UUID;

-- DropEnum
DROP TYPE "BookmarkFor";

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_apiId_fkey" FOREIGN KEY ("apiId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_codeId_fkey" FOREIGN KEY ("codeId") REFERENCES "code"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_noteId_fkey" FOREIGN KEY ("noteId") REFERENCES "note"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_postId_fkey" FOREIGN KEY ("postId") REFERENCES "post"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_questionId_fkey" FOREIGN KEY ("questionId") REFERENCES "question"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_questionAnswerId_fkey" FOREIGN KEY ("questionAnswerId") REFERENCES "question_answer"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_quizId_fkey" FOREIGN KEY ("quizId") REFERENCES "quiz"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_tagId_fkey" FOREIGN KEY ("tagId") REFERENCES "tag"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;
