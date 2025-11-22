#!/usr/bin/env node
'use strict';

/**
 * Unit tests for completeness scoring system
 * Run with: node completeness.test.js
 */

const {
  loadThresholds,
  getCategoryThresholds,
  calculateQualityScore,
  calculateCoverageScore,
  calculateQuantityScore,
  calculateCompletenessScore,
  classifyScore,
  checkStaleness,
  generateRecommendations,
  getMaxDepth
} = require('./completeness');

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

function assertCloseTo(actual, expected, tolerance, message) {
  testsRun++;
  if (Math.abs(actual - expected) <= tolerance) {
    testsPassed++;
    console.log(`  ✓ ${message}`);
  } else {
    testsFailed++;
    console.error(`  ✗ ${message} (expected ~${expected}, got ${actual})`);
  }
}

console.log('\n=== Completeness Scoring Unit Tests ===\n');

// Test 1: loadThresholds
console.log('Test Suite: loadThresholds()');
try {
  const thresholds = loadThresholds();
  assert(thresholds.version === '1.0.0', 'loads version');
  assert(thresholds.default_category === 'utility', 'has default category');
  assert(thresholds.categories.utility !== undefined, 'has utility category');
  assert(thresholds.categories['business-application'] !== undefined, 'has business-application category');
} catch (error) {
  console.error(`  ✗ loadThresholds failed: ${error.message}`);
  testsFailed++;
}

// Test 2: getCategoryThresholds
console.log('\nTest Suite: getCategoryThresholds()');
const thresholds = loadThresholds();
const utilityThresholds = getCategoryThresholds('utility', thresholds);
assert(utilityThresholds.requirements.good === 15, 'utility requirements.good === 15');
assert(utilityThresholds.targets.good === 12, 'utility targets.good === 12');
assert(utilityThresholds.tests.good === 25, 'utility tests.good === 25');

const platformThresholds = getCategoryThresholds('platform', thresholds);
assert(platformThresholds.requirements.good === 50, 'platform requirements.good === 50');

const unknownThresholds = getCategoryThresholds('unknown', thresholds);
assertEquals(unknownThresholds.requirements.good, 15, 'unknown category defaults to utility');

// Test 3: calculateQualityScore
console.log('\nTest Suite: calculateQualityScore()');
const perfectQuality = calculateQualityScore({
  requirements: { total: 10, passing: 10 },
  targets: { total: 10, passing: 10 },
  tests: { total: 10, passing: 10 }
});
assertEquals(perfectQuality.score, 70, 'perfect quality scores 70/70');
assertEquals(perfectQuality.requirement_pass_rate.points, 30, 'perfect reqs score 30');
assertEquals(perfectQuality.target_pass_rate.points, 20, 'perfect targets score 20');
assertEquals(perfectQuality.test_pass_rate.points, 20, 'perfect tests score 20');

const halfQuality = calculateQualityScore({
  requirements: { total: 10, passing: 5 },
  targets: { total: 10, passing: 5 },
  tests: { total: 10, passing: 5 }
});
assertEquals(halfQuality.score, 35, '50% quality scores 35/70');

const zeroQuality = calculateQualityScore({
  requirements: { total: 0, passing: 0 },
  targets: { total: 0, passing: 0 },
  tests: { total: 0, passing: 0 }
});
assertEquals(zeroQuality.score, 0, 'empty scenario scores 0/70');

// Test 4: getMaxDepth
console.log('\nTest Suite: getMaxDepth()');
const flatReq = { id: 'REQ-001', children: [] };
assertEquals(getMaxDepth(flatReq), 1, 'flat requirement has depth 1');

const nested2 = { id: 'REQ-001', children: [{ id: 'REQ-002', children: [] }] };
assertEquals(getMaxDepth(nested2), 2, 'nested 2 levels has depth 2');

const nested3 = {
  id: 'REQ-001',
  children: [
    {
      id: 'REQ-002',
      children: [
        { id: 'REQ-003', children: [] }
      ]
    }
  ]
};
assertEquals(getMaxDepth(nested3), 3, 'nested 3 levels has depth 3');

// Test 5: calculateCoverageScore
console.log('\nTest Suite: calculateCoverageScore()');
const perfectCoverage = calculateCoverageScore(
  { requirements: { total: 10 }, tests: { total: 20 } },
  [nested3, nested3, nested3] // 3 requirements with avg depth 3
);
assertEquals(perfectCoverage.score, 20, 'perfect coverage scores 20/20');
assertCloseTo(perfectCoverage.test_coverage_ratio.ratio, 2.0, 0.01, '2:1 test ratio');
assertEquals(perfectCoverage.test_coverage_ratio.points, 10, 'perfect test ratio scores 10');
assertEquals(perfectCoverage.depth_score.points, 10, 'depth 3 scores 10');

const shallowCoverage = calculateCoverageScore(
  { requirements: { total: 10 }, tests: { total: 10 } },
  [flatReq, flatReq, flatReq]
);
assert(shallowCoverage.score < 12, 'shallow coverage scores low (< 12/20)');

