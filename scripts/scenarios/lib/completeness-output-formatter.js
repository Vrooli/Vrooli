'use strict';

/**
 * Completeness output formatter
 * Provides improved, user-friendly output for completeness scoring
 * @module scenarios/lib/completeness-output-formatter
 */

/**
 * Format validation issues section (prioritized at top of output)
 * @param {object} validationQualityAnalysis - Validation quality analysis
 * @param {object} options - Formatting options
 * @returns {string} Formatted validation issues section
 */
function formatValidationIssues(validationQualityAnalysis, options = {}) {
  const { verbose = false } = options;

  if (!validationQualityAnalysis.has_issues) {
    return `${'='.repeat(68)}
✅ No Validation Issues Detected
${'='.repeat(68)}

All tests follow recommended patterns and best practices.
`;
  }

  const lines = [];
  const severity = validationQualityAnalysis.overall_severity.toUpperCase();
  const penalty = validationQualityAnalysis.total_penalty;

  // Determine severity level based on both overall_severity and penalty amount
  const isCritical = severity === 'HIGH' && penalty >= 50;
  const isHigh = severity === 'HIGH' && penalty >= 20;
  const severityIcon = isCritical ? '🚨' : '⚠️';
  const severityLabel = isCritical ? 'CRITICAL ' : '';

  lines.push('='.repeat(68));
  lines.push(`${severityIcon} ${severityLabel}VALIDATION ISSUES DETECTED`);
  lines.push('='.repeat(68));
  lines.push('');

  if (isCritical) {
    lines.push(`Overall Assessment: HIGH SEVERITY gaming patterns detected (-${penalty}pts)`);
    lines.push('');
    lines.push('This scenario shows signs of test gaming rather than genuine validation.');
    lines.push('Tests appear created to inflate metrics rather than validate functionality.');
  } else if (isHigh) {
    lines.push(`Overall Assessment: MEDIUM-HIGH severity issues found (-${penalty}pts)`);
    lines.push('');
    lines.push('This scenario has test quality issues that need attention.');
  } else {
    lines.push(`Overall Assessment: MEDIUM severity issues found (-${penalty}pts)`);
    lines.push('');
    lines.push('This scenario has a solid foundation but needs test quality improvements.');
  }
  lines.push('');

  // Separate issues by severity
  const highSeverityIssues = validationQualityAnalysis.issues.filter(i => i.severity === 'high');
  const mediumSeverityIssues = validationQualityAnalysis.issues.filter(i => i.severity === 'medium');

  // Show high severity issues
  if (highSeverityIssues.length > 0) {
    lines.push('Top Issues (Fix These First):');
    lines.push('━'.repeat(68));
    lines.push('');

    highSeverityIssues.forEach(issue => {
      lines.push(...formatIssueDetail(issue, verbose));
    });
  }

  // Show medium severity issues (collapsed by default)
  if (mediumSeverityIssues.length > 0) {
    if (verbose || highSeverityIssues.length === 0) {
      lines.push('━'.repeat(68));
      lines.push('');
      lines.push(`🟡 ${highSeverityIssues.length > 0 ? 'Minor Issues' : 'Issues Found'} (${mediumSeverityIssues.length} issues, -${mediumSeverityIssues.reduce((sum, i) => sum + i.penalty, 0)}pts total)`);
      lines.push('');

      mediumSeverityIssues.forEach(issue => {
        lines.push(...formatIssueDetail(issue, verbose));
      });
    } else {
      lines.push('━'.repeat(68));
      lines.push('');
      lines.push(`🟡 Minor Issues (${mediumSeverityIssues.length} issues, -${mediumSeverityIssues.reduce((sum, i) => sum + i.penalty, 0)}pts total)`);
      lines.push('');
      mediumSeverityIssues.forEach(issue => {
        const icon = issue.severity === 'high' ? '🔴' : '🟡';
        lines.push(`${icon} ${issue.message.split('\n')[0]}`);
        lines.push(`   Penalty: -${issue.penalty} pts`);
        lines.push('');
      });
    }
  }

  if (!verbose) {
    lines.push('Run with --verbose to see detailed explanations and per-requirement breakdown.');
    lines.push('');
  }

  return lines.join('\n');
}

/**
 * Format individual issue detail
 * @param {object} issue - Issue object
 * @param {boolean} verbose - Whether to show full details
 * @returns {array} Array of formatted lines
 */
function formatIssueDetail(issue, verbose) {
  const lines = [];
  const icon = issue.severity === 'high' ? '🔴' : '🟡';

  lines.push(`${icon} ${issue.message.split('\n').join('\n   ')}`);

  if (verbose) {
    lines.push('');
    lines.push(`   Why this matters:`);
    lines.push(`   ${issue.why_it_matters}`);
  }

  lines.push('');
  lines.push(`   Next Steps:`);

  // Add contextual, actionable recommendations based on issue type
  const recommendations = generateContextualRecommendations(issue);
  recommendations.forEach(rec => {
    lines.push(`     → ${rec}`);
  });

  if (verbose && issue.description) {
    lines.push('');
    lines.push(`   Background:`);
    lines.push(`   ${issue.description}`);
  }

  lines.push('');

  return lines;
}

