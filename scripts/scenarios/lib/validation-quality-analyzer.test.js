#!/usr/bin/env node
'use strict';

/**
 * Unit tests for validation-quality-analyzer
 * Run with: node validation-quality-analyzer.test.js
 */

const { detectValidationQualityIssues } = require('./validation-quality-analyzer');
const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');

let testsRun = 0;
let testsPassed = 0;
let testsFailed = 0;

function assert(condition, message) {
  testsRun++;
  if (condition) {
    testsPassed++;
    console.log(`  ✓ ${message}`);
  } else {
    testsFailed++;
    console.error(`  ✗ ${message}`);
  }
}

function assertEquals(actual, expected, message) {
  testsRun++;
  if (actual === expected) {
    testsPassed++;
    console.log(`  ✓ ${message}`);
  } else {
    testsFailed++;
    console.error(`  ✗ ${message} (expected ${expected}, got ${actual})`);
  }
}

console.log('\n=== Validation Quality Analyzer Unit Tests ===\n');

// Create temporary test directory
const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'validation-quality-test-'));
const scenarioRoot = path.join(tmpDir, 'test-scenario');
fs.mkdirSync(scenarioRoot);
fs.mkdirSync(path.join(scenarioRoot, 'api'));
fs.writeFileSync(path.join(scenarioRoot, 'api', 'main.go'), '// Go code');

