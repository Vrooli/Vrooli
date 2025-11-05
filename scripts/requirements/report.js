#!/usr/bin/env node

/**
 * requirements/report.js
 *
 * Lightweight coverage reporter for scenario requirements.
 * - Parses either <scenario>/docs/requirements.json or <scenario>/requirements/<module>.json files
 *   (index-first) to support modular requirement registries
 * - Computes aggregate counts by status and criticality gap
 * - Reads live phase results from coverage/phase-results to surface pass/fail state
 * - Emits JSON (default) or markdown/trace outputs summarising the requirements
 */

'use strict';

const fs = require('node:fs');
const path = require('node:path');

function parseArgs(argv) {
  const options = {
    scenario: '',
    format: 'json',
    includePending: false,
    output: '',
    mode: 'report',
    phase: '',
    pruneStale: false,
  };

  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    switch (arg) {
      case '--scenario':
        options.scenario = argv[i + 1] || '';
        i += 1;
        break;
      case '--format': {
        const value = (argv[i + 1] || 'json').toLowerCase();
        if (value === 'markdown' || value === 'json' || value === 'trace') {
          options.format = value;
        }
        i += 1;
        break;
      }
      case '--mode': {
        const mode = (argv[i + 1] || '').toLowerCase();
        if (['report', 'phase-inspect', 'validate', 'sync'].includes(mode)) {
          options.mode = mode;
        }
        i += 1;
        break;
      }
      case '--phase':
        options.phase = argv[i + 1] || '';
        i += 1;
        break;
      case '--include-pending':
        options.includePending = true;
        break;
      case '--output':
        options.output = argv[i + 1] || '';
        i += 1;
        break;
      case '--prune-stale':
        options.pruneStale = true;
        break;
      default:
        break;
    }
  }

  if (!options.scenario) {
    throw new Error('Missing required --scenario argument');
  }

  return options;
}

function isScenarioRoot(directory, scenario) {
  if (!directory) {
    return false;
  }
  const basename = path.basename(directory);
  if (basename !== scenario) {
    return false;
  }
  const testDir = path.join(directory, 'test');
  return fs.existsSync(testDir);
}

function resolveScenarioRoot(baseDir, scenario) {
  let current = path.resolve(baseDir);

  while (true) {
    if (isScenarioRoot(current, scenario)) {
      return current;
    }

    const candidate = path.join(current, 'scenarios', scenario);
    if (fs.existsSync(candidate)) {
      return candidate;
    }

    const parent = path.dirname(current);
    if (parent === current) {
      break;
    }
    current = parent;
  }

  if (process.env.VROOLI_ROOT) {
    const candidate = path.join(process.env.VROOLI_ROOT, 'scenarios', scenario);
    if (fs.existsSync(candidate)) {
      return candidate;
    }
  }

  throw new Error(`Unable to locate scenario '${scenario}'`);
}

function parseImports(indexPath) {
  if (!fs.existsSync(indexPath)) {
    return [];
  }

  try {
    const content = fs.readFileSync(indexPath, 'utf8');
    const parsed = JSON.parse(content);
    return parsed.imports || [];
  } catch (error) {
    console.warn(`Failed to parse imports from ${indexPath}: ${error.message}`);
    return [];
  }
}

function collectRequirementFiles(scenarioRoot) {
  const docsPath = path.join(scenarioRoot, 'docs', 'requirements.json');
  if (fs.existsSync(docsPath)) {
    return [{ path: docsPath, relative: 'docs/requirements.json', isIndex: true }];
  }

  const requirementsDir = path.join(scenarioRoot, 'requirements');
  if (!fs.existsSync(requirementsDir)) {
    throw new Error('No requirements registry found (expected docs/requirements.json or requirements/)');
  }

  const results = [];
  const seen = new Set();
  const enqueue = (filePath, isIndex = false) => {
    if (!fs.existsSync(filePath)) {
      return;
    }
    const resolved = path.resolve(filePath);
    if (seen.has(resolved)) {
      return;
    }
    seen.add(resolved);
    const relative = path.relative(scenarioRoot, resolved);
    results.push({ path: resolved, relative, isIndex });
  };

  const indexPath = path.join(requirementsDir, 'index.json');
  if (fs.existsSync(indexPath)) {
    enqueue(indexPath, true);
    const imports = parseImports(indexPath);
    imports.forEach((importEntry) => {
      const candidate = path.join(requirementsDir, importEntry);
      enqueue(candidate, false);
    });
  }

  const stack = [requirementsDir];
  while (stack.length > 0) {
    const current = stack.pop();
    const entries = fs.readdirSync(current, { withFileTypes: true });
    entries.forEach((entry) => {
      if (entry.name.startsWith('.')) {
        return;
      }
      const fullPath = path.join(current, entry.name);
      if (entry.isDirectory()) {
        stack.push(fullPath);
        return;
      }
      if (!entry.isFile()) {
        return;
      }
      if (!entry.name.endsWith('.json')) {
        return;
      }
      enqueue(fullPath, path.resolve(fullPath) === path.resolve(indexPath));
    });
  }

  results.sort((a, b) => {
    if (a.isIndex && !b.isIndex) {
      return -1;
    }
    if (!a.isIndex && b.isIndex) {
      return 1;
    }
    return a.relative.localeCompare(b.relative);
  });

  return results;
}