/**
 * Generate contextual recommendations based on issue type and data
 * @param {object} issue - Issue object
 * @returns {array} Array of recommendation strings
 */
function generateContextualRecommendations(issue) {
  const recommendations = [];

  // Extract issue type from patterns
  if (issue.message.includes('invalid test locations') || issue.message.includes('unsupported test/')) {
    const validSources = issue.message.match(/Valid sources.*?:\n(.*?)$/ms);
    if (validSources) {
      recommendations.push('Move requirement validation refs to actual test files');
      recommendations.push('Valid test locations are listed above in the error message');
    } else {
      recommendations.push(issue.recommendation);
    }

    if (issue.count === issue.total) {
      recommendations.push('');
      recommendations.push('⚠️  ALL requirements use invalid locations - suggests systematic gaming');
      recommendations.push('Study browser-automation-studio scenario for proper test structure');
    }
  } else if (issue.message.includes('multi-layer')) {
    recommendations.push('For each P0/P1 requirement:');
    recommendations.push('  1. Identify which layers apply (API/UI/e2e)');
    recommendations.push('  2. Add automated tests in valid locations for each layer');
    recommendations.push('  3. Update requirement validation refs to include all layers');

    if (issue.count > 0) {
      recommendations.push('');
      recommendations.push(`Affected requirements: ${issue.count} total`);
      recommendations.push('Prioritize P0 requirements first for highest impact');
    }
  } else if (issue.message.includes('monolithic test files')) {
    recommendations.push('Create focused tests that validate single requirements');
    recommendations.push('Use appropriate test locations (API/UI/e2e, not CLI wrappers)');

    if (issue.worst_offender) {
      recommendations.push('');
      recommendations.push(`Worst offender: ${issue.worst_offender.test_ref}`);
      recommendations.push(`  → Validates ${issue.worst_offender.count} requirements`);
      recommendations.push('  → Split into focused tests in proper locations');
    }
  } else if (issue.message.includes('operational targets') && issue.message.includes('1:1')) {
    recommendations.push('Review requirements and group related ones under shared targets');
    recommendations.push('Update operational_target_id in requirements/*/module.json');
    recommendations.push('Reference PRD operational targets for proper grouping');

    if (issue.ratio === 1.0) {
      recommendations.push('');
      recommendations.push('⚠️  100% 1:1 mapping suggests requirements were auto-generated');
      recommendations.push('Targets should group 3-10 related requirements each');
    }
  } else if (issue.message.includes('superficial')) {
    recommendations.push('Add meaningful test steps with assertions');
    recommendations.push('Add edge case coverage and negative test scenarios');
    recommendations.push('Ensure tests actually verify behavior, not just existence');
  } else if (issue.message.includes('manual validation')) {
    recommendations.push('Replace manual validations with automated tests');
    recommendations.push('Create API tests, UI tests, or e2e playbooks as appropriate');
    recommendations.push('Manual validations should only be temporary for in-progress work');
  } else if (issue.message.includes('1:1 test-to-requirement ratio')) {
    recommendations.push('Add multiple test types per requirement (API + UI + e2e where applicable)');
    recommendations.push('Avoid single monolithic tests - create layered validation');
  } else {
    // Fallback to generic recommendation
    recommendations.push(issue.recommendation);
  }

  return recommendations;
}

/**
 * Format score summary section
 * @param {number} totalScore - Final score
 * @param {object} breakdown - Score breakdown
 * @param {string} classification - Classification level
 * @param {object} validationQualityAnalysis - Validation quality analysis
 * @returns {string} Formatted score summary
 */
function formatScoreSummary(totalScore, breakdown, classification, validationQualityAnalysis) {
  const lines = [];

  lines.push('='.repeat(68));
  lines.push(`📊 COMPLETENESS SCORE: ${totalScore}/100 (${classification.replace(/_/g, '_')})`);
  lines.push('='.repeat(68));
  lines.push('');
  lines.push(`  Final Score:        ${totalScore}/100`);
  lines.push(`  Base Score:         ${breakdown.base_score}/100`);

  if (validationQualityAnalysis.has_issues) {
    const severity = validationQualityAnalysis.overall_severity === 'high' ? '⚠️  SEVERE' : '';
    lines.push(`  Validation Penalty: -${breakdown.validation_penalty}pts ${severity}`);
  }

  lines.push('');
  lines.push(`  Classification: ${classification.replace(/_/g, ' ')}`);
  lines.push(`  Status: ${getClassificationDescription(classification)}`);

  return lines.join('\n');
}

