/*
  Warnings:

  - You are about to alter the column `configFormInput` on the `routine_version` table. The data in that column could be lost. The data in that column will be cast from `VarChar(16384)` to `VarChar(8192)`.
  - You are about to alter the column `configFormOutput` on the `routine_version` table. The data in that column could be lost. The data in that column will be cast from `VarChar(16384)` to `VarChar(8192)`.

*/
-- AlterTable
ALTER TABLE "routine_version" ALTER COLUMN "configFormInput" SET DATA TYPE VARCHAR(8192),
ALTER COLUMN "configFormOutput" SET DATA TYPE VARCHAR(8192);

-- AlterTable
ALTER TABLE "routine_version_translation" ALTER COLUMN "instructions" DROP NOT NULL;
