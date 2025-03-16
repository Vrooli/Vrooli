/*
  Warnings:

  - Added the required column `directoryInId` to the `run_project_step` table without a default value. This is not possible if the table is not empty.
  - Added the required column `subroutineInId` to the `run_routine_step` table without a default value. This is not possible if the table is not empty.
  - Made the column `nodeId` on table `run_routine_step` required. This step will fail if there are existing NULL values in that column.

*/
-- AlterTable
ALTER TABLE "run_project_step" ADD COLUMN     "complexity" INTEGER NOT NULL DEFAULT 0,
ADD COLUMN     "directoryInId" UUID NOT NULL,
ALTER COLUMN "order" SET DEFAULT 0;

-- AlterTable
ALTER TABLE "run_routine_step" ADD COLUMN     "complexity" INTEGER NOT NULL DEFAULT 0,
ADD COLUMN     "subroutineInId" UUID NOT NULL,
ALTER COLUMN "order" SET DEFAULT 0,
ALTER COLUMN "nodeId" SET NOT NULL;