/**
 * Get classification description
 * @param {string} classification - Classification level
 * @returns {string} Description
 */
function getClassificationDescription(classification) {
  const descriptions = {
    production_ready: 'Production ready, excellent validation coverage',
    nearly_ready: 'Nearly ready, final polish and edge cases',
    mostly_complete: 'Mostly complete, needs refinement and validation',
    functional_incomplete: 'Functional but incomplete, needs more features/tests',
    foundation_laid: 'Foundation laid, core features in progress',
    early_stage: 'Just starting, needs significant development'
  };
  return descriptions[classification] || 'Status unclear';
}

/**
 * Format quality assessment (brief summary) - deprecated, use formatBaseMetrics instead
 * @param {object} breakdown - Score breakdown
 * @returns {string} Formatted quality assessment
 */
function formatQualityAssessment(breakdown) {
  const lines = [];

  lines.push('');
  lines.push('Quality Assessment:');

  const reqRate = breakdown.quality.requirement_pass_rate;
  const targetRate = breakdown.quality.target_pass_rate;
  const testRate = breakdown.quality.test_pass_rate;

  const reqIcon = reqRate.rate >= 0.9 ? '✅' : '⚠️';
  const targetIcon = targetRate.rate >= 0.9 ? '✅' : '⚠️';
  const testIcon = testRate.rate >= 0.9 ? '✅' : '⚠️';

  lines.push(`  Requirements: ${reqRate.passing}/${reqRate.total} passing (${Math.round(reqRate.rate * 100)}%)   ${reqIcon}${reqRate.rate < 0.9 ? '  Target: 90%+' : ''}`);
  lines.push(`  Op Targets:   ${targetRate.passing}/${targetRate.total} passing (${Math.round(targetRate.rate * 100)}%) ${targetIcon}${targetRate.rate < 0.9 ? '  Target: 90%+' : ''}`);
  lines.push(`  Tests:        ${testRate.passing}/${testRate.total} passing (${Math.round(testRate.rate * 100)}%) ${testIcon}${testRate.rate < 0.9 ? '  Target: 90%+' : ''}`);

  return lines.join('\n');
}

/**
 * Format base metrics breakdown (always shown in default output)
 * @param {object} breakdown - Score breakdown
 * @param {object} thresholds - Category thresholds
 * @returns {string} Formatted base metrics
 */
