'use strict';

/**
 * Test quality analyzer - Analyzes test file quality using basic heuristics
 * @module scenarios/lib/validators/test-quality-analyzer
 */

const fs = require('node:fs');
const path = require('node:path');

/**
 * Analyze test file quality using basic heuristics
 * @param {string} testFilePath - Relative path to test file
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} Quality analysis results
 */
function analyzeTestFileQuality(testFilePath, scenarioRoot) {
  const fullPath = path.join(scenarioRoot, testFilePath);

  if (!fs.existsSync(fullPath)) {
    return {
      exists: false,
      is_meaningful: false,
      reason: 'file_not_found'
    };
  }

  const content = fs.readFileSync(fullPath, 'utf8');
  const lines = content.split('\n');
  const nonEmptyLines = lines.filter(l => l.trim().length > 0);

  // Remove comment-only lines
  const nonCommentLines = nonEmptyLines.filter(l => {
    const t = l.trim();
    return !t.startsWith('//') &&
           !t.startsWith('/*') &&
           !t.startsWith('*') &&
           !t.startsWith('#');  // Python/shell comments
  });

  // Count test functions
  const testFunctionMatches = content.match(/func Test|@test|test\(|it\(|describe\(|def test_/gi);
  const testFunctionCount = testFunctionMatches ? testFunctionMatches.length : 0;

  // Count assertions
  const assertionMatches = content.match(/assert|expect|require|Should|Equal|Contains|Error|True|False|toBe|toHaveBeenCalled/gi);
  const assertionCount = assertionMatches ? assertionMatches.length : 0;
  const assertionDensity = nonCommentLines.length > 0 ? assertionCount / nonCommentLines.length : 0;

  // Quality heuristics (ENHANCED)
  const hasMinimumCode = nonCommentLines.length >= 20;  // At least 20 LOC
  const hasAssertions = assertionCount > 0;
  const hasMultipleTestFunctions = testFunctionCount >= 3;  // Require ≥3 test functions
  const hasGoodAssertionDensity = assertionDensity >= 0.1;  // ≥1 assertion per 10 LOC
  const hasTestFunctions = testFunctionCount > 0;

  // Calculate quality score (0-5, need ≥4 to pass)
  const qualityScore =
    (hasMinimumCode ? 1 : 0) +
    (hasAssertions ? 1 : 0) +
    (hasTestFunctions ? 1 : 0) +
    (hasMultipleTestFunctions ? 1 : 0) +
    (hasGoodAssertionDensity ? 1 : 0);

  return {
    exists: true,
    loc: nonCommentLines.length,
    has_assertions: hasAssertions,
    has_test_functions: hasTestFunctions,
    test_function_count: testFunctionCount,
    assertion_count: assertionCount,
    assertion_density: assertionDensity,
    is_meaningful: qualityScore >= 4,  // Need 4+ indicators (was 2)
    quality_score: qualityScore,
    reason: qualityScore < 4 ? 'insufficient_quality' : 'ok'
  };
}

/**
 * Analyze e2e playbook quality
 * @param {string} playbookPath - Relative path to playbook file
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} Quality analysis results
 */
function analyzePlaybookQuality(playbookPath, scenarioRoot) {
  const fullPath = path.join(scenarioRoot, playbookPath);

  if (!fs.existsSync(fullPath)) {
    return { exists: false, is_meaningful: false, reason: 'file_not_found' };
  }

  try {
    const content = fs.readFileSync(fullPath, 'utf8');
    const parsed = JSON.parse(content);
    const fileSize = Buffer.byteLength(content, 'utf8');

    // Support two playbook formats:
    // 1. Old format: {"steps": [...]}
    // 2. BAS format: {"nodes": [...], "edges": [...]}

    // Check for BAS format (nodes + edges)
    const hasNodes = Array.isArray(parsed.nodes) && parsed.nodes.length > 0;
    const nodeCount = parsed.nodes ? parsed.nodes.length : 0;
    const hasNodeActions = parsed.nodes && parsed.nodes.some(n => n.type || n.action);

    // Check for old format (steps)
    const hasSteps = Array.isArray(parsed.steps) && parsed.steps.length > 0;
    const stepCount = parsed.steps ? parsed.steps.length : 0;
    const hasStepActions = parsed.steps && parsed.steps.some(s => s.action || s.type);

    // Accept either format
    const isValid = (hasNodes && hasNodeActions && fileSize >= 100) ||
                    (hasSteps && hasStepActions && fileSize >= 100);

    return {
      exists: true,
      step_count: stepCount || nodeCount,
      has_actions: hasStepActions || hasNodeActions,
      file_size: fileSize,
      is_meaningful: isValid,
      reason: isValid ? 'ok' : (!hasSteps && !hasNodes) ? 'no_steps_or_nodes' : 'no_actions'
    };
  } catch (error) {
    return { exists: true, is_meaningful: false, reason: 'parse_error' };
  }
}

module.exports = {
  analyzeTestFileQuality,
  analyzePlaybookQuality
};
