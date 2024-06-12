-- DropIndex
DROP INDEX "phone_verificationCode_key";

-- AlterTable
ALTER TABLE "premium" ADD COLUMN     "hasReceivedFreeTrial" BOOLEAN NOT NULL DEFAULT false;
