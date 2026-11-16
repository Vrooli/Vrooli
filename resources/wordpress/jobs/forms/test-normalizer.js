#!/usr/bin/env node
'use strict';

const fs = require('node:fs');
const path = require('node:path');
const { execFileSync } = require('node:child_process');

const root = path.resolve(__dirname, '..', '..');
const workflow = JSON.parse(fs.readFileSync(path.join(__dirname, 'a3-n8n-workflow.json'), 'utf8'));
const fixtureDoc = JSON.parse(fs.readFileSync(path.join(root, 'ground', 'fixtures', 'forms.json'), 'utf8'));
const node = workflow.nodes.find(candidate => candidate.name === 'Normalize and Validate Submission');
if (!node || workflow.active !== false) throw new Error('A3 workflow is missing its inactive normalizer node');
const execute = new Function('$input', node.parameters.jsCode);

function wrapper(test) {
  const payload = structuredClone(test.payload);
  let query = {};
  if (test.provider === 'contact-form-7') payload.provider = test.provider;
  if (test.provider === 'wpforms-lite') payload.provider = test.provider;
  if (test.provider === 'forminator') query = { provider: test.provider, form_id: payload.form_id };
  if (test.provider === 'elementor-shaped-synthetic') payload.provider = test.provider;
  return { body: payload, query };
}

const accepted = new Map();
const destinationCases = [];
let passed = 0;
for (const [caseID, test] of Object.entries(fixtureDoc.cases)) {
  const expected = test.expected;
  const providerInput = wrapper(test);
  try {
    const result = execute({ first: () => ({ json: providerInput }) })[0].json;
    if (!expected.accepted) {
      if (expected.reason === 'duplicate' || expected.reason === 'identity_conflict') {
        const key = `${result.provider}:${result.formId}:${result.eventId}`;
        const serialized = JSON.stringify(result);
        const prior = accepted.get(key);
        const reason = prior === serialized ? 'duplicate' : 'identity_conflict';
        if (reason !== expected.reason) throw new Error(`${caseID}: expected ${expected.reason}, got ${reason}`);
        destinationCases.push({ id: caseID, event: result, expectedStatus: expected.reason === 'duplicate' ? 200 : 409 });
      } else {
        throw new Error(`${caseID}: expected rejection (${expected.reason}), got ${JSON.stringify(result)}`);
      }
    } else {
      if (result.eventId !== expected.eventId || result.name !== expected.name || result.email !== expected.email) {
        throw new Error(`${caseID}: known answer mismatch ${JSON.stringify(result)}`);
      }
      if (expected.messageCodepoints !== undefined && Array.from(result.message).length !== expected.messageCodepoints) {
        throw new Error(`${caseID}: message codepoint count mismatch`);
      }
      if (test.provider === 'wpforms-lite' && Number(test.payload.entry_id) !== 0) {
        throw new Error(`${caseID}: expected WPForms Lite entry_id zero fixture`);
      }
      const key = `${result.provider}:${result.formId}:${result.eventId}`;
      accepted.set(key, JSON.stringify(result));
      if (test.provider !== 'elementor-shaped-synthetic') {
        destinationCases.push({ id: caseID, event: result, expectedStatus: 201 });
      }
    }
    passed++;
  } catch (error) {
    if (expected.accepted) throw error;
    if (expected.reason !== 'duplicate' && expected.reason !== 'identity_conflict' && !String(error.message).includes(expected.reason)) {
      throw new Error(`${caseID}: expected ${expected.reason}; ${error.message}`);
    }
    if (expected.reason === 'duplicate' || expected.reason === 'identity_conflict') throw error;
    passed++;
  }
}

const unknown = { body: { provider: 'unsupported', form_id: '1' }, query: {} };
try {
  execute({ first: () => ({ json: unknown }) });
  throw new Error('unsupported provider was accepted');
} catch (error) {
  if (!String(error.message).includes('Unsupported provider')) throw error;
}
passed++;

const destinationRunner = path.join(root, 'jobs', 'shared', 'test-adapters.php');
const destinationRun = JSON.parse(execFileSync('php', [destinationRunner], {
  cwd: path.resolve(__dirname, '..', '..', '..', '..'),
  input: JSON.stringify(destinationCases),
  encoding: 'utf8',
}));
if (!destinationRun.integration || destinationRun.integration.passed !== destinationCases.length
    || destinationRun.integration.total !== destinationCases.length
    || destinationRun.integration.failed.length !== 0) {
  throw new Error(`destination contract integration failed: ${JSON.stringify(destinationRun.integration)}`);
}
passed++;
console.log(JSON.stringify({ total: Object.keys(fixtureDoc.cases).length + 2, passed, workflow: workflow.name, providerAdapters: ['contact-form-7', 'wpforms-lite', 'forminator'], syntheticCompatibility: 'elementor-shaped-synthetic', destinationIntegration: { total: destinationRun.integration.total, passed: destinationRun.integration.passed } }));
