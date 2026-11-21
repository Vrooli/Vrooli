#!/usr/bin/env node
'use strict';

const crypto = require('node:crypto');
const fs = require('node:fs');
const path = require('node:path');

const root = path.resolve(__dirname, '..', '..');
const workflow = JSON.parse(fs.readFileSync(path.join(__dirname, 'order-n8n-workflow.json'), 'utf8'));
const fixtureDoc = JSON.parse(fs.readFileSync(path.join(root, 'ground', 'fixtures', 'shop.json'), 'utf8'));
const node = workflow.nodes.find(candidate => candidate.name === 'Normalize Recipe Order Revision');
if (!node || workflow.active !== false) throw new Error('shop-order-sync workflow is missing its inactive normalizer node');
const execute = new Function('$', node.parameters.jsCode);
const signatureNode = workflow.nodes.find(candidate => candidate.name === 'Verify WooCommerce Signature');
if (!signatureNode || signatureNode.parameters.type !== 'SHA256' || signatureNode.parameters.encoding !== 'base64') {
  throw new Error('WooCommerce signature contract must be HMAC-SHA256 base64');
}

function normalize(payload) {
  const input = { body: payload };
  return execute(name => {
    if (name === 'WooCommerce Order Webhook') return { first: () => ({ json: input }) };
    throw new Error(`unknown node reference ${name}`);
  })[0].json;
}

const accepted = new Map();
let passed = 0;
for (const [caseID, test] of Object.entries(fixtureDoc.cases)) {
  if (test.operation !== 'upsert_order') continue;
  try {
    const result = normalize(test.payload);
    if (!test.expected.accepted) {
      if (test.expected.reason === 'duplicate') {
        const key = `${result.siteId}:${result.orderId}:${result.revision}`;
        if (accepted.get(key) !== JSON.stringify(result)) throw new Error(`${caseID}: expected exact duplicate revision`);
      } else {
        throw new Error(`${caseID}: expected rejection (${test.expected.reason})`);
      }
    } else {
      if (result.orderId !== String(test.expected.orderId) || result.status !== test.expected.status || result.revision !== test.expected.revision) {
        throw new Error(`${caseID}: order known answer mismatch ${JSON.stringify(result)}`);
      }
      const key = `${result.siteId}:${result.orderId}:${result.revision}`;
      accepted.set(key, JSON.stringify(result));
    }
    passed++;
  } catch (error) {
    if (test.expected.accepted || test.expected.reason === 'duplicate' || !String(error.message).includes(test.expected.reason)) throw error;
    passed++;
  }
}

const webhookSecret = 'synthetic-only-secret';
const rawBody = JSON.stringify(fixtureDoc.cases['order-1001'].payload);
const knownAnswerSignature = 'sf/x8eqU5VBwdMIyx7wnBTtzfgREZ2mo/Id2RYREiVE=';
const receivedSignature = crypto.createHmac('sha256', webhookSecret).update(rawBody).digest('base64');
const invalidSignature = crypto.createHmac('sha256', 'wrong-synthetic-secret').update(rawBody).digest('base64');
if (knownAnswerSignature !== receivedSignature || knownAnswerSignature === invalidSignature) throw new Error('WooCommerce HMAC known answer failed');
passed++;
console.log(JSON.stringify({ total: 7, passed, workflow: workflow.name, signature: 'HMAC-SHA256/base64', revisionIdentity: 'siteId+orderId+date_modified_gmt' }));
