'use strict';

/**
 * Centralized gaming pattern detection
 * @module scenarios/lib/gaming-detection
 */

const componentDetector = require('./validators/component-detector');
const layerDetector = require('./validators/layer-detector');
const duplicateDetector = require('./validators/duplicate-detector');
const targetGroupingValidator = require('./validators/target-grouping-validator');
const testQualityAnalyzer = require('./validators/test-quality-analyzer');

/**
 * Detect all gaming patterns in a scenario
 * @param {object} metrics - Completeness metrics
 * @param {array} requirements - Requirements array
 * @param {array} targets - Operational targets array
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} Detection results with warnings
 */
function detectGamingPatterns(metrics, requirements, targets, scenarioRoot) {
  const warnings = [];
  const patterns = {};

  // Pattern 1: Perfect 1:1 requirement-to-test ratio
  const testReqRatio = metrics.tests.total / Math.max(metrics.requirements.total, 1);
  if (Math.abs(testReqRatio - 1.0) < 0.1) {  // Within 10% of 1:1
    patterns.one_to_one_test_ratio = {
      severity: 'medium',
      detected: true,
      ratio: testReqRatio,
      message: `Suspicious 1:1 test-to-requirement ratio (${metrics.tests.total} tests for ${metrics.requirements.total} requirements). Expected: 1.5-2.0x ratio with diverse test sources.`,
      recommendation: 'Add multiple test types per requirement (API tests, UI tests, e2e automation)'
    };
    warnings.push(patterns.one_to_one_test_ratio);
  }

  // Pattern 2: Unsupported test directory usage
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

    patterns.unsupported_test_directory = {
      severity: ratio > 0.5 ? 'high' : 'medium',
      detected: true,
      count: unsupportedRefCount,
      total: metrics.requirements.total,
      ratio: ratio,
      message: `${unsupportedRefCount}/${metrics.requirements.total} requirements reference unsupported test/ directories. Valid sources for this scenario:\n${validSources.map(s => `  • ${s}`).join('\n')}`,
      recommendation: 'Move validation refs to supported test sources listed above'
    };
    warnings.push(patterns.unsupported_test_directory);
  }

  // Pattern 3: Duplicate test refs
  const refAnalysis = duplicateDetector.analyzeTestRefUsage(requirements);
  if (refAnalysis.violations.length > 0) {
    patterns.duplicate_test_refs = {
      severity: refAnalysis.violations.some(v => v.severity === 'high') ? 'high' : 'medium',
      detected: true,
      violations: refAnalysis.violations.length,
      worst_offender: refAnalysis.violations[0],
      message: `${refAnalysis.violations.length} test files validate ≥4 requirements each. Expected: unique tests per requirement.`,
      recommendation: 'Break monolithic test files into focused tests for each requirement'
    };
    warnings.push(patterns.duplicate_test_refs);
  }

  // Pattern 4: Excessive 1:1 target-to-requirement mappings
  const groupingAnalysis = targetGroupingValidator.analyzeTargetGrouping(targets, requirements);
  if (groupingAnalysis.violations.length > 0) {
    patterns.one_to_one_target_mapping = {
      severity: groupingAnalysis.one_to_one_ratio > 0.5 ? 'high' : 'medium',
      detected: true,
      ratio: groupingAnalysis.one_to_one_ratio,
      count: groupingAnalysis.one_to_one_count,
      message: `${Math.round(groupingAnalysis.one_to_one_ratio * 100)}% of operational targets have 1:1 requirement mapping (max ${Math.round(groupingAnalysis.acceptable_ratio * 100)}% recommended)`,
      recommendation: 'Group related requirements under shared operational targets from the PRD'
    };
    warnings.push(patterns.one_to_one_target_mapping);
  }

  // Pattern 5: Missing validation diversity
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
    patterns.missing_validation_diversity = {
      severity: 'high',
      detected: true,
      count: diversityIssues.length,
      total: requirements.length,
      message: `${diversityIssues.length} critical requirements (P0/P1) lack multi-layer AUTOMATED validation`,
      recommendation: 'Add automated validations across API, UI, and e2e layers for critical requirements (manual validations don\'t count toward diversity)'
    };
    warnings.push(patterns.missing_validation_diversity);
  }

  // Pattern 6: Low-quality test files
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
    patterns.low_quality_tests = {
      severity: 'medium',
      detected: true,
      count: lowQualityTests.length,
      files: lowQualityTests,
      message: `${lowQualityTests.length} test file(s) appear superficial (< 20 LOC, missing assertions, or no test functions)`,
      recommendation: 'Add meaningful test logic with assertions and edge case coverage'
    };
    warnings.push(patterns.low_quality_tests);
  }

  // Pattern 7: Excessive manual validation usage
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
    patterns.excessive_manual_validations = {
      severity: completeWithManual.length >= 10 ? 'high' : 'medium',
      detected: true,
      ratio: manualRatio,
      count: manualValidations,
      complete_with_manual: completeWithManual.length,
      message: manualRatio > 0.10
        ? `${Math.round(manualRatio * 100)}% of validations are manual (max 10% recommended). Manual validations are intended as temporary measures before automated tests are implemented.`
        : `${completeWithManual.length} requirements marked complete with ONLY manual validations. Automated tests should replace manual validations for completed requirements.`,
      recommendation: 'Replace manual validations with automated tests (API tests, UI tests, or e2e automation). Manual validations should be temporary measures for pending/in_progress requirements.'
    };
    warnings.push(patterns.excessive_manual_validations);
  }

  return {
    has_warnings: warnings.length > 0,
    warning_count: warnings.length,
    warnings,
    patterns,
    overall_severity: warnings.some(w => w.severity === 'high') ? 'high' : 'medium'
  };
}

module.exports = {
  detectGamingPatterns
};
