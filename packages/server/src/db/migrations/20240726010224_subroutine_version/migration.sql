/*
  Warnings:

  - You are about to drop the column `subroutineVersionId` on the `run_routine_step` table. All the data in the column will be lost.

*/
-- DropForeignKey
ALTER TABLE "run_routine_step" DROP CONSTRAINT "run_routine_step_subroutineVersionId_fkey";

-- AlterTable
ALTER TABLE "run_routine_step" DROP COLUMN "subroutineVersionId",
ADD COLUMN     "subroutineId" UUID;

-- AddForeignKey
ALTER TABLE "run_routine_step" ADD CONSTRAINT "run_routine_step_subroutineId_fkey" FOREIGN KEY ("subroutineId") REFERENCES "routine_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;
