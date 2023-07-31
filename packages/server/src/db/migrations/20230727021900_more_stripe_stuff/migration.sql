/*
  Warnings:

  - You are about to drop the column `cardExpDate` on the `payment` table. All the data in the column will be lost.
  - You are about to drop the column `cardLast4` on the `payment` table. All the data in the column will be lost.
  - You are about to drop the column `cardType` on the `payment` table. All the data in the column will be lost.
  - A unique constraint covering the columns `[stripeCustomerId]` on the table `organization` will be added. If there are existing duplicate values, this will fail.

*/
-- AlterTable
ALTER TABLE "organization" ADD COLUMN     "stripeCustomerId" VARCHAR(255);

-- AlterTable
ALTER TABLE "payment" DROP COLUMN "cardExpDate",
DROP COLUMN "cardLast4",
DROP COLUMN "cardType",
ADD COLUMN     "stripeSessionId" VARCHAR(255);

-- CreateIndex
CREATE UNIQUE INDEX "organization_stripeCustomerId_key" ON "organization"("stripeCustomerId");
