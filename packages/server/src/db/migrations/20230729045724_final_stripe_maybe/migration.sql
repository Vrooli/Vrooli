/*
  Warnings:

  - You are about to drop the column `stripeSessionId` on the `payment` table. All the data in the column will be lost.

*/
-- AlterTable
ALTER TABLE "payment" DROP COLUMN "stripeSessionId";
