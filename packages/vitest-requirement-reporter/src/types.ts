/**
 * Result for a single requirement tracked by the reporter
 */
export interface RequirementResult {
  /** Requirement ID (e.g., BAS-WORKFLOW-PERSIST-CRUD) */
  id: string;

  /** Worst status across all tests for this requirement */
  status: 'passed' | 'failed';

  /** Evidence string showing which tests validated this requirement */
  evidence: string;

  /** Total duration in milliseconds across all tests */
  duration_ms: number;

  /** Number of tests that validated this requirement */
  test_count: number;
}

/**
 * Complete requirement coverage report output
 */
export interface RequirementReport {
  /** ISO timestamp when report was generated */
  generated_at: string;

  /** Scenario name (auto-detected or configured) */
  scenario: string;

  /** Testing phase (always 'unit' for vitest) */
  phase: 'unit';

  /** Test framework identifier */
  test_framework: 'vitest';

  /** Total number of tests executed */
  total_tests: number;

  /** Number of tests that passed */
  passed_tests: number;

  /** Number of tests that failed */
  failed_tests: number;

  /** Number of tests that were skipped (NOT tracked in requirements) */
  skipped_tests: number;

  /** Total test execution time in milliseconds */
  duration_ms: number;

  /** Array of requirement results, sorted by ID */
  requirements: RequirementResult[];
}

/**
 * Configuration options for RequirementReporter
 */
export interface RequirementReporterOptions {
  /** Output file path (default: coverage/vitest-requirements.json) */
  outputFile?: string;

  /** Scenario name (auto-detected from cwd if not provided) */
  scenario?: string;

  /** Enable verbose console output (default: true) */
  verbose?: boolean;

  /** Pattern to extract requirement IDs (default: /\[REQ:([A-Z0-9_-]+(?:,\s*[A-Z0-9_-]+)*)\]/gi) */
  pattern?: RegExp;

  /** Emit parseable stdout for existing shell infrastructure (default: true) */
  emitStdout?: boolean;

  /** When true, merge with an existing output file instead of overwriting */
  append?: boolean;
}
