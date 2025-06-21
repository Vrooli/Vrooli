// Test file to validate imports
import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";

console.log("Testing imports...");
console.log("generatePK type:", typeof generatePK);
console.log("generatePublicId type:", typeof generatePublicId);
console.log("nanoid type:", typeof nanoid);

// Test the functions
try {
    const pk = generatePK();
    console.log("Generated PK:", pk, "type:", typeof pk);
    
    const pubId = generatePublicId();
    console.log("Generated Public ID:", pubId, "type:", typeof pubId);
    
    const nano = nanoid();
    console.log("Generated nanoid:", nano, "type:", typeof nano);
    
    // Test Prisma types
    console.log("Prisma namespace available:", typeof Prisma !== 'undefined');
    console.log("PrismaClient available:", typeof PrismaClient !== 'undefined');
    
} catch (error) {
    console.error("Error testing imports:", error);
}

export {};