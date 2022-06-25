/*
  Warnings:

  - You are about to drop the column `pickups` on the `run_step` table. All the data in the column will be lost.

*/
-- AlterTable
ALTER TABLE "run_step" DROP COLUMN "pickups",
ADD COLUMN     "contextSwitches" INTEGER NOT NULL DEFAULT 0;
