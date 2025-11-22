#!/usr/bin/env node
'use strict';

/**
 * Test to verify refactored completeness-data.js produces same output as original
 */

const path = require('node:path');

// Load both versions
const refactored = require('./completeness-data');

// We need to temporarily use the backup to compare
const fs = require('node:fs');
const backupPath = path.join(__dirname, 'completeness-data.js.backup');
const backupContent = fs.readFileSync(backupPath, 'utf8');

// Create a temporary module to load the backup
const Module = require('module');
const originalModule = new Module();
originalModule._compile(backupContent, backupPath);
const original = originalModule.exports;

// Test scenario
const testScenario = path.join(__dirname, '../../../scenarios/picker-wheel');

console.log('Testing refactored completeness-data.js...\n');
console.log(`Test scenario: ${testScenario}\n`);

// Collect metrics with both versions
console.log('Collecting metrics with original version...');
const originalMetrics = original.collectMetrics(testScenario);

console.log('Collecting metrics with refactored version...');
const refactoredMetrics = refactored.collectMetrics(testScenario);

// Compare results
console.log('\n=== Comparison ===\n');

function deepEqual(obj1, obj2, path = '') {
  if (obj1 === obj2) return true;

  if (typeof obj1 !== 'object' || typeof obj2 !== 'object' || obj1 === null || obj2 === null) {
    console.log(`❌ Mismatch at ${path}: ${JSON.stringify(obj1)} !== ${JSON.stringify(obj2)}`);
    return false;
  }

  const keys1 = Object.keys(obj1);
  const keys2 = Object.keys(obj2);

  if (keys1.length !== keys2.length) {
    console.log(`❌ Different number of keys at ${path}: ${keys1.length} !== ${keys2.length}`);
    return false;
  }

  for (const key of keys1) {
    if (!keys2.includes(key)) {
      console.log(`❌ Key missing at ${path}.${key}`);
      return false;
    }

    if (!deepEqual(obj1[key], obj2[key], `${path}.${key}`)) {
      return false;
    }
  }

  return true;
}

if (deepEqual(originalMetrics, refactoredMetrics)) {
  console.log('✅ SUCCESS: Refactored version produces identical output!\n');
  console.log('Sample output:');
  console.log(JSON.stringify(refactoredMetrics, null, 2));
  process.exit(0);
} else {
  console.log('\n❌ FAILURE: Outputs differ\n');
  console.log('Original:');
  console.log(JSON.stringify(originalMetrics, null, 2));
  console.log('\nRefactored:');
  console.log(JSON.stringify(refactoredMetrics, null, 2));
  process.exit(1);
}
