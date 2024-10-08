/*
  Warnings:

  - You are about to drop the column `standardType` on the `standard_version` table. All the data in the column will be lost.

*/
-- CreateEnum
CREATE TYPE "StandardType" AS ENUM ('DataStructure', 'Prompt');

-- AlterTable
ALTER TABLE "standard_version" DROP COLUMN "standardType",
ADD COLUMN     "codeLanguage" VARCHAR(128) NOT NULL DEFAULT 'json',
ADD COLUMN     "variant" "StandardType" NOT NULL DEFAULT 'DataStructure';
