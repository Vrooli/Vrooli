/**
 * Test Pattern Linter
 * 
 * A custom linting tool that checks test files for compliance with
 * the standardized cleanup patterns. This provides immediate feedback
 * and can be run as part of CI/CD pipelines.
 */

import fs from 'fs';
import path from 'path';
import { TestMigrationAnalyzer, type TestFileAnalysis } from './testMigrationAnalyzer.js';

export interface LintResult {
    filePath: string;
    fileName: string;
    errors: LintError[];
    warnings: LintWarning[];
    score: number; // 0-100 compliance score
}

export interface LintError {
    type: 'manual-cleanup' | 'missing-import' | 'inconsistent-pattern' | 'forbidden-table';
    message: string;
    line?: number;
    severity: 'error' | 'warning';
    fixable: boolean;
    suggestion?: string;
}

export interface LintWarning {
    type: 'missing-validation' | 'complex-factories' | 'performance' | 'best-practice';
    message: string;
    line?: number;
    suggestion?: string;
}

export interface LintSummary {
    totalFiles: number;
    compliantFiles: number;
    filesWithErrors: number;
    filesWithWarnings: number;
    overallScore: number;
    topIssues: { type: string; count: number; message: string }[];
}

export class TestPatternLinter {
    private analyzer = new TestMigrationAnalyzer();

    async lintFile(filePath: string): Promise<LintResult> {
        const fileName = path.basename(filePath);
        const errors: LintError[] = [];
        const warnings: LintWarning[] = [];

        try {
            const content = await fs.promises.readFile(filePath, 'utf-8');
            const analysis = await this.analyzer.analyzeFile(filePath);

            // Check for manual cleanup patterns
            this.checkManualCleanup(content, analysis, errors);
            
            // Check for missing imports
            this.checkMissingImports(content, analysis, errors);
            
            // Check for inconsistent patterns
            this.checkInconsistentPatterns(content, analysis, errors);
            
            // Check for missing validation
            this.checkMissingValidation(content, analysis, warnings);
            
            // Check for complex factory usage
            this.checkComplexFactoryUsage(content, analysis, warnings);
            
            // Check for performance issues
            this.checkPerformanceIssues(content, analysis, warnings);

            // Calculate compliance score
            const score = this.calculateComplianceScore(analysis, errors, warnings);

            return {
                filePath,
                fileName,
                errors,
                warnings,
                score,
            };

        } catch (error) {
            errors.push({
                type: 'manual-cleanup',
                message: `Failed to analyze file: ${error instanceof Error ? error.message : String(error)}`,
                severity: 'error',
                fixable: false,
            });

            return {
                filePath,
                fileName,
                errors,
                warnings,
                score: 0,
            };
        }
    }

    async lintDirectory(dirPath: string): Promise<LintResult[]> {
        const testFiles = await this.findTestFiles(dirPath);
        const results: LintResult[] = [];

        for (const filePath of testFiles) {
            const result = await this.lintFile(filePath);
            results.push(result);
        }

        return results;
    }

    generateSummary(results: LintResult[]): LintSummary {
        const totalFiles = results.length;
        const compliantFiles = results.filter(r => r.errors.length === 0 && r.warnings.length === 0).length;
        const filesWithErrors = results.filter(r => r.errors.length > 0).length;
        const filesWithWarnings = results.filter(r => r.warnings.length > 0).length;
        const overallScore = totalFiles > 0 ? 
            Math.round(results.reduce((sum, r) => sum + r.score, 0) / totalFiles) : 100;

        // Count issue types
        const issueCount: Record<string, number> = {};
        results.forEach(result => {
            result.errors.forEach(error => {
                issueCount[error.type] = (issueCount[error.type] || 0) + 1;
            });
            result.warnings.forEach(warning => {
                issueCount[warning.type] = (issueCount[warning.type] || 0) + 1;
            });
        });

        // Generate top issues
        const topIssues = Object.entries(issueCount)
            .sort(([, a], [, b]) => b - a)
            .slice(0, 5)
            .map(([type, count]) => ({
                type,
                count,
                message: this.getIssueDescription(type),
            }));

        return {
            totalFiles,
            compliantFiles,
            filesWithErrors,
            filesWithWarnings,
            overallScore,
            topIssues,
        };
    }

