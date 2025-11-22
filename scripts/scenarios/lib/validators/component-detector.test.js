#!/usr/bin/env node
'use strict';

/**
 * Unit tests for component-detector
 * Run with: node component-detector.test.js
 */

const { detectScenarioComponents, getApplicableLayers } = require('./component-detector');
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

console.log('\n=== Component Detector Unit Tests ===\n');

// Create temporary test directories
const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'component-detector-test-'));

try {
  // Test 1: Empty scenario (no components)
  console.log('Test Suite: detectScenarioComponents() - Empty scenario');
  const emptyScenario = path.join(tmpDir, 'empty-scenario');
  fs.mkdirSync(emptyScenario);

  const components1 = detectScenarioComponents(emptyScenario);
  assertEquals(components1.size, 0, 'empty scenario has 0 components');

  // Test 2: Scenario with API component
  console.log('\nTest Suite: detectScenarioComponents() - API component');
  const apiScenario = path.join(tmpDir, 'api-scenario');
  fs.mkdirSync(apiScenario);
  fs.mkdirSync(path.join(apiScenario, 'api'));
  fs.writeFileSync(path.join(apiScenario, 'api', 'main.go'), '// Go code');

  const components2 = detectScenarioComponents(apiScenario);
  assertEquals(components2.size, 1, 'API scenario has 1 component');
  assert(components2.has('API'), 'detects API component');

  // Test 3: Scenario with UI component
  console.log('\nTest Suite: detectScenarioComponents() - UI component');
  const uiScenario = path.join(tmpDir, 'ui-scenario');
  fs.mkdirSync(uiScenario);
  fs.mkdirSync(path.join(uiScenario, 'ui'));
  fs.writeFileSync(path.join(uiScenario, 'ui', 'package.json'), '{}');

  const components3 = detectScenarioComponents(uiScenario);
  assertEquals(components3.size, 1, 'UI scenario has 1 component');
  assert(components3.has('UI'), 'detects UI component');

  // Test 4: Full-stack scenario (API + UI)
  console.log('\nTest Suite: detectScenarioComponents() - Full-stack scenario');
  const fullStackScenario = path.join(tmpDir, 'fullstack-scenario');
  fs.mkdirSync(fullStackScenario);
  fs.mkdirSync(path.join(fullStackScenario, 'api'));
  fs.mkdirSync(path.join(fullStackScenario, 'ui'));
  fs.writeFileSync(path.join(fullStackScenario, 'api', 'main.go'), '// Go code');
  fs.writeFileSync(path.join(fullStackScenario, 'ui', 'package.json'), '{}');

  const components4 = detectScenarioComponents(fullStackScenario);
  assertEquals(components4.size, 2, 'full-stack scenario has 2 components');
  assert(components4.has('API'), 'detects API component');
  assert(components4.has('UI'), 'detects UI component');

  // Test 5: getApplicableLayers() - Empty components
  console.log('\nTest Suite: getApplicableLayers() - Applicable layers');
  const layers1 = getApplicableLayers(new Set());
  assertEquals(layers1.size, 1, 'empty components have 1 layer (E2E)');
  assert(layers1.has('E2E'), 'always includes E2E layer');

  // Test 6: getApplicableLayers() - API component
  const layers2 = getApplicableLayers(new Set(['API']));
  assertEquals(layers2.size, 2, 'API component adds 2 layers');
  assert(layers2.has('E2E'), 'includes E2E layer');
  assert(layers2.has('API'), 'includes API layer');

  // Test 7: getApplicableLayers() - UI component
  const layers3 = getApplicableLayers(new Set(['UI']));
  assertEquals(layers3.size, 2, 'UI component adds 2 layers');
  assert(layers3.has('E2E'), 'includes E2E layer');
  assert(layers3.has('UI'), 'includes UI layer');

  // Test 8: getApplicableLayers() - Full-stack
  const layers4 = getApplicableLayers(new Set(['API', 'UI']));
  assertEquals(layers4.size, 3, 'full-stack has all 3 layers');
  assert(layers4.has('E2E'), 'includes E2E layer');
  assert(layers4.has('API'), 'includes API layer');
  assert(layers4.has('UI'), 'includes UI layer');

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