function parseRequirementFile(filePath) {
  try {
    const content = fs.readFileSync(filePath, 'utf8');
    const parsed = JSON.parse(content);

    // Extract requirements array
    const requirements = (parsed.requirements || []).map((req) => {
      // Normalize requirement structure
      const requirement = {
        id: req.id || '',
        status: req.status || 'pending',
        criticality: req.criticality || '',
        title: req.title || '',
        category: req.category || '',
        validations: (req.validation || []).map((val) => ({
          type: val.type || '',
          ref: val.ref || '',
          status: val.status || '',
          notes: val.notes || '',
          phase: val.phase || '',
          workflow_id: val.workflow_id || '',
          scenario: val.scenario || '',
          folder: val.folder || '',
          metadata: val.metadata || null,
        })),
        children: req.children || [],
        description: req.description || '',
        prd_ref: req.prd_ref || '',
        tags: req.tags || [],
        depends_on: req.depends_on || [],
        blocks: req.blocks || [],
      };

      // Add metadata for tracking (used by sync operations)
      const meta = {
        filePath,
        originalStatus: requirement.status,
      };
      Object.defineProperty(requirement, '__meta', {
        value: meta,
        enumerable: false,
      });

      // Add metadata to validations
      requirement.validations.forEach((validation) => {
        const validationMeta = {
          filePath,
        };
        Object.defineProperty(validation, '__meta', {
          value: validationMeta,
          enumerable: false,
        });
      });

      return requirement;
    });

    return { requirements };
  } catch (error) {
    throw new Error(`Failed to parse ${filePath}: ${error.message}`);
  }
}

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

const REQUIREMENT_STATUS_PRIORITY = {
  failed: 4,
  skipped: 3,
  passed: 2,
  not_run: 1,
  unknown: 0,
};

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

