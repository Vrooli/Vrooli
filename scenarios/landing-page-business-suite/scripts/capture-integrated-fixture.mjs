import { createServer } from 'node:http';
import { once } from 'node:events';
import { access, mkdir, writeFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';

import { createLandingServer } from '../ui/server.js';
import { allocateOutputDir, captureProductionEvidence, presentationExpectation } from './capture-public-evidence.mjs';

// This runner is deliberately test-owned. It starts the real UI factory over a
// real production Vite dist directory, but never starts the LPBS API binary or
// enables a live/public membership. The Go harness owns the temporary API.

async function freePort() {
  const probe = createServer();
  probe.listen(0, '127.0.0.1');
  await once(probe, 'listening');
  const port = probe.address().port;
  await new Promise((resolveClose, rejectClose) => probe.close(error => error ? rejectClose(error) : resolveClose()));
  return port;
}

async function waitForHTTP(url, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  let lastError;
  while (Date.now() < deadline) {
    try {
      const response = await fetch(url, { redirect: 'error', signal: AbortSignal.timeout(Math.max(1, deadline - Date.now())) });
      if (response.ok) return;
      lastError = new Error(`UI readiness returned HTTP ${response.status}`);
    } catch (error) {
      lastError = error;
    }
    await new Promise(resolveDelay => setTimeout(resolveDelay, 50));
  }
  throw new Error(`UI server did not become ready: ${lastError?.message || 'timeout'}`);
}

function captureOptions({ baseUrl, outputDir, variant, revision, route = '/', detailRoute, bundle, expectedPresentations, checkpointSpec }) {
  const checkpoints = checkpointSpec ?? (bundle
    ? {
        hero: '[data-capture-landmark="hero"]',
        catalog: '[data-capture-landmark="catalog"]',
        closing: '.site-footer',
        detail: '[data-capture-landmark="app-detail"]',
      }
    : {
        hero: '[data-capture-landmark="hero"]',
        product: '[data-block="product-story"]',
        artifacts: '[data-block="artifact-explorer"]',
        voice: '[data-block="voice-story"]',
        roadmap: '[data-block="capability-roadmap"]',
        closing: '.site-footer',
        detail: '[data-capture-landmark="app-detail"]',
      });
  const checkpointSelectors = { ...checkpoints };
  if (checkpoints.detail) checkpointSelectors['app-detail'] = checkpoints.detail;
  delete checkpointSelectors.detail;
  return {
    baseUrl,
    outputDir,
    variant,
    revision,
    route,
    detailRoute,
    width: 1440,
    height: 1000,
    dpr: 1,
    timeoutMs: 30000,
    settleMs: 900,
    recordMs: 1200,
    sampleFps: 4,
    decodeWidth: 320,
    maxCaptureMs: 180000,
    testMode: true,
    expectedPresentations,
    checkpoints,
    checkpointSelectors,
    routeLandmarks: {
      root: Object.keys(checkpoints).filter(key => key !== 'detail'),
      detail: detailRoute ? ['app-detail'] : [],
    },
  };
}

async function preflightDocuments(baseUrl, options, outputDir) {
  const checks = [];
  const expectations = {};
  for (const [variant, revision] of [[options.singleVariant, options.singleRevision], [options.bundleVariant, options.bundleRevision]]) {
    expectations[variant] = {};
    const routes = variant === options.bundleVariant ? ['/', '/apps/aquila', '/apps/backdrop-studio'] : ['/', '/apps/aquila'];
    for (const route of routes) {
      const pageKey = route === '/' ? 'root' : route === '/apps/aquila' ? 'detail' : 'backdrop';
      const url = new URL(route, baseUrl); url.searchParams.set('variant', variant);
      try {
        const response = await fetch(url, { headers: { 'x-vrooli-test-mode': '1' }, redirect: 'error', signal: AbortSignal.timeout(10000) });
        const html = await response.text();
        const match = /<script\b[^>]*\bid="lpbs-presentation-bootstrap"[^>]*>([\s\S]*?)<\/script>/i.exec(html);
        const bootstrap = match ? JSON.parse(match[1]) : undefined;
        if (bootstrap) await writeFile(join(outputDir, `${variant}-${pageKey}.bootstrap.json`), JSON.stringify(bootstrap, null, 2), { flag: 'wx' });
        const presentation = bootstrap?.config?.presentation;
        const diagnostics = presentation?.diagnostics;
        const actualRevision = diagnostics?.resolvedRevision ?? diagnostics?.resolved_revision;
        const actualVariant = diagnostics?.resolvedVariant ?? diagnostics?.resolved_variant;
        const actualRoute = diagnostics?.resolvedRoute ?? diagnostics?.resolved_route;
        const expectation = presentation ? presentationExpectation(presentation) : undefined;
        expectations[variant][pageKey] = expectation;
        const passed = response.status === 200 && actualRevision === revision && actualVariant === variant && actualRoute === route && !diagnostics?.preview && !diagnostics?.fallback && expectation?.revision === revision && expectation?.variant === variant && expectation?.route === route && /^sha256:[a-f0-9]{64}$/.test(expectation?.digest ?? '') && /^[a-f0-9]{64}$/.test(expectation?.payloadHash ?? '') && !!expectation?.pageID && Array.isArray(expectation?.blocks) && expectation.blocks.length > 0 && expectation.blocks.every(block => typeof block?.id === 'string' && typeof block?.kind === 'string');
        checks.push({ variant, revision, route, status: response.status, actualRevision, actualVariant, actualRoute, expectation, passed });
      } catch (error) { checks.push({ variant, revision, route, passed: false, error: error.message }); }
    }
  }
  const reportPath = join(outputDir, 'document-preflight.json');
  await writeFile(reportPath, JSON.stringify({ evidence_scope: 'isolated-integration-fixture', checks, expectations }, null, 2), { flag: 'wx' });
  if (checks.some(check => !check.passed)) throw new Error(`Public document preflight failed before recording: ${reportPath}`);
  return expectations;
}

function publicPageURL(baseUrl, route, variant) {
  const url = new URL(route, baseUrl);
  url.searchParams.set('variant', variant);
  return url;
}

async function ssrCheck(baseUrl, route, variant, expectedStatus, headers, name) {
  const response = await fetch(publicPageURL(baseUrl, route, variant), {
    headers, redirect: 'error', signal: AbortSignal.timeout(10000),
  });
  const body = await response.text();
  const robots = body.match(/<meta\b[^>]*\bname=["']robots["'][^>]*\bcontent=["']([^"']+)["'][^>]*>/i)?.[1] || '';
  const xRobots = response.headers.get('x-robots-tag') || '';
  const hasBootstrap = /<script\b[^>]*\bid=["']lpbs-presentation-bootstrap["']/i.test(body);
  return {
    name, variant, route, expectedStatus, status: response.status,
    robots, x_robots_tag: xRobots, has_bootstrap: hasBootstrap,
    passed: response.status === expectedStatus && robots === 'noindex, nofollow' && xRobots === 'noindex, nofollow' && !hasBootstrap,
  };
}

async function preflightSSRNegatives(baseUrl, options, outputDir) {
  const checks = [];
  for (const variant of [options.singleVariant, options.bundleVariant]) {
    checks.push(await ssrCheck(baseUrl, '/apps/browser-automation-studio', variant, 404, { 'x-vrooli-test-mode': '1' }, 'private-bas-404'));
    // The temporary primary root is intentionally empty, so an unmarked SSR
    // request resolves to the owner's normal no-publication response rather than
    // reading the leased fixture document.
    // The primary roots have no active publication. Root unavailability is
    // deliberately 503; a private/unknown detail is the distinct 404 contract.
    checks.push(await ssrCheck(baseUrl, '/', variant, 503, {}, 'missing-test-mode-header'));
  }
  const reportPath = join(outputDir, 'ssr-negative-preflight.json');
  await writeFile(reportPath, JSON.stringify({ evidence_scope: 'isolated-integration-fixture', checks }, null, 2), { flag: 'wx' });
  if (checks.some(check => !check.passed)) throw new Error(`SSR negative preflight failed: ${reportPath}`);
  return checks;
}

async function expireFixtureLease(apiPort) {
  const response = await fetch(`http://127.0.0.1:${apiPort}/__lpbs_test__/expire-presentation-lease`, {
    method: 'POST', headers: { 'x-vrooli-test-mode': '1' }, redirect: 'error', signal: AbortSignal.timeout(10000),
  });
  await response.arrayBuffer();
  if (response.status !== 204) throw new Error(`fixture lease expiry route returned HTTP ${response.status}`);
}

async function preflightExpiredLease(baseUrl, options, outputDir) {
  await expireFixtureLease(options.apiPort);
  const checks = [];
  for (const variant of [options.singleVariant, options.bundleVariant]) {
    checks.push(await ssrCheck(baseUrl, '/', variant, 503, { 'x-vrooli-test-mode': '1' }, 'expired-test-lease'));
  }
  const reportPath = join(outputDir, 'ssr-expired-lease-preflight.json');
  await writeFile(reportPath, JSON.stringify({ evidence_scope: 'isolated-integration-fixture', checks }, null, 2), { flag: 'wx' });
  if (checks.some(check => !check.passed)) throw new Error(`SSR expired-lease preflight failed: ${reportPath}`);
  return checks;
}

export async function captureIntegratedFixture(options) {
  if (!options || !options.apiPort || !options.distDir || !options.outputDir) {
    throw new Error('apiPort, distDir, and outputDir are required');
  }
  const rootOutput = await allocateOutputDir(resolve(options.outputDir));
  await mkdir(rootOutput, { recursive: true });
  process.env.NODE_ENV = 'test';
  if (!process.env.MOCKUP_PLAYWRIGHT_MODULE && !process.env.LPBS_PLAYWRIGHT_MODULE) {
    // Reuse the repository-pinned module used by capture-public-evidence.test;
    // this is discovery of an installed test tool, not a dependency install.
    const repositoryModule = resolve(new URL('../../..', import.meta.url).pathname, 'node_modules/.pnpm/playwright@1.62.1/node_modules/playwright/index.mjs');
    try {
      await access(repositoryModule);
      process.env.MOCKUP_PLAYWRIGHT_MODULE = repositoryModule;
    } catch {
      // captureProductionEvidence will return its normal actionable missing
      // Playwright receipt when the pinned test tool is not installed.
    }
  }
  const uiPort = await freePort();
  const app = createLandingServer({
    uiPort: String(uiPort),
    apiPort: String(options.apiPort),
    distDir: resolve(options.distDir),
  });
  const server = app.listen(uiPort, '127.0.0.1');
  const baseUrl = `http://127.0.0.1:${uiPort}`;
  try {
    await once(server, 'listening');
    // The Go harness is an httptest public mux, not the full API binary, so its
    // database-backed /health contract is intentionally absent. /config is the
    // real UI server's own bounded readiness endpoint for this composition.
    await waitForHTTP(`${baseUrl}/config`, Number(options.timeoutMs || 30000));
    const expectations = await preflightDocuments(baseUrl, options, rootOutput);
    const negativeChecks = await preflightSSRNegatives(baseUrl, options, rootOutput);
    const single = await captureProductionEvidence(captureOptions({
      baseUrl,
      outputDir: join(rootOutput, 'single'),
      variant: options.singleVariant,
      revision: options.singleRevision,
      detailRoute: '/apps/aquila',
      bundle: false,
      expectedPresentations: expectations[options.singleVariant],
    }));
    const bundle = await captureProductionEvidence(captureOptions({
      baseUrl,
      outputDir: join(rootOutput, 'bundle'),
      variant: options.bundleVariant,
      revision: options.bundleRevision,
      detailRoute: '/apps/aquila',
      bundle: true,
      expectedPresentations: expectations[options.bundleVariant],
    }));
    // A second-app journey starts on Backdrop, records its full section path,
    // then navigates explicitly to Aquila. This proves route transitions, not
    // a particular CTA click, and retains the production two-route contract.
    const backdrop = await captureProductionEvidence(captureOptions({
      baseUrl,
      outputDir: join(rootOutput, 'backdrop-detail'),
      variant: options.bundleVariant,
      revision: options.bundleRevision,
      route: '/apps/backdrop-studio',
      detailRoute: '/apps/aquila',
      expectedPresentations: {
        root: expectations[options.bundleVariant].backdrop,
        detail: expectations[options.bundleVariant].detail,
      },
      checkpointSpec: {
        hero: '[data-capture-landmark="hero"]',
        product: '[data-block="product-story"]',
        closing: '.site-footer',
        detail: '[data-capture-landmark="app-detail"]',
      },
    }));
    const expiredLeaseChecks = await preflightExpiredLease(baseUrl, options, rootOutput);
    const report = {
      schema_version: 1,
      evidence_scope: 'isolated-integration-fixture',
      qualification: 'fake_fixture_owner_only',
      public_cutover: false,
      real_release_proof: false,
      commerce_owner_proof: false,
      fixture_disclosure: 'Isolated integration fixture — not a public release',
      presentation_expectations: expectations,
      ssr_negative_checks: [...negativeChecks, ...expiredLeaseChecks],
      ui_server_factory: 'landing-page-business-suite/ui/server.js:createLandingServer',
      captures: [single, bundle, backdrop].map(receipt => ({
        status: receipt.status,
        evidence_scope: receipt.evidenceScope,
        receipt_path: receipt.receiptPath,
        output_dir: receipt.outputDir,
        requested_variant: receipt.requested.variant,
        requested_revision: receipt.requested.revision,
        requested_route: receipt.requested.route,
        requested_detail_route: receipt.requested.detailRoute,
        navigation_method: 'explicit-browser-navigation',
        errors: receipt.errors,
      })),
    };
    const reportPath = join(rootOutput, 'integrated-fixture-report.json');
    await writeFile(reportPath, `${JSON.stringify(report, null, 2)}\n`, { encoding: 'utf8', flag: 'wx' });
    if ([single, bundle, backdrop].some(receipt => receipt.status !== 'passed')) {
      throw new Error(`one or more isolated captures failed; report: ${reportPath}`);
    }
    return { rootOutput, reportPath, captures: [single, bundle, backdrop] };
  } finally {
    await new Promise(resolveClose => {
      if (typeof server.closeAllConnections === 'function') server.closeAllConnections();
      server.close(() => resolveClose());
    }).catch(() => {});
  }
}

function cliOptions(argv) {
  const options = {};
  const values = new Map([
    ['--api-port', 'apiPort'], ['--dist-dir', 'distDir'], ['--output-dir', 'outputDir'],
    ['--single-variant', 'singleVariant'], ['--single-revision', 'singleRevision'],
    ['--bundle-variant', 'bundleVariant'], ['--bundle-revision', 'bundleRevision'],
  ]);
  for (let index = 0; index < argv.length; index += 1) {
    const key = argv[index];
    if (!values.has(key)) throw new Error(`unknown option: ${key}`);
    options[values.get(key)] = argv[++index];
  }
  return options;
}

if (import.meta.url === `file://${process.argv[1]}`) {
  try {
    const result = await captureIntegratedFixture(cliOptions(process.argv.slice(2)));
    console.log(JSON.stringify({ status: 'passed', rootOutput: result.rootOutput, reportPath: result.reportPath,
      captures: result.captures.map(receipt => ({ status: receipt.status, outputDir: receipt.outputDir, evidence: receipt.evidence })),
    }, null, 2));
  } catch (error) {
    console.error(JSON.stringify({ status: 'failed', error: error.message }, null, 2));
    process.exitCode = 1;
  }
}
