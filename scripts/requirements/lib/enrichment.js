#!/usr/bin/env node
'use strict';

/**
 * Status enrichment and aggregation for requirements
 * @module requirements/lib/enrichment
 */

const {
  REQUIREMENT_STATUS_PRIORITY,
  selectBestRequirementEvidence,
  deriveDeclaredRollup,
  deriveLiveRollup
} = require('./utils');
const { detectValidationSource } = require('./evidence');

/**
 * Enrich validations with live test results
 * @param {import('./types').Requirement[]} requirements - Requirements to enrich
 * @param {import('./types').EnrichmentContext} context - Enrichment context
 */
function enrichValidationResults(requirements, context) {
  const phaseResults = context.phaseResults || {};
  const requirementEvidence = context.requirementEvidence || {};
  const statusPriority = REQUIREMENT_STATUS_PRIORITY;

  requirements.forEach((requirement) => {
    const evidenceRecords = requirementEvidence[requirement.id] || [];
    if (evidenceRecords.length > 0) {
      requirement.liveEvidence = evidenceRecords;
    }

    if (!Array.isArray(requirement.validations)) {
      const bestOnlyEvidence = selectBestRequirementEvidence(evidenceRecords);
      requirement.liveStatus = bestOnlyEvidence ? bestOnlyEvidence.status : 'unknown';
      return;
    }

    let aggregateStatus = 'unknown';
    let aggregateScore = statusPriority[aggregateStatus];

    requirement.validations = requirement.validations.map((validation) => {
      const validationMeta = validation.__meta;
      const enriched = { ...validation };
      if (validationMeta) {
        Object.defineProperty(enriched, '__meta', {
          value: validationMeta,
          enumerable: false,
        });
      }
      const source = detectValidationSource(validation);
      enriched.liveSource = source || null;

      let liveStatus = 'unknown';
      let liveDetails = null;

      if (source && source.kind === 'phase') {
        const phaseResult = phaseResults[source.name];
        const phaseEvidence = selectBestRequirementEvidence(evidenceRecords, source.name);
        if (phaseEvidence) {
          liveStatus = phaseEvidence.status;
          liveDetails = {
            updated_at: phaseEvidence.updated_at || (phaseResult && phaseResult.updated_at) || null,
            duration_seconds:
              phaseEvidence.duration_seconds || (phaseResult && (phaseResult.duration_seconds || phaseResult.duration)) || null,
            requirement: {
              id: requirement.id,
              phase: phaseEvidence.phase,
              status: phaseEvidence.status,
              evidence: phaseEvidence.evidence || null,
            },
          };
        } else if (phaseResult && typeof phaseResult.status === 'string') {
          liveStatus = phaseResult.status === 'passed' ? 'passed' : 'failed';
          liveDetails = {
            updated_at: phaseResult.updated_at || null,
            duration_seconds: phaseResult.duration_seconds || phaseResult.duration || null,
          };
        } else {
          liveStatus = 'not_run';
        }
      } else if (source && source.kind === 'automation') {
        const automationEvidence = selectBestRequirementEvidence(evidenceRecords);
        if (automationEvidence) {
          liveStatus = automationEvidence.status;
          liveDetails = {
            updated_at: automationEvidence.updated_at || null,
            duration_seconds: automationEvidence.duration_seconds || null,
            requirement: {
              id: requirement.id,
              phase: automationEvidence.phase,
              status: automationEvidence.status,
              evidence: automationEvidence.evidence || null,
            },
          };
        } else {
          liveStatus = 'not_run';
        }
      } else {
        const genericEvidence = selectBestRequirementEvidence(evidenceRecords);
        if (genericEvidence) {
          liveStatus = genericEvidence.status;
          liveDetails = {
            updated_at: genericEvidence.updated_at || null,
            duration_seconds: genericEvidence.duration_seconds || null,
            requirement: {
              id: requirement.id,
              phase: genericEvidence.phase,
              status: genericEvidence.status,
              evidence: genericEvidence.evidence || null,
            },
          };
        }
      }

      enriched.liveStatus = liveStatus;
      if (liveDetails) {
        enriched.liveDetails = liveDetails;
      }

      const score = statusPriority[liveStatus] ?? 0;
      if (score > aggregateScore) {
        aggregateStatus = liveStatus;
        aggregateScore = score;
      }

      return enriched;
    });

    const bestOverallEvidence = selectBestRequirementEvidence(evidenceRecords);
    if (bestOverallEvidence) {
      const evidenceScore = statusPriority[bestOverallEvidence.status] ?? 0;
      if (evidenceScore > aggregateScore) {
        aggregateStatus = bestOverallEvidence.status;
        aggregateScore = evidenceScore;
      }
    }

    requirement.liveStatus = aggregateStatus;
  });
}

