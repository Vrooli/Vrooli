#!/usr/bin/env node
'use strict';

const fs = require('node:fs');
const path = require('node:path');

function safeJsonParse(payload, label) {
  if (!payload) {
    return null;
  }
  try {
    return JSON.parse(payload);
  } catch (error) {
    console.warn(`requirements/snapshot: unable to parse ${label}: ${error.message}`);
    return null;
  }
}

function readLatestManifestEntry(manifestPath) {
  if (!manifestPath || !fs.existsSync(manifestPath)) {
    return null;
  }
  try {
    const lines = fs.readFileSync(manifestPath, 'utf8')
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter((line) => line.length > 0);
    if (!lines.length) {
      return null;
    }
    return safeJsonParse(lines[lines.length - 1], 'manifest log');
  } catch (error) {
    console.warn(`requirements/snapshot: unable to read manifest log ${manifestPath}: ${error.message}`);
    return null;
  }
}

function parseManifestEntry(envEntry, manifestPath) {
  const directEntry = safeJsonParse(envEntry, 'manifest entry env');
  if (directEntry) {
    return directEntry;
  }
  return readLatestManifestEntry(manifestPath);
}

function normalizeRequirementSnapshots(fileSnapshots) {
  const requirementMap = {};
  if (!Array.isArray(fileSnapshots)) {
    return requirementMap;
  }
  fileSnapshots.forEach((file) => {
    const requirements = Array.isArray(file.requirements) ? file.requirements : [];
    requirements.forEach((req) => {
      if (!req || !req.id) {
        return;
      }
      requirementMap[req.id] = {
        id: req.id,
        status: req.status || 'pending',
        criticality: req.criticality || null,
        prd_ref: req.prd_ref || null,
        module: req.module || file.module || null,
        file: file.relative_path,
        sync_metadata: req.sync_metadata || null,
        validations: Array.isArray(req.validations)
          ? req.validations.map((validation) => ({
            type: validation.type || null,
            ref: validation.ref || null,
            status: validation.status || null,
            phase: validation.phase || null,
            workflow_id: validation.workflow_id || null,
            sync_metadata: validation.sync_metadata || null,
          }))
          : [],
      };
    });
  });
  return requirementMap;
}

function extractTargetId(requirement) {
  if (!requirement || !requirement.prd_ref) {
    return null;
  }
  const match = requirement.prd_ref.match(/OT-[Pp][0-2]-\d{3}/);
  return match ? match[0].toUpperCase() : null;
}

function extractFolderHint(relativePath) {
  if (!relativePath) {
    return null;
  }
  const parts = relativePath.split(path.sep);
  const requirementsIndex = parts.indexOf('requirements');
  if (requirementsIndex >= 0 && requirementsIndex + 1 < parts.length) {
    return parts[requirementsIndex + 1];
  }
  return null;
}

function shouldReplaceCriticality(current, incoming) {
  if (!incoming) {
    return false;
  }
  if (!current) {
    return true;
  }
  const priority = { P0: 1, P1: 2, P2: 3 };
  const currentScore = priority[current] || 99;
  const incomingScore = priority[incoming] || 99;
  return incomingScore < currentScore;
}

function rollupStatus(counts) {
  if (!counts || counts.total === 0) {
    return 'unknown';
  }
  if (counts.complete === counts.total) {
    return 'complete';
  }
  if (counts.complete > 0 || counts.in_progress > 0) {
    return 'in_progress';
  }
  if (counts.planned > 0) {
    return 'planned';
  }
  return 'pending';
}

