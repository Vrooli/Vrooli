import { createRequire } from 'node:module';
import { createHash } from 'node:crypto';
import { Linter } from 'eslint';
import tseslint from 'typescript-eslint';
import vitest from '@vitest/eslint-plugin';

const require = createRequire(import.meta.url);
export const vitestLintVersion = require('@vitest/eslint-plugin/package.json').version;
export const selectedVitestRules = Object.freeze({
  'vitest/no-focused-tests': 'warn',
  // Returned promises are valid; do not force an await-only convention.
  'vitest/valid-expect': ['warn', { alwaysAwait: false }],
});
export const vitestSyntaxConfig = {
  files: ['**/*.{test,spec}.{ts,tsx,js,jsx,mts,mjs}'],
  plugins: { vitest },
  rules: selectedVitestRules,
};
const asyncMessages = new Set(['asyncMustBeAwaited', 'promisesWithAsyncAssertionsMustBeAwaited']);
export function canonicalRule(message) {
  if (message.ruleId === 'vitest/no-focused-tests') return 'focused-test';
  if (message.ruleId === 'vitest/valid-expect') return asyncMessages.has(message.messageId) ? 'async-assertion' : 'malformed-expectation';
  return undefined;
}

// Bound the upstream name-based focus rule's applicability; this does not
// implement a competing focused-test detector. Native parser scope identifies
// aliases and local lookalikes that the pinned upstream rule cannot distinguish.
function focusSupport(source, filename) {
  const parsed = tseslint.parser.parseForESLint(source, { filePath: filename, ecmaFeatures: { jsx: true }, ecmaVersion: 2022, sourceType: 'module', range: true, loc: true });
  const names = new Set(['test', 'it', 'describe']);
  for (const scope of parsed.scopeManager.scopes) for (const variable of scope.variables) {
    for (const definition of variable.defs) {
      if (definition.type === 'ImportBinding') {
        const node = definition.node;
        const source = definition.parent.source.value;
        const imported = node.imported?.name;
        if (source === 'vitest' && (node.type === 'ImportNamespaceSpecifier' || (names.has(imported) && node.local.name !== imported))) return false;
        if (names.has(variable.name) && (source !== 'vitest' || imported !== variable.name)) return false;
      } else if (names.has(variable.name)) return false;
      // Native .extend identities are outside the pinned focus rule's profile.
      if (definition.node?.init?.callee?.property?.name === 'extend') return false;
    }
  }
  const pending = [{ node: parsed.ast }];
  while (pending.length) {
    const { node, parent, grandparent } = pending.pop();
    if (node.type === 'MemberExpression' && (node.property.name === 'only' || node.property.value === 'only')) {
      // The pinned native rule checks direct expression-statement calls.
      // Computed access and chained only/each/concurrent forms are not covered.
      if (node.computed || !names.has(node.object.name) || parent?.type !== 'CallExpression' || parent.callee !== node || grandparent?.type !== 'ExpressionStatement') return false;
    }
    for (const [key, value] of Object.entries(node)) {
      if (key === 'parent' || key === 'tokens' || key === 'comments') continue;
      for (const child of Array.isArray(value) ? value : [value]) {
        if (child && typeof child === 'object' && typeof child.type === 'string') pending.push({ node: child, parent: node, grandparent: parent });
      }
    }
  }
  return true;
}

// Quality Health owns native lint execution. Consumers reuse this observation;
// they do not run a second lint process to interpret it.
export function observeVitestSyntax(source, filename = 'case.test.ts') {
  const identity = { schemaVersion: 'vitest-lint/v1', pluginVersion: vitestLintVersion,
    configDigest: createHash('sha256').update(JSON.stringify(selectedVitestRules)).digest('hex'),
    eslintVersion: require('eslint/package.json').version,
    parserVersion: require('typescript-eslint/package.json').version,
    profile: 'vitest-syntax-1.6.9', file: filename };
  if (vitestLintVersion !== '1.6.9' || identity.eslintVersion !== '9.39.4' || identity.parserVersion !== '8.59.2') return { ...identity, status: 'unknown', reason: 'unsupported-version', diagnostics: [],
    checks: ['focused-test','malformed-expectation','async-assertion'].map(rule=>({rule,status:'unknown',reason:'unsupported-version'})) };
  const messages = new Linter().verify(source, [{
    files: ['**/*.{ts,tsx,js,jsx,mts,mjs}'], languageOptions: { parser: tseslint.parser, ecmaVersion: 2022, sourceType: 'module' },
    plugins: { vitest }, rules: selectedVitestRules,
  }], { filename });
  const diagnostics = messages.map(message => ({ ...message, canonicalRule: canonicalRule(message) }));
  const invalid = messages.some(message => message.fatal || !canonicalRule(message));
  const supportedFocus = !invalid && focusSupport(source, filename);
  const checks = ['focused-test', 'malformed-expectation', 'async-assertion'].map(rule => {
    const supported = !invalid && (rule !== 'focused-test' || supportedFocus);
    return { rule, status: !supported ? 'unknown' : diagnostics.some(d=>d.canonicalRule===rule) ? 'violation' : 'checked_clean',
      reason: invalid ? 'parse-failure' : !supported ? 'unsupported-test-kind' : 'none' };
  });
  return { ...identity, status: invalid ? 'unknown' : 'observed', reason: invalid ? 'parse-failure' : 'none',
    selectedRules: Object.keys(selectedVitestRules), diagnostics, checks,
    limitations: ['Syntax checks do not establish runtime assertion activity or behavioral adequacy.',
      'The pinned focused-test rule does not resolve aliases, local lookalikes or extended test identities; those file scopes remain unknown.',
      'Unregistered imported test extensions and asynchronous custom matchers require separate identity evidence.'] };
}
