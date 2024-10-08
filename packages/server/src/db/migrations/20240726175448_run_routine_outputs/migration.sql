-- CreateTable
CREATE TABLE "run_routine_output" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "data" VARCHAR(8192) NOT NULL,
    "outputId" UUID NOT NULL,
    "runRoutineId" UUID NOT NULL,

    CONSTRAINT "run_routine_output_pkey" PRIMARY KEY ("id")
);

-- AddForeignKey
ALTER TABLE "run_routine_output" ADD CONSTRAINT "run_routine_output_outputId_fkey" FOREIGN KEY ("outputId") REFERENCES "routine_version_output"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine_output" ADD CONSTRAINT "run_routine_output_runRoutineId_fkey" FOREIGN KEY ("runRoutineId") REFERENCES "run_routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;
