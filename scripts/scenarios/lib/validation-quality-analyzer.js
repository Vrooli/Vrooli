'use strict';

/**
 * Validation Quality Analyzer
 *
 * Identifies testing anti-patterns and validation quality issues in scenario test suites.
 * This module analyzes requirements, tests, and operational targets to detect potential
 * "gaming" behavior where tests are created to inflate metrics rather than provide genuine
 * validation.
 *
 * Penalties Applied (see completeness-config.json for values):
 * - Invalid test locations: Up to -25pts (ratio-based, 25pt multiplier)
 * - Monolithic test files: Up to -15pts (2pts per violation)
 * - Ungrouped targets: Up to -10pts (10pt multiplier on ratio)
 * - Insufficient layers: Up to -20pts (20pt multiplier on ratio)
 * - Superficial tests: Up to -10pts (1pt per low-quality file)
 * - Missing automation: Up to -15pts (ratio-based + count-based)
 * - Suspicious 1:1 ratio: -5pts (fixed penalty)
 *
 * @module scenarios/lib/validation-quality-analyzer
 */

const componentDetector = require('./validators/component-detector');
const layerDetector = require('./validators/layer-detector');
const duplicateDetector = require('./validators/duplicate-detector');
const targetGroupingValidator = require('./validators/target-grouping-validator');
const testQualityAnalyzer = require('./validators/test-quality-analyzer');
const { getPenaltyConfig, getValidationConfig } = require('./utils/config-loader');

/**
 * Detect validation quality issues in a scenario
 *
 * Analyzes a scenario's test suite for anti-patterns and validation quality issues.
 * Returns actionable recommendations and applies penalties based on severity.
 *
 * Detection heuristics:
 * 1. **Insufficient test coverage**: Detects suspicious 1:1 test-to-requirement ratios
 * 2. **Invalid test locations**: Flags refs to unsupported test/ directories
 * 3. **Monolithic test files**: Identifies tests validating ≥4 requirements
 * 4. **Ungrouped operational targets**: Detects excessive 1:1 target-to-requirement mappings
 * 5. **Insufficient validation layers**: Checks for multi-layer validation on P0/P1 requirements
 * 6. **Superficial test implementation**: Analyzes test files for meaningful content
 * 7. **Missing test automation**: Flags excessive manual validation usage
 *
 * @param {object} metrics - Completeness metrics from collectMetrics()
 * @param {array} requirements - Requirements array with validation refs
 * @param {array} targets - Operational targets array
 * @param {string} scenarioRoot - Absolute path to scenario directory
 * @returns {object} Analysis with issues[], patterns{}, and total_penalty
 *
 * @example
 * const analysis = detectValidationQualityIssues(metrics, reqs, targets, root);
 * console.log(`Found ${analysis.issue_count} issues (-${analysis.total_penalty}pts)`);
 *
 * @example Output structure
 * {
 *   has_issues: true,
 *   issue_count: 3,
 *   issues: [
 *     {
 *       severity: 'high',
 *       detected: true,
 *       penalty: 25,
 *       message: '37/37 requirements reference unsupported test/ directories...',
 *       recommendation: 'Move validation refs to supported test sources...',
 *       why_it_matters: 'Requirements must be validated by actual tests...',
 *       description: 'Test files must live in recognized test locations...'
 *     }
 *   ],
 *   patterns: {
 *     invalid_test_location: { ... },
 *     monolithic_test_files: { ... }
 *   },
 *   total_penalty: 72,
 *   overall_severity: 'high'
 * }
 */
