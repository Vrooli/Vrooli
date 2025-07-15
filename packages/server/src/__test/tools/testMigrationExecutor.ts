/**
 * Test Migration Executor
 * 
 * Performs automated migration of test files to use the new cleanup helper system.
 * Works with the TestMigrationAnalyzer to apply recommended changes.
 */

import fs from 'fs';
import path from 'path';
import { TestMigrationAnalyzer, type TestFileAnalysis } from './testMigrationAnalyzer.js';

export interface MigrationResult {
    filePath: string;
    success: boolean;
    changes: string[];
    warnings: string[];
    errors: string[];
}

export interface MigrationOptions {
    dryRun?: boolean;
    backup?: boolean;
    validateAfter?: boolean;
    force?: boolean; // Skip safety checks
}

export class TestMigrationExecutor {
    private analyzer = new TestMigrationAnalyzer();

    /**
     * Migrate a single test file
     */
    async migrateFile(filePath: string, options: MigrationOptions = {}): Promise<MigrationResult> {
        const result: MigrationResult = {
            filePath,
            success: false,
            changes: [],
            warnings: [],
            errors: [],
        };

        try {
            // Analyze the file first
            const analysis = await this.analyzer.analyzeFile(filePath);
            
            // Check if migration is needed
            if (analysis.beforeEachPattern.type === 'new' && analysis.afterEachPattern.hasValidation) {
                result.warnings.push('File already uses new cleanup patterns');
                result.success = true;
                return result;
            }

            // Read current file content
            const originalContent = await fs.promises.readFile(filePath, 'utf-8');
            
            // Create backup if requested
            if (options.backup) {
                const backupPath = `${filePath}.backup`;
                await fs.promises.writeFile(backupPath, originalContent);
                result.changes.push(`Created backup: ${backupPath}`);
            }

            // Apply transformations
            let newContent = originalContent;
            
            // 1. Add required imports
            newContent = this.addRequiredImports(newContent, analysis);
            if (newContent !== originalContent) {
                result.changes.push('Added cleanup helper imports');
            }

            // 2. Transform beforeEach
            const beforeEachResult = this.transformBeforeEach(newContent, analysis);
            newContent = beforeEachResult.content;
            result.changes.push(...beforeEachResult.changes);
            result.warnings.push(...beforeEachResult.warnings);

            // 3. Transform afterEach 
            const afterEachResult = this.transformAfterEach(newContent, analysis);
            newContent = afterEachResult.content;
            result.changes.push(...afterEachResult.changes);

            // 4. Remove redundant imports
            newContent = this.removeRedundantImports(newContent);

            // Write the migrated file (unless dry run)
            if (!options.dryRun) {
                await fs.promises.writeFile(filePath, newContent);
                result.changes.push('File successfully migrated');
            } else {
                result.changes.push('Dry run - no changes written');
            }

            // Validate after migration
            if (options.validateAfter && !options.dryRun) {
                const validationResult = await this.validateMigration(filePath);
                if (!validationResult.valid) {
                    result.warnings.push(...validationResult.issues);
                }
            }

            result.success = true;

        } catch (error) {
            result.errors.push(`Migration failed: ${error instanceof Error ? error.message : String(error)}`);
        }

        return result;
    }

    /**
     * Migrate multiple files in batch
     */
    async migrateFiles(filePaths: string[], options: MigrationOptions = {}): Promise<MigrationResult[]> {
        const results: MigrationResult[] = [];
        
        for (const filePath of filePaths) {
            const result = await this.migrateFile(filePath, options);
            results.push(result);
            
            // Log progress
            const status = result.success ? '✓' : '✗';
            const fileName = path.basename(filePath);
            console.log(`${status} ${fileName} - ${result.changes.length} changes, ${result.warnings.length} warnings, ${result.errors.length} errors`);
        }
        
        return results;
    }

