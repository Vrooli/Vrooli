-- DropIndex
DROP INDEX "email_verificationCode_key";

-- AlterTable
ALTER TABLE "phone" ALTER COLUMN "verificationCode" SET DATA TYPE VARCHAR(16);