// Test 6: calculateQuantityScore
console.log('\nTest Suite: calculateQuantityScore()');
const perfectQuantity = calculateQuantityScore(
  { requirements: { total: 50 }, targets: { total: 30 }, tests: { total: 80 } },
  platformThresholds
);
assertEquals(perfectQuantity.score, 10, 'exceeding thresholds scores 10/10');

const okQuantity = calculateQuantityScore(
  { requirements: { total: 30 }, targets: { total: 20 }, tests: { total: 50 } },
  platformThresholds
);
assert(okQuantity.score < 10, 'ok quantities score less than 10');
assert(okQuantity.score > 5, 'ok quantities score more than 5');

const lowQuantity = calculateQuantityScore(
  { requirements: { total: 5 }, targets: { total: 3 }, tests: { total: 10 } },
  platformThresholds
);
assert(lowQuantity.score < 5, 'low quantities score less than 5');

// Test 7: calculateCompletenessScore
console.log('\nTest Suite: calculateCompletenessScore()');
const perfectScore = calculateCompletenessScore(
  { requirements: { total: 50, passing: 50 }, targets: { total: 30, passing: 30 }, tests: { total: 100, passing: 100 } },
  [nested3, nested3, nested3],
  platformThresholds
);
assertEquals(perfectScore.score, 100, 'perfect scenario scores 100/100');

const emptyScore = calculateCompletenessScore(
  { requirements: { total: 0, passing: 0 }, targets: { total: 0, passing: 0 }, tests: { total: 0, passing: 0 } },
  [],
  platformThresholds
);
assertEquals(emptyScore.score, 0, 'empty scenario scores 0/100');

// Test 8: classifyScore
console.log('\nTest Suite: classifyScore()');
assertEquals(classifyScore(100), 'production_ready', 'score 100 → production_ready');
assertEquals(classifyScore(96), 'production_ready', 'score 96 → production_ready');
assertEquals(classifyScore(95), 'nearly_ready', 'score 95 → nearly_ready');
assertEquals(classifyScore(81), 'nearly_ready', 'score 81 → nearly_ready');
assertEquals(classifyScore(80), 'mostly_complete', 'score 80 → mostly_complete');
assertEquals(classifyScore(61), 'mostly_complete', 'score 61 → mostly_complete');
assertEquals(classifyScore(60), 'functional_incomplete', 'score 60 → functional_incomplete');
assertEquals(classifyScore(41), 'functional_incomplete', 'score 41 → functional_incomplete');
assertEquals(classifyScore(40), 'foundation_laid', 'score 40 → foundation_laid');
assertEquals(classifyScore(21), 'foundation_laid', 'score 21 → foundation_laid');
assertEquals(classifyScore(20), 'early_stage', 'score 20 → early_stage');
assertEquals(classifyScore(0), 'early_stage', 'score 0 → early_stage');

// Test 9: checkStaleness
console.log('\nTest Suite: checkStaleness()');
const recentTest = new Date(Date.now() - 1000 * 60 * 60).toISOString(); // 1 hour ago
const recentCheck = checkStaleness(recentTest);
assertEquals(recentCheck.warning, false, 'recent test (1h) is not stale');

const staleTest = new Date(Date.now() - 1000 * 60 * 60 * 72).toISOString(); // 72 hours ago
const staleCheck = checkStaleness(staleTest);
assertEquals(staleCheck.warning, true, 'old test (72h) is stale');
assert(staleCheck.hoursStale >= 72, 'staleness hours calculated correctly');

const noTest = checkStaleness(null);
assertEquals(noTest.warning, true, 'missing test results triggers warning');

// Test 10: generateRecommendations
console.log('\nTest Suite: generateRecommendations()');
const needsTests = calculateCompletenessScore(
  { requirements: { total: 10, passing: 10 }, targets: { total: 10, passing: 10 }, tests: { total: 5, passing: 3 } },
  [flatReq],
  utilityThresholds
);
const testRecs = generateRecommendations(needsTests, utilityThresholds);
assert(testRecs.some(r => r.includes('test pass rate')), 'recommends improving test pass rate');
assert(testRecs.some(r => r.includes('tests to reach')), 'recommends adding more tests');

const needsReqs = calculateCompletenessScore(
  { requirements: { total: 5, passing: 3 }, targets: { total: 5, passing: 3 }, tests: { total: 10, passing: 8 } },
  [flatReq],
  utilityThresholds
);
const reqRecs = generateRecommendations(needsReqs, utilityThresholds);
assert(reqRecs.some(r => r.includes('requirement pass rate')), 'recommends improving requirement pass rate');

// Summary
console.log('\n=== Test Summary ===');
console.log(`Total tests: ${testsRun}`);
console.log(`Passed: ${testsPassed} (${Math.round(testsPassed / testsRun * 100)}%)`);
console.log(`Failed: ${testsFailed}`);

if (testsFailed === 0) {
  console.log('\n✅ All tests passed!\n');
  process.exit(0);
} else {
  console.log(`\n❌ ${testsFailed} tests failed\n`);
  process.exit(1);
}
