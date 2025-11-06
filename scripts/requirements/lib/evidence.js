#!/usr/bin/env node
'use strict';

/**
 * Phase results and test evidence loading
 * @module requirements/lib/evidence
 */

const fs = require('node:fs');
const path = require('node:path');
const { normalizeRequirementStatus } = require('./utils');

/**
 * Detect the source of a validation entry
 * @param {import('./types').Validation} validation - Validation object
 * @returns {import('./types').ValidationSource|null} Source information or null
 */
function detectValidationSource(validation) {
  const ref = (validation.ref || '').toLowerCase();
  const workflowId = (validation.workflow_id || '').toLowerCase();

  if (!ref && !workflowId) {
    return null;
  }

  if (validation.type === 'test') {
    const phaseMatch = ref.match(/test\/phases\/test-([a-z0-9_-]+)\.sh$/);
    if (phaseMatch) {
      return { kind: 'phase', name: phaseMatch[1] };
    }
    // Recognize vitest test files (ui/src/.../*.test.{ts,tsx})
    if (ref.startsWith('ui/src/') && (ref.endsWith('.test.ts') || ref.endsWith('.test.tsx'))) {
      return { kind: 'phase', name: 'unit' };
    }
    // Recognize Go test files
    if (ref.endsWith('_test.go') || ref.includes('/tests/')) {
      return { kind: 'phase', name: 'unit' };
    }
  }

  if (validation.type === 'automation') {
    if (ref) {
      const slug = path.basename(ref, path.extname(ref));
      return { kind: 'automation', name: slug };
    }
    if (workflowId) {
      return { kind: 'automation', name: workflowId };
    }
  }

  return null;
}

/**
 * Load phase results from coverage/phase-results directory
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {import('./types').EnrichmentContext} Phase results and requirement evidence
 */
function loadPhaseResults(scenarioRoot) {
  const results = {};
  const resultsDir = path.join(scenarioRoot, 'coverage', 'phase-results');
  if (!fs.existsSync(resultsDir)) {
    return { phaseResults: results, requirementEvidence: {} };
  }

  const files = fs.readdirSync(resultsDir).filter((name) => name.endsWith('.json'));
  const requirementEvidence = {};

  for (const file of files) {
    const fullPath = path.join(resultsDir, file);
    try {
      const parsed = JSON.parse(fs.readFileSync(fullPath, 'utf8'));
      if (parsed && typeof parsed.phase === 'string') {
        results[parsed.phase] = parsed;
        if (Array.isArray(parsed.requirements)) {
          parsed.requirements.forEach((entry) => {
            if (!entry || typeof entry !== 'object') {
              return;
            }
            const requirementId = (entry.id || '').trim();
            if (!requirementId) {
              return;
            }
            const status = normalizeRequirementStatus(entry.status);
            const evidenceRecord = {
              id: requirementId,
              status,
              phase: parsed.phase,
              evidence: entry.evidence || null,
              updated_at: entry.updated_at || parsed.updated_at || null,
              duration_seconds: entry.duration_seconds || parsed.duration_seconds || parsed.duration || null,
            };
            if (!requirementEvidence[requirementId]) {
              requirementEvidence[requirementId] = [];
            }
            requirementEvidence[requirementId].push(evidenceRecord);
          });
        }
      }
    } catch (error) {
      console.warn(`requirements/report: failed to parse phase results ${fullPath}: ${error.message}`);
    }
  }

  return { phaseResults: results, requirementEvidence };
}

/**
 * Extract vitest test file references from vitest-requirements.json
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {Map<string, import('./types').VitestFile[]>} Map of requirement ID to test files
 */
