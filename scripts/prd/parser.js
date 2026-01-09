#!/usr/bin/env node
'use strict';

const fs = require('node:fs');
const path = require('node:path');

const MODERN_HEADER = '## 🎯 Operational Targets';
const CHECKBOX_PATTERN = /^- \[(x|X| )]/;
const CHECKBOX_TOGGLE_PATTERN = /(\s*-\s*\[)(x|X| )(\])/;
const REQ_LINK_PATTERN = /`\[req:([A-Z0-9,-]+)\]`/i;

function slugify(value) {
  const lower = (value || '').toString().toLowerCase();
  let result = '';
  let prevHyphen = false;
  for (const char of lower) {
    if ((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')) {
      result += char;
      prevHyphen = false;
      continue;
    }
    if (char === ' ' || char === '-' || char === '_') {
      if (!prevHyphen) {
        result += '-';
        prevHyphen = true;
      }
    }
  }
  result = result.replace(/^-+|-+$/g, '').replace(/--+/g, '-');
  if (!result) {
    return `target-${Date.now()}`;
  }
  return result;
}

function parseModernTargetLine(line, criticalityHint, context) {
  const normalizedCriticality = (criticalityHint || '').toUpperCase() || 'P2';
  const status = line.trim().toLowerCase().startsWith('- [x]') ? 'complete' : 'pending';
  let content = line.slice(6).trim();

  let linked = [];
  const reqMatch = content.match(REQ_LINK_PATTERN);
  if (reqMatch && reqMatch[1]) {
    linked = reqMatch[1].split(',').map((entry) => entry.trim()).filter(Boolean);
    content = content.replace(REQ_LINK_PATTERN, '').trim();
  }

  const segments = content.split('|').map((segment) => segment.trim());
  const id = segments[0] || '';
  const title = segments[1] || '';
  const description = segments[2] || '';
  const fallbackId = slugify(`${context.entityName || 'scenario'}-${normalizedCriticality}-${title || description}`);
  const finalId = (id || fallbackId).toUpperCase();
  const pathLeaf = finalId || title;

  return {
    id: finalId,
    title,
    description,
    notes: description,
    status,
    criticality: normalizedCriticality,
    category: normalizedCriticality,
    linked_requirements: linked,
    path: `Operational Targets > ${normalizedCriticality} > ${pathLeaf}`,
  };
}

function extractTargetIdFromLine(line) {
  if (!line) {
    return null;
  }
  const withoutCheckbox = line.replace(CHECKBOX_PATTERN, '').trim();
  if (!withoutCheckbox) {
    return null;
  }
  const candidate = withoutCheckbox.split('|')[0].trim();
  if (/^OT-[Pp][0-2]-\d{3}$/.test(candidate)) {
    return candidate.toUpperCase();
  }
  return null;
}

function parseModernOperationalTargets(content, context) {
  const lines = content.split(/\r?\n/);
  const targets = [];
  let inSection = false;
  let currentCriticality = 'P2';

  for (const rawLine of lines) {
    const line = rawLine.trim();
    if (line.startsWith('## ')) {
      if (line.localeCompare(MODERN_HEADER, undefined, { sensitivity: 'accent' }) === 0) {
        inSection = true;
        continue;
      }
      if (inSection) {
        break;
      }
    }

    if (!inSection || !line) {
      continue;
    }

    if (line.startsWith('###')) {
      if (line.includes('P0')) {
        currentCriticality = 'P0';
      } else if (line.includes('P1')) {
        currentCriticality = 'P1';
      } else if (line.includes('P2')) {
        currentCriticality = 'P2';
      }
      continue;
    }

    if (!CHECKBOX_PATTERN.test(line)) {
      continue;
    }

    targets.push(parseModernTargetLine(line, currentCriticality, context));
  }

  return targets;
}

function parseLegacyOperationalTargets(content, context) {
  const lines = content.split(/\r?\n/);
  const targets = [];
  let inFunctional = false;
  let currentCategory = '';
  let currentCriticality = '';

  lines.forEach((rawLine) => {
    const line = rawLine.trim();
    if (line.startsWith('### ')) {
      if (line === '### Functional Requirements') {
        inFunctional = true;
        return;
      }
      if (inFunctional) {
        inFunctional = false;
      }
    }

    if (!inFunctional || !line) {
      return;
    }

    if (line.startsWith('- **') && line.endsWith('**')) {
      const segment = line.slice(4, -2);
      const parenIndex = segment.indexOf('(');
      if (parenIndex >= 0) {
        currentCategory = segment.slice(0, parenIndex).trim();
        currentCriticality = segment.slice(parenIndex + 1).replace(')', '').trim();
      } else {
        currentCategory = segment.trim();
        currentCriticality = '';
      }
      return;
    }

    if (!CHECKBOX_PATTERN.test(line)) {
      return;
    }

    const status = line.toLowerCase().startsWith('- [x]') ? 'complete' : 'pending';
    const withoutCheckbox = line.slice(6).trim();
    const parts = withoutCheckbox.split('_(');
    const title = parts[0].trim();
    const notes = parts[1] ? parts[1].replace(/\)_$/, '').trim() : '';
    const fallbackId = slugify(`${context.entityName || 'scenario'}-${currentCategory}-${title}`);

    targets.push({
      id: fallbackId.toUpperCase(),
      title,
      description: notes,
      notes,
      status,
      criticality: (currentCriticality || '').toUpperCase(),
      category: currentCategory,
      linked_requirements: [],
      path: `Functional Requirements > ${currentCategory} > ${title}`,
    });
  });

  return targets;
}

