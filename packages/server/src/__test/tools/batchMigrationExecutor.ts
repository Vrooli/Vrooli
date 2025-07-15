/**
 * Batch Migration Executor
 * 
 * Executes systematic migration of test files in prioritized batches,
 * with validation, rollback capabilities, and progress tracking.
 */

import fs from 'fs';
import path from 'path';
import { TestMigrationAnalyzer, type TestFileAnalysis } from './testMigrationAnalyzer.js';
import { TestMigrationExecutor, type MigrationResult } from './testMigrationExecutor.js';
import { TestPatternLinter } from './lintTestPatterns.js';

export interface BatchMigrationConfig {
    dryRun?: boolean;
    backup?: boolean;
    validateAfter?: boolean;
    stopOnError?: boolean;
    maxBatchSize?: number;
    batchDelay?: number; // ms between batches
}

export interface BatchResult {
    batchName: string;
    files: string[];
    results: MigrationResult[];
    preScore: number;
    postScore: number;
    improvement: number;
    success: boolean;
    errors: string[];
    warnings: string[];
}

export interface MigrationSession {
    sessionId: string;
    startTime: Date;
    endTime?: Date;
    config: BatchMigrationConfig;
    batches: BatchResult[];
    overallSuccess: boolean;
    totalFiles: number;
    successfulFiles: number;
    failedFiles: number;
    totalImprovement: number;
}

export class BatchMigrationExecutor {
    private analyzer = new TestMigrationAnalyzer();
    private executor = new TestMigrationExecutor();
    private linter = new TestPatternLinter();

    async executeBatchMigration(
        directory: string, 
        config: BatchMigrationConfig = {}
    ): Promise<MigrationSession> {
        const sessionId = this.generateSessionId();
        const session: MigrationSession = {
            sessionId,
            startTime: new Date(),
            config: {
                dryRun: false,
                backup: true,
                validateAfter: true,
                stopOnError: false,
                maxBatchSize: 5,
                batchDelay: 1000,
                ...config,
            },
            batches: [],
            overallSuccess: false,
            totalFiles: 0,
            successfulFiles: 0,
            failedFiles: 0,
            totalImprovement: 0,
        };

        console.log(`🚀 Starting batch migration session: ${sessionId}`);
        console.log(`Directory: ${directory}`);
        console.log(`Configuration:`, session.config);
        console.log('');

        try {
            // Step 1: Analyze files and create migration plan
            console.log('📊 Analyzing files and creating migration plan...');
            const analyses = await this.analyzer.analyzeDirectory(directory);
            const plan = this.analyzer.generateMigrationPlan(analyses);
            
            console.log(`Found ${analyses.length} test files`);
            console.log(`High priority: ${plan.summary.byPriority.high || 0}`);
            console.log(`Medium priority: ${plan.summary.byPriority.medium || 0}`);
            console.log(`Low priority: ${plan.summary.byPriority.low || 0}`);
            console.log('');

            // Step 2: Execute migration phases in order
            for (const phase of plan.phases) {
                if (phase.files.length === 0) {
                    console.log(`⏭️  Skipping ${phase.phase} - no files to migrate`);
                    continue;
                }

                console.log(`🔄 Executing ${phase.phase}`);
                console.log(`Files: ${phase.files.length}`);
                console.log(`Description: ${phase.description}`);
                console.log('');

                // Split phase into batches
                const batches = this.splitIntoBatches(phase.files, session.config.maxBatchSize!);
                
                for (let i = 0; i < batches.length; i++) {
                    const batchName = `${phase.phase} - Batch ${i + 1}`;
                    const batchFiles = batches[i];
                    
                    console.log(`📦 ${batchName} (${batchFiles.length} files)`);
                    
                    const batchResult = await this.executeBatch(
                        batchName,
                        batchFiles,
                        session.config
                    );
                    
                    session.batches.push(batchResult);
                    session.totalFiles += batchResult.files.length;
                    session.successfulFiles += batchResult.results.filter(r => r.success).length;
                    session.failedFiles += batchResult.results.filter(r => !r.success).length;
                    session.totalImprovement += batchResult.improvement;
                    
                    // Display batch results
                    this.displayBatchResults(batchResult);
                    
                    // Stop on error if configured
                    if (!batchResult.success && session.config.stopOnError) {
                        console.log('❌ Stopping migration due to batch failure');
                        session.overallSuccess = false;
                        session.endTime = new Date();
                        return session;
                    }
                    
                    // Delay between batches
                    if (i < batches.length - 1 && session.config.batchDelay! > 0) {
                        console.log(`⏳ Waiting ${session.config.batchDelay}ms before next batch...`);
                        await new Promise(resolve => setTimeout(resolve, session.config.batchDelay));
                    }
                }
                
                console.log(`✅ Completed ${phase.phase}`);
                console.log('');
            }

            session.overallSuccess = session.failedFiles === 0;
            session.endTime = new Date();

            // Step 3: Generate final report
            this.displaySessionSummary(session);

            return session;

        } catch (error) {
            console.error('❌ Migration session failed:', error);
            session.overallSuccess = false;
            session.endTime = new Date();
            return session;
        }
    }

