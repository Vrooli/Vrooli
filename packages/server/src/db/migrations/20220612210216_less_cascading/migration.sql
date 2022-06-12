-- DropForeignKey
ALTER TABLE "comment" DROP CONSTRAINT "comment_projectId_fkey";

-- DropForeignKey
ALTER TABLE "comment" DROP CONSTRAINT "comment_routineId_fkey";

-- DropForeignKey
ALTER TABLE "comment" DROP CONSTRAINT "comment_standardId_fkey";

-- DropForeignKey
ALTER TABLE "project" DROP CONSTRAINT "project_createdByOrganizationId_fkey";

-- DropForeignKey
ALTER TABLE "project" DROP CONSTRAINT "project_createdByUserId_fkey";

-- DropForeignKey
ALTER TABLE "project" DROP CONSTRAINT "project_organizationId_fkey";

-- DropForeignKey
ALTER TABLE "project" DROP CONSTRAINT "project_userId_fkey";

-- DropForeignKey
ALTER TABLE "report" DROP CONSTRAINT "report_fromId_fkey";

-- DropForeignKey
ALTER TABLE "routine" DROP CONSTRAINT "routine_createdByOrganizationId_fkey";

-- DropForeignKey
ALTER TABLE "routine" DROP CONSTRAINT "routine_createdByUserId_fkey";

-- DropForeignKey
ALTER TABLE "routine" DROP CONSTRAINT "routine_organizationId_fkey";

-- DropForeignKey
ALTER TABLE "routine" DROP CONSTRAINT "routine_projectId_fkey";

-- DropForeignKey
ALTER TABLE "routine" DROP CONSTRAINT "routine_userId_fkey";

-- DropForeignKey
ALTER TABLE "routine_input" DROP CONSTRAINT "routine_input_standardId_fkey";

-- DropForeignKey
ALTER TABLE "routine_output" DROP CONSTRAINT "routine_output_standardId_fkey";

-- DropForeignKey
ALTER TABLE "standard" DROP CONSTRAINT "standard_createdByOrganizationId_fkey";

-- DropForeignKey
ALTER TABLE "standard" DROP CONSTRAINT "standard_createdByUserId_fkey";

-- DropForeignKey
ALTER TABLE "tag" DROP CONSTRAINT "tag_createdByUserId_fkey";

-- DropForeignKey
ALTER TABLE "view" DROP CONSTRAINT "view_userId_fkey";

-- AlterTable
ALTER TABLE "comment" ADD COLUMN     "parentId" UUID;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "comment"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project" ADD CONSTRAINT "project_createdByOrganizationId_fkey" FOREIGN KEY ("createdByOrganizationId") REFERENCES "organization"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project" ADD CONSTRAINT "project_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project" ADD CONSTRAINT "project_createdByUserId_fkey" FOREIGN KEY ("createdByUserId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project" ADD CONSTRAINT "project_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_fromId_fkey" FOREIGN KEY ("fromId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine" ADD CONSTRAINT "routine_createdByOrganizationId_fkey" FOREIGN KEY ("createdByOrganizationId") REFERENCES "organization"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine" ADD CONSTRAINT "routine_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine" ADD CONSTRAINT "routine_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine" ADD CONSTRAINT "routine_createdByUserId_fkey" FOREIGN KEY ("createdByUserId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine" ADD CONSTRAINT "routine_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_input" ADD CONSTRAINT "routine_input_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_output" ADD CONSTRAINT "routine_output_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "standard" ADD CONSTRAINT "standard_createdByOrganizationId_fkey" FOREIGN KEY ("createdByOrganizationId") REFERENCES "organization"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "standard" ADD CONSTRAINT "standard_createdByUserId_fkey" FOREIGN KEY ("createdByUserId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "tag" ADD CONSTRAINT "tag_createdByUserId_fkey" FOREIGN KEY ("createdByUserId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;