/**
 * Aggregate requirement statuses based on parent/child relationships
 * @param {import('./types').Requirement[]} requirements - Requirements to aggregate
 * @param {Map<string, import('./types').Requirement>} requirementIndex - Requirement index
 */
function aggregateRequirementStatuses(requirements, requirementIndex) {
  if (!Array.isArray(requirements) || !requirements.length) {
    return;
  }

  const declaredCache = new Map();
  const liveCache = new Map();

  const resolvingDeclared = new Set();
  const resolvingLive = new Set();

  /**
   * Resolve declared status with cycle detection
   * @param {import('./types').Requirement} req - Requirement to resolve
   * @returns {string} Resolved status
   */
  function resolveDeclared(req) {
    if (!req || !req.id) {
      return 'pending';
    }
    if (declaredCache.has(req.id)) {
      return declaredCache.get(req.id);
    }
    if (!Array.isArray(req.children) || req.children.length === 0) {
      const status = req.status || 'pending';
      declaredCache.set(req.id, status);
      return status;
    }

    if (resolvingDeclared.has(req.id)) {
      return req.status || 'pending';
    }
    resolvingDeclared.add(req.id);

    const childStatuses = [];
    req.children.forEach((childId) => {
      const child = requirementIndex.get(childId);
      if (!child) {
        console.warn(`requirements/report: child requirement '${childId}' referenced by '${req.id}' is missing`);
        return;
      }
      childStatuses.push(resolveDeclared(child));
    });

    resolvingDeclared.delete(req.id);

    const aggregated = deriveDeclaredRollup(childStatuses, req.status);
    req.status = aggregated;
    declaredCache.set(req.id, aggregated);
    return aggregated;
  }

  /**
   * Resolve live status with cycle detection
   * @param {import('./types').Requirement} req - Requirement to resolve
   * @returns {string} Resolved live status
   */
  function resolveLive(req) {
    if (!req || !req.id) {
      return req && req.liveStatus ? req.liveStatus : 'unknown';
    }
    if (liveCache.has(req.id)) {
      return liveCache.get(req.id);
    }
    if (!Array.isArray(req.children) || req.children.length === 0) {
      const status = req.liveStatus || 'unknown';
      liveCache.set(req.id, status);
      return status;
    }

    if (resolvingLive.has(req.id)) {
      return req.liveStatus || 'unknown';
    }
    resolvingLive.add(req.id);

    const childStatuses = [];
    req.children.forEach((childId) => {
      const child = requirementIndex.get(childId);
      if (!child) {
        return;
      }
      childStatuses.push(resolveLive(child));
    });

    resolvingLive.delete(req.id);

    const aggregated = deriveLiveRollup(childStatuses, req.liveStatus);
    req.liveStatus = aggregated;
    liveCache.set(req.id, aggregated);
    return aggregated;
  }

  requirements.forEach((req) => resolveDeclared(req));
  requirements.forEach((req) => resolveLive(req));
}

/**
 * Compute summary statistics for requirements
 * @param {import('./types').Requirement[]} records - Requirements to summarize
 * @returns {import('./types').Summary} Summary statistics
 */
function computeSummary(records) {
  const byStatus = {};
  const liveStatus = {};
  let criticalityGap = 0;

  for (const record of records) {
    const key = record.status || 'unknown';
    byStatus[key] = (byStatus[key] || 0) + 1;

    const liveKey = record.liveStatus || 'unknown';
    liveStatus[liveKey] = (liveStatus[liveKey] || 0) + 1;

    if (record.criticality && record.criticality.toUpperCase() !== 'P2') {
      if (record.status !== 'complete') {
        criticalityGap += 1;
      }
    }
  }

  return {
    total: records.length,
    byStatus,
    liveStatus,
    criticalityGap,
  };
}

module.exports = {
  enrichValidationResults,
  aggregateRequirementStatuses,
  computeSummary,
};