function formatBaseMetrics(breakdown, thresholds) {
  const lines = [];

  lines.push('');
  lines.push('Quality Metrics (' + breakdown.quality.score + '/' + breakdown.quality.max + '):');
  lines.push(`  ${breakdown.quality.requirement_pass_rate.rate >= 0.9 ? '✅' : '⚠️ '} Requirements: ${breakdown.quality.requirement_pass_rate.total} total, ${breakdown.quality.requirement_pass_rate.passing} passing (${Math.round(breakdown.quality.requirement_pass_rate.rate * 100)}%) → ${breakdown.quality.requirement_pass_rate.points}/20 pts${breakdown.quality.requirement_pass_rate.rate < 0.9 ? '  [Target: 90%+]' : ''}`);
  lines.push(`  ${breakdown.quality.target_pass_rate.rate >= 0.9 ? '✅' : '⚠️ '} Op Targets: ${breakdown.quality.target_pass_rate.total} total, ${breakdown.quality.target_pass_rate.passing} passing (${Math.round(breakdown.quality.target_pass_rate.rate * 100)}%) → ${breakdown.quality.target_pass_rate.points}/15 pts${breakdown.quality.target_pass_rate.rate < 0.9 ? '  [Target: 90%+]' : ''}`);
  lines.push(`  ${breakdown.quality.test_pass_rate.rate >= 0.9 ? '✅' : '⚠️ '} Tests: ${breakdown.quality.test_pass_rate.total} total, ${breakdown.quality.test_pass_rate.passing} passing (${Math.round(breakdown.quality.test_pass_rate.rate * 100)}%) → ${breakdown.quality.test_pass_rate.points}/15 pts${breakdown.quality.test_pass_rate.rate < 0.9 ? '  [Target: 90%+]' : ''}`);

  lines.push('');
  lines.push('Coverage Metrics (' + breakdown.coverage.score + '/' + breakdown.coverage.max + '):');
  lines.push(`  ${breakdown.coverage.test_coverage_ratio.ratio >= 2.0 ? '✅' : '⚠️ '} Test Coverage: ${breakdown.coverage.test_coverage_ratio.ratio.toFixed(1)}x → ${breakdown.coverage.test_coverage_ratio.points}/8 pts${breakdown.coverage.test_coverage_ratio.ratio < 2.0 ? '  [Target: 2.0x]' : ''}`);
  lines.push(`  ${breakdown.coverage.depth_score.avg_depth >= 3.0 ? '✅' : '⚠️ '} Depth Score: ${breakdown.coverage.depth_score.avg_depth.toFixed(1)} avg levels → ${breakdown.coverage.depth_score.points}/7 pts${breakdown.coverage.depth_score.avg_depth < 3.0 ? '  [Target: 3.0+]' : ''}`);

  lines.push('');
  lines.push('Quantity Metrics (' + breakdown.quantity.score + '/' + breakdown.quantity.max + '):');
  lines.push(`  ${breakdown.quantity.requirements.threshold === 'good' || breakdown.quantity.requirements.threshold === 'excellent' ? '✅' : '⚠️ '} Requirements: ${breakdown.quantity.requirements.count} (${breakdown.quantity.requirements.threshold}) → ${breakdown.quantity.requirements.points}/4 pts${breakdown.quantity.requirements.threshold !== 'good' && breakdown.quantity.requirements.threshold !== 'excellent' ? `  [Target: ${thresholds.requirements.good}+]` : ''}`);
  lines.push(`  ${breakdown.quantity.targets.threshold === 'good' || breakdown.quantity.targets.threshold === 'excellent' ? '✅' : '⚠️ '} Targets: ${breakdown.quantity.targets.count} (${breakdown.quantity.targets.threshold}) → ${breakdown.quantity.targets.points}/3 pts${breakdown.quantity.targets.threshold !== 'good' && breakdown.quantity.targets.threshold !== 'excellent' ? `  [Target: ${thresholds.targets.good}+]` : ''}`);
  lines.push(`  ${breakdown.quantity.tests.threshold === 'good' || breakdown.quantity.tests.threshold === 'excellent' ? '✅' : '⚠️ '} Tests: ${breakdown.quantity.tests.count} (${breakdown.quantity.tests.threshold}) → ${breakdown.quantity.tests.points}/3 pts${breakdown.quantity.tests.threshold !== 'good' && breakdown.quantity.tests.threshold !== 'excellent' ? `  [Target: ${thresholds.tests.good}+]` : ''}`);

  if (breakdown.ui) {
    lines.push('');
    lines.push('UI Metrics (' + breakdown.ui.score + '/' + breakdown.ui.max + '):');

    const templateIcon = breakdown.ui.template_check.is_template ? '❌' : '✅';
    const templateStatus = breakdown.ui.template_check.is_template ? 'TEMPLATE' : 'Custom';
    lines.push(`  ${templateIcon} Template: ${templateStatus} → ${breakdown.ui.template_check.points}/10 pts${breakdown.ui.template_check.is_template ? '  [CRITICAL: Replace template UI]' : ''}`);

    const compIcon = breakdown.ui.component_complexity.threshold === 'good' || breakdown.ui.component_complexity.threshold === 'excellent' ? '✅' : '⚠️ ';
    lines.push(`  ${compIcon} Files: ${breakdown.ui.component_complexity.file_count} files (${breakdown.ui.component_complexity.threshold}) → ${breakdown.ui.component_complexity.points}/5 pts${breakdown.ui.component_complexity.threshold === 'below' || breakdown.ui.component_complexity.threshold === 'ok' ? `  [Target: ${thresholds.ui.file_count.good}+]` : ''}`);

    const apiIcon = breakdown.ui.api_integration.endpoint_count >= thresholds.ui.api_endpoints.good ? '✅' : '⚠️ ';
    lines.push(`  ${apiIcon} API Integration: ${breakdown.ui.api_integration.endpoint_count} endpoints beyond /health → ${breakdown.ui.api_integration.points}/6 pts${breakdown.ui.api_integration.endpoint_count < thresholds.ui.api_endpoints.good ? `  [Target: ${thresholds.ui.api_endpoints.good}+]` : ''}`);

    const routeIcon = breakdown.ui.routing.route_count >= 3 ? '✅' : '⚠️ ';
    lines.push(`  ${routeIcon} Routing: ${breakdown.ui.routing.route_count} routes → ${breakdown.ui.routing.points}/1.5 pts`);

    const volumeIcon = breakdown.ui.code_volume.total_loc >= thresholds.ui.total_loc.good ? '✅' : '⚠️ ';
    lines.push(`  ${volumeIcon} LOC: ${breakdown.ui.code_volume.total_loc} total → ${breakdown.ui.code_volume.points}/2.5 pts${breakdown.ui.code_volume.total_loc < thresholds.ui.total_loc.good ? `  [Target: ${thresholds.ui.total_loc.good}+]` : ''}`);
  }

  return lines.join('\n');
}

/**
 * Format detailed metrics breakdown (for verbose mode or --metrics flag)
 * @param {object} breakdown - Score breakdown
 * @param {object} thresholds - Category thresholds
 * @returns {string} Formatted metrics breakdown
 */