    /**
     * Execute migration plan from analyzer
     */
    async executeMigrationPlan(directoryPath: string, options: MigrationOptions = {}) {
        console.log('Analyzing test files for migration opportunities...');
        
        const analyses = await this.analyzer.analyzeDirectory(directoryPath);
        const plan = this.analyzer.generateMigrationPlan(analyses);
        
        console.log('\n=== Migration Plan Summary ===');
        console.log(`Total files: ${plan.summary.totalFiles}`);
        console.log(`High priority: ${plan.summary.byPriority.high || 0}`);
        console.log(`Medium priority: ${plan.summary.byPriority.medium || 0}`);
        console.log(`Low priority: ${plan.summary.byPriority.low || 0}`);
        
        console.log('\n=== Recommendations ===');
        plan.recommendations.forEach(rec => console.log(`• ${rec}`));
        
        // Execute each phase
        for (const phase of plan.phases) {
            if (phase.files.length === 0) continue;
            
            console.log(`\n=== ${phase.phase} ===`);
            console.log(phase.description);
            console.log(`Files to migrate: ${phase.files.length}`);
            
            if (!options.force && !options.dryRun) {
                // In a real implementation, you'd want user confirmation here
                console.log('Use --force flag to execute automatically');
                continue;
            }
            
            const filePaths = phase.files.map(f => f.filePath);
            const results = await this.migrateFiles(filePaths, options);
            
            // Summary for this phase
            const successful = results.filter(r => r.success).length;
            const failed = results.length - successful;
            
            console.log(`Phase completed: ${successful} successful, ${failed} failed`);
            
            if (failed > 0) {
                console.log('Failed files:');
                results.filter(r => !r.success).forEach(r => {
                    console.log(`  ${path.basename(r.filePath)}: ${r.errors.join(', ')}`);
                });
            }
        }
    }

    private addRequiredImports(content: string, analysis: TestFileAnalysis): string {
        const imports = [];
        
        // Check if cleanup helpers import is needed
        if (!content.includes('testCleanupHelpers')) {
            imports.push('import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";');
        }
        
        // Check if validation import is needed
        if (!content.includes('testValidation') && 
            analysis.migrationComplexity !== 'simple') {
            imports.push('import { validateCleanup } from "../../__test/helpers/testValidation.js";');
        }

        if (imports.length === 0) return content;

        // Find existing imports and add after them
        const lines = content.split('\n');
        let lastImportIndex = -1;
        
        for (let i = 0; i < lines.length; i++) {
            if (lines[i].trim().startsWith('import ') || lines[i].trim().startsWith('} from ')) {
                lastImportIndex = i;
            }
        }
        
        if (lastImportIndex >= 0) {
            lines.splice(lastImportIndex + 1, 0, ...imports);
        } else {
            // No imports found, add at the top
            lines.unshift(...imports, '');
        }
        
        return lines.join('\n');
    }