    private checkManualCleanup(content: string, analysis: TestFileAnalysis, errors: LintError[]): void {
        if (analysis.beforeEachPattern.type === 'manual') {
            analysis.beforeEachPattern.tables.forEach(table => {
                // Check if this table is in forbidden list
                if (!['user', 'email', 'session'].includes(table)) {
                    errors.push({
                        type: 'manual-cleanup',
                        message: `Manual cleanup of '${table}' should use cleanupGroups instead`,
                        severity: 'error',
                        fixable: true,
                        suggestion: `Replace with: await ${analysis.recommendedCleanupGroup}(DbProvider.get())`,
                    });
                }
            });

            // Check for dependency order issues
            analysis.beforeEachPattern.issues.forEach(issue => {
                if (issue.includes('Dependency order')) {
                    errors.push({
                        type: 'manual-cleanup',
                        message: issue,
                        severity: 'error',
                        fixable: true,
                        suggestion: 'Use cleanupGroups for proper dependency ordering',
                    });
                }
            });
        }
    }

    private checkMissingImports(content: string, analysis: TestFileAnalysis, errors: LintError[]): void {
        // Check for cleanup groups import
        if (analysis.beforeEachPattern.type === 'manual' && !content.includes('testCleanupHelpers')) {
            errors.push({
                type: 'missing-import',
                message: 'Missing import for cleanupGroups helper',
                severity: 'error',
                fixable: true,
                suggestion: 'Add: import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";',
            });
        }

        // Check for validation import
        if (!analysis.afterEachPattern.hasValidation && !content.includes('testValidation')) {
            errors.push({
                type: 'missing-import',
                message: 'Missing import for validateCleanup helper',
                severity: 'warning',
                fixable: true,
                suggestion: 'Add: import { validateCleanup } from "../../__test/helpers/testValidation.js";',
            });
        }
    }

    private checkInconsistentPatterns(content: string, analysis: TestFileAnalysis, errors: LintError[]): void {
        if (analysis.beforeEachPattern.type === 'mixed') {
            errors.push({
                type: 'inconsistent-pattern',
                message: 'File mixes manual cleanup and factory patterns - use cleanupGroups consistently',
                severity: 'error',
                fixable: true,
                suggestion: 'Migrate all cleanup to use cleanupGroups',
            });
        }
    }

    private checkMissingValidation(content: string, analysis: TestFileAnalysis, warnings: LintWarning[]): void {
        if (!analysis.afterEachPattern.hasValidation && analysis.testCount > 5) {
            warnings.push({
                type: 'missing-validation',
                message: 'Tests should include cleanup validation to detect test pollution',
                suggestion: 'Add afterEach with validateCleanup call',
            });
        }
    }

    private checkComplexFactoryUsage(content: string, analysis: TestFileAnalysis, warnings: LintWarning[]): void {
        if (analysis.factoryUsage > 15) {
            warnings.push({
                type: 'complex-factories',
                message: `High factory usage (${analysis.factoryUsage}) - consider FactoryComposition`,
                suggestion: 'Use FactoryComposition for coordinated factory operations',
            });
        }
    }

    private checkPerformanceIssues(content: string, analysis: TestFileAnalysis, warnings: LintWarning[]): void {
        // Check for potential over-cleaning
        if (analysis.beforeEachPattern.tables.length > 8) {
            warnings.push({
                type: 'performance',
                message: `Cleaning ${analysis.beforeEachPattern.tables.length} tables - consider focused cleanup scope`,
                suggestion: `Use ${analysis.recommendedCleanupGroup} for better performance`,
            });
        }

        // Check for missing cache cleanup
        if (analysis.beforeEachPattern.tables.length > 0 && !analysis.beforeEachPattern.hasCache) {
            warnings.push({
                type: 'performance',
                message: 'Database cleanup without cache clearing may cause test pollution',
                suggestion: 'Ensure cache is cleared with database cleanup',
            });
        }
    }

    private calculateComplianceScore(
        analysis: TestFileAnalysis, 
        errors: LintError[], 
        warnings: LintWarning[]
    ): number {
        let score = 100;

        // Deduct points for errors
        score -= errors.length * 20;

        // Deduct points for warnings
        score -= warnings.length * 5;

        // Bonus points for good patterns
        if (analysis.beforeEachPattern.type === 'new') score += 10;
        if (analysis.afterEachPattern.hasValidation) score += 10;
        if (analysis.migrationComplexity === 'simple') score += 5;

        return Math.max(0, Math.min(100, score));
    }