function formatDetailedMetrics(breakdown, thresholds) {
  const lines = [];

  lines.push('='.repeat(68));
  lines.push('📊 DETAILED METRICS BREAKDOWN');
  lines.push('='.repeat(68));
  lines.push('');

  // Quality Metrics
  lines.push(`Quality Metrics (${breakdown.quality.score}/${breakdown.quality.max}):`);
  lines.push(`  ${breakdown.quality.requirement_pass_rate.rate >= 0.9 ? '✅' : '⚠️ '} Requirements Pass Rate: ${breakdown.quality.requirement_pass_rate.passing}/${breakdown.quality.requirement_pass_rate.total} (${Math.round(breakdown.quality.requirement_pass_rate.rate * 100)}%)        → ${breakdown.quality.requirement_pass_rate.points}/20 pts   ${breakdown.quality.requirement_pass_rate.rate < 0.9 ? '⚠️  Target: 90%+' : ''}`);
  lines.push(`    Breakdown: ${breakdown.quality.requirement_pass_rate.passing} passing, ${breakdown.quality.requirement_pass_rate.total - breakdown.quality.requirement_pass_rate.passing} failing, 0 pending`);
  lines.push('');
  lines.push(`  ${breakdown.quality.target_pass_rate.rate >= 0.9 ? '✅' : '⚠️ '} Operational Targets:    ${breakdown.quality.target_pass_rate.passing}/${breakdown.quality.target_pass_rate.total} (${Math.round(breakdown.quality.target_pass_rate.rate * 100)}%)     → ${breakdown.quality.target_pass_rate.points}/15 pts  ${breakdown.quality.target_pass_rate.rate < 0.9 ? '⚠️  Target: 90%+' : ''}`);
  lines.push(`    Breakdown: ${breakdown.quality.target_pass_rate.passing} passing, ${breakdown.quality.target_pass_rate.total - breakdown.quality.target_pass_rate.passing} failing, 0 pending`);
  lines.push('');
  lines.push(`  ${breakdown.quality.test_pass_rate.rate >= 0.9 ? '✅' : '⚠️ '} Test Pass Rate:         ${breakdown.quality.test_pass_rate.passing}/${breakdown.quality.test_pass_rate.total} (${Math.round(breakdown.quality.test_pass_rate.rate * 100)}%)     → ${breakdown.quality.test_pass_rate.points}/15 pts  ${breakdown.quality.test_pass_rate.rate < 0.9 ? '⚠️  Target: 90%+' : ''}`);
  lines.push(`    Breakdown: ${breakdown.quality.test_pass_rate.passing} passing, ${breakdown.quality.test_pass_rate.total - breakdown.quality.test_pass_rate.passing} failing, 0 skipped`);

  // Coverage Metrics
  lines.push('');
  lines.push(`Coverage Metrics (${breakdown.coverage.score}/${breakdown.coverage.max}):`);
  lines.push(`  ${breakdown.coverage.test_coverage_ratio.ratio >= 2.0 ? '✅' : '⚠️ '} Test-to-Requirement Ratio: ${breakdown.coverage.test_coverage_ratio.ratio.toFixed(1)}x          → ${breakdown.coverage.test_coverage_ratio.points}/8 pts   ${breakdown.coverage.test_coverage_ratio.ratio < 2.0 ? '⚠️  Target: 2.0x+' : ''}`);
  lines.push(`    Tests: ${breakdown.quality.test_pass_rate.total} | Requirements: ${breakdown.quality.requirement_pass_rate.total}`);
  if (breakdown.coverage.test_coverage_ratio.ratio < 2.0) {
    const deficit = Math.ceil((2.0 * breakdown.quality.requirement_pass_rate.total) - breakdown.quality.test_pass_rate.total);
    lines.push(`    Deficit: Need ${deficit} more tests for optimal coverage`);
  }
  lines.push('');
  lines.push(`  ${breakdown.coverage.depth_score.avg_depth >= 3.0 ? '✅' : '⚠️ '} Validation Depth Score: ${breakdown.coverage.depth_score.avg_depth.toFixed(1)} avg levels   → ${breakdown.coverage.depth_score.points}/7 pts   ${breakdown.coverage.depth_score.avg_depth < 3.0 ? '⚠️  Target: 3.0+' : ''}`);
  lines.push(`    Requirements with 1 layer: [count not available]`);
  lines.push(`    Requirements with 2 layers: [count not available]`);
  lines.push(`    Requirements with 3+ layers: [count not available]`);

  // Quantity Metrics
  lines.push('');
  lines.push(`Quantity Metrics (${breakdown.quantity.score}/${breakdown.quantity.max}):`);
  lines.push(`  ${breakdown.quantity.requirements.threshold === 'good' || breakdown.quantity.requirements.threshold === 'excellent' ? '✅' : '⚠️ '} Requirements Count: ${breakdown.quantity.requirements.count} (${breakdown.quantity.requirements.threshold})       → ${breakdown.quantity.requirements.points}/4 pts   ${breakdown.quantity.requirements.threshold !== 'good' && breakdown.quantity.requirements.threshold !== 'excellent' ? `⚠️  Target: ${thresholds.requirements.good}+` : ''}`);
  lines.push(`  ${breakdown.quantity.targets.threshold === 'good' || breakdown.quantity.targets.threshold === 'excellent' ? '✅' : '⚠️ '} Op Targets Count:   ${breakdown.quantity.targets.count} (${breakdown.quantity.targets.threshold})       → ${breakdown.quantity.targets.points}/3 pts   ${breakdown.quantity.targets.threshold !== 'good' && breakdown.quantity.targets.threshold !== 'excellent' ? `⚠️  Target: ${thresholds.targets.good}+` : ''}`);
  lines.push(`  ${breakdown.quantity.tests.threshold === 'good' || breakdown.quantity.tests.threshold === 'excellent' ? '✅' : '⚠️ '} Test Count:         ${breakdown.quantity.tests.count} (${breakdown.quantity.tests.threshold})       → ${breakdown.quantity.tests.points}/3 pts   ${breakdown.quantity.tests.threshold !== 'good' && breakdown.quantity.tests.threshold !== 'excellent' ? `⚠️  Target: ${thresholds.tests.good}+` : ''}`);

  // UI Metrics
  if (breakdown.ui) {
    lines.push('');
    lines.push(`UI Metrics (${breakdown.ui.score}/${breakdown.ui.max}):`);

    const templateIcon = breakdown.ui.template_check.is_template ? '❌' : '✅';
    const templateStatus = breakdown.ui.template_check.is_template ? 'TEMPLATE' : 'Custom';
    lines.push(`  ${templateIcon} Template Type:      ${templateStatus}               → ${breakdown.ui.template_check.points}/10 pts ${breakdown.ui.template_check.is_template ? '⚠️  CRITICAL: Replace template UI' : ''}`);

    const compIcon = breakdown.ui.component_complexity.threshold === 'good' || breakdown.ui.component_complexity.threshold === 'excellent' ? '✅' : '⚠️ ';
    lines.push(`  ${compIcon} File Count:         ${breakdown.ui.component_complexity.file_count} files (${breakdown.ui.component_complexity.threshold})      → ${breakdown.ui.component_complexity.points}/5 pts  ${breakdown.ui.component_complexity.threshold === 'below' || breakdown.ui.component_complexity.threshold === 'ok' ? `⚠️  Target: ${thresholds.ui.file_count.good}+` : ''}`);

    const apiIcon = breakdown.ui.api_integration.endpoint_count >= thresholds.ui.api_endpoints.good ? '✅' : '⚠️ ';
    lines.push(`  ${apiIcon} API Integration:    ${breakdown.ui.api_integration.endpoint_count} endpoints          → ${breakdown.ui.api_integration.points}/6 pts  ${breakdown.ui.api_integration.endpoint_count < thresholds.ui.api_endpoints.good ? `⚠️  Target: ${thresholds.ui.api_endpoints.good}+` : ''}`);

    const routeIcon = breakdown.ui.routing.route_count >= 3 ? '✅' : '⚠️ ';
    lines.push(`  ${routeIcon} Routing:            ${breakdown.ui.routing.route_count} routes             → ${breakdown.ui.routing.points}/1.5 pts`);

    const volumeIcon = breakdown.ui.code_volume.total_loc >= thresholds.ui.total_loc.good ? '✅' : '⚠️ ';
    lines.push(`  ${volumeIcon} Lines of Code:      ${breakdown.ui.code_volume.total_loc} total             → ${breakdown.ui.code_volume.points}/2.5 pts ${breakdown.ui.code_volume.total_loc < thresholds.ui.total_loc.good ? `⚠️  Target: ${thresholds.ui.total_loc.good}+` : ''}`);
  }

  return lines.join('\n');
}

