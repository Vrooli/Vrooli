import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readTokenCSS, readBaseTokenDictionary, renderBaseTokenDictionary, buildTheme } from './design-token-generate.mjs';

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
test('generates the shared token dictionary from the authoritative base CSS and preserves editorial guidance', () => {
  const source = ':root { /* @tier Expression */ --color: white; /* @tier Contract */ --touch-target: 44px; }';
  const prior = '# Dictionary\n\nAuthored introduction.\n\n| Property | Tier | Vrooli default | Meaning | Use when |\n|---|---|---|---|---|\n| `--color` | Expression | `old` | Semantic color role for color. | Styling semantic state. |\n\nAuthored footer.';
  const tokens = readBaseTokenDictionary(source);
  assert.deepEqual(tokens, [
    { name: '--color', tier: 'Expression', value: 'white' },
    { name: '--touch-target', tier: 'Contract', value: '44px' },
  ]);
  const rendered = renderBaseTokenDictionary(source, prior, 'templates/design/_base/tokens.css');
  assert.match(rendered, /source-sha256=[a-f0-9]{64}/);
  assert.match(rendered, /Authored introduction\./);
  assert.match(rendered, /Authored footer\./);
  assert.match(rendered, /\| `--color` \| Expression \| `white` \| Semantic color role for color\. \| Styling semantic state\. \|/);
  assert.match(rendered, /\| `--touch-target` \| Contract \| `44px` \| Canonical design-system value for touch target\. \| Implementing the named shared component contract\. \|/);
  assert.equal(renderBaseTokenDictionary(source, rendered, 'templates/design/_base/tokens.css'), rendered);
  assert.throws(() => readBaseTokenDictionary(':root { --untiered: red; }'), /missing its @tier comment/);
  assert.throws(() => readBaseTokenDictionary(':root { /* @tier Expression */ --dup: red; /* @tier Expression */ --dup: blue; }'), /duplicate root token names/);
});