    private async executeBatch(
        batchName: string,
        files: TestFileAnalysis[],
        config: BatchMigrationConfig
    ): Promise<BatchResult> {
        const filePaths = files.map(f => f.filePath);
        
        // Get pre-migration scores
        const preScores = await Promise.all(
            filePaths.map(async (filePath) => {
                const lintResult = await this.linter.lintFile(filePath);
                return lintResult.score;
            })
        );
        const preScore = Math.round(preScores.reduce((sum, score) => sum + score, 0) / preScores.length);

        // Execute migrations
        const results = await this.executor.migrateFiles(filePaths, {
            dryRun: config.dryRun,
            backup: config.backup,
            validateAfter: config.validateAfter,
        });

        // Get post-migration scores (skip if dry run)
        let postScore = preScore;
        if (!config.dryRun) {
            const postScores = await Promise.all(
                filePaths.map(async (filePath) => {
                    const lintResult = await this.linter.lintFile(filePath);
                    return lintResult.score;
                })
            );
            postScore = Math.round(postScores.reduce((sum, score) => sum + score, 0) / postScores.length);
        }

        const improvement = postScore - preScore;
        const success = results.every(r => r.success);

        // Collect errors and warnings
        const errors: string[] = [];
        const warnings: string[] = [];
        
        results.forEach(result => {
            errors.push(...result.errors);
            warnings.push(...result.warnings);
        });

        return {
            batchName,
            files: filePaths,
            results,
            preScore,
            postScore,
            improvement,
            success,
            errors,
            warnings,
        };
    }

    private splitIntoBatches<T>(items: T[], batchSize: number): T[][] {
        const batches: T[][] = [];
        for (let i = 0; i < items.length; i += batchSize) {
            batches.push(items.slice(i, i + batchSize));
        }
        return batches;
    }

    private displayBatchResults(batch: BatchResult): void {
        const status = batch.success ? '✅' : '❌';
        const successCount = batch.results.filter(r => r.success).length;
        
        console.log(`${status} Batch completed: ${successCount}/${batch.results.length} files migrated`);
        console.log(`📈 Score improvement: ${batch.preScore} → ${batch.postScore} (+${batch.improvement})`);
        
        if (batch.errors.length > 0) {
            console.log(`❌ Errors: ${batch.errors.length}`);
            batch.errors.slice(0, 3).forEach(error => {
                console.log(`   • ${error}`);
            });
            if (batch.errors.length > 3) {
                console.log(`   • ... and ${batch.errors.length - 3} more`);
            }
        }
        
        if (batch.warnings.length > 0) {
            console.log(`⚠️  Warnings: ${batch.warnings.length}`);
        }
        
        console.log('');
    }

