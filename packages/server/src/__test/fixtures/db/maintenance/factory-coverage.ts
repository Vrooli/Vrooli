#!/usr/bin/env tsx
/**
 * Factory Coverage Analysis Tool
 * 
 * This tool analyzes the coverage of database factories against Prisma models.
 * It helps identify missing factories and validates existing ones.
 * 
 * Usage: npm run fixtures:coverage
 */

import { readFileSync, readdirSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));

interface CoverageReport {
    totalModels: number;
    coveredModels: number;
    missingFactories: string[];
    extraFactories: string[];
    coveragePercentage: number;
}

/**
 * Extract model names from Prisma schema
 */
function getPrismaModels(): string[] {
    const schemaPath = join(__dirname, "../../../../db/schema.prisma");
    const schemaContent = readFileSync(schemaPath, "utf-8");
    
    const modelRegex = /^model\s+(\w+)\s*{/gm;
    const models: string[] = [];
    
    let match;
    while ((match = modelRegex.exec(schemaContent)) !== null) {
        models.push(match[1]);
    }
    
    return models;
}

/**
 * Get existing factory files
 */
function getExistingFactories(): string[] {
    const factoryDir = join(__dirname, "..");
    const files = readdirSync(factoryDir);
    
    const factories: string[] = [];
    
    files.forEach(file => {
        if (file.endsWith("DbFactory.ts") && !file.includes("Enhanced") && !file.includes("TEMPLATE")) {
            // Extract model name from factory filename
            const modelName = file
                .replace("DbFactory.ts", "")
                .replace(/([A-Z])/g, (match, p1, offset) => offset > 0 ? "_" + p1 : p1)
                .toLowerCase();
            factories.push(modelName);
        }
    });
    
    return factories;
}

/**
 * Generate coverage report
 */
function generateCoverageReport(): CoverageReport {
    const prismaModels = getPrismaModels();
    const existingFactories = getExistingFactories();
    
    // Find missing factories
    const missingFactories = prismaModels.filter(model => 
        !existingFactories.includes(model) && 
        model !== "user_auth", // Special case - mapped to AuthDbFactory
    );
    
    // Find extra factories (not mapping to any model)
    const extraFactories = existingFactories.filter(factory => 
        !prismaModels.includes(factory),
    );
    
    const coveredModels = prismaModels.length - missingFactories.length;
    const coveragePercentage = Math.round((coveredModels / prismaModels.length) * 100);
    
    return {
        totalModels: prismaModels.length,
        coveredModels,
        missingFactories,
        extraFactories,
        coveragePercentage,
    };
}

/**
 * Format factory name from model name
 */
function formatFactoryName(modelName: string): string {
    return modelName
        .split("_")
        .map(part => part.charAt(0).toUpperCase() + part.slice(1))
        .join("") + "DbFactory";
}

/**
 * Print coverage report
 */
function printReport(report: CoverageReport): void {
    console.log("\n=== Database Factory Coverage Report ===\n");
    
    console.log(`Total Prisma Models: ${report.totalModels}`);
    console.log(`Covered Models: ${report.coveredModels}`);
    console.log(`Coverage: ${report.coveragePercentage}%`);
    
    if (report.missingFactories.length > 0) {
        console.log("\n❌ Missing Factories:");
        report.missingFactories.forEach(model => {
            console.log(`   - ${model} → ${formatFactoryName(model)}.ts`);
        });
    }
    
    if (report.extraFactories.length > 0) {
        console.log("\n⚠️  Extra Factories (no corresponding model):");
        report.extraFactories.forEach(factory => {
            console.log(`   - ${factory}`);
        });
    }
    
    if (report.coveragePercentage === 100 && report.extraFactories.length === 0) {
        console.log("\n✅ Perfect coverage! All models have corresponding factories.");
    }
    
    console.log("\n=== End Report ===\n");
}

/**
 * Generate creation script for missing factories
 */
function generateCreationScript(missingFactories: string[]): void {
    if (missingFactories.length === 0) return;
    
    console.log("\n📝 Script to create missing factories:\n");
    console.log("```bash");
    missingFactories.forEach(model => {
        const factoryName = formatFactoryName(model);
        console.log(`# Create ${factoryName}`);
        console.log(`cp FACTORY_TEMPLATE.ts ${factoryName}.ts`);
        console.log(`# Then replace MODEL_NAME with '${model}' and ModelName with '${factoryName.replace("DbFactory", "")}'`);
        console.log("");
    });
    console.log("```\n");
}

// Main execution
if (import.meta.url === `file://${process.argv[1]}`) {
    const report = generateCoverageReport();
    printReport(report);
    
    if (report.missingFactories.length > 0) {
        generateCreationScript(report.missingFactories);
    }
    
    // Exit with error code if coverage is not complete
    if (report.coveragePercentage < 100) {
        process.exit(1);
    }
}

export { generateCoverageReport, CoverageReport };
