#!/usr/bin/env node
// BaseStyles owns values. token-map owns consumer names, never duplicate values.
import { readFile, writeFile } from 'node:fs/promises';
import { dirname, join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import postcss from 'postcss';
import ts from 'typescript';
import { format } from 'prettier';
import { resolveLibrarySpecifier } from '../../../../packages/react-component-library/tooling/resolve-specifier.mjs';
const uiRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const libraryRoot = resolve(uiRoot, '../library');
const argument = (name) => { const index = process.argv.indexOf(name); return index < 0 ? undefined : process.argv[index + 1]; };
const reference = (name) => `var(${name})`;
export function readTokenCSS(source) {
  const ast = ts.createSourceFile('BaseStyles.ts', source, ts.ScriptTarget.Latest, true);
  let literal;
  const visit = (node) => {
    if (ts.isVariableDeclaration(node) && node.name.getText(ast) === 'baseStyles') {
      if (!node.initializer || !ts.isNoSubstitutionTemplateLiteral(node.initializer)) throw new Error('BaseStyles must declare a static CSS template');
      literal = node.initializer.text;
    }
    ts.forEachChild(node, visit);
  };
  visit(ast);
  if (!literal) throw new Error('BaseStyles CSS template is missing');
  const css = postcss.parse(literal);
  const output = postcss.root();
  const values = {};
  css.walkRules((rule) => {
    if (!rule.selector.includes(':root') && !rule.selector.split(',').some((selector) => selector.trim() === '.dark')) return;
    const declarations = rule.nodes.filter((node) => node.type === 'decl' && (node.prop.startsWith('--') || node.prop === 'color-scheme'));
    if (!declarations.length) return;
    let conditional = false;
    for (let parent = rule.parent; parent; parent = parent.parent) if (parent.type === 'atrule' && parent.name !== 'layer') conditional = true;
    if (!conditional && rule.selector === ':root') for (const declaration of declarations) if (declaration.prop.startsWith('--')) values[declaration.prop] = declaration.value;
    let generated = rule.clone({ nodes: declarations.map((node) => node.clone()) });
    for (let parent = rule.parent; parent && parent.type !== 'root'; parent = parent.parent) if (parent.type === 'atrule') generated = parent.clone({ nodes: [generated] });
    output.append(generated);
  });
  if (Object.keys(values).length === 0) throw new Error('BaseStyles declares no root tokens');
  return { css: output.toString().trim() + '\n', values };
}
export function buildTheme(aliases, values, appValues) {
  const theme = structuredClone(aliases);
  const available = new Set([...Object.keys(values), ...Object.keys(appValues)]);
  for (const [name, definition] of Object.entries(theme.fontSize)) {
    const shorthand = values[`--text-${name === 'subheading' ? 'subtitle' : name}`] ?? '';
    const weight = shorthand.match(/^\d+/)?.[0];
    if (weight) definition[1].fontWeight = weight;
    if (values[`--text-${name}-tracking`]) definition[1].letterSpacing = reference(`--text-${name}-tracking`);
  }
  for (const match of JSON.stringify(theme).matchAll(/var\((--[\w-]+)\)/g)) if (!available.has(match[1])) throw new Error(`Tailwind alias references undeclared token ${match[1]}`);
  return theme;
}
// Public export names are compatibility mappings, not authored token values.
const tokenGroups = {
  "TOKEN_RAMPS": {
    "space": [
      "--space-4xs",
      "--space-3xs",
      "--space-2xs",
      "--space-xs",
      "--space-sm",
      "--space-md",
      "--space-lg",
      "--space-xl",
      "--space-2xl",
      "--space-4xl"
    ],
    "text": [
      "--text-display",
      "--text-title",
      "--text-heading",
      "--text-body",
      "--text-label",
      "--text-caption",
      "--text-code",
      "--text-overline"
    ],
    "radius": [
      "--radius-control",
      "--radius-panel",
      "--radius-sheet",
      "--radius-overlay",
      "--radius-pill"
    ],
    "elevation": [
      "--elev-flat",
      "--elev-subtle",
      "--elev-raised",
      "--elev-floating",
      "--elev-overlay",
      "--elev-modal"
    ],
    "layer": [
      "--layer-base",
      "--layer-raised",
      "--layer-sticky",
      "--layer-dropdown",
      "--layer-menu",
      "--layer-popover",
      "--layer-overlay",
      "--layer-modal",
      "--layer-toast",
      "--layer-tooltip",
      "--layer-alert"
    ],
    "motion": [
      "--dur-instant",
      "--dur-fast",
      "--dur-quick",
      "--dur-normal",
      "--dur-moderate",
      "--dur-slow",
      "--dur-deliberate",
      "--dur-enter"
    ]
  },
  "SEMANTIC_TOKENS": {
    "background": "--color-background",
    "foreground": "--color-foreground",
    "surface": "--color-surface",
    "surfaceMuted": "--color-surface-muted",
    "border": "--color-border",
    "muted": "--color-muted-foreground",
    "primary": "--color-primary",
    "primaryForeground": "--color-primary-foreground",
    "accent": "--color-accent",
    "success": "--color-success",
    "warning": "--color-warning",
    "danger": "--color-danger",
    "info": "--color-info",
    "focus": "--color-focus"
  },
  "PROVENANCE_TOKENS": {
    "measured": "--provenance-measured",
    "cached": "--provenance-cached",
    "sample": "--provenance-sample",
    "absent": "--provenance-absent",
    "glow": "--glow-primary"
  },
  "COMPONENT_TOKENS": {
    "controlSize": "--control-size-md",
    "controlRadius": "--control-radius",
    "controlPadding": "--control-padding",
    "panelRadius": "--panel-radius",
    "panelPadding": "--panel-padding",
    "focusRing": "--focus-ring"
  },
  "TEXT_STYLES": {
    "display": "--text-display",
    "title": "--text-title",
    "heading": "--text-heading",
    "body": "--text-body",
    "label": "--text-label",
    "caption": "--text-caption",
    "code": "--text-code",
    "overline": "--text-overline",
    "wall": "--text-wall"
  }
};
export function buildTokenModule(header, values, source) {
  const declarations = Object.entries(tokenGroups).map(([name, definition]) => {
    const mapped = JSON.stringify(definition, null, 2).replace(/"(--[\w-]+)"/g, (_, token) => {
      if (!(token in values)) throw new Error(`Tokens public API references undeclared ${token}`);
      return JSON.stringify(reference(token));
    });
    return `export const ${name} = ${mapped} as const;`;
  });
  return `${header}\n// Code generated by node ui/scripts/design-token-generate.mjs; DO NOT EDIT.\n// Source: ${source}\n${declarations.join('\n\n')}\n\nexport type TextStyle = keyof typeof TEXT_STYLES;\nexport const TOKEN_VALUES = ${JSON.stringify(values, null, 2)} as const;\nexport const tokens = { ramps: TOKEN_RAMPS, semantic: SEMANTIC_TOKENS, component: COMPONENT_TOKENS, text: TEXT_STYLES } as const;\n`;
}
async function main() {
  const base = await resolveLibrarySpecifier('@vrooli/react-component-library/BaseStyles/1', { libraryRoot });
  const currentTokens = await resolveLibrarySpecifier('@vrooli/react-component-library/Tokens/1', { libraryRoot });
  const { css, values } = readTokenCSS(await readFile(base.sourcePath, 'utf8'));
  const appCSS = postcss.parse(await readFile(join(uiRoot, 'src/app-tokens.css'), 'utf8'));
  const appValues = {};
  appCSS.walkDecls(/^--/, (decl) => {
    if (decl.prop in values) throw new Error(`App token duplicates library authority: ${decl.prop}`);
    appValues[decl.prop] = decl.value;
  });
  const mapping = JSON.parse(await readFile(join(uiRoot, 'token-map.json'), 'utf8'));
  const source = relative(resolve(uiRoot, '..'), base.sourcePath);
  const theme = { _generated: `node ui/scripts/design-token-generate.mjs from ${source}`, ...buildTheme(mapping.tailwindAliases, values, appValues) };
  const outputs = new Map([
    [join(uiRoot, 'src/design-tokens.css'), `/* Generated by node ui/scripts/design-token-generate.mjs from ${source}. DO NOT EDIT. */\n${css}`],
    [join(uiRoot, 'src/theme/tailwind.theme.json'), JSON.stringify(theme, null, 2) + '\n'],
  ]);
  const prior = await readFile(currentTokens.sourcePath, 'utf8');
  const version = argument('--tokens-version') ?? currentTokens.version;
  const header = prior.slice(0, prior.indexOf('*/') + 2).replace(/@version\s+\S+/, `@version ${version}`) + '\n/** @vrooliComponentSource react-component-library:Tokens */';
  const formatting = JSON.parse(await readFile(join(uiRoot, '.prettierrc.json'), 'utf8'));
  const tokenModule = await format(buildTokenModule(header, values, source), { ...formatting, parser: 'typescript' });
  const tokenOut = argument('--tokens-out');
  if (tokenOut) {
    if (resolve(tokenOut).startsWith(libraryRoot + '/')) throw new Error('Use content-set and draft-publish for catalog writes; tokens-out must be outside library/');
    await writeFile(tokenOut, tokenModule);
  } else outputs.set(currentTokens.sourcePath, tokenModule);
  const check = process.argv.includes('--check');
  const drift = [];
  for (const [path, content] of outputs) {
    const current = await readFile(path, 'utf8').catch((error) => { if (error.code === 'ENOENT') return ''; throw error; });
    if (current === content) continue;
    if (check || path === currentTokens.sourcePath) drift.push(relative(uiRoot, path));
    else await writeFile(path, content);
  }
  if (drift.length) throw new Error(`Generated token drift: ${drift.join(', ')}. Regenerate consumers; publish Tokens changes through the draft ladder using --tokens-out.`);
  console.log(`Token ${check ? 'check' : 'generation'} passed: ${Object.keys(values).length} library defaults, ${Object.keys(appValues).length} app-local tokens; source ${source}.`);
}
if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main().catch((error) => { console.error(error.message); process.exitCode = 1; });
