'use strict';

/**
 * Target grouping validator - Validates operational target grouping patterns
 * @module scenarios/lib/validators/target-grouping-validator
 */

/**
 * Analyze operational target grouping patterns
 * @param {array} targets - Array of operational target objects
 * @param {array} requirements - Array of requirement objects
 * @returns {object} Analysis results
 */
function analyzeTargetGrouping(targets, requirements) {
  const targetMap = new Map();  // OT-XXX → [req_id, ...]

  // Build target-to-requirements mapping
  requirements.forEach(req => {
    const match = (req.prd_ref || '').match(/OT-[Pp][0-2]-\d{3}/);
    if (match) {
      const targetId = match[0].toUpperCase();
      if (!targetMap.has(targetId)) {
        targetMap.set(targetId, []);
      }
      targetMap.get(targetId).push(req.id);
    }
  });

  // Count 1:1 mappings
  const oneToOneMappings = Array.from(targetMap.entries())
    .filter(([targetId, reqIds]) => reqIds.length === 1);

  const totalTargets = targetMap.size;
  const oneToOneCount = oneToOneMappings.length;
  const oneToOneRatio = totalTargets > 0 ? oneToOneCount / totalTargets : 0;

  // Dynamic threshold: min(20%, 5/total_targets)
  const acceptableRatio = Math.min(0.2, totalTargets > 0 ? 5 / totalTargets : 0);

  const violations = [];
  if (oneToOneRatio > acceptableRatio) {
    violations.push({
      type: 'excessive_one_to_one_mappings',
      current_ratio: oneToOneRatio,
      acceptable_ratio: acceptableRatio,
      one_to_one_count: oneToOneCount,
      total_targets: totalTargets,
      message: `${Math.round(oneToOneRatio * 100)}% of targets have 1:1 requirement mapping (max ${Math.round(acceptableRatio * 100)}% recommended)`,
      targets: oneToOneMappings.map(([targetId, reqIds]) => ({
        target: targetId,
        requirement: reqIds[0]
      }))
    });
  }

  return {
    total_targets: totalTargets,
    one_to_one_count: oneToOneCount,
    one_to_one_ratio: oneToOneRatio,
    acceptable_ratio: acceptableRatio,
    average_requirements_per_target: requirements.length / Math.max(totalTargets, 1),
    violations
  };
}

module.exports = {
  analyzeTargetGrouping
};
