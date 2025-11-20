#!/usr/bin/env node
'use strict';

const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const discovery = require('./lib/discovery');
const prdParser = require('../prd/parser');
const manual = require('./lib/manual');

function parseArgs(argv) {
  const options = { scenario: '' };
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === '--scenario') {
      options.scenario = argv[i + 1] || '';
      i += 1;
    }
  }
  if (!options.scenario) {
    throw new Error('Missing required --scenario argument');
  }
  return options;
}

function readJSON(filePath) {
  if (!fs.existsSync(filePath)) {
    return null;
  }
  try {
    const payload = fs.readFileSync(filePath, 'utf8');
    return JSON.parse(payload);
  } catch (error) {
    console.warn(`requirements/drift-check: unable to parse ${filePath}: ${error.message}`);
    return null;
  }
}

function computeHash(content) {
  return crypto.createHash('sha256').update(content, 'utf8').digest('hex');
}

function normalizeRelative(relativePath) {
  return relativePath.replace(/\\/g, '/');
}

function collectRequirementHashes(scenarioRoot) {
  const files = discovery.collectRequirementFiles(scenarioRoot);
  return files.map((file) => {
    const content = fs.readFileSync(file.path, 'utf8');
    const stats = fs.statSync(file.path);
    return {
      relative_path: normalizeRelative(file.relative),
      hash: computeHash(content),
      mtime: stats.mtime.toISOString(),
    };
  });
}

function compareFiles(currentFiles, snapshotFiles) {
  const drift = {
    mismatched: [],
    missing_from_disk: [],
    new_files: [],
  };
  const snapshotMap = new Map();
  (snapshotFiles || []).forEach((file) => {
    if (file && file.relative_path) {
      snapshotMap.set(normalizeRelative(file.relative_path), file);
    }
  });

  const seen = new Set();
  currentFiles.forEach((file) => {
    const recorded = snapshotMap.get(file.relative_path);
    seen.add(file.relative_path);
    if (!recorded) {
      drift.new_files.push(file);
      return;
    }
    if (recorded.hash && recorded.hash !== file.hash) {
      drift.mismatched.push({
        file: file.relative_path,
        snapshot_hash: recorded.hash,
        current_hash: file.hash,
      });
    }
  });

  snapshotMap.forEach((file, key) => {
    if (!seen.has(key)) {
      drift.missing_from_disk.push({ file: key, snapshot_hash: file.hash || null });
    }
  });

  drift.has_drift = Boolean(
    drift.mismatched.length || drift.new_files.length || drift.missing_from_disk.length,
  );
  return drift;
}

function collectCandidateTimes(dirPath, accumulator) {
  if (!fs.existsSync(dirPath)) {
    return;
  }
  const stats = fs.statSync(dirPath);
  if (stats.isFile()) {
    accumulator.push(stats.mtimeMs);
    return;
  }
  if (!stats.isDirectory()) {
    return;
  }
  const entries = fs.readdirSync(dirPath, { withFileTypes: true });
  entries.forEach((entry) => {
    if (entry.name.startsWith('.')) {
      return;
    }
    collectCandidateTimes(path.join(dirPath, entry.name), accumulator);
  });
}

function latestArtifactTimestamp(scenarioRoot) {
  const times = [];
  const phaseDir = path.join(scenarioRoot, 'coverage', 'phase-results');
  if (fs.existsSync(phaseDir)) {
    const files = fs.readdirSync(phaseDir);
    files.forEach((file) => {
      if (!file.endsWith('.json')) {
        return;
      }
      const target = path.join(phaseDir, file);
      const stats = fs.statSync(target);
      times.push(stats.mtimeMs);
    });
  }

  const vitestReport = path.join(scenarioRoot, 'ui', 'coverage', 'vitest-requirements.json');
  if (fs.existsSync(vitestReport)) {
    const stats = fs.statSync(vitestReport);
    times.push(stats.mtimeMs);
  }

  const manualDir = path.join(scenarioRoot, 'coverage', 'manual-validations');
  collectCandidateTimes(manualDir, times);

  if (!times.length) {
    return null;
  }
  return new Date(Math.max(...times)).toISOString();
}

function normalizeStatus(status) {
  const value = (status || '').toString().toLowerCase();
  if (value === 'complete' || value === 'done') {
    return 'complete';
  }
  if (value === 'in_progress' || value === 'in-progress') {
    return 'in_progress';
  }
  return 'pending';
}

function summarizePrdTargets(scenarioRoot, scenarioName, snapshot) {
  const result = {
    parsed: false,
    mismatches: [],
    missing_in_snapshot: [],
    snapshot_only: [],
  };
  const { targets, status } = prdParser.loadOperationalTargets(scenarioRoot, 'scenario', scenarioName);
  result.status = status;
  if (status === 'missing') {
    return result;
  }
  result.parsed = targets.length > 0;
  const prdMap = new Map();
  targets.forEach((target) => {
    if (!target || !target.id) {
      return;
    }
    prdMap.set(target.id.toUpperCase(), normalizeStatus(target.status));
  });

  const snapshotMap = new Map();
  const snapshotTargets = Array.isArray(snapshot && snapshot.operational_targets)
    ? snapshot.operational_targets
    : [];
  snapshotTargets.forEach((target) => {
    if (!target || !target.target_id) {
      return;
    }
    snapshotMap.set(target.target_id.toUpperCase(), normalizeStatus(target.status));
  });

  prdMap.forEach((prdStatus, targetId) => {
    if (!snapshotMap.has(targetId)) {
      result.missing_in_snapshot.push(targetId);
      return;
    }
    const snapshotStatus = snapshotMap.get(targetId);
    if (prdStatus !== snapshotStatus) {
      result.mismatches.push({ target_id: targetId, prd_status: prdStatus, snapshot_status: snapshotStatus });
    }
  });

  snapshotMap.forEach((value, targetId) => {
    if (!prdMap.has(targetId)) {
      result.snapshot_only.push(targetId);
    }
  });

  result.has_drift = Boolean(
    result.mismatches.length || result.missing_in_snapshot.length || result.snapshot_only.length,
  );
  return result;
}

