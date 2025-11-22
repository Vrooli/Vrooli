#!/usr/bin/env node
'use strict';

/**
 * Unit tests for duplicate-detector
 * Run with: node duplicate-detector.test.js
 */

const { analyzeTestRefUsage } = require('./duplicate-detector');

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

console.log('\n=== Duplicate Detector Unit Tests ===\n');

// Test 1: No requirements
console.log('Test Suite: analyzeTestRefUsage() - Empty requirements');
const analysis1 = analyzeTestRefUsage([]);
assertEquals(analysis1.total_refs, 0, 'no requirements = 0 refs');
assertEquals(analysis1.total_requirements, 0, 'no requirements = 0 total');
assertEquals(analysis1.violations.length, 0, 'no requirements = 0 violations');

// Test 2: Unique test refs (no duplicates)
console.log('\nTest Suite: analyzeTestRefUsage() - Unique refs');
const requirements2 = [
  { id: 'req1', validation: [{ ref: 'test1.go' }] },
  { id: 'req2', validation: [{ ref: 'test2.go' }] },
  { id: 'req3', validation: [{ ref: 'test3.go' }] }
];
const analysis2 = analyzeTestRefUsage(requirements2);
assertEquals(analysis2.total_refs, 3, 'unique refs = 3 total');
assertEquals(analysis2.total_requirements, 3, 'unique refs = 3 requirements');
assertEquals(analysis2.violations.length, 0, 'unique refs = 0 violations');

// Test 3: Monolithic test file (≥4 requirements)
console.log('\nTest Suite: analyzeTestRefUsage() - Monolithic test');
const requirements3 = [
  { id: 'req1', validation: [{ ref: 'test1.go' }] },
  { id: 'req2', validation: [{ ref: 'test1.go' }] },
  { id: 'req3', validation: [{ ref: 'test1.go' }] },
  { id: 'req4', validation: [{ ref: 'test1.go' }] }
];
const analysis3 = analyzeTestRefUsage(requirements3);
assertEquals(analysis3.total_refs, 1, 'monolithic = 1 ref');
assertEquals(analysis3.total_requirements, 4, 'monolithic = 4 requirements');
assertEquals(analysis3.violations.length, 1, 'monolithic = 1 violation');
assert(analysis3.violations[0].count === 4, 'violation count = 4');
assertEquals(analysis3.violations[0].severity, 'medium', 'severity = medium for 4 reqs');

// Test 4: High severity monolithic (≥6 requirements)
console.log('\nTest Suite: analyzeTestRefUsage() - High severity');
const requirements4 = [
  { id: 'req1', validation: [{ ref: 'test1.go' }] },
  { id: 'req2', validation: [{ ref: 'test1.go' }] },
  { id: 'req3', validation: [{ ref: 'test1.go' }] },
  { id: 'req4', validation: [{ ref: 'test1.go' }] },
  { id: 'req5', validation: [{ ref: 'test1.go' }] },
  { id: 'req6', validation: [{ ref: 'test1.go' }] }
];
const analysis4 = analyzeTestRefUsage(requirements4);
assertEquals(analysis4.violations.length, 1, 'high severity = 1 violation');
assertEquals(analysis4.violations[0].severity, 'high', 'severity = high for 6 reqs');

// Test 5: Multiple violations
console.log('\nTest Suite: analyzeTestRefUsage() - Multiple violations');
const requirements5 = [
  { id: 'req1', validation: [{ ref: 'test1.go' }] },
  { id: 'req2', validation: [{ ref: 'test1.go' }] },
  { id: 'req3', validation: [{ ref: 'test1.go' }] },
  { id: 'req4', validation: [{ ref: 'test1.go' }] },
  { id: 'req5', validation: [{ ref: 'test2.go' }] },
  { id: 'req6', validation: [{ ref: 'test2.go' }] },
  { id: 'req7', validation: [{ ref: 'test2.go' }] },
  { id: 'req8', validation: [{ ref: 'test2.go' }] }
];
const analysis5 = analyzeTestRefUsage(requirements5);
assertEquals(analysis5.violations.length, 2, 'multiple monoliths = 2 violations');

// Test 6: workflow_id support (BAS format)
console.log('\nTest Suite: analyzeTestRefUsage() - workflow_id support');
const requirements6 = [
  { id: 'req1', validation: [{ workflow_id: 'workflow1' }] },
  { id: 'req2', validation: [{ workflow_id: 'workflow1' }] },
  { id: 'req3', validation: [{ workflow_id: 'workflow1' }] },
  { id: 'req4', validation: [{ workflow_id: 'workflow1' }] }
];
const analysis6 = analyzeTestRefUsage(requirements6);
assertEquals(analysis6.violations.length, 1, 'workflow_id supported');
assert(analysis6.violations[0].test_ref === 'workflow1', 'uses workflow_id as ref');

console.log(`\n=== Test Summary ===`);
console.log(`Total: ${testsRun}, Passed: ${testsPassed}, Failed: ${testsFailed}`);

if (testsFailed > 0) {
  console.log('\n❌ Tests failed');
  process.exit(1);
} else {
  console.log('\n✅ All tests passed');
  process.exit(0);
}
