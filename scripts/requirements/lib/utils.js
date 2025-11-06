#!/usr/bin/env node
'use strict';

/**
 * Shared utility functions for requirements processing
 * @module requirements/lib/utils
 */

const REQUIREMENT_STATUS_PRIORITY = {
  failed: 4,
  skipped: 3,
  passed: 2,
  not_run: 1,
  unknown: 0,
};

/**
 * Normalize requirement status values to canonical form
 * @param {string|null|undefined} value - Status value to normalize
 * @returns {string} Normalized status
 */
function normalizeRequirementStatus(value) {
  if (value === undefined || value === null) {
    return 'unknown';
  }
  const normalized = String(value).trim().toLowerCase();
  switch (normalized) {
    case 'pass':
    case 'passed':
    case 'success':
      return 'passed';
    case 'fail':
    case 'failed':
    case 'error':
    case 'failure':
    case 'errored':
      return 'failed';
    case 'skip':
    case 'skipped':
      return 'skipped';
    case 'not_run':
    case 'not-run':
    case 'not run':
    case 'pending':
    case 'planned':
      return 'not_run';
    case 'passed_with_warnings':
      return 'passed';
    default:
      return normalized || 'unknown';
  }
}

/**
 * Compare two evidence records and return the one with higher priority
 * @param {import('./types').EvidenceRecord|null} current - Current evidence
 * @param {import('./types').EvidenceRecord|null} candidate - Candidate evidence
 * @returns {import('./types').EvidenceRecord|null} Evidence with higher priority
 */
function compareRequirementEvidence(current, candidate) {
  if (!candidate) {
    return current || null;
  }
  if (!current) {
    return candidate;
  }
  const currentScore = REQUIREMENT_STATUS_PRIORITY[current.status] ?? 0;
  const candidateScore = REQUIREMENT_STATUS_PRIORITY[candidate.status] ?? 0;
  if (candidateScore > currentScore) {
    return candidate;
  }
  if (candidateScore === currentScore) {
    const currentTime = current.updated_at ? Date.parse(current.updated_at) : 0;
    const candidateTime = candidate.updated_at ? Date.parse(candidate.updated_at) : 0;
    if (candidateTime > currentTime) {
      return candidate;
    }
  }
  return current;
}

/**
 * Select the best evidence record from an array
 * @param {import('./types').EvidenceRecord[]} records - Evidence records
 * @param {string|null} [phaseName] - Optional phase name to filter by
 * @returns {import('./types').EvidenceRecord|null} Best evidence record
 */
function selectBestRequirementEvidence(records, phaseName = null) {
  if (!Array.isArray(records) || records.length === 0) {
    return null;
  }

  let bestForPhase = null;
  if (phaseName) {
    records.forEach((record) => {
      if (record.phase === phaseName) {
        bestForPhase = compareRequirementEvidence(bestForPhase, record);
      }
    });
  }

  if (bestForPhase) {
    return bestForPhase;
  }

  let bestOverall = null;
  records.forEach((record) => {
    bestOverall = compareRequirementEvidence(bestOverall, record);
  });

  return bestOverall;
}

/**
 * Derive declared status rollup from child statuses
 * @param {string[]} childStatuses - Child requirement statuses
 * @param {string} [fallbackStatus] - Fallback status if no children
 * @returns {string} Rolled up status
 */
