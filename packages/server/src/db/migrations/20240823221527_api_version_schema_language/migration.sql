/*
  Warnings:

  - Added the required column `schemaLanguage` to the `api_version` table without a default value. This is not possible if the table is not empty.

*/
-- AlterTable
ALTER TABLE "api_version" ADD COLUMN     "schemaLanguage" VARCHAR(128) NOT NULL;
