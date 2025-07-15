/**
 * Test Migration Analyzer
 * 
 * Analyzes test files to identify cleanup patterns and recommend migrations
 * to the new cleanup helper system.
 */

import fs from 'fs';
import path from 'path';

export interface CleanupPattern {
    type: 'manual' | 'factory' | 'mixed' | 'new' | 'none';
    tables: string[];
    hasCache: boolean;
    hasValidation: boolean;
    complexity: 'simple' | 'medium' | 'complex';
    issues: string[];
}

export interface TestFileAnalysis {
    filePath: string;
    fileName: string;
    lineCount: number;
    testCount: number;
    beforeEachPattern: CleanupPattern;
    afterEachPattern: CleanupPattern;
    factoryUsage: number;
    sessionMocks: number;
    migrationPriority: 'high' | 'medium' | 'low';
    recommendedCleanupGroup: string;
    migrationComplexity: 'simple' | 'medium' | 'complex';
    potentialIssues: string[];
}

export class TestMigrationAnalyzer {
    private readonly dependencyOrder = [
        // Level 1: Deep children
        'run_step', 'run_io', 'chat_message', 'chat_participants', 'chat_invite',
        'meeting_attendees', 'meeting_invite', 'member', 'member_invite',
        'session', 'user_auth', 'email', 'phone', 'push_device',
        'notification', 'notification_subscription',
        
        // Level 2: Intermediate
        'credit_ledger_entry', 'payment', 'comment', 'bookmark', 'reaction',
        'view', 'report', 'resource_version_relation', 'resource_version',
        'api_key', 'api_key_external', 'issue', 'run', 'schedule',
        
        // Level 3: Parents
        'meeting', 'chat', 'resource', 'team', 'user'
    ];

    /**
     * Analyze a single test file for migration opportunities
     */
    async analyzeFile(filePath: string): Promise<TestFileAnalysis> {
        const content = await fs.promises.readFile(filePath, 'utf-8');
        const fileName = path.basename(filePath);
        
        return {
            filePath,
            fileName,
            lineCount: content.split('\n').length,
            testCount: this.countTests(content),
            beforeEachPattern: this.analyzeBeforeEach(content),
            afterEachPattern: this.analyzeAfterEach(content),
            factoryUsage: this.countFactoryUsage(content),
            sessionMocks: this.countSessionMocks(content),
            migrationPriority: this.calculateMigrationPriority(content, fileName),
            recommendedCleanupGroup: this.recommendCleanupGroup(content, fileName),
            migrationComplexity: this.calculateMigrationComplexity(content),
            potentialIssues: this.identifyPotentialIssues(content),
        };
    }

    /**
     * Analyze multiple test files
     */
    async analyzeDirectory(dirPath: string): Promise<TestFileAnalysis[]> {
        const files = await this.findTestFiles(dirPath);
        const analyses: TestFileAnalysis[] = [];
        
        for (const file of files) {
            try {
                const analysis = await this.analyzeFile(file);
                analyses.push(analysis);
            } catch (error) {
                console.warn(`Failed to analyze ${file}:`, error);
            }
        }
        
        return analyses;
    }