function extractVitestFilesFromPhaseResults(scenarioRoot) {
  const vitestReportPath = path.join(scenarioRoot, 'ui/coverage/vitest-requirements.json');
  if (!fs.existsSync(vitestReportPath)) {
    return new Map();
  }

  const vitestReport = JSON.parse(fs.readFileSync(vitestReportPath, 'utf8'));
  const vitestFiles = new Map();

  if (!Array.isArray(vitestReport.requirements)) {
    return vitestFiles;
  }

  vitestReport.requirements.forEach(reqResult => {
    const evidence = reqResult.evidence || '';

    // Extract unique test file paths from evidence
    // Evidence format: "src/components/__tests__/ProjectModal.test.tsx:Suite > Test; src/stores/..."
    const fileRefs = new Set();

    // Match all file paths in the evidence string
    const fileMatches = evidence.matchAll(/([^;]+\.test\.tsx?)/g);

    for (const match of fileMatches) {
      // Extract just the file path part before the colon
      const fullMatch = match[0];
      const colonIndex = fullMatch.indexOf(':');
      const filePath = colonIndex > 0 ? fullMatch.substring(0, colonIndex).trim() : fullMatch.trim();

      // Prepend "ui/" if not already present
      const testRef = filePath.startsWith('ui/') ? filePath : `ui/${filePath}`;

      // Only track ui/ vitest files
      if (testRef.startsWith('ui/src/') && testRef.match(/\.test\.(ts|tsx)$/)) {
        fileRefs.add(testRef);
      }
    }

    // Add all unique file refs for this requirement
    if (fileRefs.size > 0) {
      if (!vitestFiles.has(reqResult.id)) {
        vitestFiles.set(reqResult.id, []);
      }

      fileRefs.forEach(testRef => {
        vitestFiles.get(reqResult.id).push({
          ref: testRef,
          phase: 'unit',
          status: reqResult.status === 'passed' ? 'implemented' : 'failing',
        });
      });
    }
  });

  return vitestFiles;
}

/**
 * Collect validations for a specific phase
 * @param {import('./types').Requirement[]} requirements - Requirements array
 * @param {string} phaseName - Phase name to filter by
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {Array<{id: string, criticality: string, validations: Array}>} Phase validations
 */
function collectValidationsForPhase(requirements, phaseName, scenarioRoot) {
  if (!Array.isArray(requirements) || !phaseName) {
    return [];
  }

  const normalizedPhase = phaseName.trim().toLowerCase();
  const grouped = new Map();

  requirements.forEach((requirement) => {
    if (!Array.isArray(requirement.validations)) {
      return;
    }

    requirement.validations.forEach((validation) => {
      if (!validation || typeof validation !== 'object') {
        return;
      }

      const source = detectValidationSource(validation);
      const explicitPhase = (validation.phase || '').trim().toLowerCase();
      const phaseMatch = (explicitPhase && explicitPhase === normalizedPhase) ||
        (source && source.kind === 'phase' && source.name === normalizedPhase);
      const ref = validation.ref || '';
      const workflowId = validation.workflow_id || '';
      const referenceLabel = ref || workflowId;
      const refNormalized = ref.replace(/\\/g, '/').toLowerCase();
      const directMatch = refNormalized === `test/phases/test-${normalizedPhase}.sh`;

      if (!phaseMatch && !directMatch) {
        return;
      }

      const resolved = ref ? path.resolve(scenarioRoot, ref) : '';
      const exists = ref ? fs.existsSync(resolved) : false;

      if (!grouped.has(requirement.id)) {
        grouped.set(requirement.id, {
          id: requirement.id,
          criticality: requirement.criticality || '',
          validations: [],
        });
      }

      const bucket = grouped.get(requirement.id);
      bucket.validations.push({
        type: validation.type || '',
        ref,
        workflow_id: workflowId,
        reference: referenceLabel,
        status: validation.status || '',
        exists,
        scenario: validation.scenario || '',
        folder: validation.folder || '',
        notes: validation.notes || '',
        metadata: validation.metadata || null,
      });
    });
  });

  return Array.from(grouped.values());
}

module.exports = {
  detectValidationSource,
  loadPhaseResults,
  extractVitestFilesFromPhaseResults,
  collectValidationsForPhase,
};
