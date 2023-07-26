/*
  Warnings:

  - You are about to drop the column `lastSessionVerified` on the `user` table. All the data in the column will be lost.
  - A unique constraint covering the columns `[stripeCustomerId]` on the table `user` will be added. If there are existing duplicate values, this will fail.

*/
-- AlterTable
ALTER TABLE "user" DROP COLUMN "lastSessionVerified",
ADD COLUMN     "stripeCustomerId" VARCHAR(255);

-- CreateIndex
CREATE UNIQUE INDEX "user_stripeCustomerId_key" ON "user"("stripeCustomerId");
