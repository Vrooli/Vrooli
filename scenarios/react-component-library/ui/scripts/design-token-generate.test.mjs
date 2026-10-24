import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readTokenCSS, buildTheme } from './design-token-generate.mjs';

test('preserves root, dark and media cascade while excluding component state overrides', () => {
  const { css, values } = readTokenCSS('export const baseStyles = `@layer tokens { :root { --color: white; } } .dark { --color: black; } @media (prefers-color-scheme: dark) { :root { --color: navy; } } [data-control] { --color: red; }`;');
  assert.deepEqual(values, { '--color': 'white' });
  assert.match(css, /@layer tokens/);
  assert.match(css, /\.dark \{ --color: black/);
  assert.match(css, /@media \(prefers-color-scheme: dark\)/);
  assert.doesNotMatch(css, /red|data-control/);
});
test('rejects dynamic token source and missing consumer references', () => {
  assert.throws(() => readTokenCSS('export const baseStyles = `:root { --color: ${color}; }`;'), /static CSS template/);
  assert.throws(() => buildTheme({ fontSize: {}, spacing: { missing: 'var(--missing)' } }, {}, {}), /undeclared token --missing/);
});
test('derives typography weight from BaseStyles and retains scoped app aliases', () => {
  const result = buildTheme({ fontSize: { title: ['var(--text-title-size)', { lineHeight: 'var(--text-title-line)' }] }, spacing: { panel: 'var(--local-panel)' } }, { '--text-title': '700 var(--text-title-size)', '--text-title-size': '24px', '--text-title-line': '30px' }, { '--local-panel': '20rem' });
  assert.equal(result.fontSize.title[1].fontWeight, '700');
  assert.equal(result.spacing.panel, 'var(--local-panel)');
});
