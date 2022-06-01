-- AlterTable
ALTER TABLE "resource_list" ADD COLUMN     "standardId" UUID;

-- AddForeignKey
ALTER TABLE "resource_list" ADD CONSTRAINT "resource_list_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;