/**
 * Format action plan section
 * @param {object} breakdown - Score breakdown
 * @param {object} validationQualityAnalysis - Validation quality analysis
 * @param {object} thresholds - Category thresholds
 * @returns {string} Formatted action plan
 */
function formatActionPlan(breakdown, validationQualityAnalysis, thresholds) {
  const lines = [];

  lines.push('='.repeat(68));
  lines.push('🎯 RECOMMENDED ACTION PLAN');
  lines.push('='.repeat(68));
  lines.push('');

  if (validationQualityAnalysis.has_issues) {
    lines.push('To improve this score, fix validation issues first (highest ROI):');
    lines.push('');

    // Generate phased approach based on issues
    const phases = generateActionPhases(breakdown, validationQualityAnalysis, thresholds);
    phases.forEach((phase, index) => {
      lines.push(`Phase ${index + 1}: ${phase.title} (+${phase.estimatedPoints}pts estimated)`);
      phase.actions.forEach(action => {
        lines.push(`  ${action}`);
      });
      lines.push('');
    });

    const estimatedFinalScore = Math.min(breakdown.base_score, 100);
    const newClassification = estimatedFinalScore >= 61 ? 'functional_complete' :
                              estimatedFinalScore >= 41 ? 'functional_incomplete' :
                              estimatedFinalScore >= 21 ? 'foundation_laid' : 'early_stage';
    lines.push(`Estimated Score After Fixes: ~${estimatedFinalScore}/100 (${newClassification})`);
  } else {
    // No gaming issues, show regular recommendations
    const recs = generatePriorityRecommendations(breakdown, thresholds);
    if (recs.length > 0) {
      recs.forEach((rec, index) => {
        lines.push(`${index + 1}. ${rec}`);
      });
    } else {
      lines.push('No priority actions needed. Continue maintaining quality!');
    }
  }

  return lines.join('\n');
}

