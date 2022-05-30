-- AlterTable
ALTER TABLE "run_step" ADD COLUMN     "subroutineId" UUID;

-- AddForeignKey
ALTER TABLE "run_step" ADD CONSTRAINT "run_step_subroutineId_fkey" FOREIGN KEY ("subroutineId") REFERENCES "routine"("id") ON DELETE SET NULL ON UPDATE CASCADE;
