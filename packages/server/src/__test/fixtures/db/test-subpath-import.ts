// Test subpath imports from @vrooli/shared
import { generatePK } from "@vrooli/shared/id/snowflake.js";
import { generatePublicId, nanoid } from "@vrooli/shared/id/publicId.js";

console.log("Testing subpath imports...");
console.log("generatePK:", typeof generatePK);
console.log("generatePublicId:", typeof generatePublicId);
console.log("nanoid:", typeof nanoid);

export {};