/**
 * Generate phased action plan
 * @param {object} breakdown - Score breakdown
 * @param {object} validationQualityAnalysis - Validation quality analysis
 * @param {object} thresholds - Category thresholds
 * @returns {array} Array of phase objects
 */
function generateActionPhases(breakdown, validationQualityAnalysis, thresholds) {
  const phases = [];

  // Find the highest impact issues
  const highSeverityIssues = validationQualityAnalysis.issues
    .filter(i => i.severity === 'high')
    .sort((a, b) => b.penalty - a.penalty);

  // Phase 1: Fix test locations (if applicable)
  const invalidLocationIssue = highSeverityIssues.find(i =>
    i.message.includes('invalid test locations') || i.message.includes('unsupported test/')
  );

  if (invalidLocationIssue) {
    phases.push({
      title: 'Fix Test Locations',
      estimatedPoints: invalidLocationIssue.penalty,
      actions: [
        `Current: ${invalidLocationIssue.count || 0} requirements use invalid test locations`,
        'Target: Move all requirement validation refs to valid locations',
        '',
        'Actions:',
        '  1. Audit each requirement to determine appropriate test layers:',
        '     - Business logic → API tests (api/**/*_test.go)',
        '     - UI components → UI tests (ui/src/**/*.test.tsx)',
        '     - User workflows → e2e playbooks (test/playbooks/**/*.json)',
        '',
        '  2. Create tests in valid locations (or reference existing ones)',
        '',
        '  3. Update requirements/*/module.json validation refs',
        '',
        `Note: You currently have 0 API tests, 0 UI tests, ${breakdown.quality.test_pass_rate.total} test files.`,
        '      Start by creating test files in the appropriate locations.'
      ]
    });
  }

  // Phase 2: Add multi-layer validation (if applicable)
  const multiLayerIssue = highSeverityIssues.find(i =>
    i.message.includes('multi-layer')
  );

  if (multiLayerIssue) {
    phases.push({
      title: 'Add Multi-Layer Validation',
      estimatedPoints: multiLayerIssue.penalty,
      actions: [
        `Current: ${multiLayerIssue.count || 0}/${multiLayerIssue.total || 0} critical requirements have multi-layer validation`,
        'Target: All P0/P1 requirements validated at ≥2 layers',
        '',
        'Actions:',
        '  → For each P0 requirement:',
        '    Ensure validation at 2+ layers (API + UI, API + e2e, or all 3)',
        '',
        '  → For each P1 requirement:',
        '    Ensure validation at 2+ layers where applicable'
      ]
    });
  }

  // Phase 3: Break up monolithic tests (if applicable)
  const monolithicIssue = validationQualityAnalysis.issues.find(i =>
    i.message.includes('monolithic test files')
  );

  if (monolithicIssue) {
    phases.push({
      title: 'Create Focused Tests',
      estimatedPoints: monolithicIssue.penalty,
      actions: [
        `Current: ${monolithicIssue.violations || 0} monolithic test files`,
        'Target: Focused tests per requirement',
        '',
        'Actions:',
        '  → Instead of test files that validate many requirements,',
        '    create focused tests that validate individual requirements',
        '  → Use appropriate test types (API/UI/e2e) instead of CLI wrappers'
      ]
    });
  }

  // Phase 4: Restructure operational targets (if applicable)
  const targetIssue = validationQualityAnalysis.issues.find(i =>
    i.message.includes('operational targets') && i.message.includes('1:1')
  );

  if (targetIssue) {
    phases.push({
      title: 'Restructure Operational Targets',
      estimatedPoints: targetIssue.penalty,
      actions: [
        `Current: ${Math.round((targetIssue.ratio || 0) * 100)}% of targets have 1:1 requirement mapping`,
        'Target: <15% of targets have 1:1 mapping',
        '',
        'Actions:',
        '  → Review requirements and group related ones under shared targets',
        '  → Update operational_target_id in requirements/*/module.json',
        '  → Reference PRD operational targets for proper grouping'
      ]
    });
  }

  return phases;
}

