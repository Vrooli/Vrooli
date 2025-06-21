#!/usr/bin/env tsx
/**
 * Factory Validation Tool
 * 
 * This tool validates that database factories follow the correct patterns
 * and implement all required methods.
 * 
 * Usage: npm run fixtures:validate
 */

import { readFileSync, readdirSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import * as ts from 'typescript';

const __dirname = dirname(fileURLToPath(import.meta.url));

interface ValidationIssue {
    file: string;
    type: 'error' | 'warning';
    message: string;
    line?: number;
}

interface ValidationReport {
    totalFactories: number;
    validFactories: number;
    issues: ValidationIssue[];
}

/**
 * Required methods that every factory must implement
 */
const REQUIRED_METHODS = [
    'getPrismaDelegate',
    'generateMinimalData',
    'generateCompleteData',
    'getFixtures',
];

/**
 * Required fixture categories
 */
const REQUIRED_FIXTURES = [
    'minimal',
    'complete',
    'invalid',
    'edgeCases',
    'updates',
];

/**
 * Validate a single factory file
 */
function validateFactory(filePath: string): ValidationIssue[] {
    const issues: ValidationIssue[] = [];
    const fileName = filePath.split('/').pop() || '';
    
    try {
        const content = readFileSync(filePath, 'utf-8');
        
        // Check for extends EnhancedDatabaseFactory
        if (!content.includes('extends EnhancedDatabaseFactory')) {
            issues.push({
                file: fileName,
                type: 'error',
                message: 'Factory must extend EnhancedDatabaseFactory',
            });
        }
        
        // Check for required methods
        REQUIRED_METHODS.forEach(method => {
            const methodRegex = new RegExp(`protected\\s+${method}\\s*\\(`, 'g');
            if (!methodRegex.test(content)) {
                issues.push({
                    file: fileName,
                    type: 'error',
                    message: `Missing required method: ${method}()`,
                });
            }
        });
        
        // Check for factory function export
        const factoryName = fileName.replace('.ts', '');
        const createFunctionName = `create${factoryName}`;
        if (!content.includes(`export function ${createFunctionName}`) && 
            !content.includes(`export const ${createFunctionName}`)) {
            issues.push({
                file: fileName,
                type: 'warning',
                message: `Missing factory creation function: ${createFunctionName}()`,
            });
        }
        
        // Check for proper type parameters
        if (content.includes('extends EnhancedDatabaseFactory<any')) {
            issues.push({
                file: fileName,
                type: 'warning',
                message: 'Using "any" type parameter - should use proper Prisma types',
            });
        }
        
        // Check fixture structure
        const fixturesMatch = content.match(/getFixtures\(\)[^{]*{([\s\S]*?)};[\s]*}/s);
        if (fixturesMatch) {
            const fixturesContent = fixturesMatch[1];
            REQUIRED_FIXTURES.forEach(fixture => {
                if (!fixturesContent.includes(`${fixture}:`)) {
                    issues.push({
                        file: fileName,
                        type: 'warning',
                        message: `Missing fixture category: ${fixture}`,
                    });
                }
            });
        }
        
        // Check for proper imports
        if (!content.includes('import { type Prisma, type PrismaClient } from "@prisma/client"') &&
            !content.includes('import type { Prisma, PrismaClient } from "@prisma/client"')) {
            issues.push({
                file: fileName,
                type: 'error',
                message: 'Missing or incorrect Prisma imports',
            });
        }
        
        // Check for ID generation
        if (!content.includes('generatePK') && !content.includes('BigInt(')) {
            issues.push({
                file: fileName,
                type: 'warning',
                message: 'No ID generation found - ensure IDs are properly generated',
            });
        }
        
    } catch (error) {
        issues.push({
            file: fileName,
            type: 'error',
            message: `Failed to read or parse file: ${error}`,
        });
    }
    
    return issues;
}

/**
 * Get all factory files
 */
function getFactoryFiles(): string[] {
    const factoryDir = join(__dirname, '..');
    const files = readdirSync(factoryDir);
    
    return files
        .filter(file => 
            file.endsWith('DbFactory.ts') && 
            !file.includes('Enhanced') && 
            !file.includes('TEMPLATE')
        )
        .map(file => join(factoryDir, file));
}

/**
 * Validate all factories
 */
function validateAllFactories(): ValidationReport {
    const factoryFiles = getFactoryFiles();
    const allIssues: ValidationIssue[] = [];
    
    factoryFiles.forEach(file => {
        const issues = validateFactory(file);
        allIssues.push(...issues);
    });
    
    const validFactories = factoryFiles.filter(file => {
        const fileIssues = allIssues.filter(issue => 
            issue.file === file.split('/').pop() && issue.type === 'error'
        );
        return fileIssues.length === 0;
    });
    
    return {
        totalFactories: factoryFiles.length,
        validFactories: validFactories.length,
        issues: allIssues,
    };
}

/**
 * Print validation report
 */
function printReport(report: ValidationReport): void {
    console.log('\n=== Database Factory Validation Report ===\n');
    
    console.log(`Total Factories: ${report.totalFactories}`);
    console.log(`Valid Factories: ${report.validFactories}`);
    console.log(`Issues Found: ${report.issues.length}`);
    
    if (report.issues.length > 0) {
        console.log('\n⚠️  Issues:\n');
        
        // Group by file
        const issuesByFile = new Map<string, ValidationIssue[]>();
        report.issues.forEach(issue => {
            const existing = issuesByFile.get(issue.file) || [];
            existing.push(issue);
            issuesByFile.set(issue.file, existing);
        });
        
        issuesByFile.forEach((issues, file) => {
            console.log(`📄 ${file}:`);
            issues.forEach(issue => {
                const icon = issue.type === 'error' ? '❌' : '⚠️';
                console.log(`   ${icon} ${issue.message}`);
            });
            console.log('');
        });
    } else {
        console.log('\n✅ All factories are valid!');
    }
    
    console.log('=== End Report ===\n');
}

// Main execution
if (import.meta.url === `file://${process.argv[1]}`) {
    const report = validateAllFactories();
    printReport(report);
    
    // Exit with error code if there are errors
    const hasErrors = report.issues.some(issue => issue.type === 'error');
    if (hasErrors) {
        process.exit(1);
    }
}

export { validateAllFactories, ValidationReport };