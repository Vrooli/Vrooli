#!/usr/bin/env node

/**
 * requirements/report.js
 *
 * Lightweight coverage reporter for scenario requirements.
 * - Parses either <scenario>/docs/requirements.yaml or <scenario>/requirements/<module>.yaml files
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
  const imports = [];
  if (!fs.existsSync(indexPath)) {
    return imports;
  }

  const lines = fs.readFileSync(indexPath, 'utf8').split(/\r?\n/);
  let inImports = false;

  lines.forEach((line) => {
    const trimmed = line.trim();
    if (trimmed.startsWith('#')) {
      return;
    }
    if (!inImports && /^imports:\s*$/.test(trimmed)) {
      inImports = true;
      return;
    }
    if (inImports) {
      const match = line.match(/^\s*-\s*(.+)$/);
      if (match) {
        imports.push(match[1].trim());
      } else if (trimmed.length === 0) {
        return;
      } else if (!/^\s/.test(line)) {
        inImports = false;
      }
    }
  });

  return imports;
}

function collectRequirementFiles(scenarioRoot) {
  const docsPath = path.join(scenarioRoot, 'docs', 'requirements.yaml');
  if (fs.existsSync(docsPath)) {
    return [{ path: docsPath, relative: 'docs/requirements.yaml', isIndex: true }];
  }

  const requirementsDir = path.join(scenarioRoot, 'requirements');
  if (!fs.existsSync(requirementsDir)) {
    throw new Error('No requirements registry found (expected docs/requirements.yaml or requirements/)');
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

  const indexPath = path.join(requirementsDir, 'index.yaml');
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
      if (!entry.name.endsWith('.yaml')) {
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
  const content = fs.readFileSync(filePath, 'utf8');
  const lines = content.split(/\r?\n/);

  const requirements = [];
  let current = null;
  let inValidation = false;
  let currentValidation = null;

  const startPattern = /^ {2}-\s+id:\s*(.+)$/;
  const propertyPattern = /^ {4}([a-zA-Z_]+):\s*(.+)$/;
  const listSectionPattern = /^ {4}([a-zA-Z_]+):\s*$/;
  const listItemPattern = /^ {6}-\s*(.+)$/;
  const validationSectionPattern = /^ {4}validation:/;
  const validationStartPattern = /^ {6}-\s+type:\s*(.+)$/;
  const validationPropertyPattern = /^ {8}([a-zA-Z_]+):\s*(.+)$/;

  for (let lineIndex = 0; lineIndex < lines.length; lineIndex += 1) {
    const line = lines[lineIndex];
    const startMatch = line.match(startPattern);
    if (startMatch) {
      if (current) {
        if (currentValidation) {
          current.validations.push(currentValidation);
          currentValidation = null;
        }
        requirements.push(current);
      }
      current = {
        id: startMatch[1].trim().replace(/^"|"$/g, ''),
        status: 'pending',
        criticality: '',
        title: '',
        category: '',
        validations: [],
      };
      const meta = { statusLine: null, filePath, startLine: lineIndex, originalStatus: 'pending' };
      Object.defineProperty(current, '__meta', {
        value: meta,
        enumerable: false,
      });
      inValidation = false;
      currentValidation = null;
      continue;
    }

    if (!current) {
      continue;
    }

    const propMatch = line.match(propertyPattern);
    if (propMatch && !inValidation) {
      const key = propMatch[1].trim();
      const rawValue = propMatch[2].trim();
      const value = rawValue.replace(/^"|"$/g, '');
      if (key === 'status') {
        current.status = value;
        const meta = current.__meta;
        if (meta) {
          meta.statusLine = lineIndex;
          meta.originalStatus = value;
        }
      } else if (key === 'criticality') {
        current.criticality = value;
      } else if (key === 'title') {
        current.title = value;
      } else if (key === 'category') {
        current.category = value;
      }
      if (key === 'validation') {
        inValidation = true;
        if (currentValidation) {
          current.validations.push(currentValidation);
          currentValidation = null;
        }
      }
      continue;
    }

    const listMatch = line.match(listSectionPattern);
    if (listMatch && !inValidation) {
      const key = listMatch[1].trim();
      if (key === 'validation') {
        inValidation = true;
        if (currentValidation) {
          current.validations.push(currentValidation);
          currentValidation = null;
        }
        continue;
      }
      const items = [];
      let offset = lineIndex + 1;
      while (offset < lines.length) {
        const candidate = lines[offset];
        const itemMatch = candidate.match(listItemPattern);
        if (itemMatch) {
          items.push(itemMatch[1].trim().replace(/^"|"$/g, ''));
          offset += 1;
          continue;
        }
        const indent = candidate.search(/\S|$/);
        if (indent <= 4) {
          break;
        }
        offset += 1;
      }
      current[key] = items;
      lineIndex = offset - 1;
      continue;
    }

    if (validationSectionPattern.test(line)) {
      inValidation = true;
      if (currentValidation) {
        current.validations.push(currentValidation);
        currentValidation = null;
      }
      continue;
    }

    if (inValidation) {
      const validationStartMatch = line.match(validationStartPattern);
      if (validationStartMatch) {
        if (currentValidation) {
          current.validations.push(currentValidation);
        }
        currentValidation = {
          type: validationStartMatch[1].trim().replace(/^"|"$/g, ''),
          ref: '',
          status: '',
          notes: '',
        };
        const validationMeta = { statusLine: null, filePath, startLine: lineIndex };
        Object.defineProperty(currentValidation, '__meta', {
          value: validationMeta,
          enumerable: false,
        });
        continue;
      }

      const validationPropMatch = line.match(validationPropertyPattern);
      if (validationPropMatch && currentValidation) {
        const key = validationPropMatch[1].trim();
        const rawValue = validationPropMatch[2].trim();
        const value = rawValue.replace(/^"|"$/g, '');
        currentValidation[key] = value;
        if (key === 'status') {
          const validationMeta = currentValidation.__meta;
          if (validationMeta) {
            validationMeta.statusLine = lineIndex;
          }
        }
        continue;
      }

      const indent = line.search(/\S|$/);
      if (indent <= 4) {
        if (currentValidation) {
          current.validations.push(currentValidation);
          currentValidation = null;
        }
        inValidation = false;
        // fall through to allow the line to be processed as a top-level property
      } else {
        continue;
      }
    }
  }

  if (current) {
    if (currentValidation) {
      current.validations.push(currentValidation);
    }
    requirements.push(current);
  }

  return { requirements };
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

function replaceStatusValue(line, newValue) {
  if (!line.includes('status:')) {
    return line;
  }

  const commentIndex = line.indexOf('#');
  if (commentIndex >= 0) {
    const beforeComment = line.slice(0, commentIndex);
    const comment = line.slice(commentIndex);
    const updatedBefore = beforeComment.replace(/(status:\s*)([^#]+)/, (_, prefix) => `${prefix}${newValue}`);
    return `${updatedBefore}${comment}`;
  }

  return line.replace(/(status:\s*)(.+?)(\s*)$/, (_, prefix, _value, suffix) => `${prefix}${newValue}${suffix}`);
}

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

  const originalContent = fs.readFileSync(filePath, 'utf8');
  const newline = originalContent.includes('\r\n') ? '\r\n' : '\n';
  const lines = originalContent.split(/\r?\n/);

  const updates = [];

  requirements.forEach((requirement) => {
    const requirementMeta = requirement.__meta;
    const validationStatuses = [];
    const originalStatus = requirementMeta && typeof requirementMeta.originalStatus === 'string'
      ? requirementMeta.originalStatus
      : requirement.status;

    if (Array.isArray(requirement.validations)) {
      requirement.validations.forEach((validation, index) => {
        const validationMeta = validation.__meta;
        const derivedStatus = deriveValidationStatus(validation);
        validationStatuses.push(derivedStatus);

        if (!validationMeta || typeof validationMeta.statusLine !== 'number') {
          return;
        }

        if (derivedStatus && derivedStatus !== validation.status) {
          const lineIndex = validationMeta.statusLine;
          if (lineIndex >= 0 && lineIndex < lines.length) {
            lines[lineIndex] = replaceStatusValue(lines[lineIndex], derivedStatus);
            validation.status = derivedStatus;
            updates.push({
              type: 'validation',
              requirement: requirement.id,
              index,
              status: derivedStatus,
              file: filePath,
            });
          }
        }
      });
    }

    if (!requirementMeta || typeof requirementMeta.statusLine !== 'number') {
      return;
    }

    const nextStatus = deriveRequirementStatus(requirement.status, validationStatuses);
    if (nextStatus && nextStatus !== originalStatus) {
      const lineIndex = requirementMeta.statusLine;
      if (lineIndex >= 0 && lineIndex < lines.length) {
        lines[lineIndex] = replaceStatusValue(lines[lineIndex], nextStatus);
        requirement.status = nextStatus;
        requirementMeta.originalStatus = nextStatus;
        updates.push({
          type: 'requirement',
          requirement: requirement.id,
          status: nextStatus,
          file: filePath,
        });
      }
    }
  });

  if (updates.length > 0) {
    fs.writeFileSync(filePath, lines.join(newline), 'utf8');
  }

  return updates;
}

function syncRequirementRegistry(fileRequirementMap) {
  const updates = [];
  for (const [filePath, requirements] of fileRequirementMap.entries()) {
    updates.push(...syncRequirementFile(filePath, requirements));
  }
  return updates;
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
    const updates = syncRequirementRegistry(fileRequirementMap);
    if (updates.length === 0) {
      process.stdout.write('requirements/report: no requirement updates needed\n');
    } else {
      process.stdout.write(`requirements/report: updated ${updates.length} requirement entries\n`);
    }
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