/**
 * Generate priority recommendations (for scenarios without gaming issues)
 * @param {object} breakdown - Score breakdown
 * @param {object} thresholds - Category thresholds
 * @returns {array} Array of recommendation strings
 */
function generatePriorityRecommendations(breakdown, thresholds) {
  const recommendations = [];

  // Check test pass rate
  if (breakdown.quality.test_pass_rate.rate < 1.0) {
    const failingTests = breakdown.quality.test_pass_rate.total - breakdown.quality.test_pass_rate.passing;
    recommendations.push(`Fix ${failingTests} failing tests (+${Math.round((1 - breakdown.quality.test_pass_rate.rate) * 15)}pts estimated)`);
  }

  // Check requirement pass rate
  if (breakdown.quality.requirement_pass_rate.rate < 0.9) {
    const failingReqs = breakdown.quality.requirement_pass_rate.total - breakdown.quality.requirement_pass_rate.passing;
    const p0Count = failingReqs; // We don't have priority breakdown here
    recommendations.push(`Complete ${failingReqs} pending requirements (+${Math.round((0.9 - breakdown.quality.requirement_pass_rate.rate) * 20)}pts estimated)`);
  }

  // Check test coverage ratio
  if (breakdown.coverage.test_coverage_ratio.ratio < 2.0) {
    const gap = Math.ceil((2.0 * breakdown.quality.requirement_pass_rate.total) - breakdown.quality.test_pass_rate.total);
    if (gap > 0) {
      recommendations.push(`Add ${gap} more tests to reach 2:1 ratio (+${Math.round((2.0 - breakdown.coverage.test_coverage_ratio.ratio) / 2.0 * 8)}pts estimated)`);
    }
  }

  return recommendations;
}

/**
 * Format comparison context
 * @param {string} scenarioName - Current scenario name
 * @param {number} totalScore - Current scenario score
 * @param {number} validationPenalty - Current validation penalty
 * @param {number} invalidRefRatio - Ratio of invalid refs
 * @returns {string} Formatted comparison
 */
function formatComparison(scenarioName, totalScore, validationPenalty, invalidRefRatio) {
  const lines = [];

  lines.push('='.repeat(68));
  lines.push('');

  // Only show comparison if there are severe issues OR if score is low
  if (validationPenalty > 30 || totalScore < 50) {
    lines.push('💡 Compare to similar scenarios:');

    // Show reference scenarios (excluding current scenario)
    const referenceScenarios = [
      { name: 'browser-automation-studio', score: 43, penalty: 24, invalidRefs: 11 },
      { name: 'landing-manager', score: 8, penalty: 52, invalidRefs: 15 },
    ].filter(s => s.name !== scenarioName);

    referenceScenarios.forEach(s => {
      lines.push(`   ${s.name.padEnd(27)} ${s.score}/100 (-${s.penalty}pts penalty, ${s.invalidRefs}% invalid refs)`);
    });
    lines.push(`   ${scenarioName.padEnd(27)} ${totalScore}/100 (-${validationPenalty}pts penalty, ${Math.round(invalidRefRatio * 100)}% invalid refs) ← YOU ARE HERE`);
    lines.push('');
  }

  if (validationPenalty > 50) {
    lines.push('🎓 Study browser-automation-studio as reference for proper test structure:');
    lines.push('   • Has API tests: api/**/*_test.go');
    lines.push('   • Has UI tests: ui/src/**/*.test.tsx');
    lines.push('   • Has e2e playbooks: test/playbooks/capabilities/**/ui/*.json');
    lines.push('   • Requirements reference appropriate test types');
  } else if (totalScore >= 80 && validationPenalty < 10) {
    lines.push('🌟 Excellent work! This scenario demonstrates:');
    lines.push('   ✓ Comprehensive multi-layer testing');
    lines.push('   ✓ Proper test organization');
    lines.push('   ✓ High pass rates across all metrics');
    lines.push('   ✓ Minimal gaming patterns detected');
  } else if (totalScore >= 40 && validationPenalty < 30) {
    lines.push('✨ This scenario has good test structure - continue improving:');
    lines.push('   • Proper use of test/playbooks/ for e2e testing');
    lines.push('   • Good mix of test types where present');
    lines.push('   • Focus on increasing test coverage and pass rates');
  }

  return lines.join('\n');
}

module.exports = {
  formatValidationIssues,
  formatScoreSummary,
  formatQualityAssessment,
  formatBaseMetrics,
  formatDetailedMetrics,
  formatActionPlan,
  formatComparison
};