function deriveDeclaredRollup(childStatuses, fallbackStatus) {
  if (!Array.isArray(childStatuses) || childStatuses.length === 0) {
    return fallbackStatus || 'pending';
  }

  let allComplete = true;
  let anyInProgress = false;
  let anyComplete = false;
  let anyPending = false;
  let anyPlanned = false;

  childStatuses.forEach((statusRaw) => {
    const status = (statusRaw || '').toLowerCase();
    if (status !== 'complete') {
      allComplete = false;
    }
    switch (status) {
      case 'complete':
        anyComplete = true;
        break;
      case 'in_progress':
        anyInProgress = true;
        break;
      case 'planned':
      case 'not_implemented':
        anyPlanned = true;
        break;
      case 'pending':
      case 'incomplete':
      case 'in_review':
        anyPending = true;
        break;
      default:
        if (status && status !== 'complete') {
          anyPending = true;
        }
        if (!status) {
          allComplete = false;
        }
        break;
    }
  });

  if (allComplete) {
    return 'complete';
  }

  if (anyInProgress) {
    return 'in_progress';
  }

  if (anyComplete && (anyPending || anyPlanned)) {
    return 'in_progress';
  }

  if (anyPending) {
    return 'pending';
  }

  if (anyPlanned) {
    return 'planned';
  }

  if (anyComplete) {
    return 'in_progress';
  }

  return fallbackStatus || 'pending';
}

/**
 * Derive live status rollup from child live statuses
 * @param {string[]} childStatuses - Child live statuses
 * @param {string} [fallbackStatus] - Fallback status if no children
 * @returns {string} Rolled up live status
 */
function deriveLiveRollup(childStatuses, fallbackStatus) {
  if (!Array.isArray(childStatuses) || childStatuses.length === 0) {
    return fallbackStatus || 'unknown';
  }

  const normalized = childStatuses
    .map((status) => (status ? status.toLowerCase() : 'unknown'));

  if (normalized.some((status) => status === 'failed')) {
    return 'failed';
  }

  if (normalized.every((status) => status === 'passed')) {
    return 'passed';
  }

  if (normalized.every((status) => status === 'passed' || status === 'skipped')) {
    return normalized.includes('passed') ? 'passed' : 'skipped';
  }

  if (normalized.some((status) => status === 'skipped')) {
    return 'skipped';
  }

  if (normalized.some((status) => status === 'not_run')) {
    return 'not_run';
  }

  if (normalized.every((status) => status === 'unknown')) {
    return 'unknown';
  }

  if (normalized.some((status) => status === 'passed')) {
    return 'not_run';
  }

  return fallbackStatus || 'unknown';
}

/**
 * Derive validation status from live and original status
 * @param {import('./types').Validation} validation - Validation object
 * @returns {string} Derived status
 */
function deriveValidationStatus(validation) {
  const originalStatus = validation.status || '';
  const liveStatus = validation.liveStatus || '';

  if (liveStatus === 'passed') {
    return 'implemented';
  }
  if (liveStatus === 'failed') {
    return 'failing';
  }
  if (liveStatus === 'skipped') {
    // Keep successful historical status, otherwise fall back to planned work.
    return originalStatus && originalStatus !== 'not_implemented' ? originalStatus : 'planned';
  }
  if ((liveStatus === 'not_run' || liveStatus === 'unknown') && originalStatus) {
    return originalStatus;
  }
  if (!originalStatus) {
    return 'not_implemented';
  }
  return originalStatus;
}

/**
 * Derive requirement status from validation statuses
 * @param {string} existingStatus - Current requirement status
 * @param {string[]} validationStatuses - Array of validation statuses
 * @returns {string} Derived requirement status
 */
function deriveRequirementStatus(existingStatus, validationStatuses) {
  if (!Array.isArray(validationStatuses) || validationStatuses.length === 0) {
    return existingStatus;
  }

  const hasFailing = validationStatuses.some((status) => status === 'failing');
  const allImplemented = validationStatuses.every((status) => status === 'implemented');
  const hasImplemented = validationStatuses.some((status) => status === 'implemented');

  if (hasFailing) {
    return 'in_progress';
  }
  if (allImplemented) {
    return 'complete';
  }
  if (hasImplemented && (existingStatus === 'pending' || existingStatus === 'not_implemented')) {
    return 'in_progress';
  }

  return existingStatus;
}

module.exports = {
  REQUIREMENT_STATUS_PRIORITY,
  normalizeRequirementStatus,
  compareRequirementEvidence,
  selectBestRequirementEvidence,
  deriveDeclaredRollup,
  deriveLiveRollup,
  deriveValidationStatus,
  deriveRequirementStatus,
};