    private transformBeforeEach(content: string, analysis: TestFileAnalysis): {
        content: string;
        changes: string[];
        warnings: string[];
    } {
        const changes: string[] = [];
        const warnings: string[] = [];
        
        // Find beforeEach block
        const beforeEachRegex = /beforeEach\s*\([^)]*\)\s*=>\s*\{([^}]+(?:\{[^}]*\}[^}]*)*)\}/;
        const match = content.match(beforeEachRegex);
        
        if (!match) {
            // No beforeEach found, add one
            const newBeforeEach = this.generateBeforeEach(analysis.recommendedCleanupGroup);
            
            // Find describe block to insert beforeEach
            const describeMatch = content.match(/describe\s*\([^)]+\)\s*=>\s*\{/);
            if (describeMatch) {
                const insertIndex = content.indexOf(describeMatch[0]) + describeMatch[0].length;
                const newContent = 
                    content.slice(0, insertIndex) + 
                    '\n    ' + newBeforeEach + '\n' +
                    content.slice(insertIndex);
                changes.push('Added new beforeEach with cleanup');
                return { content: newContent, changes, warnings };
            } else {
                warnings.push('Could not find describe block to insert beforeEach');
                return { content, changes, warnings };
            }
        }

        // Transform existing beforeEach
        const beforeEachContent = match[1];
        
        // Check if already using new patterns
        if (beforeEachContent.includes('cleanupGroups.')) {
            warnings.push('BeforeEach already uses cleanupGroups');
            return { content, changes, warnings };
        }

        // Replace manual deleteMany calls with cleanup group
        const newBeforeEachContent = this.generateBeforeEachContent(analysis.recommendedCleanupGroup);
        const newContent = content.replace(beforeEachRegex, `beforeEach(async () => {${newBeforeEachContent}\n    });`);
        
        changes.push('Replaced manual cleanup with cleanupGroups');
        if (beforeEachContent.includes('deleteMany')) {
            changes.push('Removed manual deleteMany calls');
        }
        
        return { content: newContent, changes, warnings };
    }

    private transformAfterEach(content: string, analysis: TestFileAnalysis): {
        content: string;
        changes: string[];
    } {
        const changes: string[] = [];
        
        // Check if afterEach already exists
        if (content.includes('afterEach')) {
            // Already has afterEach, check if it has validation
            if (!content.includes('validateCleanup')) {
                // Add validation to existing afterEach
                const afterEachRegex = /afterEach\s*\([^)]*\)\s*=>\s*\{([^}]+(?:\{[^}]*\}[^}]*)*)\}/;
                const match = content.match(afterEachRegex);
                
                if (match) {
                    const validationCode = this.generateValidationCode(analysis);
                    const newAfterEachContent = match[1] + '\n' + validationCode;
                    const newContent = content.replace(afterEachRegex, 
                        `afterEach(async () => {${newAfterEachContent}\n    });`
                    );
                    changes.push('Added cleanup validation to existing afterEach');
                    return { content: newContent, changes };
                }
            }
        } else {
            // No afterEach found, add one with validation
            const newAfterEach = this.generateAfterEach(analysis);
            
            // Find where to insert afterEach (after beforeEach if it exists)
            let insertPoint = content.indexOf('beforeEach');
            if (insertPoint >= 0) {
                // Find the end of beforeEach block
                const afterBeforeEach = content.indexOf('});', insertPoint) + 3;
                const newContent = 
                    content.slice(0, afterBeforeEach) + 
                    '\n\n    ' + newAfterEach + 
                    content.slice(afterBeforeEach);
                changes.push('Added afterEach with cleanup validation');
                return { content: newContent, changes };
            } else {
                // Insert after describe opening
                const describeMatch = content.match(/describe\s*\([^)]+\)\s*=>\s*\{/);
                if (describeMatch) {
                    const insertIndex = content.indexOf(describeMatch[0]) + describeMatch[0].length;
                    const newContent = 
                        content.slice(0, insertIndex) + 
                        '\n    ' + newAfterEach + '\n' +
                        content.slice(insertIndex);
                    changes.push('Added afterEach with cleanup validation');
                    return { content: newContent, changes };
                }
            }
        }
        
        return { content, changes };
    }

    private generateBeforeEach(cleanupGroup: string): string {
        return `beforeEach(async () => {${this.generateBeforeEachContent(cleanupGroup)}\n    });`;
    }

    private generateBeforeEachContent(cleanupGroup: string): string {
        const group = cleanupGroup.replace('cleanupGroups.', '');
        return `
        // Clean up using dependency-ordered cleanup helpers
        await ${cleanupGroup}(DbProvider.get());`;
    }

    private generateAfterEach(analysis: TestFileAnalysis): string {
        return `afterEach(async () => {${this.generateValidationCode(analysis)}\n    });`;
    }

    private generateValidationCode(analysis: TestFileAnalysis): string {
        // Determine tables to validate based on the analysis
        const tables = this.getValidationTables(analysis);
        
        return `
        // Validate cleanup to detect any missed records
        const orphans = await validateCleanup(DbProvider.get(), {
            tables: ${JSON.stringify(tables)},
            logOrphans: true,
        });
        if (orphans.length > 0) {
            console.warn('Test cleanup incomplete:', orphans);
        }`;
    }

    private getValidationTables(analysis: TestFileAnalysis): string[] {
        // Map cleanup groups to validation tables
        const groupTableMap: Record<string, string[]> = {
            'cleanupGroups.minimal': ['user', 'user_auth', 'email', 'session'],
            'cleanupGroups.userAuth': ['user', 'user_auth', 'email', 'phone', 'push_device', 'session'],
            'cleanupGroups.chat': ['chat', 'chat_message', 'chat_participants', 'chat_invite', 'user'],
            'cleanupGroups.team': ['team', 'member', 'member_invite', 'meeting', 'user'],
            'cleanupGroups.execution': ['run', 'run_step', 'run_io', 'resource', 'resource_version', 'user'],
            'cleanupGroups.financial': ['payment', 'credit_account', 'credit_ledger_entry', 'wallet', 'transfer'],
            'cleanupGroups.content': ['comment', 'reaction', 'bookmark', 'view', 'report', 'issue'],
            'cleanupGroups.full': ['user', 'team', 'chat', 'run', 'resource', 'comment', 'payment'],
        };
        
        return groupTableMap[analysis.recommendedCleanupGroup] || ['user'];
    }

    private removeRedundantImports(content: string): string {
        // Remove imports that are no longer needed after migration
        const redundantPatterns = [
            // Remove individual table delete imports if they exist
            /import\s+\{[^}]*deleteMany[^}]*\}\s+from[^;]+;?\n?/g,
        ];
        
        let newContent = content;
        redundantPatterns.forEach(pattern => {
            newContent = newContent.replace(pattern, '');
        });
        
        return newContent;
    }

    private async validateMigration(filePath: string): Promise<{ valid: boolean; issues: string[] }> {
        const issues: string[] = [];
        
        try {
            const content = await fs.promises.readFile(filePath, 'utf-8');
            
            // Check that new patterns are in place
            if (!content.includes('cleanupGroups.')) {
                issues.push('File does not use cleanupGroups');
            }
            
            // Check for remaining manual deleteMany calls in beforeEach
            const beforeEachMatch = content.match(/beforeEach\s*\([^)]*\)\s*=>\s*\{([^}]+(?:\{[^}]*\}[^}]*)*)\}/);
            if (beforeEachMatch && beforeEachMatch[1].includes('deleteMany')) {
                issues.push('Manual deleteMany calls still present in beforeEach');
            }
            
            // Check for syntax errors (basic check)
            if (content.includes('await cleanupGroups.') && !content.includes('import')) {
                issues.push('Missing import for cleanupGroups');
            }
            
        } catch (error) {
            issues.push(`Validation failed: ${error instanceof Error ? error.message : String(error)}`);
        }
        
        return { valid: issues.length === 0, issues };
    }
}

// CLI interface for running migrations
if (import.meta.url === `file://${process.argv[1]}`) {
    const executor = new TestMigrationExecutor();
    const args = process.argv.slice(2);
    
    const options: MigrationOptions = {
        dryRun: args.includes('--dry-run'),
        backup: args.includes('--backup'),
        validateAfter: args.includes('--validate'),
        force: args.includes('--force'),
    };
    
    const directoryArg = args.find(arg => !arg.startsWith('--'));
    const directory = directoryArg || '/root/Vrooli/packages/server/src/__test';
    
    console.log('Starting test migration...');
    console.log(`Directory: ${directory}`);
    console.log(`Options:`, options);
    
    executor.executeMigrationPlan(directory, options)
        .then(() => console.log('Migration completed'))
        .catch(error => {
            console.error('Migration failed:', error);
            process.exit(1);
        });
}