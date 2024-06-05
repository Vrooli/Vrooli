/*
  Warnings:

  - You are about to drop the column `contractType` on the `code_version` table. All the data in the column will be lost.
  - You are about to drop the column `apiCallData` on the `routine_version` table. All the data in the column will be lost.
  - You are about to drop the column `codeCallData` on the `routine_version` table. All the data in the column will be lost.
  - Added the required column `codeLanguage` to the `code_version` table without a default value. This is not possible if the table is not empty.

*/
-- CreateEnum
CREATE TYPE "CodeType" AS ENUM ('DataConvert', 'SmartContract');

-- CreateEnum
CREATE TYPE "RoutineType" AS ENUM ('Action', 'Api', 'Code', 'Data', 'Generate', 'Informational', 'MultiStep', 'SmartContract');

-- AlterTable
ALTER TABLE "code_version" DROP COLUMN "contractType",
ADD COLUMN     "codeLanguage" VARCHAR(128) NOT NULL,
ADD COLUMN     "codeType" "CodeType" NOT NULL DEFAULT 'DataConvert';

-- AlterTable
ALTER TABLE "routine_version" DROP COLUMN "apiCallData",
DROP COLUMN "codeCallData",
ADD COLUMN     "configCallData" VARCHAR(8192),
ADD COLUMN     "routineType" "RoutineType" NOT NULL DEFAULT 'Informational';

-- CreateTable
CREATE TABLE "code_version_contract" (
    "id" UUID NOT NULL,
    "address" VARCHAR(256),
    "blockchain" VARCHAR(128) NOT NULL,
    "codeVersionId" UUID NOT NULL,
    "contractType" VARCHAR(256),
    "hash" VARCHAR(256),
    "isAddressVerified" BOOLEAN NOT NULL DEFAULT false,
    "isHashVerified" BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT "code_version_contract_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "code_version_contract_codeVersionId_key" ON "code_version_contract"("codeVersionId");

-- AddForeignKey
ALTER TABLE "code_version_contract" ADD CONSTRAINT "code_version_contract_codeVersionId_fkey" FOREIGN KEY ("codeVersionId") REFERENCES "code_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;
