/*
  Warnings:

  - You are about to drop the column `runProjectId` on the `run_routine` table. All the data in the column will be lost.

*/
-- DropForeignKey
ALTER TABLE "run_routine" DROP CONSTRAINT "run_routine_runProjectId_fkey";

-- AlterTable
ALTER TABLE "run_routine" DROP COLUMN "runProjectId";