    private displaySessionSummary(session: MigrationSession): void {
        const duration = session.endTime!.getTime() - session.startTime.getTime();
        const durationSeconds = Math.round(duration / 1000);
        const successRate = Math.round((session.successfulFiles / session.totalFiles) * 100);

        console.log('🎯 Migration Session Summary');
        console.log('═'.repeat(50));
        console.log(`Session ID: ${session.sessionId}`);
        console.log(`Duration: ${durationSeconds}s`);
        console.log(`Overall Success: ${session.overallSuccess ? '✅' : '❌'}`);
        console.log('');
        
        console.log('📊 File Statistics:');
        console.log(`Total files: ${session.totalFiles}`);
        console.log(`Successful: ${session.successfulFiles} (${successRate}%)`);
        console.log(`Failed: ${session.failedFiles}`);
        console.log(`Total improvement: +${session.totalImprovement} points`);
        console.log('');
        
        console.log('📦 Batch Results:');
        session.batches.forEach((batch, i) => {
            const status = batch.success ? '✅' : '❌';
            console.log(`${i + 1}. ${status} ${batch.batchName}: ${batch.files.length} files, +${batch.improvement} points`);
        });
        console.log('');
        
        if (session.config.dryRun) {
            console.log('🔍 This was a dry run - no files were actually modified');
            console.log('Run without --dry-run to apply changes');
        } else {
            console.log('✨ Migration completed! Files have been updated.');
            console.log('🔍 Verify results with: ./scripts/lint-test-patterns.sh');
        }
        
        if (session.config.backup && !session.config.dryRun) {
            console.log('💾 Backups created with .backup extension');
        }
    }

    private generateSessionId(): string {
        const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, -5);
        const random = Math.random().toString(36).substring(2, 8);
        return `migration-${timestamp}-${random}`;
    }

    async migrateHighPriorityFiles(directory: string, config: BatchMigrationConfig = {}): Promise<MigrationSession> {
        console.log('🎯 High-Priority File Migration');
        console.log('══════════════════════════════════');
        console.log('Focusing on files with the highest impact potential');
        console.log('');

        // Override config for high-priority migration
        const highPriorityConfig: BatchMigrationConfig = {
            ...config,
            maxBatchSize: 3, // Smaller batches for careful migration
            validateAfter: true, // Always validate high-priority files
            stopOnError: true, // Stop if any high-priority file fails
        };

        return this.executeBatchMigration(directory, highPriorityConfig);
    }
}

// CLI interface
if (import.meta.url === `file://${process.argv[1]}`) {
    const executor = new BatchMigrationExecutor();
    const args = process.argv.slice(2);
    
    const config: BatchMigrationConfig = {
        dryRun: args.includes('--dry-run'),
        backup: !args.includes('--no-backup'),
        validateAfter: !args.includes('--no-validate'),
        stopOnError: args.includes('--stop-on-error'),
        maxBatchSize: 5,
        batchDelay: 1000,
    };
    
    const directoryArg = args.find(arg => !arg.startsWith('--'));
    const directory = directoryArg || '/root/Vrooli/packages/server/src/endpoints/logic';
    
    const highPriorityOnly = args.includes('--high-priority');
    
    console.log('🚀 Batch Migration Executor');
    console.log(`Directory: ${directory}`);
    console.log(`High priority only: ${highPriorityOnly}`);
    console.log(`Configuration:`, config);
    console.log('');
    
    const migrationPromise = highPriorityOnly ? 
        executor.migrateHighPriorityFiles(directory, config) :
        executor.executeBatchMigration(directory, config);
    
    migrationPromise
        .then(session => {
            if (session.overallSuccess) {
                console.log('🎉 Migration session completed successfully!');
                process.exit(0);
            } else {
                console.log('⚠️  Migration session completed with issues');
                process.exit(1);
            }
        })
        .catch(error => {
            console.error('💥 Migration session failed:', error);
            process.exit(1);
        });
}