    /**
     * Generate migration recommendations
     */
    generateMigrationPlan(analyses: TestFileAnalysis[]): {
        summary: {
            totalFiles: number;
            byPriority: Record<string, number>;
            byComplexity: Record<string, number>;
            byPattern: Record<string, number>;
        };
        phases: {
            phase: string;
            files: TestFileAnalysis[];
            description: string;
            expectedBenefits: string[];
        }[];
        recommendations: string[];
    } {
        const summary = {
            totalFiles: analyses.length,
            byPriority: this.groupByProperty(analyses, 'migrationPriority'),
            byComplexity: this.groupByProperty(analyses, 'migrationComplexity'),
            byPattern: this.groupByProperty(analyses, a => a.beforeEachPattern.type),
        };

        // Define phases step by step to avoid reference issues
        const highImpactFiles = analyses
            .filter(a => a.migrationPriority === 'high' && a.migrationComplexity !== 'complex')
            .slice(0, 5);
        const complexHighPriorityFiles = analyses
            .filter(a => a.migrationPriority === 'high' && a.migrationComplexity === 'complex');
        const mediumPriorityFiles = analyses
            .filter(a => a.migrationPriority === 'medium')
            .slice(0, 15);
        
        // Calculate remaining files for phase 4
        const assignedFiles = new Set([
            ...highImpactFiles.map(f => f.filePath),
            ...complexHighPriorityFiles.map(f => f.filePath),
            ...mediumPriorityFiles.map(f => f.filePath)
        ]);
        const remainingFiles = analyses.filter(a => !assignedFiles.has(a.filePath));

        const phases = [
            {
                phase: 'Phase 3.1 - High Impact Quick Wins',
                files: highImpactFiles,
                description: 'High-priority files with manageable migration complexity',
                expectedBenefits: [
                    'Immediate reliability improvements',
                    'Template for other migrations',
                    'Reduced maintenance burden on complex tests'
                ]
            },
            {
                phase: 'Phase 3.2 - Complex High-Priority Files',
                files: complexHighPriorityFiles,
                description: 'Complex files requiring careful migration',
                expectedBenefits: [
                    'Elimination of most critical cleanup issues',
                    'Improved reliability for core functionality tests'
                ]
            },
            {
                phase: 'Phase 3.3 - Medium Priority Standardization',
                files: mediumPriorityFiles,
                description: 'Standard endpoint tests with good standardization potential',
                expectedBenefits: [
                    'Pattern consistency across codebase',
                    'Reduced boilerplate code',
                    'Easier maintenance'
                ]
            },
            {
                phase: 'Phase 3.4 - Completion and Cleanup',
                files: remainingFiles,
                description: 'Remaining files for pattern completion',
                expectedBenefits: [
                    '100% pattern consistency',
                    'Future-proofing for new tests',
                    'Complete cleanup validation coverage'
                ]
            }
        ];

        const recommendations = this.generateRecommendations(analyses, summary);

        return { summary, phases, recommendations };
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

    private countTests(content: string): number {
        const testRegex = /^\s*(it|test)\s*\(/gm;
        return (content.match(testRegex) || []).length;
    }

    private analyzeBeforeEach(content: string): CleanupPattern {
        const beforeEachMatch = content.match(/beforeEach\s*\([^)]*\)\s*=>\s*\{([^}]+(?:\{[^}]*\}[^}]*)*)\}/);
        if (!beforeEachMatch) {
            return {
                type: 'none',
                tables: [],
                hasCache: false,
                hasValidation: false,
                complexity: 'simple',
                issues: ['No beforeEach cleanup found']
            };
        }

        const beforeEachContent = beforeEachMatch[1];
        
        // Check for new cleanup patterns
        if (beforeEachContent.includes('cleanupGroups.')) {
            return {
                type: 'new',
                tables: this.extractCleanupGroupTables(beforeEachContent),
                hasCache: beforeEachContent.includes('CacheService'),
                hasValidation: false,
                complexity: 'simple',
                issues: []
            };
        }

        // Check for factory cleanup
        const hasFactoryCleanup = beforeEachContent.includes('factory.cleanup') || 
                                beforeEachContent.includes('cleanupAll');

        // Extract manual deleteMany calls
        const deleteManyCalls = this.extractDeleteManyCalls(beforeEachContent);
        
        const type = hasFactoryCleanup ? 
            (deleteManyCalls.length > 0 ? 'mixed' : 'factory') : 
            (deleteManyCalls.length > 0 ? 'manual' : 'none');

        const issues = this.identifyCleanupIssues(deleteManyCalls, beforeEachContent);

        return {
            type,
            tables: deleteManyCalls,
            hasCache: beforeEachContent.includes('CacheService') || beforeEachContent.includes('flushAll'),
            hasValidation: false,
            complexity: this.calculateCleanupComplexity(deleteManyCalls, beforeEachContent),
            issues
        };
    }

    private analyzeAfterEach(content: string): CleanupPattern {
        const afterEachMatch = content.match(/afterEach\s*\([^)]*\)\s*=>\s*\{([^}]+(?:\{[^}]*\}[^}]*)*)\}/);
        if (!afterEachMatch) {
            return {
                type: 'none',
                tables: [],
                hasCache: false,
                hasValidation: false,
                complexity: 'simple',
                issues: []
            };
        }

        const afterEachContent = afterEachMatch[1];
        const hasValidation = afterEachContent.includes('validateCleanup') || 
                            afterEachContent.includes('validateStrictCleanup');

        return {
            type: hasValidation ? 'new' : 'manual',
            tables: [],
            hasCache: false,
            hasValidation,
            complexity: 'simple',
            issues: hasValidation ? [] : ['No cleanup validation']
        };
    }

    private extractDeleteManyCalls(content: string): string[] {
        const regex = /await\s+prisma\.(\w+)\.deleteMany\(\)/g;
        const matches = [];
        let match;
        
        while ((match = regex.exec(content)) !== null) {
            matches.push(match[1]);
        }
        
        return matches;
    }

    private extractCleanupGroupTables(content: string): string[] {
        const match = content.match(/cleanupGroups\.(\w+)/);
        if (!match) return [];
        
        // Map cleanup groups to their table coverage
        const groupTables: Record<string, string[]> = {
            minimal: ['user', 'user_auth', 'email', 'session'],
            userAuth: ['user', 'user_auth', 'email', 'phone', 'push_device', 'session', 'plan'],
            chat: ['chat', 'chat_message', 'chat_participants', 'chat_invite', 'user'],
            team: ['team', 'member', 'member_invite', 'meeting', 'user'],
            execution: ['run', 'run_step', 'run_io', 'resource', 'resource_version', 'user'],
            financial: ['payment', 'credit_account', 'credit_ledger_entry', 'plan', 'wallet', 'transfer'],
            content: ['comment', 'reaction', 'bookmark', 'view', 'report', 'issue', 'pull_request'],
            full: ['all_tables']
        };
        
        return groupTables[match[1]] || [];
    }

    private identifyCleanupIssues(tables: string[], content: string): string[] {
        const issues: string[] = [];
        
        // Check dependency order
        const orderIssues = this.checkDependencyOrder(tables);
        issues.push(...orderIssues);
        
        // Check for missing cache cleanup
        if (!content.includes('CacheService') && !content.includes('flushAll')) {
            issues.push('Missing cache cleanup');
        }
        
        // Check for over-cleaning (unnecessary tables)
        if (tables.length > 10) {
            issues.push('Possible over-cleaning - consider focused cleanup group');
        }
        
        // Check for under-cleaning (missing common related tables)
        if (tables.includes('user') && !tables.includes('user_auth') && !tables.includes('session')) {
            issues.push('Possible under-cleaning - missing auth tables');
        }
        
        return issues;
    }

    private checkDependencyOrder(tables: string[]): string[] {
        const issues: string[] = [];
        
        for (let i = 0; i < tables.length - 1; i++) {
            const currentTable = tables[i];
            const nextTable = tables[i + 1];
            
            const currentIndex = this.dependencyOrder.indexOf(currentTable);
            const nextIndex = this.dependencyOrder.indexOf(nextTable);
            
            // If both tables are in our dependency order and current comes after next
            if (currentIndex !== -1 && nextIndex !== -1 && currentIndex > nextIndex) {
                issues.push(`Dependency order issue: ${currentTable} should be cleaned before ${nextTable}`);
            }
        }
        
        return issues;
    }

    private calculateCleanupComplexity(tables: string[], content: string): 'simple' | 'medium' | 'complex' {
        if (tables.length === 0) return 'simple';
        if (tables.length <= 3) return 'simple';
        if (tables.length <= 7) return 'medium';
        return 'complex';
    }

    private countFactoryUsage(content: string): number {
        const factoryRegex = /\w+Factory\.|factory\./g;
        return (content.match(factoryRegex) || []).length;
    }

    private countSessionMocks(content: string): number {
        const sessionRegex = /mockAuthenticatedSession|mockLoggedOutSession|mockApiSession/g;
        return (content.match(sessionRegex) || []).length;
    }

    private calculateMigrationPriority(content: string, fileName: string): 'high' | 'medium' | 'low' {
        const lineCount = content.split('\n').length;
        const testCount = this.countTests(content);
        const factoryUsage = this.countFactoryUsage(content);
        
        // High priority: Large complex tests with lots of factory usage
        if (lineCount > 600 || testCount > 25 || factoryUsage > 30) {
            return 'high';
        }
        
        // High priority: Critical endpoint tests
        if (fileName.includes('auth.test') || fileName.includes('user.test') || 
            fileName.includes('team.test') || fileName.includes('run.test') ||
            fileName.includes('resource.test')) {
            return 'high';
        }
        
        // Medium priority: Standard endpoint tests
        if (fileName.includes('endpoints/logic/') && (lineCount > 200 || testCount > 10)) {
            return 'medium';
        }
        
        return 'low';
    }

    private recommendCleanupGroup(content: string, fileName: string): string {
        // Analyze table usage to recommend appropriate cleanup group
        const deleteManyCalls = this.extractDeleteManyCalls(content);
        
        if (deleteManyCalls.includes('user_auth') || deleteManyCalls.includes('session')) {
            return 'cleanupGroups.userAuth';
        }
        
        if (deleteManyCalls.includes('chat') || deleteManyCalls.includes('chat_message')) {
            return 'cleanupGroups.chat';
        }
        
        if (deleteManyCalls.includes('team') || deleteManyCalls.includes('member')) {
            return 'cleanupGroups.team';
        }
        
        if (deleteManyCalls.includes('run') || deleteManyCalls.includes('resource')) {
            return 'cleanupGroups.execution';
        }
        
        if (deleteManyCalls.includes('payment') || deleteManyCalls.includes('credit_account')) {
            return 'cleanupGroups.financial';
        }
        
        if (deleteManyCalls.includes('comment') || deleteManyCalls.includes('reaction')) {
            return 'cleanupGroups.content';
        }
        
        if (deleteManyCalls.length <= 3) {
            return 'cleanupGroups.minimal';
        }
        
        return 'cleanupGroups.full';
    }

    private calculateMigrationComplexity(content: string): 'simple' | 'medium' | 'complex' {
        const hasCustomCleanup = content.includes('cleanup') && !content.includes('deleteMany');
        const hasComplexMocking = content.includes('mockImplementation') || content.includes('spyOn');
        const hasTransactions = content.includes('$transaction');
        const lineCount = content.split('\n').length;
        
        if (hasTransactions || hasCustomCleanup || lineCount > 800) {
            return 'complex';
        }
        
        if (hasComplexMocking || lineCount > 400) {
            return 'medium';
        }
        
        return 'simple';
    }

    private identifyPotentialIssues(content: string): string[] {
        const issues: string[] = [];
        
        // Check for common anti-patterns
        if (content.includes('deleteMany') && !content.includes('CacheService')) {
            issues.push('Database cleanup without cache clearing');
        }
        
        if (content.includes('createUser') && !content.includes('user.deleteMany')) {
            issues.push('User creation without user cleanup');
        }
        
        if (content.includes('$transaction') && content.includes('deleteMany')) {
            issues.push('Manual cleanup in transaction-using test - consider transaction wrapper');
        }
        
        return issues;
    }

    private groupByProperty<T>(items: T[], property: keyof T | ((item: T) => string)): Record<string, number> {
        const groups: Record<string, number> = {};
        
        for (const item of items) {
            const key = typeof property === 'function' ? property(item) : String(item[property]);
            groups[key] = (groups[key] || 0) + 1;
        }
        
        return groups;
    }

    private generateRecommendations(analyses: TestFileAnalysis[], summary: any): string[] {
        const recommendations: string[] = [];
        
        // Priority recommendations
        if (summary.byPriority.high > 0) {
            recommendations.push(
                `Migrate ${summary.byPriority.high} high-priority files first for maximum impact`
            );
        }
        
        // Pattern-specific recommendations
        const manualCount = summary.byPattern.manual || 0;
        if (manualCount > 0) {
            recommendations.push(
                `${manualCount} files use manual deleteMany - convert to cleanupGroups for reliability`
            );
        }
        
        // Validation recommendations
        const filesWithoutValidation = analyses.filter(a => !a.afterEachPattern.hasValidation).length;
        if (filesWithoutValidation > 0) {
            recommendations.push(
                `Add cleanup validation to ${filesWithoutValidation} files to detect test pollution`
            );
        }
        
        // Factory recommendations
        const highFactoryUsage = analyses.filter(a => a.factoryUsage > 20).length;
        if (highFactoryUsage > 0) {
            recommendations.push(
                `${highFactoryUsage} files with heavy factory usage - consider FactoryComposition utilities`
            );
        }
        
        return recommendations;
    }
}