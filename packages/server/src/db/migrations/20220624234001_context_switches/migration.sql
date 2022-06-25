/*
  Warnings:

  - You are about to drop the column `pickups` on the `run` table. All the data in the column will be lost.

*/
-- AlterTable
ALTER TABLE "run" DROP COLUMN "pickups",
ADD COLUMN     "contextSwitches" INTEGER NOT NULL DEFAULT 0;
