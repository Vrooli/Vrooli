/*
  Warnings:

  - You are about to drop the column `multiStepParentId` on the `routine_version` table. All the data in the column will be lost.

*/
-- DropForeignKey
ALTER TABLE "routine_version" DROP CONSTRAINT "routine_version_multiStepParentId_fkey";

-- AlterTable
ALTER TABLE "routine_version" DROP COLUMN "multiStepParentId";

-- CreateTable
CREATE TABLE "routine_version_subroutine" (
    "id" UUID NOT NULL,
    "parentRoutineId" UUID NOT NULL,
    "subroutineId" UUID NOT NULL,

    CONSTRAINT "routine_version_subroutine_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "routine_version_subroutine_parentRoutineId_subroutineId_key" ON "routine_version_subroutine"("parentRoutineId", "subroutineId");

-- AddForeignKey
ALTER TABLE "routine_version_subroutine" ADD CONSTRAINT "routine_version_subroutine_parentRoutineId_fkey" FOREIGN KEY ("parentRoutineId") REFERENCES "routine_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_version_subroutine" ADD CONSTRAINT "routine_version_subroutine_subroutineId_fkey" FOREIGN KEY ("subroutineId") REFERENCES "routine_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;
