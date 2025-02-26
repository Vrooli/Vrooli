/*
  Warnings:

  - You are about to drop the column `step` on the `run_project_step` table. All the data in the column will be lost.
  - You are about to drop the column `step` on the `run_routine_step` table. All the data in the column will be lost.

*/
-- AlterTable
ALTER TABLE "run_project_step" DROP COLUMN "step";

-- AlterTable
ALTER TABLE "run_routine_step" DROP COLUMN "step";