    private async findTestFiles(dirPath: string): Promise<string[]> {
        const files: string[] = [];
        
        async function traverse(currentPath: string) {
            const entries = await fs.promises.readdir(currentPath, { withFileTypes: true });
            
            for (const entry of entries) {
                const fullPath = path.join(currentPath, entry.name);
                
                if (entry.isDirectory() && !entry.name.startsWith('.') && entry.name !== 'node_modules') {
                    await traverse(fullPath);
                } else if (entry.isFile() && entry.name.endsWith('.test.ts')) {
                    files.push(fullPath);
                }
            }
        }
        
        await traverse(dirPath);
        return files;
    }

    private getIssueDescription(type: string): string {
        const descriptions: Record<string, string> = {
            'manual-cleanup': 'Manual deleteMany calls should use cleanupGroups',
            'missing-import': 'Required helper imports are missing',
            'inconsistent-pattern': 'Mixing cleanup patterns reduces reliability',
            'forbidden-table': 'Direct table cleanup violates best practices',
            'missing-validation': 'Missing cleanup validation can hide test pollution',
            'complex-factories': 'Complex factory usage benefits from composition patterns',
            'performance': 'Cleanup operations could be optimized',
            'best-practice': 'Code could follow established best practices',
        };
        return descriptions[type] || 'Unknown issue type';
    }
}

// CLI interface
if (import.meta.url === `file://${process.argv[1]}`) {
    const linter = new TestPatternLinter();
    const args = process.argv.slice(2);
    
    const directory = args[0] || '/root/Vrooli/packages/server/src/endpoints/logic';
    const outputFormat = args.includes('--json') ? 'json' : 'console';
    const showWarnings = !args.includes('--errors-only');
    
    console.log('🔍 Linting test patterns...\n');
    
    linter.lintDirectory(directory)
        .then(results => {
            const summary = linter.generateSummary(results);
            
            if (outputFormat === 'json') {
                console.log(JSON.stringify({ summary, results }, null, 2));
                return;
            }
            
            // Console output
            console.log('📊 Test Pattern Lint Summary');
            console.log('═'.repeat(50));
            console.log(`Total files: ${summary.totalFiles}`);
            console.log(`Compliant files: ${summary.compliantFiles} (${Math.round(summary.compliantFiles / summary.totalFiles * 100)}%)`);
            console.log(`Files with errors: ${summary.filesWithErrors}`);
            console.log(`Files with warnings: ${summary.filesWithWarnings}`);
            console.log(`Overall compliance score: ${summary.overallScore}/100`);
            console.log('');
            
            if (summary.topIssues.length > 0) {
                console.log('🔴 Top Issues:');
                summary.topIssues.forEach((issue, i) => {
                    console.log(`  ${i + 1}. ${issue.message} (${issue.count} files)`);
                });
                console.log('');
            }
            
            // Show detailed results for files with issues
            const problemFiles = results.filter(r => r.errors.length > 0 || (showWarnings && r.warnings.length > 0));
            
            if (problemFiles.length > 0) {
                console.log('📋 Detailed Results:');
                console.log('─'.repeat(50));
                
                problemFiles.forEach(result => {
                    console.log(`\n📁 ${result.fileName} (Score: ${result.score}/100)`);
                    
                    if (result.errors.length > 0) {
                        console.log('  ❌ Errors:');
                        result.errors.forEach(error => {
                            console.log(`    • ${error.message}`);
                            if (error.suggestion) {
                                console.log(`      💡 ${error.suggestion}`);
                            }
                        });
                    }
                    
                    if (showWarnings && result.warnings.length > 0) {
                        console.log('  ⚠️  Warnings:');
                        result.warnings.forEach(warning => {
                            console.log(`    • ${warning.message}`);
                            if (warning.suggestion) {
                                console.log(`      💡 ${warning.suggestion}`);
                            }
                        });
                    }
                });
            } else {
                console.log('✅ All files are compliant with test patterns!');
            }
            
            // Exit with appropriate code
            if (summary.filesWithErrors > 0) {
                process.exit(1);
            }
        })
        .catch(error => {
            console.error('❌ Linting failed:', error);
            process.exit(1);
        });
}