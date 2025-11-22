'use strict';

/**
 * Layer detector - Identifies which validation layers cover a requirement
 * @module scenarios/lib/validators/layer-detector
 */

const VALIDATION_LAYERS = {
  API: {
    patterns: [
      /\/?api\/.*_test\.go$/,           // Make leading slash optional
      /\/?api\/.*\/tests\//,             // Make leading slash optional
    ],
    description: 'API unit tests (Go)'
  },
  UI: {
    patterns: [
      /\/?ui\/src\/.*\.test\.(ts|tsx|js|jsx)$/,  // Make leading slash optional
    ],
    description: 'UI unit tests (Vitest/Jest)'
  },
  E2E: {
    patterns: [
      /\/?test\/playbooks\/.*\.(json|yaml)$/,    // Make leading slash optional
    ],
    description: 'End-to-end automation (BAS playbooks)'
  }
};

/**
 * Derive requirement criticality from operational target reference
 * @param {object} requirement - Requirement object
 * @returns {string} Criticality level (P0, P1, P2)
 */
function deriveRequirementCriticality(requirement) {
  const prdRef = requirement.prd_ref || '';
  const match = prdRef.match(/OT-([Pp][0-2])-\d{3}/);

  if (match) {
    return match[1].toUpperCase();  // Extract P0, P1, or P2 from OT-P0-001
  }

  // Default to P2 if no operational target reference
  return 'P2';
}

/**
 * Detect which validation layers a requirement is tested in (basic version without quality checks)
 * @param {object} requirement - Requirement object with validations
 * @returns {Set<string>} Set of layer names (API, UI, E2E)
 */
function detectValidationLayersBasic(requirement) {
  const layers = new Set();

  (requirement.validation || []).forEach(v => {
    const ref = (v.ref || '').toLowerCase();

    // Check API layer
    if (VALIDATION_LAYERS.API.patterns.some(p => p.test(ref))) {
      layers.add('API');
    }

    // Check UI layer
    if (VALIDATION_LAYERS.UI.patterns.some(p => p.test(ref))) {
      layers.add('UI');
    }

    // Check E2E layer
    if (VALIDATION_LAYERS.E2E.patterns.some(p => p.test(ref))) {
      layers.add('E2E');
    }

    // Check manual layer
    if (v.type === 'manual') {
      layers.add('MANUAL');
    }
  });

  return layers;
}

/**
 * Check if requirement meets diversity requirements
 * @param {object} requirement - Requirement object
 * @param {Set<string>} applicableLayers - Layers applicable to scenario
 * @param {string} criticality - Derived criticality (P0, P1, P2)
 * @returns {{passes: boolean, layers: Set<string>, minRequired: number}}
 */
function checkDiversityRequirement(requirement, applicableLayers, criticality) {
  const minLayers = (criticality === 'P0' || criticality === 'P1') ? 2 : 1;
  const layers = detectValidationLayersBasic(requirement);

  // Filter to applicable layers only
  const applicable = new Set();
  layers.forEach(layer => {
    if (applicableLayers.has(layer)) {
      applicable.add(layer);
    }
  });

  return {
    passes: applicable.size >= minLayers,
    layers: applicable,
    minRequired: minLayers
  };
}

/**
 * Detect validation layers with quality filtering (full version)
 * @param {object} requirement - Requirement object with validations
 * @param {string} scenarioRoot - Scenario root for quality checks
 * @returns {{automated: Set<string>, has_manual: boolean}}
 */
function detectValidationLayers(requirement, scenarioRoot) {
  const testQualityAnalyzer = require('./test-quality-analyzer');
  const automatedLayers = new Set();
  const hasManual = (requirement.validation || []).some(v => v.type === 'manual');

  (requirement.validation || []).forEach(v => {
    const refOriginal = v.ref || '';  // Keep original case for file operations
    const ref = refOriginal.toLowerCase();  // Lowercase for pattern matching

    // IMPORTANT: Skip manual validations for layer diversity
    // Manual validations are temporary measures, not automated test evidence
    if (v.type === 'manual') {
      return;
    }

    // Skip validations that don't reference actual test files
    if (!refOriginal && !v.workflow_id) {
      return;
    }

    // Reject unsupported test/ directories early (except test/playbooks/)
    // These are already rejected by evidence.js during loading, but may still exist in requirement JSON
    if (refOriginal.startsWith('test/')) {
      const isPlaybook = refOriginal.startsWith('test/playbooks/') &&
                         (refOriginal.endsWith('.json') || refOriginal.endsWith('.yaml'));
      if (!isPlaybook) {
        // Silently skip (already warned by evidence.js during loading)
        return;
      }
    }

    // Check quality for test type validations with file refs
    if (v.type === 'test' && refOriginal) {
      // Only quality-check actual test files (prevent checking non-test files marked as type="test")
      const isActualTestFile =
        refOriginal.endsWith('_test.go') ||
        refOriginal.includes('/tests/') ||
        refOriginal.match(/\.test\.(ts|tsx|js|jsx)$/);

      if (isActualTestFile) {
        const quality = testQualityAnalyzer.analyzeTestFileQuality(refOriginal, scenarioRoot);
        if (!quality.is_meaningful) {
          // Don't count low-quality tests toward layer diversity
          // Note: Low-quality tests are reported separately via superficial_test_implementation detection
          return;
        }
      } else {
        // Silently skip non-test files with type="test" (data issue in requirements)
        return;
      }
    }

    // Check quality for automation type validations (playbooks)
    if (v.type === 'automation' && refOriginal && refOriginal.match(/\.(json|yaml)$/)) {
      const quality = testQualityAnalyzer.analyzePlaybookQuality(refOriginal, scenarioRoot);
      if (!quality.is_meaningful) {
        // Don't count low-quality playbooks toward layer diversity
        // Note: Low-quality playbooks are reported separately via superficial_test_implementation detection
        return;
      }
    }

    // Check API layer (use lowercase ref for pattern matching)
    if (VALIDATION_LAYERS.API.patterns.some(p => p.test(ref))) {
      automatedLayers.add('API');
    }

    // Check UI layer (use lowercase ref for pattern matching)
    if (VALIDATION_LAYERS.UI.patterns.some(p => p.test(ref))) {
      automatedLayers.add('UI');
    }

    // Check E2E layer (use lowercase ref for pattern matching)
    if (VALIDATION_LAYERS.E2E.patterns.some(p => p.test(ref))) {
      automatedLayers.add('E2E');
    }
  });

  return {
    automated: automatedLayers,  // Use this for diversity requirement
    has_manual: hasManual        // Track separately, don't count toward diversity
  };
}

module.exports = {
  VALIDATION_LAYERS,
  deriveRequirementCriticality,
  detectValidationLayersBasic,
  checkDiversityRequirement,
  detectValidationLayers
};
