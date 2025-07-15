/**
 * Post-Migration Validator
 * 
 * Validates migrated files for syntax errors, proper functionality,
 * and compliance with patterns.
 */

import fs from 'fs';
import path from 'path';
import { exec } from 'child_process';
import { promisify } from 'util';

const execAsync = promisify(exec);

export interface ValidationResult {
    filePath: string;
    fileName: string;
    syntaxValid: boolean;
    compileValid: boolean;
    lintValid: boolean;
    testValid: boolean;
    errors: string[];
    warnings: string[];
}

export class PostMigrationValidator {
    async validateFile(filePath: string): Promise<ValidationResult> {
        const fileName = path.basename(filePath);
        const result: ValidationResult = {
            filePath,
            fileName,
            syntaxValid: false,
            compileValid: false,
            lintValid: false,
            testValid: false,
            errors: [],
            warnings: [],
        };

        try {
            // Check syntax
            await this.checkSyntax(filePath, result);
            
            // Check compilation
            await this.checkCompilation(filePath, result);
            
            // Check linting
            await this.checkLinting(filePath, result);
            
            // Check test functionality (basic)
            await this.checkTestStructure(filePath, result);

        } catch (error) {
            result.errors.push(`Validation failed: ${error instanceof Error ? error.message : String(error)}`);
        }

        return result;
    }

    async validateBatch(filePaths: string[]): Promise<ValidationResult[]> {
        const results: ValidationResult[] = [];
        
        for (const filePath of filePaths) {
            const result = await this.validateFile(filePath);
            results.push(result);
        }
        
        return results;
    }

