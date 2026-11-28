#!/usr/bin/env node
// Sync Business Suite catalog branding from scenario declarations.
//
// For every scenario that declares `bundle_key: "business_suite"` in
// `.vrooli/monetization.json`, this resolves the display name from
// `.vrooli/service.json` and the brand-manager-applied icon from
// `ui/public/public/`, then keeps the LPBS delivery seed in sync:
//
//   * copies the icon into `ui/public/public/apps/<app_key>.<ext>`
//   * writes `/public/apps/<app_key>.<ext>` as the row's `icon_url`
//   * adopts the declared `service.displayName` as the catalog name
//   * refreshes the embedded-seed digest in `download_seed_test.go`
//
// The run makes no changes unless `--write` is passed. `--check` fails when
// drift exists, for use in verification. Adding a scenario to the bundle is
// therefore: declare it in monetization.json, give it a brand-manager branding
// block, make sure its public icon set is applied, then run this tool.
//
// Usage:
//   node scripts/sync-bundle-catalog.mjs           # report (dry run)
//   node scripts/sync-bundle-catalog.mjs --write   # apply + re-digest
//   node scripts/sync-bundle-catalog.mjs --check   # fail on drift

import { createHash } from 'node:crypto';
import { existsSync, mkdirSync, readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { dirname, extname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const lpbsRoot = resolve(here, '..');
const scenariosRoot = resolve(lpbsRoot, '..');
const bundleKey = 'business_suite';
const seedPath = join(lpbsRoot, 'api', 'internal', 'delivery', 'download_seed.json');
const seedTestPath = join(lpbsRoot, 'api', 'internal', 'delivery', 'download_seed_test.go');
const publicAppsDir = join(lpbsRoot, 'ui', 'public', 'public', 'apps');
const platformLogoPath = join(lpbsRoot, 'ui', 'public', 'public', 'logo.webp');
// The suite authority is the platform itself, not a separately branded app;
// its catalog icon is the platform logo rather than a per-app upload.
const platformAppKeys = new Set(['landing-page-business-suite']);

// Brand-manager writes these into ui/public/public/. Prefer the largest raster
// icon, then a vector mark. The order is deliberate: it avoids the placeholder
// wireframe SVGs some scenarios keep for the PWA manifest.
const iconPreference = [
  'icon-512.png',
  'icon-192.png',
  'maskable-icon-512.png',
  'apple-touch-icon.png',
  'logo.webp',
  'logo.svg',
  'icon.svg',
];

const args = new Set(process.argv.slice(2));
const writeMode = args.has('--write');
const checkMode = args.has('--check');

const readJson = path => JSON.parse(readFileSync(path, 'utf8'));
const nonEmpty = value => (typeof value === 'string' ? value.trim() : '');

function discoverBundledApps() {
  const apps = [];
  for (const entry of readdirSync(scenariosRoot, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    const scenario = entry.name;
    const monetizationPath = join(scenariosRoot, scenario, '.vrooli', 'monetization.json');
    if (!existsSync(monetizationPath)) continue;
    let monetization;
    try {
      monetization = readJson(monetizationPath);
    } catch {
      continue;
    }
    if (nonEmpty(monetization.bundle_key) !== bundleKey) continue;
    const servicePath = join(scenariosRoot, scenario, '.vrooli', 'service.json');
    const service = existsSync(servicePath) ? readJson(servicePath) : {};
    const serviceBlock = service.service ?? {};
    apps.push({
      scenario,
      appKey: nonEmpty(monetization.app_key) || scenario,
      name: nonEmpty(serviceBlock.displayName) || nonEmpty(serviceBlock.name) || scenario,
      description: nonEmpty(serviceBlock.description),
      branding: service.branding ?? null,
      iconDir: join(scenariosRoot, scenario, 'ui', 'public', 'public'),
    });
  }
  apps.sort((a, b) => a.appKey.localeCompare(b.appKey));
  return apps;
}

function resolveBrandIcon(app) {
  if (!app.branding || !nonEmpty(app.branding.brand)) return null;
  if (!existsSync(app.iconDir)) return null;
  const files = new Set(readdirSync(app.iconDir));
  const candidate = iconPreference.find(name => files.has(name));
  return candidate ? { name: candidate, path: join(app.iconDir, candidate) } : null;
}

// Go's encoding/json marshals map keys sorted and escapes HTML by default.
// The seed digest guard uses sha256 over that canonical form, so reproduce it
// exactly rather than relying on JSON.stringify's key order.
function goCanonical(value) {
  if (value === null) return 'null';
  if (Array.isArray(value)) return `[${value.map(goCanonical).join(',')}]`;
  if (typeof value === 'object') {
    const keys = Object.keys(value).sort();
    return `{${keys.map(key => `${goCanonical(key)}:${goCanonical(value[key])}`).join(',')}}`;
  }
  if (typeof value === 'string') {
    const escaped = value
      .replace(/\\/g, '\\\\')
      .replace(/"/g, '\\"')
      .replace(/</g, '\\u003c')
      .replace(/>/g, '\\u003e')
      .replace(/&/g, '\\u0026')
      .replace(/\u2028/g, '\\u2028')
      .replace(/\u2029/g, '\\u2029')
      .replace(/[\b]/g, '\\b')
      .replace(/\f/g, '\\f')
      .replace(/\n/g, '\\n')
      .replace(/\r/g, '\\r')
      .replace(/\t/g, '\\t');
    return `"${escaped.replace(/[\u0000-\u001f]/g, ch => `\\u${ch.charCodeAt(0).toString(16).padStart(4, '0')}`)}"`;
  }
  return String(value);
}

function seedDigest(seed) {
  return createHash('sha256').update(goCanonical(seed)).digest('hex');
}

function readDigestGuard() {
  const match = readFileSync(seedTestPath, 'utf8').match(/"([a-f0-9]{64})"/);
  return match ? match[1] : null;
}

function planSync() {
  const apps = discoverBundledApps();
  const raw = readFileSync(seedPath, 'utf8');
  const trailing = raw.match(/\n+$/)?.[0] ?? '\n';
  const seed = JSON.parse(raw);
  const rows = new Map(seed.map(app => [app.app_key, app]));

  const actions = [];
  const warnings = [];

  for (const app of apps) {
    const row = rows.get(app.appKey);
    const icon = resolveBrandIcon(app);
    let iconUrl = null;
    if (icon) {
      const fileBase = nonEmpty(app.branding?.brand) || app.appKey;
      const file = `${fileBase}${extname(icon.name)}`;
      iconUrl = `/public/apps/${file}`;
      const target = join(publicAppsDir, file);
      const bytes = readFileSync(icon.path);
      const current = existsSync(target) ? readFileSync(target) : null;
      if (!current || !current.equals(bytes)) {
        actions.push({ kind: 'icon', appKey: app.appKey, source: icon.path, target, iconUrl });
      }
    } else if (platformAppKeys.has(app.appKey) && existsSync(platformLogoPath)) {
      iconUrl = '/public/logo.webp';
    } else if (app.branding && nonEmpty(app.branding.brand)) {
      warnings.push(`${app.appKey}: branding "${app.branding.brand}" is declared but no public icon set was found under ui/public/public/`);
    } else {
      warnings.push(`${app.appKey}: no brand-manager branding declared; run brand-manager before expecting an icon`);
    }

    if (!row) {
      warnings.push(`${app.appKey}: bundled in monetization.json but absent from download_seed.json; add a catalog row (public visibility is a deliberate choice)`);
      continue;
    }

    if (app.branding && nonEmpty(app.branding.brand) && row.name !== app.name) {
      actions.push({ kind: 'name', appKey: app.appKey, from: row.name, to: app.name });
    }
    const nextIcon = iconUrl ?? null;
    const currentIcon = nonEmpty(row.icon_url) || null;
    if (iconUrl && currentIcon !== nextIcon) {
      actions.push({ kind: 'icon_url', appKey: app.appKey, from: currentIcon, to: nextIcon });
    }
  }

  return { apps, seed, rows, trailing, actions, warnings };
}

function applySync({ seed, rows, trailing, actions }) {
  for (const action of actions) {
    if (action.kind === 'icon') {
      mkdirSync(publicAppsDir, { recursive: true });
      writeFileSync(action.target, readFileSync(action.source));
      continue;
    }
    const row = rows.get(action.appKey);
    if (action.kind === 'name') row.name = action.to;
    if (action.kind === 'icon_url') row.icon_url = action.to;
  }
  writeFileSync(seedPath, `${JSON.stringify(seed, null, 2)}${trailing}`);

  const digest = seedDigest(seed);
  const guard = readDigestGuard();
  if (guard && guard !== digest) {
    writeFileSync(seedTestPath, readFileSync(seedTestPath, 'utf8').replace(guard, digest));
  }
  return digest;
}

function main() {
  const plan = planSync();
  const digest = seedDigest(plan.seed);
  const guard = readDigestGuard();

  console.log(`Bundled apps discovered: ${String(plan.apps.length)}`);
  for (const action of plan.actions) {
    if (action.kind === 'icon') console.log(`  + icon      ${action.appKey} -> ${action.target} (${action.iconUrl})`);
    if (action.kind === 'name') console.log(`  ~ name      ${action.appKey}: "${action.from}" -> "${action.to}"`);
    if (action.kind === 'icon_url') console.log(`  ~ icon_url  ${action.appKey}: ${action.from ?? '(none)'} -> ${action.to}`);
  }
  if (plan.actions.length === 0) console.log('  (catalog branding already in sync)');
  for (const warning of plan.warnings) console.log(`  ! ${warning}`);

  const digestDrift = guard !== null && guard !== digest;
  if (digestDrift) console.log(`  ~ digest    ${guard} -> ${digest}`);
  const drift = plan.actions.length > 0 || digestDrift;

  if (checkMode && drift) {
    console.error('Catalog branding is out of sync. Run: node scripts/sync-bundle-catalog.mjs --write');
    process.exitCode = 1;
    return;
  }
  if (!writeMode) {
    if (!drift) console.log('Catalog branding is in sync.');
    else console.log('Dry run. Re-run with --write to apply.');
    return;
  }

  const applied = applySync(plan);
  console.log(`Applied. Seed digest: ${applied}`);
}

main();