try {
  // Test 1: Basic functionality test
  console.log('Test Suite: detectValidationQualityIssues() - Basic functionality');
  const metrics1 = {
    requirements: { total: 10, passing: 10 },
    targets: { total: 5, passing: 5 },
    tests: { total: 20, passing: 20 }
  };
  const requirements1 = [];
  const targets1 = [];

  const analysis1 = detectValidationQualityIssues(metrics1, requirements1, targets1, scenarioRoot);
  assertEquals(typeof analysis1.total_penalty, 'number', 'returns total_penalty');
  assertEquals(typeof analysis1.has_issues, 'boolean', 'returns has_issues');
  assert(Array.isArray(analysis1.issues), 'returns issues array');

  // Test 2: Suspicious 1:1 ratio
  console.log('\nTest Suite: detectValidationQualityIssues() - Suspicious 1:1 ratio');
  const metrics2 = {
    requirements: { total: 10, passing: 10 },
    targets: { total: 5, passing: 5 },
    tests: { total: 10, passing: 10 }  // Exactly 1:1
  };
  const analysis2 = detectValidationQualityIssues(metrics2, [], [], scenarioRoot);
  assert(analysis2.patterns.insufficient_test_coverage, 'detects 1:1 ratio');
  assertEquals(analysis2.patterns.insufficient_test_coverage.severity, 'medium', '1:1 ratio is medium severity');
  assert(analysis2.patterns.insufficient_test_coverage.penalty > 0, 'applies penalty for 1:1 ratio');

  // Test 3: Invalid test location
  console.log('\nTest Suite: detectValidationQualityIssues() - Invalid test locations');
  const requirements3 = [
    { id: 'req1', validation: [{ type: 'test', ref: 'test/cli/invalid.bats' }] },
    { id: 'req2', validation: [{ type: 'test', ref: 'test/phases/invalid.sh' }] }
  ];
  const metrics3 = {
    requirements: { total: 2, passing: 0 },
    targets: { total: 1, passing: 0 },
    tests: { total: 2, passing: 2 }
  };
  const analysis3 = detectValidationQualityIssues(metrics3, requirements3, [], scenarioRoot);
  assert(analysis3.patterns.invalid_test_location, 'detects invalid test locations');
  assertEquals(analysis3.patterns.invalid_test_location.count, 2, 'counts invalid refs');
  assert(analysis3.patterns.invalid_test_location.penalty > 0, 'applies penalty');

  // Test 4: Monolithic test files
  console.log('\nTest Suite: detectValidationQualityIssues() - Monolithic tests');
  const requirements4 = [
    { id: 'req1', validation: [{ type: 'test', ref: 'api/mega_test.go' }] },
    { id: 'req2', validation: [{ type: 'test', ref: 'api/mega_test.go' }] },
    { id: 'req3', validation: [{ type: 'test', ref: 'api/mega_test.go' }] },
    { id: 'req4', validation: [{ type: 'test', ref: 'api/mega_test.go' }] }
  ];
  const metrics4 = {
    requirements: { total: 4, passing: 0 },
    targets: { total: 1, passing: 0 },
    tests: { total: 1, passing: 1 }
  };
  const analysis4 = detectValidationQualityIssues(metrics4, requirements4, [], scenarioRoot);
  assert(analysis4.patterns.monolithic_test_files, 'detects monolithic tests');
  assertEquals(analysis4.patterns.monolithic_test_files.violations, 1, 'counts violations');

  // Test 5: Multiple issue types detected
  console.log('\nTest Suite: detectValidationQualityIssues() - Multiple issues');
  const requirements5 = [
    { id: 'req1', validation: [{ type: 'test', ref: 'test/cli/invalid1.bats' }] },
    { id: 'req2', validation: [{ type: 'test', ref: 'test/cli/invalid2.bats' }] },
    { id: 'req3', validation: [{ type: 'test', ref: 'test/cli/mega.bats' }] },
    { id: 'req4', validation: [{ type: 'test', ref: 'test/cli/mega.bats' }] },
    { id: 'req5', validation: [{ type: 'test', ref: 'test/cli/mega.bats' }] },
    { id: 'req6', validation: [{ type: 'test', ref: 'test/cli/mega.bats' }] }
  ];
  const metrics5 = {
    requirements: { total: 6, passing: 0 },
    targets: { total: 1, passing: 0 },
    tests: { total: 6, passing: 6 }
  };
  const analysis5 = detectValidationQualityIssues(metrics5, requirements5, [], scenarioRoot);
  assert(analysis5.has_issues, 'detects multiple issues');
  assert(analysis5.total_penalty > 0, 'accumulates penalties from multiple issues');

  // Test 6: Excessive manual validations
  console.log('\nTest Suite: detectValidationQualityIssues() - Manual validations');
  const requirements6 = [
    { id: 'req1', status: 'complete', validation: [{ type: 'manual', ref: 'manual check' }] },
    { id: 'req2', status: 'complete', validation: [{ type: 'manual', ref: 'manual check' }] },
    { id: 'req3', status: 'complete', validation: [{ type: 'manual', ref: 'manual check' }] },
    { id: 'req4', status: 'complete', validation: [{ type: 'manual', ref: 'manual check' }] },
    { id: 'req5', status: 'complete', validation: [{ type: 'manual', ref: 'manual check' }] },
    { id: 'req6', status: 'complete', validation: [{ type: 'manual', ref: 'manual check' }] }
  ];
  const metrics6 = {
    requirements: { total: 6, passing: 0 },
    targets: { total: 1, passing: 0 },
    tests: { total: 0, passing: 0 }
  };
  const analysis6 = detectValidationQualityIssues(metrics6, requirements6, [], scenarioRoot);
  assert(analysis6.patterns.missing_test_automation, 'detects missing automation');
  assert(analysis6.patterns.missing_test_automation.complete_with_manual >= 5, 'counts complete reqs with only manual');

  console.log('\n=== Integration Test: Multiple issues ===');
  const multiIssueAnalysis = detectValidationQualityIssues(metrics6, requirements6, [], scenarioRoot);
  assert(multiIssueAnalysis.has_issues, 'detects has_issues=true');
  assert(multiIssueAnalysis.issue_count > 0, 'counts multiple issues');
  assert(multiIssueAnalysis.total_penalty > 0, 'accumulates penalties');
  assertEquals(typeof multiIssueAnalysis.overall_severity, 'string', 'calculates overall_severity');

} finally {
  // Cleanup
  fs.rmSync(tmpDir, { recursive: true, force: true });
}

console.log(`\n=== Test Summary ===`);
console.log(`Total: ${testsRun}, Passed: ${testsPassed}, Failed: ${testsFailed}`);

if (testsFailed > 0) {
  console.log('\n❌ Tests failed');
  process.exit(1);
} else {
  console.log('\n✅ All tests passed');
  process.exit(0);
}