function parseOperationalTargetsFromContent(content, options = {}) {
  if (!content) {
    return [];
  }
  const context = {
    entityName: options.entityName || 'scenario',
  };
  const modern = parseModernOperationalTargets(content, context);
  if (modern.length > 0) {
    return modern;
  }
  return parseLegacyOperationalTargets(content, context);
}

function loadOperationalTargets(scenarioRoot, entityType = 'scenario', entityName = null) {
  const prdPath = path.join(scenarioRoot, 'PRD.md');
  if (!fs.existsSync(prdPath)) {
    return { targets: [], status: 'missing', path: prdPath };
  }
  const content = fs.readFileSync(prdPath, 'utf8');
  const targets = parseOperationalTargetsFromContent(content, { entityName: entityName || path.basename(scenarioRoot) });
  return {
    targets,
    status: targets.length > 0 ? 'parsed' : 'empty',
    path: prdPath,
  };
}

function normalizeTargetStatusMap(source) {
  if (source instanceof Map) {
    return source;
  }
  const lookup = new Map();
  if (Array.isArray(source)) {
    source.forEach((entry) => {
      if (entry && entry.target_id) {
        lookup.set(entry.target_id.toUpperCase(), entry.status || 'pending');
      }
    });
    return lookup;
  }
  if (source && typeof source === 'object') {
    Object.keys(source).forEach((key) => {
      if (key) {
        lookup.set(key.toUpperCase(), source[key]);
      }
    });
  }
  return lookup;
}

function toggleCheckbox(rawLine, shouldCheck) {
  const desired = shouldCheck ? 'x' : ' ';
  if (!CHECKBOX_TOGGLE_PATTERN.test(rawLine)) {
    return rawLine;
  }
  return rawLine.replace(
    CHECKBOX_TOGGLE_PATTERN,
    (match, prefix, _current, suffix) => `${prefix}${desired}${suffix}`,
  );
}

function syncOperationalTargetCheckboxes(scenarioRoot, targetStatusSource) {
  const prdPath = path.join(scenarioRoot, 'PRD.md');
  if (!fs.existsSync(prdPath)) {
    return { updated: false, reason: 'missing', path: prdPath, changedTargets: [] };
  }

  const statusMap = normalizeTargetStatusMap(targetStatusSource);
  if (statusMap.size === 0) {
    return { updated: false, reason: 'no-targets', path: prdPath, changedTargets: [] };
  }

  const content = fs.readFileSync(prdPath, 'utf8');
  const lines = content.split(/\r?\n/);
  let inTargets = false;
  let modified = false;
  const changedTargets = [];

  for (let i = 0; i < lines.length; i += 1) {
    const rawLine = lines[i];
    const trimmed = rawLine.trim();

    if (trimmed.startsWith('## ')) {
      if (trimmed.localeCompare(MODERN_HEADER, undefined, { sensitivity: 'accent' }) === 0) {
        inTargets = true;
        continue;
      }
      if (inTargets) {
        break;
      }
    }

    if (!inTargets || !CHECKBOX_PATTERN.test(trimmed)) {
      continue;
    }

    const targetId = extractTargetIdFromLine(trimmed);
    if (!targetId) {
      continue;
    }

    const desiredStatus = statusMap.get(targetId);
    if (!desiredStatus) {
      continue;
    }

    const shouldCheck = (desiredStatus || '').toLowerCase() === 'complete';
    const currentlyChecked = trimmed.toLowerCase().startsWith('- [x');
    if (shouldCheck === currentlyChecked) {
      continue;
    }

    lines[i] = toggleCheckbox(rawLine, shouldCheck);
    modified = true;
    changedTargets.push({ id: targetId, status: desiredStatus });
  }

  if (modified) {
    fs.writeFileSync(prdPath, lines.join('\n'), 'utf8');
  }

  return { updated: modified, path: prdPath, changedTargets };
}

module.exports = {
  parseOperationalTargetsFromContent,
  loadOperationalTargets,
  syncOperationalTargetCheckboxes,
};