function enrichValidationResults(requirements, context) {
  const phaseResults = context.phaseResults || {};
  const requirementEvidence = context.requirementEvidence || {};
  const statusPriority = { failed: 4, skipped: 3, passed: 2, not_run: 1, unknown: 0 };

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

function aggregateRequirementStatuses(requirements, requirementIndex) {
  if (!Array.isArray(requirements) || !requirements.length) {
    return;
  }

  const declaredCache = new Map();
  const liveCache = new Map();

  const resolvingDeclared = new Set();
  const resolvingLive = new Set();

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

function renderJSON(summary, requirements) {
  const output = {
    generatedAt: new Date().toISOString(),
    summary,
    requirements,
  };
  return `${JSON.stringify(output, null, 2)}\n`;
}

function renderMarkdown(summary, requirements, includePending) {
  const lines = [];
  lines.push('# Requirement Coverage');
  lines.push('');
  lines.push(`*Generated:* ${new Date().toISOString()}`);
  lines.push('');
  lines.push(`Total requirements: **${summary.total}**`);
  lines.push(`Criticality gap (P0/P1 incomplete): **${summary.criticalityGap}**`);
  lines.push('');
  lines.push('| ID | Status | Criticality | Live |');
  lines.push('| --- | --- | --- | --- |');

  const rows = includePending ? requirements : requirements.filter((r) => r.status !== 'pending');
  for (const req of rows) {
    lines.push(`| ${req.id} | ${req.status || 'pending'} | ${req.criticality || '-'} | ${req.liveStatus || 'unknown'} |`);
  }

  if (rows.length === 0) {
    lines.push('| _no entries_ |  |  |  |');
  }

  lines.push('');
  lines.push('## Status Breakdown');
  for (const [status, count] of Object.entries(summary.byStatus)) {
    lines.push(`- **${status}:** ${count}`);
  }

  lines.push('');
  lines.push('## Live Validation Status');
  for (const [status, count] of Object.entries(summary.liveStatus || {})) {
    lines.push(`- **${status}:** ${count}`);
  }

  lines.push('');
  return `${lines.join('\n')}\n`;
}

function renderTrace(requirements, scenarioRoot) {
  const lines = [];
  lines.push('# Requirement Traceability');
  lines.push('');
  lines.push(`*Generated:* ${new Date().toISOString()}`);
  lines.push('');

  requirements.forEach((req) => {
    lines.push(`## ${req.id}${req.title ? ` — ${req.title}` : ''}`);
    lines.push('');
    lines.push(`- **Status:** ${req.status || 'pending'}`);
    lines.push(`- **Criticality:** ${req.criticality || 'n/a'}`);
    if (req.category) {
      lines.push(`- **Category:** ${req.category}`);
    }
    lines.push('');

    if (Array.isArray(req.validations) && req.validations.length > 0) {
      lines.push('| Type | Reference | Status | Exists | Live | Notes |');
      lines.push('| --- | --- | --- | --- | --- | --- |');
      req.validations.forEach((validation) => {
        const reference = validation.ref || validation.workflow_id || '';
        let exists = '';
        const usesWorkflowId = !validation.ref && validation.workflow_id;
        if (reference && !usesWorkflowId) {
          const candidate = path.resolve(scenarioRoot, reference);
          exists = fs.existsSync(candidate) ? '✅' : '⚠️';
        } else if (usesWorkflowId) {
            exists = '—';
        }
        const liveStatus = validation.liveStatus || 'unknown';
        const liveSource = validation.liveSource && validation.liveSource.name ? ` (${validation.liveSource.name})` : '';
        const notes = validation.notes ? validation.notes.replace(/\|/g, '\\|') : '';
        lines.push(`| ${validation.type || '-'} | ${reference || '-'} | ${validation.status || '-'} | ${exists} | ${liveStatus}${liveSource} | ${notes || ''} |`);
      });
    } else {
      lines.push('_No validations listed._');
    }

    lines.push('');
  });

  return `${lines.join('\n')}\n`;
}

function renderPhaseInspect(options, requirements, scenarioRoot) {
  if (!options.phase) {
    throw new Error('phase-inspect mode requires --phase');
  }

  const phaseName = options.phase.trim().toLowerCase();
  const entries = collectValidationsForPhase(requirements, phaseName, scenarioRoot);

  const missingReferences = [];
  entries.forEach((entry) => {
    if (!Array.isArray(entry.validations)) {
      return;
    }
    entry.validations.forEach((validation) => {
      const ref = validation.ref || '';
      if (ref && validation.exists === false) {
        missingReferences.push(ref);
      }
    });
  });

  const payload = {
    scenario: options.scenario,
    phase: phaseName,
    requirements: entries,
  };

  if (missingReferences.length > 0) {
    payload.missingReferences = Array.from(new Set(missingReferences));
  }

  return `${JSON.stringify(payload, null, 2)}\n`;
}

// Removed replaceStatusValue - no longer needed with JSON

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

function syncRequirementFile(filePath, requirements) {
  if (!requirements || requirements.length === 0) {
    return [];
  }

  // Read the original JSON file
  const originalContent = fs.readFileSync(filePath, 'utf8');
  const parsed = JSON.parse(originalContent);
  const updates = [];

  // Create a map of requirement IDs to their index in the file
  const requirementMap = new Map();
  (parsed.requirements || []).forEach((req, idx) => {
    requirementMap.set(req.id, idx);
  });

  requirements.forEach((requirement) => {
    const requirementMeta = requirement.__meta;
    const validationStatuses = [];
    const originalStatus = requirementMeta && typeof requirementMeta.originalStatus === 'string'
      ? requirementMeta.originalStatus
      : requirement.status;

    // Update validation statuses
    if (Array.isArray(requirement.validations)) {
      requirement.validations.forEach((validation, index) => {
        const derivedStatus = deriveValidationStatus(validation);
        validationStatuses.push(derivedStatus);

        if (derivedStatus && derivedStatus !== validation.status) {
          validation.status = derivedStatus;
          updates.push({
            type: 'validation',
            requirement: requirement.id,
            index,
            status: derivedStatus,
            file: filePath,
          });
        }
      });
    }

    // Update requirement status based on validation rollup
    const nextStatus = deriveRequirementStatus(requirement.status, validationStatuses);
    if (nextStatus && nextStatus !== originalStatus) {
      requirement.status = nextStatus;
      if (requirementMeta) {
        requirementMeta.originalStatus = nextStatus;
      }
      updates.push({
        type: 'requirement',
        requirement: requirement.id,
        status: nextStatus,
        file: filePath,
      });
    }

    // Update the parsed JSON with new statuses
    const reqIndex = requirementMap.get(requirement.id);
    if (reqIndex !== undefined && parsed.requirements[reqIndex]) {
      parsed.requirements[reqIndex].status = requirement.status;

      // Update validation statuses in parsed JSON
      if (Array.isArray(requirement.validations) && Array.isArray(parsed.requirements[reqIndex].validation)) {
        requirement.validations.forEach((validation, idx) => {
          if (parsed.requirements[reqIndex].validation[idx]) {
            parsed.requirements[reqIndex].validation[idx].status = validation.status;
          }
        });
      }
    }
  });

  // Write back to file if there were updates
  if (updates.length > 0) {
    // Update metadata
    if (!parsed._metadata) {
      parsed._metadata = {};
    }
    parsed._metadata.last_synced_at = new Date().toISOString();

    // Write with 2-space indentation
    fs.writeFileSync(filePath, JSON.stringify(parsed, null, 2) + '\n', 'utf8');
  }

  return updates;
}

/**
 * Extract vitest test file references from live vitest-requirements.json
 * SCOPE: Only processes vitest test files (ui/src/.../...test.{ts,tsx})
 * @param {string} scenarioRoot - Path to scenario directory
 * @returns {Map<string, Array<{ref, phase, status}>>} - Map of requirement ID to test files
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
 * Add missing vitest validation entries to requirement JSON
 * @param {string} filePath - Path to requirements JSON file
 * @param {Array} requirements - Requirements to check
 * @param {Map} vitestFiles - Map of requirement ID to test files from live evidence
 * @param {string} scenarioRoot - Scenario directory path
 * @returns {Array} - Array of changes made
 */
function addMissingValidations(filePath, requirements, vitestFiles, scenarioRoot) {
  const changes = [];

  // 1. Read and parse JSON
  const parsed = JSON.parse(fs.readFileSync(filePath, 'utf8'));

  if (!parsed || !Array.isArray(parsed.requirements)) {
    return changes;
  }

  let modified = false;

  parsed.requirements.forEach((requirement) => {
    const liveTestFiles = vitestFiles.get(requirement.id) || [];

    if (liveTestFiles.length === 0) {
      return; // No vitest evidence for this requirement
    }

    // 2. Ensure validation array exists
    if (!Array.isArray(requirement.validation)) {
      requirement.validation = [];
    }

    // 3. Build set of existing vitest refs
    const existingRefs = new Set(
      requirement.validation
        .filter(v => v.type === 'test' && v.ref && v.ref.startsWith('ui/src/'))
        .map(v => v.ref)
    );

    // 4. Add missing validations
    liveTestFiles.forEach(testFile => {
      if (existingRefs.has(testFile.ref)) {
        return; // Already exists
      }

      const newValidation = {
        type: 'test',
        ref: testFile.ref,
        phase: testFile.phase,
        status: testFile.status,
        notes: 'Auto-added from vitest evidence',
      };

      requirement.validation.push(newValidation);
      modified = true;

      changes.push({
        type: 'add_validation',
        requirement: requirement.id,
        validation: testFile.ref,
        status: testFile.status,
      });
    });
  });

  // 5. Write back if modified
  if (modified) {
    if (!parsed._metadata) {
      parsed._metadata = {};
    }
    parsed._metadata.last_synced_at = new Date().toISOString();

    fs.writeFileSync(filePath, JSON.stringify(parsed, null, 2) + '\n', 'utf8');
  }

  return changes;
}

/**
 * Detect and optionally remove orphaned vitest validation entries
 * SCOPE: Only checks type:test validations pointing to ui/src/.../...test.{ts,tsx}
 * @param {string} filePath - Path to requirements JSON file
 * @param {string} scenarioRoot - Scenario directory
 * @param {Object} options - Sync options (pruneStale flag)
 * @returns {Object} - { orphaned: Array, removed: Array }
 */
function detectOrphanedValidations(filePath, scenarioRoot, options) {
  const orphaned = [];
  const removed = [];

  // 1. Read and parse JSON
  const parsed = JSON.parse(fs.readFileSync(filePath, 'utf8'));

  if (!parsed || !Array.isArray(parsed.requirements)) {
    return { orphaned, removed };
  }

  let modified = false;

  parsed.requirements.forEach(requirement => {
    if (!Array.isArray(requirement.validation)) {
      return;
    }

    const validValidations = [];

    requirement.validation.forEach(validation => {
      // PRESERVE all non-test validations (automation, manual, etc.)
      if (validation.type !== 'test') {
        validValidations.push(validation);
        return;
      }

      // PRESERVE Go/Python tests (not vitest scope)
      const ref = validation.ref || '';
      if (!ref.startsWith('ui/src/') || !ref.match(/\.test\.(ts|tsx)$/)) {
        validValidations.push(validation);
        return;
      }

      // Check if vitest test file exists
      const exists = fs.existsSync(path.join(scenarioRoot, ref));

      if (exists) {
        validValidations.push(validation);
      } else {
        // Orphaned vitest validation found!
        orphaned.push({
          requirement: requirement.id,
          ref,
          phase: validation.phase,
          file: filePath,
        });

        if (options.pruneStale) {
          removed.push({
            requirement: requirement.id,
            ref,
            file: filePath,
          });
          modified = true;
          // Don't push to validValidations (removes it)
        } else {
          // Keep but mark as orphaned
          validValidations.push(validation);
        }
      }
    });

    // Replace validation array if pruning
    if (modified && options.pruneStale) {
      requirement.validation = validValidations;
    }
  });

  // Write back if modified
  if (modified) {
    if (!parsed._metadata) {
      parsed._metadata = {};
    }
    parsed._metadata.last_synced_at = new Date().toISOString();

    fs.writeFileSync(filePath, JSON.stringify(parsed, null, 2) + '\n', 'utf8');
  }

  return { orphaned, removed };
}

function syncRequirementRegistry(fileRequirementMap, scenarioRoot, options) {
  const updates = [];
  const addedValidations = [];
  const orphanedValidations = [];
  const removedValidations = [];

  // Extract live vitest evidence once (from phase-results)
  const vitestFiles = extractVitestFilesFromPhaseResults(scenarioRoot);

  for (const [filePath, requirements] of fileRequirementMap.entries()) {
    // Phase 1: Detect and optionally remove orphaned validations
    const orphanResult = detectOrphanedValidations(filePath, scenarioRoot, options);
    orphanedValidations.push(...orphanResult.orphaned);
    removedValidations.push(...orphanResult.removed);

    // Phase 2: Add missing validations from live evidence
    const added = addMissingValidations(filePath, requirements, vitestFiles, scenarioRoot);
    addedValidations.push(...added);

    // Phase 3: Update status fields (existing logic)
    updates.push(...syncRequirementFile(filePath, requirements));
  }

  return {
    statusUpdates: updates,
    addedValidations,
    orphanedValidations,
    removedValidations,
  };
}

/**
 * Print sync summary to console
 * @param {Object} syncResult - Result from syncRequirementRegistry
 */
function printSyncSummary(syncResult) {
  console.log('\n📋 Requirements Sync Report:\n');

  // Status updates (existing)
  if (syncResult.statusUpdates.length > 0) {
    console.log('✏️  Status Updates:');
    syncResult.statusUpdates.forEach(update => {
      if (update.type === 'requirement') {
        console.log(`   ${update.requirement}: status → ${update.status}`);
      }
    });
    console.log();
  }

  // Added validations (new)
  if (syncResult.addedValidations.length > 0) {
    console.log('✅ Added Validations:');
    syncResult.addedValidations.forEach(added => {
      console.log(`   ${added.requirement}: + ${added.validation}`);
    });
    console.log();
  }

  // Orphaned validations (new)
  if (syncResult.orphanedValidations.length > 0) {
    console.log('⚠️  Orphaned Validations (file not found):');
    syncResult.orphanedValidations.forEach(orphan => {
      console.log(`   ${orphan.requirement}: × ${orphan.ref}`);
    });
    console.log();

    if (!syncResult.removedValidations.length) {
      console.log('   💡 Use --prune-stale to remove orphaned validations\n');
    }
  }

  // Removed validations (new)
  if (syncResult.removedValidations.length > 0) {
    console.log('🗑️  Removed Orphaned Validations:');
    syncResult.removedValidations.forEach(removed => {
      console.log(`   ${removed.requirement}: - ${removed.ref}`);
    });
    console.log();
  }

  // Summary
  const totalChanges = syncResult.statusUpdates.length +
                      syncResult.addedValidations.length +
                      syncResult.removedValidations.length;

  if (totalChanges === 0) {
    console.log('✅ No changes needed - all requirements are up to date\n');
  } else {
    console.log(`✅ Sync complete: ${totalChanges} total changes\n`);
  }
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const scenarioRoot = resolveScenarioRoot(process.cwd(), options.scenario);
  const requirementSources = collectRequirementFiles(scenarioRoot);
  if (requirementSources.length === 0) {
    throw new Error('No requirements files discovered');
  }

  const requirements = [];
  const fileRequirementMap = new Map();
  const requirementIndex = new Map();

  requirementSources.forEach((source) => {
    const { requirements: fileRequirements } = parseRequirementFile(source.path);
    if (!fileRequirements || fileRequirements.length === 0) {
      return;
    }

    if (!fileRequirementMap.has(source.path)) {
      fileRequirementMap.set(source.path, []);
    }

    fileRequirements.forEach((requirement) => {
      if (!requirement || !requirement.id) {
        return;
      }
      if (requirementIndex.has(requirement.id)) {
        const previous = requirementIndex.get(requirement.id);
        const previousPath = previous.__meta && previous.__meta.filePath ? previous.__meta.filePath : 'unknown';
        throw new Error(`Duplicate requirement id '${requirement.id}' in ${source.path} (already defined in ${previousPath})`);
      }
      if (requirement.__meta) {
        requirement.__meta.filePath = source.path;
      }
      if (Array.isArray(requirement.validations)) {
        requirement.validations.forEach((validation) => {
          if (validation && validation.__meta) {
            validation.__meta.filePath = source.path;
          }
        });
      }
      fileRequirementMap.get(source.path).push(requirement);
      requirementIndex.set(requirement.id, requirement);
      requirements.push(requirement);
    });
  });

  if (options.mode === 'phase-inspect') {
    const phaseOutput = renderPhaseInspect(options, requirements, scenarioRoot);
    if (options.output) {
      const resolvedOutput = path.resolve(process.cwd(), options.output);
      fs.mkdirSync(path.dirname(resolvedOutput), { recursive: true });
      fs.writeFileSync(resolvedOutput, phaseOutput, 'utf8');
    } else {
      process.stdout.write(phaseOutput);
    }
    return;
  }

  const { phaseResults, requirementEvidence } = loadPhaseResults(scenarioRoot);
  enrichValidationResults(requirements, { phaseResults, requirementEvidence });
  aggregateRequirementStatuses(requirements, requirementIndex);

  if (options.mode === 'sync') {
    const syncResult = syncRequirementRegistry(
      fileRequirementMap,
      scenarioRoot,
      options
    );

    printSyncSummary(syncResult);
    return;
  }

  const summary = computeSummary(requirements);

  let outputContent = '';
  if (options.format === 'markdown') {
    outputContent = renderMarkdown(summary, requirements, options.includePending);
  } else if (options.format === 'trace') {
    outputContent = renderTrace(requirements, scenarioRoot);
  } else {
    outputContent = renderJSON(summary, requirements);
  }

  if (options.output) {
    const resolvedOutput = path.resolve(process.cwd(), options.output);
    fs.mkdirSync(path.dirname(resolvedOutput), { recursive: true });
    fs.writeFileSync(resolvedOutput, outputContent, 'utf8');
  } else {
    process.stdout.write(outputContent);
  }
}

try {
  main();
} catch (error) {
  console.error(`requirements/report failed: ${error.message}`);
  process.exitCode = 1;
}
