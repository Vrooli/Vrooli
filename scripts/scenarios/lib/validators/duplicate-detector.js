'use strict';

/**
 * Duplicate detector - Detects when test files validate multiple requirements
 * @module scenarios/lib/validators/duplicate-detector
 */

/**
 * Analyze test ref usage across requirements
 * @param {array} requirements - Array of requirement objects
 * @returns {object} Analysis results with violations
 */
function analyzeTestRefUsage(requirements) {
  const refUsage = new Map();  // ref → [req_id, req_id, ...]

  requirements.forEach(req => {
    (req.validation || []).forEach(v => {
      const ref = v.ref || v.workflow_id;
      if (!ref) return;

      if (!refUsage.has(ref)) {
        refUsage.set(ref, []);
      }
      refUsage.get(ref).push(req.id);
    });
  });

  // Detect violations (1 test file validates ≥4 requirements)
  const violations = [];
  refUsage.forEach((reqIds, ref) => {
    if (reqIds.length >= 4) {  // Threshold: 4+ requirements per test file
      violations.push({
        test_ref: ref,
        shared_by: reqIds,
        count: reqIds.length,
        severity: reqIds.length >= 6 ? 'high' : 'medium'
      });
    }
  });

  return {
    total_refs: refUsage.size,
    total_requirements: requirements.length,
    average_reqs_per_ref: requirements.length / Math.max(refUsage.size, 1),
    violations,
    duplicate_ratio: violations.length / Math.max(requirements.length, 1)
  };
}

module.exports = {
  analyzeTestRefUsage
};