function detectValidationQualityIssues(metrics, requirements, targets, scenarioRoot) {
  const issues = [];
  const patterns = {};
  let totalPenalty = 0;

  // Load configuration
  const validationConfig = getValidationConfig();

  // ========================================================================
  // Issue 1: Insufficient test coverage (suspicious 1:1 ratio)
  // ========================================================================
  const penaltyConfig1 = getPenaltyConfig('insufficient_test_coverage');
  const testReqRatio = metrics.tests.total / Math.max(metrics.requirements.total, 1);
  const ratioTolerance = validationConfig.acceptable_ratios?.test_to_requirement_tolerance || 0.10;

  if (Math.abs(testReqRatio - 1.0) < ratioTolerance) {
    const penalty = penaltyConfig1.base_penalty;
    totalPenalty += penalty;

    patterns.insufficient_test_coverage = {
      severity: penaltyConfig1.severity,
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

  // ========================================================================
  // Issue 2: Invalid test location (unsupported test/ directories)
  // ========================================================================
  const penaltyConfig2 = getPenaltyConfig('invalid_test_location');
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
    const penalty = Math.min(
      Math.round(ratio * penaltyConfig2.multiplier),
      penaltyConfig2.max_penalty
    );
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

    const severity = ratio > (penaltyConfig2.severity_threshold || 0.5) ? 'high' : 'medium';

    patterns.invalid_test_location = {
      severity,
      detected: true,
      penalty: penalty,
      count: unsupportedRefCount,
      total: metrics.requirements.total,
      ratio: ratio,
      valid_sources: validSources,
      message: `${unsupportedRefCount}/${metrics.requirements.total} requirements reference unsupported test/ directories`,
      recommendation: 'Move validation refs to supported test sources listed above',
      why_it_matters: 'Requirements must be validated by actual tests (API unit tests, UI component tests, or e2e automation playbooks). Infrastructure test files like test/phases/ or CLI wrapper tests in test/cli/ are not acceptable validation sources.',
      description: 'Test files must live in recognized test locations where they can be executed as part of the test suite. Files in unsupported directories may not run during testing or may be infrastructure scripts rather than actual tests.'
    };
    issues.push(patterns.invalid_test_location);
  }

  // ========================================================================
  // Issue 3: Monolithic test files (duplicate test refs)
  // ========================================================================
  const penaltyConfig3 = getPenaltyConfig('monolithic_test_files');
  const refAnalysis = duplicateDetector.analyzeTestRefUsage(requirements);

  if (refAnalysis.violations.length > 0) {
    const penalty = Math.min(
      refAnalysis.violations.length * penaltyConfig3.per_violation,
      penaltyConfig3.max_penalty
    );
    totalPenalty += penalty;

    const hasSevereViolations = refAnalysis.violations.some(v => v.severity === 'high');
    const severity = hasSevereViolations ? 'high' : 'medium';

    patterns.monolithic_test_files = {
      severity,
      detected: true,
      penalty: penalty,
      violations: refAnalysis.violations.length,
      worst_offender: refAnalysis.violations[0],
      message: `${refAnalysis.violations.length} test files validate ≥4 requirements each`,
      recommendation: 'Break monolithic test files into focused tests for each requirement',
      why_it_matters: 'Monolithic test files create several problems: (1) Hard to trace which requirement failed, (2) Reduces test isolation and makes debugging difficult, (3) Encourages gaming by linking many requirements to one superficial test, (4) Poor code organization and maintainability. Each requirement deserves focused validation.',
      description: 'Gaming Prevention: This penalty detects when test files validate many requirements (≥4), which often indicates a single broad test being reused to check off multiple requirements without genuine validation. While some grouping is natural (e.g., CRUD operations), excessive sharing suggests superficial testing. The threshold of 4 requirements encourages smaller, more focused test files and better organization.'
    };
    issues.push(patterns.monolithic_test_files);
  }

  // ========================================================================
  // Issue 4: Ungrouped operational targets (excessive 1:1 mappings)
  // ========================================================================
  const penaltyConfig4 = getPenaltyConfig('ungrouped_operational_targets');
  const groupingAnalysis = targetGroupingValidator.analyzeTargetGrouping(targets, requirements);

  if (groupingAnalysis.violations.length > 0) {
    const penalty = Math.min(
      Math.round(groupingAnalysis.one_to_one_ratio * penaltyConfig4.multiplier),
      penaltyConfig4.max_penalty
    );
    totalPenalty += penalty;

    const severity = groupingAnalysis.one_to_one_ratio > (penaltyConfig4.severity_threshold || 0.5)
      ? 'high'
      : 'medium';

    const acceptableRatio = validationConfig.acceptable_ratios?.one_to_one_target_mapping || 0.15;

    patterns.ungrouped_operational_targets = {
      severity,
      detected: true,
      penalty: penalty,
      ratio: groupingAnalysis.one_to_one_ratio,
      count: groupingAnalysis.one_to_one_count,
      message: `${Math.round(groupingAnalysis.one_to_one_ratio * 100)}% of operational targets have 1:1 requirement mapping (max ${Math.round(acceptableRatio * 100)}% recommended)`,
      recommendation: 'Group related requirements under shared operational targets from the PRD',
      why_it_matters: 'Operational targets from the PRD typically represent high-level business capabilities that encompass multiple related requirements. A 1:1 mapping suggests requirements were auto-generated from targets rather than properly decomposed.',
      description: 'Operational targets should group related requirements under cohesive business objectives. Excessive 1:1 mappings indicate lack of proper requirement decomposition and PRD understanding.'
    };
    issues.push(patterns.ungrouped_operational_targets);
  }

  // ========================================================================
  // Issue 5: Insufficient validation layers (missing validation diversity)
  // ========================================================================
  const penaltyConfig5 = getPenaltyConfig('insufficient_validation_layers');
  const scenarioComponents = componentDetector.detectScenarioComponents(scenarioRoot);

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

    const minLayersP0 = validationConfig.min_layers?.p0_requirements || 2;
    const minLayersP1 = validationConfig.min_layers?.p1_requirements || 2;
    const minLayers = (criticality === 'P0' || criticality === 'P1')
      ? (criticality === 'P0' ? minLayersP0 : minLayersP1)
      : 1;

    return applicable.size < minLayers;
  });

  if (diversityIssues.length > 0) {
    const penalty = Math.min(
      Math.round((diversityIssues.length / Math.max(requirements.length, 1)) * penaltyConfig5.multiplier),
      penaltyConfig5.max_penalty
    );
    totalPenalty += penalty;

    patterns.insufficient_validation_layers = {
      severity: penaltyConfig5.severity,
      detected: true,
      penalty: penalty,
      count: diversityIssues.length,
      total: requirements.length,
      has_api: scenarioComponents.has('API'),
      has_ui: scenarioComponents.has('UI'),
      message: `${diversityIssues.length} critical requirements (P0/P1) lack multi-layer AUTOMATED validation`,
      recommendation: 'Add automated validations across API, UI, and e2e layers for critical requirements (manual validations don\'t count toward diversity)',
      why_it_matters: 'Multi-layer validation catches different bug types: API tests verify business logic, UI tests verify user interface behavior, and e2e tests verify complete workflows. Single-layer validation misses integration issues and edge cases. Gaming Prevention: This ensures agents don\'t just link requirements to the easiest/most basic tests.',
      description: `Determining applicable layers: This scenario has ${scenarioComponents.has('API') ? 'API' : 'no API'} and ${scenarioComponents.has('UI') ? 'UI' : 'no UI'} components. P0/P1 requirements need 2+ AUTOMATED layers from applicable types. ${scenarioComponents.has('API') && scenarioComponents.has('UI') ? 'For full-stack scenarios: validate at API + UI, API + e2e, or all three layers.' : scenarioComponents.has('API') ? 'For API-only scenarios: validate at API + e2e layers.' : scenarioComponents.has('UI') ? 'For UI-only scenarios: validate at UI + e2e layers.' : 'Add e2e validation.'} Manual validations are excluded as they don't provide automated regression protection.`
    };
    issues.push(patterns.insufficient_validation_layers);
  }

  // ========================================================================
  // Issue 6: Superficial test implementation (low-quality test files)
  // ========================================================================
  const penaltyConfig6 = getPenaltyConfig('superficial_test_implementation');
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
    const penalty = Math.min(
      lowQualityTests.length * penaltyConfig6.per_file,
      penaltyConfig6.max_penalty
    );
    totalPenalty += penalty;

    patterns.superficial_test_implementation = {
      severity: penaltyConfig6.severity,
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

  // ========================================================================
  // Issue 7: Missing test automation (excessive manual validation usage)
  // ========================================================================
  const penaltyConfig7 = getPenaltyConfig('missing_test_automation');
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
  const acceptableManualRatio = validationConfig.acceptable_ratios?.manual_validation || 0.10;
  const completeThreshold = validationConfig.complete_with_manual_threshold || 5;

  // Flag if: (a) >10% manual overall OR (b) ≥5 complete requirements with ONLY manual validation
  if (manualRatio > acceptableManualRatio || completeWithManual.length >= completeThreshold) {
    const penalty = Math.min(
      Math.round(manualRatio * penaltyConfig7.ratio_multiplier) +
        Math.min(completeWithManual.length * penaltyConfig7.per_complete_requirement, penaltyConfig7.max_complete_count),
      penaltyConfig7.max_penalty
    );
    totalPenalty += penalty;

    const severity = completeWithManual.length >= (penaltyConfig7.severity_threshold || 10)
      ? 'high'
      : 'medium';

    patterns.missing_test_automation = {
      severity,
      detected: true,
      penalty: penalty,
      ratio: manualRatio,
      count: manualValidations,
      complete_with_manual: completeWithManual.length,
      message: manualRatio > acceptableManualRatio
        ? `${Math.round(manualRatio * 100)}% of validations are manual (max ${Math.round(acceptableManualRatio * 100)}% recommended). Manual validations are intended as temporary measures before automated tests are implemented.`
        : `${completeWithManual.length} requirements marked complete with ONLY manual validations. Automated tests should replace manual validations for completed requirements.`,
      recommendation: 'Replace manual validations with automated tests (API tests, UI tests, or e2e automation). Manual validations should be temporary measures for pending/in_progress requirements.',
      why_it_matters: 'Manual validations are not repeatable, not run in CI/CD, and provide no regression protection. Completed requirements must have automated tests to ensure they stay working as the codebase evolves.',
      description: 'Manual validations are acceptable for in-progress work, but completed requirements must be validated by automated tests. Manual testing doesn\'t scale and can\'t catch regressions.'
    };
    issues.push(patterns.missing_test_automation);
  }

  // ========================================================================
  // Return analysis results
  // ========================================================================
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
