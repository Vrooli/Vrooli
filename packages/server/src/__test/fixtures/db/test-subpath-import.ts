// Test subpath imports from @vrooli/shared
import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";

console.log("Testing subpath imports...");
console.log("generatePK:", typeof generatePK);
console.log("generatePublicId:", typeof generatePublicId);
console.log("nanoid:", typeof nanoid);

export {};
