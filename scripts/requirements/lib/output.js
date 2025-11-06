#!/usr/bin/env node
'use strict';

/**
 * Output rendering for requirement reports
 * @module requirements/lib/output
 */

const fs = require('node:fs');
const path = require('node:path');
const { collectValidationsForPhase } = require('./evidence');

/**
 * Render requirements as JSON
 * @param {import('./types').Summary} summary - Summary statistics
 * @param {import('./types').Requirement[]} requirements - Requirements to render
 * @returns {string} JSON output
 */
function renderJSON(summary, requirements) {
  const output = {
    generatedAt: new Date().toISOString(),
    summary,
    requirements,
  };
  return `${JSON.stringify(output, null, 2)}\n`;
}

/**
 * Render requirements as markdown table
 * @param {import('./types').Summary} summary - Summary statistics
 * @param {import('./types').Requirement[]} requirements - Requirements to render
 * @param {boolean} includePending - Include pending requirements
 * @returns {string} Markdown output
 */
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

/**
 * Render requirements with full traceability information
 * @param {import('./types').Requirement[]} requirements - Requirements to render
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {string} Trace output
 */
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

/**
 * Render phase inspection report
 * @param {import('./types').ParseOptions} options - Parse options
 * @param {import('./types').Requirement[]} requirements - Requirements to inspect
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {string} Phase inspection output
 * @throws {Error} If phase name not provided
 */
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

module.exports = {
  renderJSON,
  renderMarkdown,
  renderTrace,
  renderPhaseInspect,
};
