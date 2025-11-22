'use strict';

/**
 * Validation quality issue detection
 * Identifies testing anti-patterns and validation quality issues
 * @module scenarios/lib/validation-quality
 */

const componentDetector = require('./validators/component-detector');
const layerDetector = require('./validators/layer-detector');
const duplicateDetector = require('./validators/duplicate-detector');
const targetGroupingValidator = require('./validators/target-grouping-validator');
const testQualityAnalyzer = require('./validators/test-quality-analyzer');

/**
 * Detect validation quality issues in a scenario
 * @param {object} metrics - Completeness metrics
 * @param {array} requirements - Requirements array
 * @param {array} targets - Operational targets array
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} Detection results with issues and penalties
 */
function detectValidationQualityIssues(metrics, requirements, targets, scenarioRoot) {
  const issues = [];
  const patterns = {};
  let totalPenalty = 0;

  // Issue 1: Insufficient test coverage (suspicious 1:1 ratio)
  const testReqRatio = metrics.tests.total / Math.max(metrics.requirements.total, 1);
  if (Math.abs(testReqRatio - 1.0) < 0.1) {  // Within 10% of 1:1
    const penalty = 5;
    totalPenalty += penalty;

    patterns.insufficient_test_coverage = {
      severity: 'medium',
      detected: true,
      penalty: penalty,
      ratio: testReqRatio,
      message: `Suspicious 1:1 test-to-requirement ratio (${metrics.tests.total} tests for ${metrics.requirements.total} requirements). Expected: 1.5-2.0x ratio with diverse test sources.`,
      recommendation: 'Add multiple test types per requirement (API tests, UI tests, e2e automation)',
      why_it_matters: 'Each requirement should be validated by multiple test types (unit, integration, e2e) to ensure comprehensive coverage and catch different types of issues.',
      description: 'Perfect 1:1 test-to-requirement ratios often indicate that requirements are being validated by single monolithic tests rather than comprehensive, layered testing.'
    };
    issues.push(patterns.insufficient_test_coverage);
  }

  // Issue 2: Invalid test location (unsupported test/ directories)
  const unsupportedRefCount = requirements.filter(r =>
    (r.validation || []).some(v => {
      const ref = (v.ref || '').toLowerCase();
      // Reject anything in test/ that's not test/playbooks/*.{json,yaml}
      return ref.startsWith('test/') &&
             !(ref.startsWith('test/playbooks/') && (ref.endsWith('.json') || ref.endsWith('.yaml')));
    })
  ).length;

  if (unsupportedRefCount > 0) {
    const ratio = unsupportedRefCount / Math.max(metrics.requirements.total, 1);
    const penalty = Math.round(ratio * 25);
    totalPenalty += penalty;

    // Build component-aware recommendation
    const scenarioComponents = componentDetector.detectScenarioComponents(scenarioRoot);
    const validSources = [];
    if (scenarioComponents.has('API')) {
      validSources.push('api/**/*_test.go (API unit tests)');
    }
    if (scenarioComponents.has('UI')) {
      validSources.push('ui/src/**/*.test.tsx (UI unit tests)');
    }
    validSources.push('test/playbooks/**/*.{json,yaml} (e2e automation)');

    patterns.invalid_test_location = {
      severity: ratio > 0.5 ? 'high' : 'medium',
      detected: true,
      penalty: penalty,
      count: unsupportedRefCount,
      total: metrics.requirements.total,
      ratio: ratio,
      message: `${unsupportedRefCount}/${metrics.requirements.total} requirements reference unsupported test/ directories. Valid sources for this scenario:\n${validSources.map(s => `  • ${s}`).join('\n')}`,
      recommendation: 'Move validation refs to supported test sources listed above',
      why_it_matters: 'Requirements must be validated by actual tests (API unit tests, UI component tests, or e2e automation playbooks). Infrastructure test files like test/phases/ or CLI wrapper tests in test/cli/ are not acceptable validation sources.',
      description: 'Test files must live in recognized test locations where they can be executed as part of the test suite. Files in unsupported directories may not run during testing or may be infrastructure scripts rather than actual tests.'
    };
    issues.push(patterns.invalid_test_location);
  }

  // Issue 3: Monolithic test files (duplicate test refs)
  const refAnalysis = duplicateDetector.analyzeTestRefUsage(requirements);
  if (refAnalysis.violations.length > 0) {
    const penalty = Math.min(refAnalysis.violations.length * 2, 15);
    totalPenalty += penalty;

    patterns.monolithic_test_files = {
      severity: refAnalysis.violations.some(v => v.severity === 'high') ? 'high' : 'medium',
      detected: true,
      penalty: penalty,
      violations: refAnalysis.violations.length,
      worst_offender: refAnalysis.violations[0],
      message: `${refAnalysis.violations.length} test files validate ≥4 requirements each. Expected: unique tests per requirement.`,
      recommendation: 'Break monolithic test files into focused tests for each requirement',
      why_it_matters: 'When a single test file validates many requirements, it becomes difficult to isolate failures, maintain test clarity, and ensure each requirement is properly validated. Focused tests provide better diagnostics and are easier to maintain.',
      description: 'Monolithic test files that validate multiple requirements are a testing anti-pattern. Each requirement should have dedicated, focused tests that clearly validate its specific functionality.'
    };
    issues.push(patterns.monolithic_test_files);
  }

  // Issue 4: Ungrouped operational targets (excessive 1:1 target-to-requirement mappings)
  const groupingAnalysis = targetGroupingValidator.analyzeTargetGrouping(targets, requirements);
  if (groupingAnalysis.violations.length > 0) {
    const penalty = Math.round(groupingAnalysis.one_to_one_ratio * 10);
    totalPenalty += penalty;

    patterns.ungrouped_operational_targets = {
      severity: groupingAnalysis.one_to_one_ratio > 0.5 ? 'high' : 'medium',
      detected: true,
      penalty: penalty,
      ratio: groupingAnalysis.one_to_one_ratio,
      count: groupingAnalysis.one_to_one_count,
      message: `${Math.round(groupingAnalysis.one_to_one_ratio * 100)}% of operational targets have 1:1 requirement mapping (max ${Math.round(groupingAnalysis.acceptable_ratio * 100)}% recommended)`,
      recommendation: 'Group related requirements under shared operational targets from the PRD',
      why_it_matters: 'Operational targets from the PRD typically represent high-level business capabilities that encompass multiple related requirements. A 1:1 mapping suggests requirements were auto-generated from targets rather than properly decomposed.',
      description: 'Operational targets should group related requirements under cohesive business objectives. Excessive 1:1 mappings indicate lack of proper requirement decomposition and PRD understanding.'
    };
    issues.push(patterns.ungrouped_operational_targets);
  }

  // Issue 5: Insufficient validation layers (missing validation diversity)
  const scenarioComponents = componentDetector.detectScenarioComponents(scenarioRoot);
  const applicableLayers = componentDetector.getApplicableLayers(scenarioComponents);

  const diversityIssues = requirements.filter(req => {
    const criticality = layerDetector.deriveRequirementCriticality(req);
    const layerAnalysis = layerDetector.detectValidationLayers(req, scenarioRoot);

    // Filter to only applicable AUTOMATED layers based on scenario components
    const applicable = new Set();
    if (scenarioComponents.has('API') && layerAnalysis.automated.has('API')) {
      applicable.add('API');
    }
    if (scenarioComponents.has('UI') && layerAnalysis.automated.has('UI')) {
      applicable.add('UI');
    }
    // E2E is always applicable (AUTOMATED layer)
    if (layerAnalysis.automated.has('E2E')) {
      applicable.add('E2E');
    }

    const minLayers = (criticality === 'P0' || criticality === 'P1') ? 2 : 1;
    return applicable.size < minLayers;
  });

  if (diversityIssues.length > 0) {
    const penalty = Math.round((diversityIssues.length / Math.max(requirements.length, 1)) * 20);
    totalPenalty += penalty;

    patterns.insufficient_validation_layers = {
      severity: 'high',
      detected: true,
      penalty: penalty,
      count: diversityIssues.length,
      total: requirements.length,
      message: `${diversityIssues.length} critical requirements (P0/P1) lack multi-layer AUTOMATED validation`,
      recommendation: 'Add automated validations across API, UI, and e2e layers for critical requirements (manual validations don\'t count toward diversity)',
      why_it_matters: 'Critical requirements (P0/P1) need validation at multiple layers to catch different types of bugs. API tests validate logic, UI tests validate interface, and e2e tests validate end-to-end flows. Single-layer validation misses integration issues.',
      description: 'Validation diversity ensures critical features are tested comprehensively. Requirements should have automated tests at the API layer, UI layer, and e2e layer when applicable. Manual validations do not count toward diversity.'
    };
    issues.push(patterns.insufficient_validation_layers);
  }

  // Issue 6: Superficial test implementation (low-quality test files)
  const lowQualityTests = [];
  requirements.forEach(req => {
    (req.validation || []).forEach(v => {
      if (v.type === 'test' && v.ref) {
        const quality = testQualityAnalyzer.analyzeTestFileQuality(v.ref, scenarioRoot);
        if (quality.exists && !quality.is_meaningful) {
          lowQualityTests.push({
            requirement: req.id,
            ref: v.ref,
            quality_score: quality.quality_score,
            reason: quality.reason
          });
        }
      }
    });
  });

  if (lowQualityTests.length > 0) {
    const penalty = Math.min(lowQualityTests.length, 10);
    totalPenalty += penalty;

    patterns.superficial_test_implementation = {
      severity: 'medium',
      detected: true,
      penalty: penalty,
      count: lowQualityTests.length,
      files: lowQualityTests,
      message: `${lowQualityTests.length} test file(s) appear superficial (< 20 LOC, missing assertions, or no test functions)`,
      recommendation: 'Add meaningful test logic with assertions and edge case coverage',
      why_it_matters: 'Tests must actually verify behavior through assertions and cover edge cases. Superficial tests that merely exist without meaningful validation provide false confidence and won\'t catch real bugs.',
      description: 'Test quality matters as much as test quantity. Tests should have sufficient logic (>20 LOC typically), include assertions that verify expected behavior, and cover both happy paths and edge cases.'
    };
    issues.push(patterns.superficial_test_implementation);
  }

  // Issue 7: Missing test automation (excessive manual validation usage)
  const totalValidations = requirements.reduce((sum, r) =>
    sum + (r.validation || []).length, 0
  );
  const manualValidations = requirements.reduce((sum, r) =>
    sum + (r.validation || []).filter(v => v.type === 'manual').length, 0
  );

  // Count manual validations for COMPLETE requirements (suspicious)
  const completeWithManual = requirements.filter(req => {
    const status = (req.status || '').toLowerCase();
    const isComplete = status === 'complete' || status === 'validated' || status === 'implemented';
    const hasManual = (req.validation || []).some(v => v.type === 'manual');
    const hasAutomated = (req.validation || []).some(v => v.type !== 'manual');

    // Only flag if COMPLETE with manual validation but NO automated tests
    return isComplete && hasManual && !hasAutomated;
  });

  const manualRatio = manualValidations / Math.max(totalValidations, 1);

  // Flag if: (a) >10% manual overall OR (b) ≥5 complete requirements with ONLY manual validation
  if (manualRatio > 0.10 || completeWithManual.length >= 5) {
    const penalty = Math.round(manualRatio * 10) + Math.min(completeWithManual.length, 5);
    totalPenalty += penalty;

    patterns.missing_test_automation = {
      severity: completeWithManual.length >= 10 ? 'high' : 'medium',
      detected: true,
      penalty: penalty,
      ratio: manualRatio,
      count: manualValidations,
      complete_with_manual: completeWithManual.length,
      message: manualRatio > 0.10
        ? `${Math.round(manualRatio * 100)}% of validations are manual (max 10% recommended). Manual validations are intended as temporary measures before automated tests are implemented.`
        : `${completeWithManual.length} requirements marked complete with ONLY manual validations. Automated tests should replace manual validations for completed requirements.`,
      recommendation: 'Replace manual validations with automated tests (API tests, UI tests, or e2e automation). Manual validations should be temporary measures for pending/in_progress requirements.',
      why_it_matters: 'Manual validations are not repeatable, not run in CI/CD, and provide no regression protection. Completed requirements must have automated tests to ensure they stay working as the codebase evolves.',
      description: 'Manual validations are acceptable for in-progress work, but completed requirements must be validated by automated tests. Manual testing doesn\'t scale and can\'t catch regressions.'
    };
    issues.push(patterns.missing_test_automation);
  }

  return {
    has_issues: issues.length > 0,
    issue_count: issues.length,
    issues,
    patterns,
    total_penalty: totalPenalty,
    overall_severity: issues.some(i => i.severity === 'high') ? 'high' : 'medium'
  };
}

module.exports = {
  detectValidationQualityIssues
};
