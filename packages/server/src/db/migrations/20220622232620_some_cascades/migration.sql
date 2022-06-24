-- DropForeignKey
ALTER TABLE "comment" DROP CONSTRAINT "comment_parentId_fkey";

-- DropForeignKey
ALTER TABLE "star" DROP CONSTRAINT "star_commentId_fkey";

-- DropForeignKey
ALTER TABLE "star" DROP CONSTRAINT "star_organizationId_fkey";

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "star" ADD CONSTRAINT "star_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "star" ADD CONSTRAINT "star_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;
