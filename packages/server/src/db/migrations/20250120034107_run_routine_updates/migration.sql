/*
  Warnings:

  - You are about to drop the `run_routine_input` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `run_routine_output` table. If the table is not empty, all the data it contains will be lost.

*/
-- DropForeignKey
ALTER TABLE "run_routine_input" DROP CONSTRAINT "run_routine_input_inputId_fkey";

-- DropForeignKey
ALTER TABLE "run_routine_input" DROP CONSTRAINT "run_routine_input_runRoutineId_fkey";

-- DropForeignKey
ALTER TABLE "run_routine_output" DROP CONSTRAINT "run_routine_output_outputId_fkey";

-- DropForeignKey
ALTER TABLE "run_routine_output" DROP CONSTRAINT "run_routine_output_runRoutineId_fkey";

-- AlterTable
ALTER TABLE "run_routine" ADD COLUMN     "data" VARCHAR(16384);

-- DropTable
DROP TABLE "run_routine_input";

-- DropTable
DROP TABLE "run_routine_output";

-- CreateTable
CREATE TABLE "run_routine_io" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "data" VARCHAR(8192) NOT NULL,
    "nodeInputName" VARCHAR(128) NOT NULL,
    "nodeName" VARCHAR(128) NOT NULL,
    "runRoutineId" UUID NOT NULL,
    "runRoutineInputId" UUID,
    "runRoutineOutputId" UUID,

    CONSTRAINT "run_routine_io_pkey" PRIMARY KEY ("id")
);

-- AddForeignKey
ALTER TABLE "run_routine_io" ADD CONSTRAINT "run_routine_io_runRoutineId_fkey" FOREIGN KEY ("runRoutineId") REFERENCES "run_routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine_io" ADD CONSTRAINT "run_routine_io_runRoutineInputId_fkey" FOREIGN KEY ("runRoutineInputId") REFERENCES "routine_version_input"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine_io" ADD CONSTRAINT "run_routine_io_runRoutineOutputId_fkey" FOREIGN KEY ("runRoutineOutputId") REFERENCES "routine_version_output"("id") ON DELETE SET NULL ON UPDATE CASCADE;