function deriveOperationalTargets(fileSnapshots) {
  const targetMap = new Map();
  if (!Array.isArray(fileSnapshots)) {
    return [];
  }

  fileSnapshots.forEach((file) => {
    const requirements = Array.isArray(file.requirements) ? file.requirements : [];
    const folderHint = extractFolderHint(file.relative_path);
    requirements.forEach((requirement) => {
      if (!requirement || !requirement.id) {
        return;
      }
      const targetId = extractTargetId(requirement);
      const key = targetId || folderHint || `unmapped:${file.relative_path}`;
      let target = targetMap.get(key);
      if (!target) {
        target = {
          key,
          target_id: targetId,
          folder_hint: folderHint,
          module: file.module || null,
          prd_refs: new Set(),
          requirements: [],
          counts: {
            total: 0,
            complete: 0,
            in_progress: 0,
            pending: 0,
            planned: 0,
            not_implemented: 0,
          },
          criticality: null,
        };
        targetMap.set(key, target);
      }
      const normalizedStatus = (requirement.status || '').toLowerCase();
      target.counts.total += 1;
      if (normalizedStatus === 'complete') {
        target.counts.complete += 1;
      } else if (normalizedStatus === 'in_progress') {
        target.counts.in_progress += 1;
      } else if (normalizedStatus === 'planned') {
        target.counts.planned += 1;
      } else if (normalizedStatus === 'not_implemented') {
        target.counts.not_implemented += 1;
      } else {
        target.counts.pending += 1;
      }
      target.prd_refs.add(requirement.prd_ref || null);
      target.requirements.push({
        id: requirement.id,
        status: requirement.status || 'pending',
        criticality: requirement.criticality || null,
      });
      if (shouldReplaceCriticality(target.criticality, requirement.criticality || null)) {
        target.criticality = requirement.criticality;
      }
    });
  });

  return Array.from(targetMap.values()).map((target) => ({
    key: target.key,
    target_id: target.target_id || null,
    folder_hint: target.folder_hint || null,
    module: target.module || null,
    criticality: target.criticality || null,
    status: rollupStatus(target.counts),
    counts: target.counts,
    prd_refs: Array.from(target.prd_refs).filter(Boolean),
    requirements: target.requirements,
  })).sort((a, b) => {
    if (a.target_id && b.target_id) {
      return a.target_id.localeCompare(b.target_id);
    }
    if (a.target_id) {
      return -1;
    }
    if (b.target_id) {
      return 1;
    }
    return (a.folder_hint || '').localeCompare(b.folder_hint || '');
  });
}

function writeSnapshot({
  scenarioRoot,
  scenarioName,
  summary,
  testCommands,
  fileSnapshots,
  manifestEntry,
  manifestPath,
  manualManifest,
}) {
  if (!scenarioRoot || !scenarioName) {
    return null;
  }

  const requirementMap = normalizeRequirementSnapshots(fileSnapshots);
  const targets = deriveOperationalTargets(fileSnapshots);
  const payload = {
    scenario: scenarioName,
    synced_at: new Date().toISOString(),
    tests_run: Array.isArray(testCommands) ? testCommands : [],
    manifest: manifestEntry || null,
    manifest_log: manifestPath || null,
    summary: summary || null,
    files: Array.isArray(fileSnapshots) ? fileSnapshots : [],
    requirements: requirementMap,
    operational_targets: targets,
  };

  if (manualManifest) {
    const entries = manualManifest.latestByRequirement
      ? Array.from(manualManifest.latestByRequirement.entries()).map(([requirementId, entry]) => ({
        requirement_id: requirementId,
        status: entry.status || 'unknown',
        validated_at: entry.validated_at || null,
        expires_at: entry.expires_at || null,
        validated_by: entry.validated_by || null,
        artifact_path: entry.artifact_path || null,
      }))
      : [];
    payload.manual_validations = {
      manifest_path: manualManifest.relativePath || manualManifest.manifestPath || null,
      total: entries.length,
      newest_validated_at: manualManifest.newestValidatedAt || null,
      entries,
    };
  }

  const snapshotDir = path.join(scenarioRoot, 'coverage', 'requirements-sync');
  fs.mkdirSync(snapshotDir, { recursive: true });
  const snapshotPath = path.join(snapshotDir, 'latest.json');
  fs.writeFileSync(snapshotPath, `${JSON.stringify(payload, null, 2)}\n`, 'utf8');
  return { path: snapshotPath, payload };
}

module.exports = {
  parseManifestEntry,
  writeSnapshot,
};