function collectManualValidations(snapshot) {
  const results = [];
  if (!snapshot || !snapshot.requirements) {
    return results;
  }
  const values = Object.values(snapshot.requirements);
  values.forEach((requirement) => {
    if (!requirement || !requirement.id) {
      return;
    }
    if (!Array.isArray(requirement.validations)) {
      return;
    }
    requirement.validations.forEach((validation) => {
      if (validation && validation.type === 'manual') {
        results.push({
          requirement_id: requirement.id,
          sync_metadata: validation.sync_metadata || validation._sync_metadata || null,
        });
      }
    });
  });
  return results;
}

function summarizeManualDrift(scenarioRoot, snapshot) {
  const manualSummary = {
    manifest_path: null,
    total: 0,
    expired: [],
    missing_metadata: [],
    manifest_missing_entries: [],
    unsynced: [],
    manifest_missing: false,
  };
  const validations = collectManualValidations(snapshot);
  manualSummary.total = validations.length;
  const manifest = manual.loadManifest(scenarioRoot);
  manualSummary.manifest_path = manifest.relativePath || manifest.manifestPath || null;

  if (!validations.length) {
    manualSummary.issue_count = 0;
    return manualSummary;
  }

  if (!manifest.entries.length) {
    manualSummary.manifest_missing = true;
  }

  const expired = new Set();
  const missingMetadata = new Set();
  const manifestMissing = new Set();
  const unsynced = new Set();
  const now = Date.now();

  validations.forEach((validation) => {
    const reqId = validation.requirement_id;
    const syncMeta = validation.sync_metadata;
    const manualMeta = syncMeta && syncMeta.manual ? syncMeta.manual : null;
    if (!manualMeta) {
      missingMetadata.add(reqId);
    } else if (manualMeta.expires_at && Date.parse(manualMeta.expires_at) < now) {
      expired.add(reqId);
    }
    const manifestEntry = manifest.latestByRequirement
      ? manifest.latestByRequirement.get(reqId)
      : null;
    if (!manifestEntry) {
      manifestMissing.add(reqId);
    } else if (manualMeta && manualMeta.validated_at && manifestEntry.validated_at
      && Date.parse(manifestEntry.validated_at) > Date.parse(manualMeta.validated_at)) {
      unsynced.add(reqId);
    }
  });

  manualSummary.expired = Array.from(expired);
  manualSummary.missing_metadata = Array.from(missingMetadata);
  manualSummary.manifest_missing_entries = Array.from(manifestMissing);
  manualSummary.unsynced = Array.from(unsynced);
  manualSummary.issue_count = manualSummary.expired.length
    + manualSummary.missing_metadata.length
    + manualSummary.manifest_missing_entries.length
    + manualSummary.unsynced.length
    + (manualSummary.manifest_missing ? 1 : 0);
  return manualSummary;
}

function run() {
  const options = parseArgs(process.argv.slice(2));
  const scenarioRoot = discovery.resolveScenarioRoot(process.cwd(), options.scenario);
  const snapshotPath = path.join(scenarioRoot, 'coverage', 'requirements-sync', 'latest.json');
  const snapshot = readJSON(snapshotPath);

  const response = {
    snapshot_path: snapshotPath,
    snapshot_synced_at: snapshot && snapshot.synced_at ? snapshot.synced_at : null,
  };

  if (!snapshot) {
    response.status = 'missing_snapshot';
    response.issue_count = 1;
    response.recommendation = 'Run the scenario test suite (test/run-tests.sh) to regenerate the requirements snapshot.';
    console.log(JSON.stringify(response));
    return;
  }

  let fileHashes = [];
  try {
    fileHashes = collectRequirementHashes(scenarioRoot);
  } catch (error) {
    response.status = 'error';
    response.message = error.message;
    console.log(JSON.stringify(response));
    return;
  }

  const fileDrift = compareFiles(fileHashes, snapshot.files || []);
  const newestArtifact = latestArtifactTimestamp(scenarioRoot);
  const artifactStale = Boolean(
    newestArtifact && response.snapshot_synced_at
      && Date.parse(newestArtifact) > Date.parse(response.snapshot_synced_at)
  );
  const prdSummary = summarizePrdTargets(scenarioRoot, options.scenario, snapshot);
  const manualSummary = summarizeManualDrift(scenarioRoot, snapshot);

  const issueCount =
    (fileDrift.mismatched.length + fileDrift.new_files.length + fileDrift.missing_from_disk.length)
    + (artifactStale ? 1 : 0)
    + (prdSummary.mismatches.length + prdSummary.missing_in_snapshot.length + prdSummary.snapshot_only.length)
    + (manualSummary.issue_count || 0);

  response.status = issueCount > 0 ? 'drift_detected' : 'ok';
  response.issue_count = issueCount;
  response.file_drift = fileDrift;
  response.latest_artifact_at = newestArtifact;
  response.artifact_stale = artifactStale;
  response.prd = prdSummary;
  response.manual = manualSummary;

  if (response.status !== 'ok') {
    response.recommendation = 'Re-run the scenario test suite so scripts/requirements/report.js --mode sync can refresh requirements and PRD targets.';
  }

  console.log(JSON.stringify(response));
}

try {
  run();
} catch (error) {
  console.log(JSON.stringify({ status: 'error', message: error.message }));
}
