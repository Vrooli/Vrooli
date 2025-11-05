import type { Reporter, File, Task } from 'vitest';
import { existsSync, mkdirSync, writeFileSync } from 'fs';
import { dirname, relative } from 'path';
import type {
  RequirementReporterOptions,
  RequirementResult,
  RequirementReport,
} from './types.js';

/**
 * Custom Vitest reporter that extracts requirement IDs from test descriptions
 * and correlates them with test results.
 *
 * @example
 * // vitest.config.ts
 * import RequirementReporter from '@vrooli/vitest-requirement-reporter';
 *
 * export default defineConfig({
 *   test: {
 *     reporters: [
 *       'default',
 *       new RequirementReporter({
 *         outputFile: 'coverage/vitest-requirements.json',
 *         emitStdout: true,  // Required for phase helper integration
 *         verbose: true,
 *       }),
 *     ],
 *   },
 * });
 */
export default class RequirementReporter implements Reporter {
  private options: Required<RequirementReporterOptions>;
  private requirementMap: Map<string, RequirementResult> = new Map();
  private stats = { total: 0, passed: 0, failed: 0, skipped: 0 };
  private startTime = 0;

  constructor(options: RequirementReporterOptions = {}) {
    this.options = {
      outputFile: options.outputFile || 'coverage/vitest-requirements.json',
      scenario: options.scenario || this.detectScenario(),
      verbose: options.verbose ?? true,
      pattern: options.pattern || /\[REQ:([A-Z0-9_-]+(?:,\s*[A-Z0-9_-]+)*)\]/gi,
      emitStdout: options.emitStdout ?? true,
    };
  }

  /**
   * Auto-detect scenario name from current working directory
   */
  private detectScenario(): string {
    const cwd = process.cwd();
    const scenariosMatch = cwd.match(/scenarios\/([^/]+)/);
    return scenariosMatch?.[1] || 'unknown';
  }

  /**
   * Validate requirement ID against schema pattern: [A-Z][A-Z0-9]+-[A-Z0-9-]+
   */
  private validateRequirementId(id: string): boolean {
    const pattern = /^[A-Z][A-Z0-9]+-[A-Z0-9-]+$/;
    if (!pattern.test(id)) {
      console.warn(`⚠️  Invalid requirement ID format: ${id} (expected: [A-Z][A-Z0-9]+-[A-Z0-9-]+)`);
      return false;
    }
    return true;
  }

  /**
   * Initialize reporter state at the start of test run
   */
  onInit(): void {
    this.startTime = Date.now();
    this.requirementMap.clear();
    this.stats = { total: 0, passed: 0, failed: 0, skipped: 0 };
  }

  /**
   * Extract requirement IDs from test name hierarchy
   * Walks up from test → suite → parent suite, collecting all [REQ:ID] tags
   */
  private extractRequirements(task: Task): string[] {
    const requirements: Set<string> = new Set();
    let current: Task | undefined = task;

    // Walk up the test hierarchy
    while (current) {
      const matches = current.name.matchAll(this.options.pattern);
      for (const match of matches) {
        // Handle comma-separated requirements: [REQ:ID1, ID2, ID3]
        const ids = match[1].split(/,\s*/);
        ids.forEach(id => {
          const trimmedId = id.trim();
          if (this.validateRequirementId(trimmedId)) {
            requirements.add(trimmedId);
          }
        });
      }
      current = current.suite;
    }

    return Array.from(requirements);
  }

  /**
   * Generate human-readable test path for evidence string
   * Format: relative/path/to/file.test.ts:Suite > Nested Suite > Test Name
   */
  private getTestPath(task: Task, file: File): string {
    const parts: string[] = [];
    let current: Task | undefined = task;

    while (current && (current.type !== 'suite' || current.name)) {
      if (current.name) {
        // Remove [REQ:...] tags from display path for cleaner evidence
        const cleanName = current.name.replace(this.options.pattern, '').trim();
        parts.unshift(cleanName);
      }
      current = current.suite;
    }

    const relativePath = relative(process.cwd(), file.filepath);
    return `${relativePath}:${parts.join(' > ')}`;
  }