    async fixCommonIssues(filePath: string): Promise<{ fixed: boolean; changes: string[] }> {
        const changes: string[] = [];
        let content = await fs.promises.readFile(filePath, 'utf-8');
        const originalContent = content;

        // Fix common syntax issues
        
        // 1. Double closing parentheses
        const doubleParenRegex = /\}\);?\);/g;
        if (doubleParenRegex.test(content)) {
            content = content.replace(doubleParenRegex, '});');
            changes.push('Fixed double closing parentheses');
        }

        // 2. Missing closing braces
        const beforeEachRegex = /beforeEach\s*\([^)]*\)\s*=>\s*\{([^}]*(?:\{[^}]*\}[^}]*)*)\};?\);?/g;
        content = content.replace(beforeEachRegex, (match, body) => {
            if (!body.includes('}')) {
                return match;
            }
            return `beforeEach(async () => {${body}\n    });`;
        });

        // 3. Missing async keyword
        const beforeEachAsyncRegex = /beforeEach\s*\(\s*\(\s*\)\s*=>\s*\{/g;
        if (beforeEachAsyncRegex.test(content)) {
            content = content.replace(beforeEachAsyncRegex, 'beforeEach(async () => {');
            changes.push('Added missing async keyword to beforeEach');
        }

        // 4. Incorrect table array formatting
        const tableArrayRegex = /tables:\s*\[([^\]]*)\]/g;
        content = content.replace(tableArrayRegex, (match, tables) => {
            // Ensure proper string formatting
            const cleanTables = tables
                .split(',')
                .map((table: string) => table.trim().replace(/^["']|["']$/g, ''))
                .filter((table: string) => table.length > 0)
                .map((table: string) => `"${table}"`)
                .join(',');
            return `tables: [${cleanTables}]`;
        });

        // 5. Fix import formatting
        const importRegex = /import\s*\{\s*([^}]+)\s*\}\s*from\s*["']([^"']+)["']/g;
        content = content.replace(importRegex, (match, imports, path) => {
            const cleanImports = imports
                .split(',')
                .map((imp: string) => imp.trim())
                .join(', ');
            return `import { ${cleanImports} } from "${path}"`;
        });

        const fixed = content !== originalContent;
        
        if (fixed) {
            await fs.promises.writeFile(filePath, content);
        }

        return { fixed, changes };
    }

    private async checkSyntax(filePath: string, result: ValidationResult): Promise<void> {
        try {
            const content = await fs.promises.readFile(filePath, 'utf-8');
            
            // Basic syntax checks
            const openBraces = (content.match(/\{/g) || []).length;
            const closeBraces = (content.match(/\}/g) || []).length;
            const openParens = (content.match(/\(/g) || []).length;
            const closeParens = (content.match(/\)/g) || []).length;

            if (openBraces !== closeBraces) {
                result.errors.push(`Brace mismatch: ${openBraces} open, ${closeBraces} close`);
                return;
            }

            if (openParens !== closeParens) {
                result.errors.push(`Parentheses mismatch: ${openParens} open, ${closeParens} close`);
                return;
            }

            // Check for common issues
            if (content.includes('}););')) {
                result.errors.push('Double closing parentheses detected');
                return;
            }

            if (content.includes('beforeEach(()')) {
                result.warnings.push('beforeEach missing async keyword');
            }

            result.syntaxValid = true;

        } catch (error) {
            result.errors.push(`Syntax check failed: ${error instanceof Error ? error.message : String(error)}`);
        }
    }

    private async checkCompilation(filePath: string, result: ValidationResult): Promise<void> {
        try {
            // Use TypeScript compiler to check for compilation errors
            const { stderr } = await execAsync(`npx tsc --noEmit "${filePath}"`);
            
            if (stderr && stderr.includes('error')) {
                result.errors.push(`Compilation errors: ${stderr}`);
                return;
            }

            result.compileValid = true;

        } catch (error) {
            // TypeScript compiler exits with non-zero on errors
            const errorMessage = error instanceof Error ? error.message : String(error);
            if (errorMessage.includes('error TS')) {
                result.errors.push(`TypeScript errors found: ${errorMessage}`);
            } else {
                result.warnings.push(`Compilation check inconclusive: ${errorMessage}`);
            }
        }
    }

    private async checkLinting(filePath: string, result: ValidationResult): Promise<void> {
        try {
            await execAsync(`npx eslint "${filePath}" --no-eslintrc --config .eslintrc.test.cjs`);
            result.lintValid = true;
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            if (errorMessage.includes('Parse error') || errorMessage.includes('Parsing error')) {
                result.errors.push(`ESLint parse errors: ${errorMessage}`);
            } else {
                result.warnings.push(`Linting issues: ${errorMessage}`);
                result.lintValid = true; // Don't fail on lint warnings
            }
        }
    }

    private async checkTestStructure(filePath: string, result: ValidationResult): Promise<void> {
        try {
            const content = await fs.promises.readFile(filePath, 'utf-8');

            // Check for required test patterns
            const hasDescribe = content.includes('describe(');
            const hasBeforeEach = content.includes('beforeEach(');
            const hasAfterEach = content.includes('afterEach(');
            const hasCleanupGroups = content.includes('cleanupGroups.');
            const hasValidateCleanup = content.includes('validateCleanup(');

            if (!hasDescribe) {
                result.warnings.push('No describe blocks found');
            }

            if (!hasBeforeEach) {
                result.warnings.push('No beforeEach found');
            }

            if (!hasAfterEach) {
                result.warnings.push('No afterEach found');
            }

            if (!hasCleanupGroups) {
                result.warnings.push('No cleanupGroups usage found');
            }

            if (!hasValidateCleanup) {
                result.warnings.push('No validateCleanup found');
            }

            result.testValid = hasDescribe && hasBeforeEach;

        } catch (error) {
            result.errors.push(`Test structure check failed: ${error instanceof Error ? error.message : String(error)}`);
        }
    }
}

// CLI interface for validation
if (import.meta.url === `file://${process.argv[1]}`) {
    const validator = new PostMigrationValidator();
    const args = process.argv.slice(2);
    
    const filePaths = args.filter(arg => !arg.startsWith('--'));
    const autoFix = args.includes('--fix');
    
    if (filePaths.length === 0) {
        console.log('Usage: npx tsx postMigrationValidator.ts [--fix] <file1> <file2> ...');
        process.exit(1);
    }
    
    console.log('🔍 Post-Migration Validation');
    console.log('═'.repeat(40));
    
    Promise.all(filePaths.map(async (filePath) => {
        console.log(`\nValidating: ${path.basename(filePath)}`);
        
        if (autoFix) {
            const fixResult = await validator.fixCommonIssues(filePath);
            if (fixResult.fixed) {
                console.log('🔧 Auto-fixes applied:', fixResult.changes.join(', '));
            }
        }
        
        const result = await validator.validateFile(filePath);
        
        const status = result.syntaxValid && result.compileValid ? '✅' : '❌';
        console.log(`${status} ${result.fileName}`);
        
        if (result.errors.length > 0) {
            console.log('  ❌ Errors:');
            result.errors.forEach(error => console.log(`    • ${error}`));
        }
        
        if (result.warnings.length > 0) {
            console.log('  ⚠️  Warnings:');
            result.warnings.forEach(warning => console.log(`    • ${warning}`));
        }
        
        return result;
    }))
    .then(results => {
        const validCount = results.filter(r => r.syntaxValid && r.compileValid).length;
        console.log(`\n📊 Summary: ${validCount}/${results.length} files valid`);
        
        if (validCount < results.length) {
            process.exit(1);
        }
    })
    .catch(error => {
        console.error('❌ Validation failed:', error);
        process.exit(1);
    });
}