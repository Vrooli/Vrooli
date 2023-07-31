-- AlterTable
ALTER TABLE "organization" ADD COLUMN     "bannerImage" VARCHAR(2048),
ADD COLUMN     "profileImage" VARCHAR(2048);

-- AlterTable
ALTER TABLE "user" ADD COLUMN     "bannerImage" VARCHAR(2048),
ADD COLUMN     "profileImage" VARCHAR(2048);
