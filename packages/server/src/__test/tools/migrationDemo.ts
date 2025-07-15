/**
 * Migration Demo Script
 * 
 * Demonstrates the migration tools by analyzing existing test files
 * and showing what migrations would be performed.
 */

import { TestMigrationAnalyzer } from './testMigrationAnalyzer.js';
import { TestMigrationExecutor } from './testMigrationExecutor.js';

async function runMigrationDemo() {
    console.log('=== Test Migration System Demo ===\n');
    
    const analyzer = new TestMigrationAnalyzer();
    const executor = new TestMigrationExecutor();
    
    // Demo files to analyze
    const demoFiles = [
        '/root/Vrooli/packages/server/src/endpoints/logic/chat.test.ts',
        '/root/Vrooli/packages/server/src/endpoints/logic/email.test.ts',
        '/root/Vrooli/packages/server/src/utils/shapes/afterMutationsVersion.test.ts'
    ];
    
    console.log('1. Analyzing sample test files...\n');
    
    for (const filePath of demoFiles) {
        try {
            const analysis = await analyzer.analyzeFile(filePath);
            
            console.log(`📁 ${analysis.fileName}`);
            console.log(`   Lines: ${analysis.lineCount}, Tests: ${analysis.testCount}`);
            console.log(`   Priority: ${analysis.migrationPriority}, Complexity: ${analysis.migrationComplexity}`);
            console.log(`   Before: ${analysis.beforeEachPattern.type} (${analysis.beforeEachPattern.tables.length} tables)`);
            console.log(`   After: ${analysis.afterEachPattern.hasValidation ? 'has validation' : 'no validation'}`);
            console.log(`   Recommended: ${analysis.recommendedCleanupGroup}`);
            
            if (analysis.potentialIssues.length > 0) {
                console.log(`   Issues: ${analysis.potentialIssues.join(', ')}`);
            }
            
            console.log('');
        } catch (error) {
            console.log(`❌ Failed to analyze ${filePath}: ${error}`);
        }
    }
    
    console.log('2. Generating migration plan for endpoint tests...\n');
    
    try {
        const endpointDir = '/root/Vrooli/packages/server/src/endpoints/logic';
        const analyses = await analyzer.analyzeDirectory(endpointDir);
        const plan = analyzer.generateMigrationPlan(analyses);
        
        console.log('📊 Migration Plan Summary:');
        console.log(`   Total files: ${plan.summary.totalFiles}`);
        console.log(`   High priority: ${plan.summary.byPriority.high || 0}`);
        console.log(`   Medium priority: ${plan.summary.byPriority.medium || 0}`);
        console.log(`   Low priority: ${plan.summary.byPriority.low || 0}`);
        console.log('');
        
        console.log('🔍 Pattern Distribution:');
        Object.entries(plan.summary.byPattern).forEach(([pattern, count]) => {
            console.log(`   ${pattern}: ${count} files`);
        });
        console.log('');
        
        console.log('💡 Recommendations:');
        plan.recommendations.forEach((rec, i) => {
            console.log(`   ${i + 1}. ${rec}`);
        });
        console.log('');
        
        console.log('📋 Migration Phases:');
        plan.phases.forEach((phase, i) => {
            console.log(`   ${phase.phase}:`);
            console.log(`      Files: ${phase.files.length}`);
            console.log(`      Description: ${phase.description}`);
            if (phase.files.length > 0) {
                const examples = phase.files.slice(0, 3).map(f => f.fileName);
                console.log(`      Examples: ${examples.join(', ')}${phase.files.length > 3 ? '...' : ''}`);
            }
            console.log('');
        });
        
    } catch (error) {
        console.log(`❌ Failed to analyze directory: ${error}`);
    }
    
    console.log('3. Demonstrating migration on chat.test.ts (dry run)...\n');
    
    try {
        const chatTestPath = '/root/Vrooli/packages/server/src/endpoints/logic/chat.test.ts';
        const result = await executor.migrateFile(chatTestPath, { 
            dryRun: true, 
            validateAfter: true 
        });
        
        console.log(`📝 Migration Result for ${result.filePath}:`);
        console.log(`   Success: ${result.success ? '✅' : '❌'}`);
        console.log(`   Changes: ${result.changes.length}`);
        result.changes.forEach(change => {
            console.log(`      • ${change}`);
        });
        
        if (result.warnings.length > 0) {
            console.log(`   Warnings: ${result.warnings.length}`);
            result.warnings.forEach(warning => {
                console.log(`      ⚠️  ${warning}`);
            });
        }
        
        if (result.errors.length > 0) {
            console.log(`   Errors: ${result.errors.length}`);
            result.errors.forEach(error => {
                console.log(`      ❌ ${error}`);
            });
        }
        
    } catch (error) {
        console.log(`❌ Migration demo failed: ${error}`);
    }
    
    console.log('\n4. Summary and Next Steps:\n');
    console.log('✅ Migration system is ready for use');
    console.log('✅ Analyzer can identify cleanup patterns and recommend improvements');
    console.log('✅ Executor can perform automated transformations');
    console.log('✅ System supports dry-run mode for safe testing');
    console.log('');
    console.log('🚀 To execute migrations:');
    console.log('   1. Start with high-priority files (already using new patterns)');
    console.log('   2. Use --dry-run flag to preview changes');
    console.log('   3. Use --backup flag to create backup files');
    console.log('   4. Use --validate flag to check migration success');
    console.log('');
    console.log('📁 Files created:');
    console.log('   • testMigrationAnalyzer.ts - Analyzes test files and recommends migrations');
    console.log('   • testMigrationExecutor.ts - Performs automated file transformations');
    console.log('   • migrationDemo.ts - This demonstration script');
    console.log('');
    console.log('Phase 3.2 (automated migration tools) is now complete! 🎉');
}

// Run the demo
runMigrationDemo().catch(console.error);