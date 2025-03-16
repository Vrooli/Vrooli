/*
  Warnings:

  - You are about to drop the column `runRoutineInputId` on the `run_routine_io` table. All the data in the column will be lost.
  - You are about to drop the column `runRoutineOutputId` on the `run_routine_io` table. All the data in the column will be lost.

*/
-- DropForeignKey
ALTER TABLE "run_routine_io" DROP CONSTRAINT "run_routine_io_runRoutineInputId_fkey";

-- DropForeignKey
ALTER TABLE "run_routine_io" DROP CONSTRAINT "run_routine_io_runRoutineOutputId_fkey";

-- AlterTable
ALTER TABLE "run_routine_io" DROP COLUMN "runRoutineInputId",
DROP COLUMN "runRoutineOutputId",
ADD COLUMN     "routineVersionInputId" UUID,
ADD COLUMN     "routineVersionOutputId" UUID;

-- AddForeignKey
ALTER TABLE "run_routine_io" ADD CONSTRAINT "run_routine_io_routineVersionInputId_fkey" FOREIGN KEY ("routineVersionInputId") REFERENCES "routine_version_input"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine_io" ADD CONSTRAINT "run_routine_io_routineVersionOutputId_fkey" FOREIGN KEY ("routineVersionOutputId") REFERENCES "routine_version_output"("id") ON DELETE SET NULL ON UPDATE CASCADE;
