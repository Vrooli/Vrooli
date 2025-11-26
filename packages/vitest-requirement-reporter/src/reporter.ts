import type { Reporter, File, Task } from 'vitest';
import { existsSync, mkdirSync, writeFileSync, readFileSync, rmSync } from 'fs';
import { dirname, relative, join, basename } from 'path';
import { execSync } from 'child_process';
import type {
  RequirementReporterOptions,
  RequirementResult,
  RequirementReport,
  TestFailure,
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
  private failures: TestFailure[] = [];
  private projectStats: Map<string, { passed: number; failed: number; duration: number }> = new Map();

  // Suppress default vitest output when in concise mode
  private suppressedOutput: boolean = false;

  constructor(options: RequirementReporterOptions = {}) {
    const conciseMode = options.conciseMode ?? process.env.VITEST_CONCISE === '1';
    this.options = {
      outputFile: options.outputFile || 'coverage/vitest-requirements.json',
      scenario: options.scenario || this.detectScenario(),
      verbose: options.verbose ?? true,
      pattern: options.pattern || /\[REQ:([A-Z0-9_-]+(?:,\s*[A-Z0-9_-]+)*)\]/gi,
      emitStdout: options.emitStdout ?? true,
      append: options.append ?? process.env.VITEST_REQUIREMENTS_APPEND === '1',
      conciseMode,
      artifactsDir: options.artifactsDir || 'coverage/unit',
      autoClear: options.autoClear ?? conciseMode,
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
   * Called when test collection starts
   */
  onCollected(): void {
    if (this.options.conciseMode) {
      // Minimal output - just show we're running
      // The project info will be shown in the final summary
    }
  }

  /**
   * Initialize reporter state at the start of test run
   */
  onInit(): void {
    this.startTime = Date.now();
    this.requirementMap.clear();
    this.stats = { total: 0, passed: 0, failed: 0, skipped: 0 };
    this.failures = [];
    this.projectStats.clear();

    // Auto-clear artifacts directory if enabled
    if (this.options.autoClear && existsSync(this.options.artifactsDir)) {
      try {
        rmSync(this.options.artifactsDir, { recursive: true, force: true });
      } catch (error) {
        // Ignore errors during cleanup
      }
    }
  }

  /**
   * Called when a task (test/suite) updates - suppress verbose output in concise mode
   */
  onTaskUpdate(): void {
    // In concise mode, we don't output per-test updates
    // All output happens in onFinished
  }

  /**
   * Called when watching for changes - not applicable for CI runs
   */
  onWatcherStart(): void {
    // No-op
  }

  /**
   * Called when watcher is ready
   */
  onWatcherRerun(): void {
    // No-op
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
   * Extract project name from test file path or meta
   */
  private getProjectName(file: File): string {
    // Try to get from file.projectName if available
    const projectName = (file as any).projectName;
    if (projectName) {
      return projectName;
    }

    // Fall back to parsing file path
    const relativePath = relative(process.cwd(), file.filepath);
    const parts = relativePath.split('/');
    if (parts.length > 1 && parts[0] === 'src') {
      // Look for test path patterns like src/components/__tests__/
      const testIdx = parts.findIndex(p => p === '__tests__' || p === 'tests' || p === 'test');
      if (testIdx > 0) {
        return parts[testIdx - 1];
      }
    }
    return 'default';
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
    const duration = task.result?.duration || 0;
    const projectName = this.getProjectName(file);

    // Update project stats
    if (!this.projectStats.has(projectName)) {
      this.projectStats.set(projectName, { passed: 0, failed: 0, duration: 0 });
    }
    const projStats = this.projectStats.get(projectName)!;
    projStats.duration += duration;

    // IMPORTANT: Skipped tests are completely ignored (user requirement)
    if (task.result?.state === 'skip') {
      this.stats.skipped++;
      return;
    }

    const status = task.result?.state === 'pass' ? 'passed' : 'failed';
    this.stats[status]++;

    if (status === 'passed') {
      projStats.passed++;
    } else {
      projStats.failed++;

      // Track failure for artifact generation
      if (this.options.conciseMode) {
        this.trackFailure(task, file, requirements, duration);
      }
    }

    if (requirements.length === 0) {
      // No requirements tagged, skip tracking
      return;
    }

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
   * Track a test failure for artifact generation
   */
  private trackFailure(task: Task, file: File, requirements: string[], duration: number): void {
    const error = task.result?.errors?.[0];
    if (!error) return;

    const testPath = this.getTestPath(task, file);
    const relativePath = relative(process.cwd(), file.filepath);

    // Get raw error string
    const errorStr = error.message || String(error);

    // Extract HTML snapshot from testing-library errors
    // Look for pattern: "Ignored nodes: ... <body>..." (may be truncated)
    let htmlSnapshot: string | undefined;
    let cleanErrorMessage = errorStr;

    if (errorStr.includes('Ignored nodes:')) {
      // Extract first line (the actual error message)
      const firstLine = errorStr.split('\n')[0];
      cleanErrorMessage = firstLine;

      // Extract HTML starting from "Ignored nodes:"
      // It may or may not have a closing </body> tag (testing-library truncates)
      const htmlStartIndex = errorStr.indexOf('Ignored nodes:');
      if (htmlStartIndex >= 0) {
        const htmlContent = errorStr.substring(htmlStartIndex);
        htmlSnapshot = this.stripAnsi(htmlContent);
      }
    }

    // Strip ANSI codes from error message
    cleanErrorMessage = this.stripAnsi(cleanErrorMessage);

    this.failures.push({
      projectName: this.getProjectName(file),
      testPath,
      filePath: relativePath,
      line: (error as any).line,
      errorMessage: cleanErrorMessage,
      stackTrace: error.stack,
      htmlSnapshot,
      duration,
      requirements,
      status: 'failed',
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

    const finalReport = this.appendExistingReportIfNeeded(report);

    // Ensure output directory exists
    const outputDir = dirname(this.options.outputFile);
    if (!existsSync(outputDir)) {
      mkdirSync(outputDir, { recursive: true });
    }

    // Write JSON report
    writeFileSync(this.options.outputFile, JSON.stringify(finalReport, null, 2));

    // Generate failure artifacts if in concise mode
    if (this.options.conciseMode) {
      this.generateFailureArtifacts();
      this.printConciseSummary(finalReport);
    } else {
      // Optional console summary (legacy mode)
      if (this.options.verbose) {
        this.printSummary(finalReport);
      }
    }

    // Emit parseable output for existing shell infrastructure
    // CRITICAL: This enables backward compatibility with _node_collect_requirement_tags()
    if (this.options.emitStdout) {
      this.printParseableOutput(finalReport);
    }
  }

  /**
   * Extract just the test name from full path (strip file path prefix)
   * Input: "src/components/__tests__/NodePalette.test.tsx:NodePalette > renders categories"
   * Output: "NodePalette > renders categories"
   */
  private extractTestName(testPath: string): string {
    // Strip file path - everything before and including the last ':'
    const colonIndex = testPath.lastIndexOf(':');
    const testNamePart = colonIndex >= 0 ? testPath.substring(colonIndex + 1) : testPath;
    return testNamePart.trim();
  }

  /**
   * Sanitize test name for use in directory/file names
   */
  private sanitizeTestName(testPath: string): string {
    // First extract just the test name (no file path)
    const testName = this.extractTestName(testPath);

    return testName
      .replace(/[^a-zA-Z0-9-_]/g, '-')
      .replace(/-+/g, '-')
      .replace(/^-|-$/g, '')
      .substring(0, 80); // Reduced from 100 to leave room for prefix
  }

  /**
   * Strip ANSI color codes from string
   */
  private stripAnsi(str: string): string {
    // eslint-disable-next-line no-control-regex
    return str.replace(/\x1b\[[0-9;]*m/g, '');
  }

  /**
   * Get git status info for context
   */
  private getGitContext(): { recentChanges: string[]; deletedFiles: string[] } {
    try {
      const status = execSync('git status --short', { encoding: 'utf-8', timeout: 2000 });
      const lines = status.split('\n').filter(l => l.trim());

      const recentChanges: string[] = [];
      const deletedFiles: string[] = [];

      lines.forEach(line => {
        const match = line.match(/^\s*([MAD?!])\s+(.+)$/);
        if (match) {
          const [, status, file] = match;
          if (status === 'D') {
            deletedFiles.push(file);
          } else {
            recentChanges.push(`${status} ${file}`);
          }
        }
      });

      return { recentChanges, deletedFiles };
    } catch {
      return { recentChanges: [], deletedFiles: [] };
    }
  }

  /**
   * Analyze failure and suggest likely causes
   */
  private analyzeLikelyCauses(failure: TestFailure): string[] {
    const causes: string[] = [];
    const errorLower = failure.errorMessage.toLowerCase();
    const { deletedFiles } = this.getGitContext();

    // Pattern: Unable to find element
    if (errorLower.includes('unable to find') || errorLower.includes('could not find')) {
      causes.push('Element not rendered or query selector incorrect');

      // Check if related component was deleted/renamed
      const testFile = basename(failure.filePath, '.test.ts').replace('.test', '');
      const relatedDeleted = deletedFiles.filter(f =>
        f.includes(testFile) || f.toLowerCase().includes(errorLower.match(/text: ([a-z\s]+)/)?.[1] || '')
      );
      if (relatedDeleted.length > 0) {
        causes.push(`Related files deleted: ${relatedDeleted.join(', ')}`);
      }
    }

    // Pattern: Expected X to be Y
    if (errorLower.includes('expected') && errorLower.includes('to be')) {
      causes.push('Assertion mismatch - check expected vs actual values');
    }

    // Pattern: Timeout
    if (errorLower.includes('timeout') || errorLower.includes('timed out')) {
      causes.push('Async operation not completing - check waitFor conditions');
    }

    // Pattern: Cannot read property
    if (errorLower.includes('cannot read prop') || errorLower.includes('undefined')) {
      causes.push('Null/undefined access - check component state initialization');
    }

    return causes.length > 0 ? causes : ['Review error message and stack trace for details'];
  }

  /**
   * Generate README.md for a test failure
   */
  private generateFailureReadme(failure: TestFailure, artifactDir: string): string {
    const { recentChanges, deletedFiles } = this.getGitContext();
    const likelyCauses = this.analyzeLikelyCauses(failure);
    const timestamp = new Date().toISOString().replace('T', ' ').substring(0, 19);

    // Use clean test name for title (no file path)
    const cleanTestName = this.extractTestName(failure.testPath);

    let readme = `# Test Failure: ${cleanTestName}\n\n`;
    readme += `**Project:** ${failure.projectName}\n`;
    readme += `**File:** ${failure.filePath}${failure.line ? `:${failure.line}` : ''}\n`;
    readme += `**Duration:** ${failure.duration}ms\n`;
    readme += `**Failed:** ${timestamp}\n\n`;

    readme += `## Error Summary\n\n`;
    const errorFirstLine = failure.errorMessage.split('\n')[0];
    readme += `${errorFirstLine}\n\n`;

    if (failure.requirements.length > 0) {
      readme += `## Requirements Tested\n\n`;
      failure.requirements.forEach(req => {
        readme += `- ${req} ❌\n`;
      });
      readme += '\n';
    }

    readme += `## Likely Causes\n\n`;
    likelyCauses.forEach((cause, idx) => {
      readme += `${idx + 1}. ${cause}\n`;
    });
    readme += '\n';

    readme += `## Debug Artifacts\n\n`;
    readme += `- \`error.txt\` - Clean error message (ANSI codes stripped)\n`;
    if (failure.stackTrace) {
      readme += `- \`stack-trace.txt\` - Full stack trace with source maps\n`;
    }
    if (failure.htmlSnapshot) {
      readme += `- \`html-snapshot.html\` - Rendered DOM at failure (formatted, ANSI codes stripped)\n`;
    }
    readme += `- \`test-context.json\` - Test metadata and timing\n\n`;

    if (deletedFiles.length > 0 || recentChanges.length > 0) {
      readme += `## Recent Changes (git status)\n\n`;
      if (deletedFiles.length > 0) {
        readme += `**Deleted files:**\n`;
        deletedFiles.slice(0, 5).forEach(f => readme += `- ${f}\n`);
        readme += '\n';
      }
      if (recentChanges.length > 0) {
        readme += `**Modified files:**\n`;
        recentChanges.slice(0, 10).forEach(f => readme += `- ${f}\n`);
        readme += '\n';
      }
    }

    return readme;
  }

  /**
   * Generate artifacts for all test failures
   */
  private generateFailureArtifacts(): void {
    if (this.failures.length === 0) return;

    this.failures.forEach(failure => {
      const sanitizedName = this.sanitizeTestName(failure.testPath);
      const artifactDir = join(this.options.artifactsDir, failure.projectName, sanitizedName);

      // Create artifact directory
      mkdirSync(artifactDir, { recursive: true });

      // Write README.md
      const readme = this.generateFailureReadme(failure, artifactDir);
      writeFileSync(join(artifactDir, 'README.md'), readme);

      // Write error.txt (already stripped of ANSI codes in trackFailure)
      writeFileSync(join(artifactDir, 'error.txt'), failure.errorMessage);

      // Write stack-trace.txt (strip ANSI codes)
      if (failure.stackTrace) {
        const cleanStack = this.stripAnsi(failure.stackTrace);
        writeFileSync(join(artifactDir, 'stack-trace.txt'), cleanStack);
      }

      // Write html-snapshot.html
      if (failure.htmlSnapshot) {
        // Format HTML for readability
        const formatted = this.formatHtmlSnapshot(failure.htmlSnapshot);
        writeFileSync(join(artifactDir, 'html-snapshot.html'), formatted);
      }

      // Write test-context.json
      const context = {
        projectName: failure.projectName,
        testPath: failure.testPath,
        filePath: failure.filePath,
        line: failure.line,
        duration: failure.duration,
        requirements: failure.requirements,
        timestamp: new Date().toISOString(),
      };
      writeFileSync(join(artifactDir, 'test-context.json'), JSON.stringify(context, null, 2));
    });
  }

  /**
   * Format HTML snapshot for better readability
   */
  private formatHtmlSnapshot(html: string): string {
    // Simple newline-based formatting for readability
    // Replace closing tags with newline + closing tag
    let formatted = html
      .replace(/></g, '>\n<')           // Add newline between tags
      .replace(/\n\s*\n/g, '\n');       // Remove multiple newlines

    // Add basic indentation
    const lines = formatted.split('\n');
    let indent = 0;
    const indented = lines.map(line => {
      const trimmed = line.trim();
      if (!trimmed) return '';

      // Decrease indent for closing tags
      if (trimmed.startsWith('</')) {
        indent = Math.max(0, indent - 1);
      }

      const result = '  '.repeat(indent) + trimmed;

      // Increase indent after opening tags (but not self-closing)
      if (trimmed.startsWith('<') && !trimmed.startsWith('</') && !trimmed.endsWith('/>')) {
        indent++;
      }

      return result;
    });

    return indented.join('\n');
  }

  /**
   * Print concise summary matching integration phase pattern
   */
  private printConciseSummary(report: RequirementReport): void {
    console.log('\n');

    // Group project names
    const projects = Array.from(this.projectStats.keys()).sort();
    if (projects.length > 0) {
      console.log(`[INFO]    Running ${projects.length} project${projects.length > 1 ? 's' : ''}: ${projects.join(', ')}`);
      console.log('');
    }

    // Print per-project results
    projects.forEach(project => {
      const stats = this.projectStats.get(project)!;
      const total = stats.passed + stats.failed;
      const durationSec = (stats.duration / 1000).toFixed(1);

      if (stats.failed === 0) {
        console.log(`✅ ${project}: ${total}/${total} passed (${durationSec}s)`);
      } else {
        console.log(`❌ ${project}: ${stats.passed}/${total} passed, ${stats.failed} failed (${durationSec}s)`);

        // Show failed test paths for this project
        const projectFailures = this.failures.filter(f => f.projectName === project);
        projectFailures.forEach(failure => {
          const sanitizedName = this.sanitizeTestName(failure.testPath);
          // Construct absolute path: if running from ui/, go up one level to scenario root
          const cwd = process.cwd();
          const scenarioRoot = cwd.endsWith('/ui') ? dirname(cwd) : cwd;
          const artifactPath = join(scenarioRoot, this.options.artifactsDir, project, sanitizedName, 'README.md');
          // Show clean test name in console (without file path)
          const cleanTestName = this.extractTestName(failure.testPath);
          console.log(`   ↳ ${cleanTestName}`);
          console.log(`   ↳ Read ${artifactPath}`);
        });
      }
    });

    // Overall summary
    const totalDurationSec = (report.duration_ms / 1000).toFixed(1);
    const totalTests = report.passed_tests + report.failed_tests;
    console.log('');
    console.log(`Summary: ${report.passed_tests}/${totalTests} passed (${report.failed_tests} failed) in ${totalDurationSec}s`);
    console.log('');
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

  private appendExistingReportIfNeeded(report: RequirementReport): RequirementReport {
    if (!this.options.append || !existsSync(this.options.outputFile)) {
      return report;
    }

    try {
      const previousRaw = readFileSync(this.options.outputFile, 'utf-8');
      const previousReport = JSON.parse(previousRaw) as RequirementReport;
      return this.mergeReports(previousReport, report);
    } catch {
      return report;
    }
  }

  private mergeReports(previous: RequirementReport, current: RequirementReport): RequirementReport {
    const merged = new Map<string, RequirementResult>();

    const addRequirement = (result: RequirementResult) => {
      const existing = merged.get(result.id);
      if (!existing) {
        merged.set(result.id, { ...result });
        return;
      }

      if (result.status === 'failed') {
        existing.status = 'failed';
      }

      if (result.evidence) {
        existing.evidence = existing.evidence
          ? `${existing.evidence}; ${result.evidence}`
          : result.evidence;
      }

      existing.duration_ms += result.duration_ms;
      existing.test_count += result.test_count;
    };

    previous.requirements.forEach(addRequirement);
    current.requirements.forEach(addRequirement);

    return {
      generated_at: current.generated_at,
      scenario: current.scenario || previous.scenario,
      phase: current.phase,
      test_framework: current.test_framework,
      total_tests: previous.total_tests + current.total_tests,
      passed_tests: previous.passed_tests + current.passed_tests,
      failed_tests: previous.failed_tests + current.failed_tests,
      skipped_tests: previous.skipped_tests + current.skipped_tests,
      duration_ms: previous.duration_ms + current.duration_ms,
      requirements: Array.from(merged.values()).sort((a, b) => a.id.localeCompare(b.id)),
    };
  }
}
