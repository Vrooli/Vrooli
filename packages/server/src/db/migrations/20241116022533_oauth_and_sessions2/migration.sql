/*
  Warnings:

  - You are about to drop the column `lastResetPasswordReqestAttempt` on the `user` table. All the data in the column will be lost.

*/
-- AlterTable
ALTER TABLE "session" ALTER COLUMN "id" DROP DEFAULT,
ALTER COLUMN "updated_at" DROP DEFAULT;

-- AlterTable
ALTER TABLE "user" DROP COLUMN "lastResetPasswordReqestAttempt";

-- AlterTable
ALTER TABLE "user_auth" ADD COLUMN     "lastResetPasswordRequestAttempt" TIMESTAMPTZ(6),
ALTER COLUMN "id" DROP DEFAULT,
ALTER COLUMN "updated_at" DROP DEFAULT;