  /**
   * Process a single test task and update requirement tracking
   */
  private processTask(task: Task, file: File): void {
    // Recursively process suite children
    if (task.type !== 'test') {
      task.tasks?.forEach(child => this.processTask(child, file));
      return;
    }

    this.stats.total++;

    const requirements = this.extractRequirements(task);
    if (requirements.length === 0) {
      // No requirements tagged, skip tracking
      return;
    }

    // IMPORTANT: Skipped tests are completely ignored (user requirement)
    if (task.result?.state === 'skip') {
      this.stats.skipped++;
      return;
    }

    const status = task.result?.state === 'pass' ? 'passed' : 'failed';
    this.stats[status]++;

    const duration = task.result?.duration || 0;
    const evidence = this.getTestPath(task, file);

    // Update each requirement this test covers
    requirements.forEach(reqId => {
      const existing = this.requirementMap.get(reqId);

      if (!existing) {
        // First test for this requirement
        this.requirementMap.set(reqId, {
          id: reqId,
          status,
          evidence,
          duration_ms: duration,
          test_count: 1,
        });
      } else {
        // Aggregate results for this requirement
        // Worst status wins: failed > passed
        if (status === 'failed') {
          existing.status = status;
        }

        // USER REQUIREMENT: List ALL tests (if 10 tests, show all 10)
        existing.evidence += `; ${evidence}`;
        existing.duration_ms += duration;
        existing.test_count++;
      }
    });
  }

  /**
   * Generate and write final report after all tests complete
   */
  async onFinished(files?: File[]): Promise<void> {
    if (!files) return;

    // Process all test files
    files.forEach(file => {
      file.tasks.forEach(task => this.processTask(task, file));
    });

    const report: RequirementReport = {
      generated_at: new Date().toISOString(),
      scenario: this.options.scenario,
      phase: 'unit',
      test_framework: 'vitest',
      total_tests: this.stats.total,
      passed_tests: this.stats.passed,
      failed_tests: this.stats.failed,
      skipped_tests: this.stats.skipped,
      duration_ms: Date.now() - this.startTime,
      requirements: Array.from(this.requirementMap.values()).sort((a, b) =>
        a.id.localeCompare(b.id)
      ),
    };

    // Ensure output directory exists
    const outputDir = dirname(this.options.outputFile);
    if (!existsSync(outputDir)) {
      mkdirSync(outputDir, { recursive: true });
    }

    // Write JSON report
    writeFileSync(this.options.outputFile, JSON.stringify(report, null, 2));

    // Optional console summary
    if (this.options.verbose) {
      this.printSummary(report);
    }

    // Emit parseable output for existing shell infrastructure
    // CRITICAL: This enables backward compatibility with _node_collect_requirement_tags()
    if (this.options.emitStdout) {
      this.printParseableOutput(report);
    }
  }

  /**
   * Print parseable format compatible with existing testing::unit::_node_collect_requirement_tags()
   * Format matches pattern expected by node.sh:353-382
   * Emits lines matching: [MARKER] REQ:ID (description)
   */
  private printParseableOutput(report: RequirementReport): void {
    console.log('\n--- Requirement Coverage (Parseable) ---');

    report.requirements.forEach(req => {
      // Use markers that existing parser recognizes (✓ PASS, ✗ FAIL)
      const marker = req.status === 'passed' ? '✓ PASS' : '✗ FAIL';

      // Format: [MARKER] REQ:ID (test count, duration)
      console.log(`${marker} REQ:${req.id} (${req.test_count} tests, ${req.duration_ms}ms)`);
    });

    console.log('--- End Requirement Coverage ---\n');
  }

  /**
   * Print human-readable summary to console
   */
  private printSummary(report: RequirementReport): void {
    console.log('\n📋 Requirement Coverage Report:');
    console.log(`   Scenario: ${report.scenario}`);
    console.log(`   Requirements: ${report.requirements.length} covered`);
    console.log(`   Tests: ${report.passed_tests}/${report.total_tests} passed`);

    const failed = report.requirements.filter(r => r.status === 'failed');
    if (failed.length > 0) {
      console.log(`   ⚠️  Failed: ${failed.map(r => r.id).join(', ')}`);
    }

    console.log(`   Output: ${this.options.outputFile}\n`);
  }
}
