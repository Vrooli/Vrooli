#!/usr/bin/env node
'use strict';

const fs = require('node:fs');
const path = require('node:path');

const DEFAULT_MANIFEST_RELATIVE = 'coverage/manual-validations/log.jsonl';
const MANIFEST_ENV_VAR = 'REQUIREMENTS_MANUAL_MANIFEST';

function resolveManifestPath(scenarioRoot, overridePath) {
  const candidate = overridePath || process.env[MANIFEST_ENV_VAR] || DEFAULT_MANIFEST_RELATIVE;
  if (!candidate) {
    return path.join(scenarioRoot, DEFAULT_MANIFEST_RELATIVE);
  }
  if (path.isAbsolute(candidate)) {
    return candidate;
  }
  return path.join(scenarioRoot, candidate);
}

function normalizeManualStatus(status) {
  const value = (status || '').toString().toLowerCase();
  if (['pass', 'passed', 'ok', 'success', 'succeeded'].includes(value)) {
    return 'passed';
  }
  if (['fail', 'failed', 'error', 'failed'].includes(value)) {
    return 'failed';
  }
  if (['skipped', 'skip'].includes(value)) {
    return 'skipped';
  }
  return 'unknown';
}

function parseManifestLine(line, lineNumber, manifestPath) {
  try {
    const entry = JSON.parse(line);
    if (!entry || typeof entry !== 'object') {
      return null;
    }
    if (!entry.requirement_id) {
      return null;
    }
    const normalized = {
      scenario: entry.scenario || null,
      requirement_id: entry.requirement_id,
      status: normalizeManualStatus(entry.status || 'passed'),
      validated_at: entry.validated_at || entry.timestamp || null,
      expires_at: entry.expires_at || null,
      validated_by: entry.validated_by || null,
      notes: entry.notes || null,
      artifact_path: entry.artifact_path || entry.artifact || null,
      recorded_at: entry.recorded_at || entry.timestamp || null,
      line_number: lineNumber,
      manifest_path: manifestPath,
    };
    return normalized;
  } catch (error) {
    console.warn(`requirements/manual: unable to parse manifest line ${lineNumber}: ${error.message}`);
    return null;
  }
}

function loadManifest(scenarioRoot, overridePath) {
  const manifestPath = resolveManifestPath(scenarioRoot, overridePath);
  if (!manifestPath) {
    return {
      manifestPath: null,
      relativePath: null,
      entries: [],
      latestByRequirement: new Map(),
      newestValidatedAt: null,
    };
  }

  if (!fs.existsSync(manifestPath)) {
    return {
      manifestPath,
      relativePath: path.relative(scenarioRoot, manifestPath),
      entries: [],
      latestByRequirement: new Map(),
      newestValidatedAt: null,
    };
  }

  let payload = '';
  try {
    payload = fs.readFileSync(manifestPath, 'utf8');
  } catch (error) {
    console.warn(`requirements/manual: unable to read manifest ${manifestPath}: ${error.message}`);
    return {
      manifestPath,
      relativePath: path.relative(scenarioRoot, manifestPath),
      entries: [],
      latestByRequirement: new Map(),
      newestValidatedAt: null,
    };
  }

  const lines = payload.split(/\r?\n/).map((line) => line.trim()).filter((line) => line.length > 0);
  const entries = [];
  const latestByRequirement = new Map();
  let newestValidatedAt = null;

  lines.forEach((line, idx) => {
    const entry = parseManifestLine(line, idx + 1, manifestPath);
    if (!entry) {
      return;
    }
    entries.push(entry);
    const current = latestByRequirement.get(entry.requirement_id);
    if (!current) {
      latestByRequirement.set(entry.requirement_id, entry);
    } else {
      const currentTime = current.validated_at ? Date.parse(current.validated_at) : 0;
      const entryTime = entry.validated_at ? Date.parse(entry.validated_at) : 0;
      if (entryTime >= currentTime) {
        latestByRequirement.set(entry.requirement_id, entry);
      }
    }
    if (entry.validated_at) {
      if (!newestValidatedAt || Date.parse(entry.validated_at) > Date.parse(newestValidatedAt)) {
        newestValidatedAt = entry.validated_at;
      }
    }
  });

  return {
    manifestPath,
    relativePath: path.relative(scenarioRoot, manifestPath),
    entries,
    latestByRequirement,
    newestValidatedAt,
  };
}

module.exports = {
  DEFAULT_MANIFEST_RELATIVE,
  MANIFEST_ENV_VAR,
  resolveManifestPath,
  loadManifest,
  normalizeManualStatus,
};
