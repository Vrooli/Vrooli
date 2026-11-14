import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
// provider-free-exception: generated-wire decoding and its prop-only renderer consumer are isolated from router/auth/public assignment providers.
import { afterEach, describe, expect, it } from 'vitest';
import { create, fromJsonString, toJsonString } from '@bufbuild/protobuf';
import { PresentationBlockSchema, PresentationFixtureSchema, PresentationPageSchema, ResolvedProductPresentationSchema } from '@vrooli/proto-types/landing-page-business-suite/v1/shared/product_presentation_pb';
import { cleanup, fireEvent, render, within } from '@testing-library/react';
import { decodeProductPresentation, parseProductPresentation } from './decode';
import { resolveResources } from './resolvedResources';
import { assertRendererContract, supportedVariants } from './contract';
import { videoWireFixture } from './videoTestFixtures';
import { PresentationPage } from './PresentationPage';
import { AssetImage } from './primitives';
import { assertResourceClosure } from './resourceClosure';
import { actionKey, type Presentation } from './types';
import { canonicalPresentationHref, resolvePublicActions, resolvePublicPresentation, withPresentationBase } from './publicIntegration';
import { publicConfig } from './publicTestFixtures';
import signal from './fixtures/signal.json';
import studio from './fixtures/studio.json';
import recommended from '../../../../../api/internal/presentationseed/recommended-signal-studio.json';

const wire = (bundle = false) => fromJsonString(ResolvedProductPresentationSchema, JSON.stringify((bundle ? studio : signal).presentation));
afterEach(cleanup);
describe('generated presentation boundary', () => {
  it.each<{ name: string; change: (p: Presentation) => void; reason: string }>([
    { name: 'unsupported document schema', change: p => { p.schema_version = 2; }, reason: 'Unsupported presentation schema' },
    { name: 'CSS instead of a finite theme color', change: p => { p.page.theme.primary = 'var(--untrusted)'; }, reason: 'Invalid presentation theme token' },
    { name: 'duplicate block identity', change: p => { const b = p.page.blocks[0]; if (!b) throw new Error('Fixture missing'); p.page.blocks.push(b); }, reason: 'Duplicate presentation block id' },
    { name: 'duplicate artifact identity', change: p => { const b = p.page.blocks.find(b => b.kind === 'artifact-explorer'); const example = b?.content.examples[0]; if (!b || !example) throw new Error('Fixture missing'); b.content.examples.push(example); }, reason: 'Duplicate artifact id' },
  ])('rejects $name before rendering a plausible but ambiguous page', ({ change, reason }) => {
    const p = decodeProductPresentation(wire()); change(p);
    expect(() => { assertRendererContract(p); }).toThrow(reason);
  });
  it('rejects a hero exceeding the configured composition capacity, not merely the selected slide count', () => {
    const p = decodeProductPresentation(wire(true)); const hero = p.page.blocks.find(b => b.kind === 'bundle-hero');
    const item = hero?.content.hero_items[0]; if (!hero || !item) throw new Error('Fixture missing');
    hero.content.hero_items.push({ ...item, app_key: 'third' }, { ...item, app_key: 'fourth' });
    expect(() => { assertRendererContract(p); }).toThrow('Hero composition exceeds capacity');
  });
  it('rejects an unsupported hero exhibit instead of substituting an image layout', () => {
    const p = decodeProductPresentation(wire(true)); const hero = p.page.blocks.find(b => b.kind === 'bundle-hero');
    const item = hero?.content.hero_items[0]; if (!item) throw new Error('Fixture missing'); item.exhibit_kind = 'embedded-html';
    expect(() => { assertRendererContract(p); }).toThrow('Unsupported hero exhibit');
  });
  it.each(['label', 'accessibleLabel'] as const)('rejects an action without configured %s', field => {
    const p = wire(); const action = p.page?.display?.shell?.headerAction; if (!action) throw new Error('Fixture missing');
    action[field] = ' ';
    expect(() => decodeProductPresentation(p)).toThrow('Missing configured action label');
  });
  it.each(['static', 'interactive'])('decodes configured %s workflow content without substituting a workspace or loading a provider', variant => {
    const input = wire(); if (!input.page) throw new Error('Fixture missing');
    input.fixtures.push(create(PresentationFixtureSchema, { id: 'configured-workflow', kind: 'workflow', data: { case: 'workflow', value: {
      title: 'Configured workflow title', label: 'Configured workflow label', steps: [{ number: '02', title: 'Configured second step', description: 'Configured step detail' }, { number: '01', title: 'Configured first step', description: 'Configured first detail' }], browserTitle: 'Configured results', browserRows: ['Configured row two', 'Configured row one'], note: 'Illustrative configured workflow',
    } } }));
    input.page.blocks.push(create(PresentationBlockSchema, { id: 'workflow-demo', kind: 'product-demo', version: 1, variant, content: { value: { case: 'productDemo', value: { heading: 'Configured demonstration', description: 'Configured description', altText: 'Configured workflow preview', rendererRef: 'workflow', fixtureRef: 'configured-workflow' } } } }));
    const p = decodeProductPresentation(input);
    const { getByRole, container } = render(<PresentationPage presentation={p} />);
    const demo = getByRole('figure', { name: 'Configured workflow preview' });
    expect(within(demo).getAllByRole('listitem').map(item => item.textContent)).toEqual(['02Configured second stepConfigured step detail', '01Configured first stepConfigured first detail']);
    expect(demo).toHaveTextContent('Configured row twoConfigured row one');
    expect(demo).toHaveTextContent('Illustrative configured workflow');
    expect(container.querySelector('iframe')).toBeNull();
    expect(within(demo).queryByRole('button')).toBeNull();
  });
  it('preserves configured pricing, FAQ and footer oneofs in page order without inventing commerce', () => {
    const input = wire(); if (!input.page) throw new Error('Fixture missing');
    input.page.blocks.push(
      create(PresentationBlockSchema, { id: 'configured-pricing', kind: 'pricing', version: 1, variant: 'compact', content: { value: { case: 'pricing', value: { heading: 'Configured plans', description: 'Configured pricing description', planRefs: ['owner-plan'], actions: [{ kind: 'purchase', label: 'Configured plan action', accessibleLabel: 'Choose configured plan', planRef: 'owner-plan' }] } } } }),
      create(PresentationBlockSchema, { id: 'configured-faq', kind: 'faq', version: 1, variant: 'accordion', content: { value: { case: 'faq', value: { heading: 'Configured questions', items: [{ question: 'Configured question?', answer: 'Configured answer.', accessibleLabel: 'Read configured answer' }] } } } }),
      create(PresentationBlockSchema, { id: 'configured-footer', kind: 'footer', version: 1, variant: 'defined', content: { value: { case: 'footer', value: { label: 'Configured legal links', links: [{ label: 'Configured terms', accessibleLabel: 'Read configured terms', target: '/terms' }] } } } }),
    );
    const p = decodeProductPresentation(input);
    const { container, getByRole, getByLabelText } = render(<PresentationPage presentation={p} />);
    expect(p.page.blocks.slice(-3).map(block => block.kind)).toEqual(['pricing', 'faq', 'footer']);
    expect(container.querySelector('#configured-pricing')).toHaveTextContent(p.page.display.shell.unavailable_reason);
    expect(container.querySelector('#configured-pricing')).not.toHaveTextContent(/\$|Free|Aquila/);
    expect(container.querySelector('#configured-pricing a')).toBeNull();
    const question = getByLabelText('Read configured answer');
    fireEvent.click(question);
    expect(container.querySelector('#configured-faq details')).toHaveAttribute('open');
    expect(getByRole('link', { name: 'Read configured terms' })).toHaveAttribute('href', '/terms');
  });
  it('rejects mismatched typed fixture data and unsafe private-preview diagnostics', () => {
    const input = wire(); const f = input.fixtures[0]; if (!f || !input.diagnostics) throw new Error('Fixture missing');
    f.kind = 'workflow';
    expect(() => decodeProductPresentation(input)).toThrow('Fixture kind and oneof disagree');
    f.kind = 'workspace'; input.diagnostics.noStore = false;
    expect(() => decodeProductPresentation(input)).toThrow('Unsafe preview diagnostics');
  });
  it('bases owner API assets and responsive alternatives once before img/srcset rendering', () => {
    const p = decodeProductPresentation(wire()); const source = p.assets?.[0];
    if (!source || !p.assets) throw new Error('fixture');
    source.public_url = '/api/v1/presentation/assets/main';
    const small = { ...source, id: 'small', width: 720, height: 360, public_url: '/proxy/api/v1/presentation/assets/small', responsive_alternatives: [] };
    p.assets.push(small); p.page.display.asset_labels.small = { alt: 'Configured small art' };
    source.responsive_alternatives = [{ surface: source.surface, asset_id: 'small' }];
    p.page.display.asset_labels[source.id] = { alt: 'Configured art', sizes: '100vw' };
    const based = withPresentationBase(p, '/proxy/');
    const view = render(<AssetImage assetRef={source.id} resources={resolveResources(based)} />);
    expect(view.container.querySelector('img')).toHaveAttribute('src', '/proxy/api/v1/presentation/assets/main');
    expect(view.container.querySelector('img')).toHaveAttribute('srcset', '/proxy/api/v1/presentation/assets/main 1440w, /proxy/api/v1/presentation/assets/small 720w');
    expect(withPresentationBase(based, '/proxy/').assets).toEqual(based.assets);
  });
  it.each(['center', 'contain', 'cover'])('retains and renders released %s crop/focal placement on every visual image path', policy => {
    for (const bundle of [false, true]) {
      const input = wire(bundle);
      for (const asset of input.assets) { asset.cropPolicy = policy; asset.focalPoint = { $typeName: 'vrooli.landing_page_business_suite.v1.shared.PresentationFocalPoint', x: 0.25, y: 0.75 }; }
      const p = decodeProductPresentation(input); const resources = resolveResources(p);
      for (const asset of Object.values(resources.assets)) {
        expect(asset.crop_policy).toBe(policy); expect(asset.focal_point).toEqual({ x: 0.25, y: 0.75 });
      }
      const view = render(<PresentationPage presentation={p} />);
      const images = view.container.querySelectorAll('img'); expect(images.length).toBeGreaterThan(5);
      for (const img of images) expect(img).toHaveStyle({ objectFit: policy === 'contain' ? 'contain' : 'cover', objectPosition: '25% 75%' });
      // Includes the closing image; its opaque copy panel/layout is not replaced.
      expect(view.container.querySelector('.closing-art img')).toHaveStyle({ objectPosition: '25% 75%' });
      expect(view.container.querySelector('.closing-content')).not.toBeNull();
      view.unmount();
    }
  });
  it.each(['', 'stretch', 'fill', 'COVER', 'cover;position:absolute'])('rejects unknown crop policy %s at the generated response boundary', policy => {
    const input = wire(); if (!input.assets[0]) throw new Error('Fixture missing'); input.assets[0].cropPolicy = policy;
    expect(() => decodeProductPresentation(input)).toThrow('Unsupported asset crop policy');
  });
  it.each([-0.01, 1.01, NaN, Infinity])('rejects invalid focal coordinate %s without clamping released placement', coordinate => {
    const p = decodeProductPresentation(wire()); if (!p.assets?.[0]) throw new Error('Fixture missing'); p.assets[0].focal_point.x = coordinate;
    expect(() => resolveResources(p)).toThrow('Invalid asset focal point');
  });
  it('preserves zero and one focal coordinates rather than substituting a center default', () => {
    const p = decodeProductPresentation(wire()); for (const asset of p.assets ?? []) asset.focal_point = { x: 0, y: 1 };
    const { container } = render(<PresentationPage presentation={p} />);
    for (const img of container.querySelectorAll('img')) expect(img).toHaveStyle({ objectPosition: '0% 100%' });
  });
  it('unwraps typed content, retains explicit order and fully configured native fixtures', () => {
    const input = wire();
    input.page?.blocks.reverse();
    const decoded = decodeProductPresentation(input);
    expect(decoded.page.blocks.map(b => b.id)).toEqual(input.page?.blocks.map(b => b.id));
    const voice = decoded.page.blocks.find(b => b.kind === 'voice-story');
    expect(voice?.content.summary_items).toHaveLength(3);
    const fixture = decoded.fixtures?.find(f => f.kind === 'workspace');
    expect(fixture?.workspace.prompt).toBeTruthy();
    expect(decoded.page.display.shell.brand_name).toBe(signal.presentation.page.display.shell.brand_name);
    expect('display' in decoded).toBe(false);
  });
  it('honors omitted protobuf false values and empty maps without inventing content', () => {
    const input = wire();
    if (!input.diagnostics || !input.page?.display) throw new Error('Fixture missing');
    input.diagnostics.preview = false; input.diagnostics.noindex = false; input.diagnostics.noStore = false;
    delete input.page.display.blocks.voice;
    const json = toJsonString(ResolvedProductPresentationSchema, input);
    const decoded = parseProductPresentation(json);
    expect(decoded.diagnostics.preview).toBe(false);
    expect(decoded.diagnostics.no_store).toBe(false);
    expect(decoded.page.display.blocks.voice).toBeUndefined();
  });
  it.each(['display', 'shell'] as const)('rejects absent mandatory %s', field => {
    const input = wire();
    if (!input.page?.display) throw new Error('Fixture missing');
    if (field === 'display') input.page.display = undefined;
    else input.page.display.shell = undefined;
    expect(() => decodeProductPresentation(input)).toThrow('Missing page.display');
  });
  it('rejects unknown JSON fields, nested binary fields, oneof drift and unsupported variants', () => {
    expect(() => parseProductPresentation(JSON.stringify({ ...signal.presentation, private_copy: 'not allowed' }))).toThrow();
    const input = wire();
    const hero = input.page?.blocks[0];
    if (!hero?.content) throw new Error('Fixture missing');
    hero.content.$unknown = [{ no: 101, wireType: 0, data: new Uint8Array([1]) }];
    expect(() => decodeProductPresentation(input)).toThrow('Unknown presentation fields');
    hero.content.$unknown = [];
    hero.kind = 'bundle-hero';
    expect(() => decodeProductPresentation(input)).toThrow('Block kind and oneof disagree');
    hero.kind = 'product-hero'; hero.variant = 'invented-layout';
    expect(() => decodeProductPresentation(input)).toThrow('Unsupported presentation block');
    hero.variant = 'centered'; hero.content.value = { case: undefined };
    expect(() => decodeProductPresentation(input)).toThrow('Missing block content');
  });
  it('keeps two eligible configured hero apps when k=0 and slide keys are empty', () => {
    const input = wire(true); input.selectedAppKeys = [];
    const decoded = decodeProductPresentation(input);
    const { container } = render(<PresentationPage presentation={decoded} />);
    expect(decoded.selected_app_keys).toEqual([]);
    expect([...container.querySelectorAll('[data-hero-app-key]')].map(el => el.getAttribute('data-hero-app-key'))).toEqual(['web-console', 'backdrop-studio']);
  });
  it('rejects a hero lacking eligibility or a public profile even if selected', () => {
    const input = wire(true);
    if (!input.diagnostics) throw new Error('Fixture missing');
    input.diagnostics.eligibleAppKeys = ['web-console'];
    expect(() => decodeProductPresentation(input)).toThrow('eligible resolved profiles');
    input.diagnostics.eligibleAppKeys = ['web-console', 'backdrop-studio']; input.spotlights.pop();
    expect(() => decodeProductPresentation(input)).toThrow('eligible resolved profiles');
  });
  it('matches finite Go variant sets without relying on map iteration order', () => {
    const source = readFileSync(resolve(process.cwd(), '../api/internal/presentation/types.go'), 'utf8');
    const names = new Map([...source.matchAll(/(Block\w+)\s+BlockKind\s*=\s*"([^"]+)"/g)].map(m => [m[1], m[2]]));
    const variants = source.match(/var validVariants = map\[BlockKind\]map\[string\]bool\{([\s\S]*?)\n\}/)?.[1];
    expect(variants).toBeTruthy();
    const go = Object.fromEntries<Set<string>>([...String(variants).matchAll(/(Block\w+):\s*\{([^}]+)\}/g)].map(m => [names.get(m[1]) ?? '', new Set([...String(m[2]).matchAll(/"([^"]+)": true/g)].map(v => v[1] ?? ''))]));
    expect(go).toEqual(Object.fromEntries(Object.entries(supportedVariants).map(([key, values]) => [key, new Set(values)])));
  });
});

describe('resolved resource closure and responsive art', () => {
  // Reproduce integrated-20260915T1153/fixture-bundle-backdrop.bootstrap.json
  // using its canonical page/fixture and existing released test assets, not a
  // duplicated capture. Detail-page labels must close over all three studies.
  function publicBackdropDetail() {
    const input = wire(true);
    const page = recommended.pages.find(page => page.id === 'backdrop-studio');
    const fixture = recommended.fixtures.find(fixture => fixture.id === 'backdrop-studies');
    const labels = input.page?.display?.assetLabels;
    if (!page || !fixture || !labels || !input.diagnostics) throw new Error('Fixture missing');
    input.page = fromJsonString(PresentationPageSchema, JSON.stringify({
      ...page,
      blocks: page.blocks.map(block => {
        const field = block.kind === 'product-hero' ? 'product_hero' : block.kind === 'product-story' ? 'product_story' : block.kind === 'closing-action' ? 'closing_action' : undefined;
        if (!field) throw new Error('Unexpected canonical detail block');
        return { ...block, content: { [field]: block.content } };
      }),
    }));
    if (!input.page.display) throw new Error('Fixture display missing');
    input.page.display.assetLabels = labels;
    input.fixtures = [fromJsonString(PresentationFixtureSchema, JSON.stringify(fixture))];
    input.mode = 'app_detail'; input.scope = 'app'; input.appKey = 'backdrop-studio';
    input.selectedAppKeys = []; input.capabilities = []; input.actions = [];
    input.spotlights = input.spotlights.filter(app => app.appKey === input.appKey);
    for (const asset of input.assets) { asset.releaseRef = 'released-studies'; asset.contentHash = 'a'.repeat(64); }
    const request = { route: '/apps/backdrop-studio', locale: '', variant: 'fixture-bundle' };
    Object.assign(input.diagnostics, {
      preview: false, noindex: false, noStore: false, requestedRoute: request.route, resolvedRoute: request.route,
      requestedVariant: request.variant, resolvedVariant: request.variant, resolvedRevision: 'qualified-revision',
      blockDigest: 'qualified-digest', locale: page.locale, assetReleaseRefs: ['released-studies'],
    });
    return { input, request };
  }
  it('resolves and renders public Backdrop detail with three page-owned asset labels after protobuf JSON roundtrip', () => {
    const { input, request } = publicBackdropDetail();
    const result = resolvePublicPresentation(fromJsonString(ResolvedProductPresentationSchema, toJsonString(ResolvedProductPresentationSchema, input)), request, '/proxy');
    expect(result.presentation.page.id).toBe('backdrop-studio');
    expect(result.presentation.page.blocks.map(block => block.id)).toEqual(['hero', 'details', 'get-started']);
    expect(Object.keys(result.presentation.page.display.asset_labels).sort()).toEqual(['pale-moon', 'survey-relief', 'tidal-halftone']);
    const { container } = render(<PresentationPage {...result} />);
    expect(container.querySelector('.presentation-page')).toHaveAttribute('data-presentation-mode', 'app_detail');
    expect(container.querySelector('.presentation-page')).toHaveAttribute('data-presentation-preview', 'false');
    const resources = resolveResources(result.presentation);
    for (const asset of input.assets) {
      const images = [...container.querySelectorAll('img')].filter(image => image.getAttribute('src') === '/proxy' + asset.publicUrl);
      expect(images.length).toBeGreaterThan(0);
      expect(resources.assets[asset.id]?.alt).toBe(input.page?.display?.assetLabels[asset.id]?.alt);
    }
  });
  it.each(['all', 'survey-relief', 'pale-moon', 'tidal-halftone'])('rejects public Backdrop detail missing %s asset labels before rendering', missing => {
    const { input, request } = publicBackdropDetail();
    if (!input.page?.display) throw new Error('Fixture missing');
    if (missing === 'all') input.page.display.assetLabels = {};
    else input.page.display.assetLabels = Object.fromEntries(Object.entries(input.page.display.assetLabels).filter(([id]) => id !== missing));
    const json = toJsonString(ResolvedProductPresentationSchema, input);
    if (missing === 'all') expect(json).not.toContain('"assetLabels"');
    expect(() => resolvePublicPresentation(fromJsonString(ResolvedProductPresentationSchema, json), request, '/proxy')).toThrow('Missing asset display label');
  });
  it('keeps all three generated backdrop study labels paired with their released images through selection', () => {
    const input = wire(true);
    const backdrop = input.fixtures.find(f => f.data.case === 'backdrop');
    if (backdrop?.data.case !== 'backdrop' || !input.page) throw new Error('Fixture missing');
    input.page.blocks.push({ $typeName: 'vrooli.landing_page_business_suite.v1.shared.PresentationBlock', id: 'study-demo', kind: 'product-demo', version: 1, variant: 'interactive', content: {
      $typeName: 'vrooli.landing_page_business_suite.v1.shared.PresentationBlockContent', value: { case: 'productDemo', value: {
        $typeName: 'vrooli.landing_page_business_suite.v1.shared.PresentationProductDemo', heading: 'Configured study collection', description: 'Configured study comparison', rendererRef: 'backdrop', fixtureRef: backdrop.id,
        posterRef: '', mediaRef: '', altText: 'Compare configured studies',
      } },
    } });
    const p = decodeProductPresentation(fromJsonString(ResolvedProductPresentationSchema, toJsonString(ResolvedProductPresentationSchema, input)));
    const { getByRole } = render(<PresentationPage presentation={p} />);
    const gallery = getByRole('figure', { name: 'Compare configured studies' });
    const studies = backdrop.data.value;
    const labels = ['Survey Relief', 'Pale Moon', 'Tidal Halftone'];
    expect(backdrop.data.value.styles).toEqual(labels);
    expect(backdrop.data.value.assetRefs).toEqual(['survey-relief', 'pale-moon', 'tidal-halftone']);
    expect(within(gallery).getAllByRole('button')).toHaveLength(3);
    labels.forEach((label, index) => {
      const button = within(gallery).getByRole('button', { name: label });
      fireEvent.click(button);
      expect(button).toHaveAttribute('aria-pressed', 'true');
      expect(within(gallery).getAllByRole('button').filter(item => item.getAttribute('aria-pressed') === 'true')).toHaveLength(1);
      const asset = input.assets.find(asset => asset.id === studies.assetRefs[index]);
      if (!asset) throw new Error('Study asset missing');
      expect(gallery.querySelector('.backdrop-canvas img')).toHaveAttribute('src', asset.publicUrl);
      expect(gallery.querySelector('.canvas-caption')).toHaveTextContent(label);
    });
  });
  // integrated-20260915T1100/fixture-bundle-root.bootstrap.json reached the
  // generated decoder with three backdrop styles but only one assetRef. Keep
  // that owner-contract failure distinct from fonts, hero defaults and actions.
  it('rejects the integrated bundle backdrop label/asset mismatch before rendering', () => {
    const input = wire(true);
    const backdrop = input.fixtures.find(f => f.data.case === 'backdrop')?.data;
    if (backdrop?.case !== 'backdrop') throw new Error('Fixture missing');
    expect(backdrop.value.styles).toEqual(['Survey Relief', 'Pale Moon', 'Tidal Halftone']);
    backdrop.value.assetRefs = ['survey-relief'];
    expect(() => decodeProductPresentation(input)).toThrow('Invalid backdrop selection');
  });
  it.each([0, 1, 2])('renders a released singleton backdrop at configured source index %s without inferring other studies', index => {
    const input = wire(true);
    const backdrop = input.fixtures.find(f => f.data.case === 'backdrop')?.data;
    const diagnostics = input.diagnostics;
    const hero = input.page?.blocks.find(b => b.kind === 'bundle-hero')?.content?.value;
    if (backdrop?.case !== 'backdrop' || !diagnostics || !input.page || hero?.case !== 'bundleHero') throw new Error('Fixture missing');
    const style = backdrop.value.styles[index]; const assetRef = backdrop.value.assetRefs[index];
    if (!style || !assetRef) throw new Error('Fixture study missing');
    backdrop.value.styles = [style]; backdrop.value.assetRefs = [assetRef]; backdrop.value.selected = style;
    // Empty proto scalar defaults are omitted by SSR JSON. Hero fixture slots,
    // not a guessed visualRef or bundle appKey, select the configured exhibits.
    input.appKey = '';
    for (const item of hero.value.heroItems) item.visualRef = '';
    for (const asset of input.assets) { asset.releaseRef = 'released-studies'; asset.contentHash = 'a'.repeat(64); }
    Object.assign(diagnostics, { preview: false, noindex: false, noStore: false, requestedRoute: '/', resolvedRoute: '/', requestedVariant: 'configured-bundle', resolvedVariant: 'configured-bundle', resolvedRevision: 'qualified-revision', blockDigest: 'qualified-digest', locale: input.page.locale, assetReleaseRefs: ['released-studies'] });
    const request = { route: '/', locale: '', variant: 'configured-bundle' };
    const json = toJsonString(ResolvedProductPresentationSchema, input);
    expect(json).not.toContain('"visualRef":""');
    expect(json).not.toContain('"appKey":""');
    const result = resolvePublicPresentation(fromJsonString(ResolvedProductPresentationSchema, json), request, '/proxy/');
    const { container } = render(<PresentationPage {...result} />);
    expect(container.querySelector('.presentation-page')).toHaveAttribute('data-presentation-mode', 'bundle');
    expect([...container.querySelectorAll('main > [data-block]')].map(node => node.id)).toEqual(input.page.blocks.map(block => block.id));
    expect(container.querySelectorAll('[data-hero-app-key]')).toHaveLength(2);
    const source = input.assets.find(asset => asset.id === assetRef);
    if (!source) throw new Error('Fixture asset missing');
    expect(container.querySelector('.backdrop-canvas img')).toHaveAttribute('src', '/proxy' + source.publicUrl);
    expect(container.querySelector('.canvas-caption')).toHaveTextContent(style);
    expect(container.querySelectorAll('.art-thumbnails img')).toHaveLength(1);
  });
  it('decodes generated playback and preserves configured labels through the native renderer', () => {
    const p = decodeProductPresentation(videoWireFixture());
    const block = p.page.blocks[0]; if (block?.kind !== 'product-demo') throw new Error('fixture');
    expect(block.content.playback).toEqual({ provider: 'youtube', external_url: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ', layout: 'stacked', play_label: 'Load configured player', caption: 'Configured caption remains visible.', unavailable_label: 'Configured player is unavailable.' });
    const view = render(<PresentationPage presentation={p} />);
    expect(view.getByRole('button', { name: 'Load configured player' })).toBeVisible();
    expect(view.container.querySelector('iframe')).toBeNull();
  });
  it.each(['provider', 'layout', 'playLabel', 'caption', 'unavailableLabel', 'externalUrl'] as const)('rejects malformed generated playback %s', field => {
    const wire = videoWireFixture(); const content = wire.page?.blocks[0]?.content?.value;
    if (content?.case !== 'productDemo' || !content.value.playback) throw new Error('fixture');
    content.value.playback[field] = field === 'provider' || field === 'layout' ? 'unknown' : '';
    expect(() => decodeProductPresentation(wire)).toThrow();
  });
  it('rejects dangling fixture, asset, display block, app and anchor references', () => {
    const missingFixture = wire(); missingFixture.fixtures = [];
    expect(() => decodeProductPresentation(missingFixture)).toThrow('Unresolved fixture');
    const missingAsset = wire(); missingAsset.assets.pop();
    expect(() => decodeProductPresentation(missingAsset)).toThrow('Unresolved asset');
    const displayBlock = wire(); const d = displayBlock.page?.display;
    if (!d?.blocks.hero) throw new Error('Fixture missing');
    d.blocks.hidden = d.blocks.hero;
    expect(() => decodeProductPresentation(displayBlock)).toThrow('Unresolved display block');
    delete d.blocks.hidden;
    const app = d.apps['web-console']; if (!app) throw new Error('Fixture missing');
    d.apps.hidden = app;
    expect(() => decodeProductPresentation(displayBlock)).toThrow('Unresolved app profile');
    delete d.apps.hidden;
    const principles = d.blocks.principles; if (!principles) throw new Error('Fixture missing');
    principles.anchors.voice = '#hidden';
    expect(() => decodeProductPresentation(displayBlock)).toThrow('Unresolved display anchor');
  });
  it('rejects ambiguous fixture/asset exhibits and missing configured accessibility copy', () => {
    const input = wire(true); const hero = input.page?.blocks[0]?.content?.value;
    if (hero?.case !== 'bundleHero' || !hero.value.heroItems[0]) throw new Error('Fixture missing');
    hero.value.heroItems[0].visualRef = 'survey-relief';
    expect(() => decodeProductPresentation(input)).toThrow('exactly one');
    const single = wire(); const decoration = single.page?.display?.fixtureDisplay.workspace;
    if (!decoration) throw new Error('Fixture missing'); decoration.tabsLabel = '';
    expect(() => decodeProductPresentation(single)).toThrow('Missing configured display copy');
  });
  it.each<{ name: string; change: (p: Presentation) => void; reason: string }>([
    { name: 'duplicate fixtures', change: p => { const f = p.fixtures?.[0]; if (f) p.fixtures?.push(f); }, reason: 'Duplicate resolved resource' },
    { name: 'duplicate assets', change: p => { const a = p.assets?.[0]; if (a) p.assets?.push(a); }, reason: 'Duplicate resolved resource' },
    { name: 'duplicate app profiles', change: p => { const a = p.spotlights?.[0]; if (a) p.spotlights?.push(a); }, reason: 'Duplicate resolved resource' },
    { name: 'unresolved capability claims', change: p => { p.capabilities = []; }, reason: 'Unresolved capability' },
    { name: 'unsafe brand destination', change: p => { p.page.display.shell.brand_target = 'javascript:alert(1)'; }, reason: 'Unsafe display target' },
    { name: 'unsafe navigation destination', change: p => { p.page.navigation.items.push({ label: 'Configured link', accessible_label: 'Configured link', target: '//other.example' }); }, reason: 'Unsafe navigation target' },
    { name: 'unsafe footer destination', change: p => { p.page.footer.links.push({ label: 'Configured footer link', accessible_label: 'Configured footer link', target: 'https://user:secret@example.test' }); }, reason: 'Unsafe navigation target' },
    { name: 'display fixture in an unsupported block slot', change: p => { p.page.display.blocks.hero = { fixture_ref: 'workspace' }; }, reason: 'Invalid display fixture slot' },
    { name: 'hero fixture binding on a non-bundle hero', change: p => { p.page.display.blocks.hero = { hero_fixture_refs: { 'web-console': 'workspace' } }; }, reason: 'Unresolved hero fixture app' },
    { name: 'workspace without localized display', change: p => { delete p.page.display.fixture_display.workspace; }, reason: 'Missing fixture display' },
    { name: 'fragment-only asset URL', change: p => { const a = p.assets?.[0]; if (a) a.public_url = '#art'; }, reason: 'Invalid released asset' },
    { name: 'asset without dimensions', change: p => { const a = p.assets?.[0]; if (a) a.width = 0; }, reason: 'Invalid released asset' },
    { name: 'asset without a configured display label', change: p => { p.page.display.asset_labels = {}; }, reason: 'Missing asset display label' },
    { name: 'artifact source referencing an absent asset', change: p => { const b = p.page.blocks.find(b => b.kind === 'artifact-explorer'); if (b?.content.examples[0]) b.content.examples[0].source.media_ref = 'missing-source'; }, reason: 'Unresolved asset' },
    { name: 'story illustration referencing an absent asset', change: p => { const b = p.page.blocks.find(b => b.kind === 'product-story'); if (b?.content.items[0]) b.content.items[0].visual_ref = 'missing-illustration'; }, reason: 'Unresolved asset' },
  ])('fails closed for $name without substituting demo resources', ({ change, reason }) => {
    const p = decodeProductPresentation(wire());
    change(p);
    expect(() => { assertResourceClosure(p); }).toThrow(reason);
    expect(() => resolveResources(p)).toThrow(reason);
  });
  it('rejects a native demo whose fixture kind differs from its configured renderer', () => {
    const p = decodeProductPresentation(wire());
    p.page.blocks.push({ id: 'demo', kind: 'product-demo', version: 1, variant: 'interactive', content: { heading: 'Configured demo', description: 'Configured description', alt_text: 'Configured demonstration', renderer_ref: 'backdrop', fixture_ref: 'workspace' } });
    expect(() => resolveResources(p)).toThrow('Demo renderer and fixture kind differ');
  });
  it('requires display for each eligible hero and catalog app rather than guessing its appearance', () => {
    const p = decodeProductPresentation(wire(true));
    delete p.page.display.apps['backdrop-studio'];
    expect(() => resolveResources(p)).toThrow('Missing hero app display');
    p.page.blocks = p.page.blocks.filter(b => b.kind !== 'bundle-hero'); delete p.page.display.blocks.hero;
    expect(() => resolveResources(p)).toThrow('Missing app display');
  });
  it('uses closed same-composition alternatives with configured sizes; does not guess art direction', () => {
    const p = decodeProductPresentation(wire()); const source = p.assets?.[0];
    if (!source || !p.assets) throw new Error('Fixture missing');
    const smaller = { ...source, id: 'small', width: 720, height: 360, public_url: '/presentation/small.png', responsive_alternatives: [] };
    p.assets.push(smaller);
    p.page.display.asset_labels.small = { alt: 'Configured small art' };
    source.responsive_alternatives = [{ surface: source.surface, asset_id: 'small' }];
    p.page.display.asset_labels[source.id] = { alt: 'Configured art', sizes: '(max-width: 720px) 100vw, 720px' };
    expect(resolveResources(p).assets[source.id]?.src_set).toBe('/presentation/survey-relief.png 1440w, /presentation/small.png 720w');
    const view = render(<AssetImage assetRef={source.id} resources={resolveResources(p)} />);
    expect(view.container.querySelector('img')).toHaveAttribute('srcset', '/presentation/survey-relief.png 1440w, /presentation/small.png 720w');
    expect(view.container.querySelector('img')).toHaveAttribute('sizes', '(max-width: 720px) 100vw, 720px');
    expect(view.container.querySelector('img')).toHaveStyle({ objectFit: 'contain', objectPosition: '50% 50%' });
    view.unmount();
    smaller.height = 720;
    expect(resolveResources(p).assets[source.id]?.src_set).toBeUndefined();
    source.responsive_alternatives[0]!.asset_id = 'missing';
    expect(() => resolveResources(p)).toThrow('Unresolved asset');
  });
  it.each(['crop', 'focal', 'aspect', 'surface', 'reference-surface'])('does not flatten a distinct %s alternative into a widths srcset', difference => {
    const p = decodeProductPresentation(wire()); const source = p.assets?.[0]; if (!source || !p.assets) throw new Error('Fixture missing');
    const candidate = { ...source, focal_point: { ...source.focal_point }, id: 'alternate', public_url: '/presentation/alternate.png', width: 720, height: 360, responsive_alternatives: [] };
    p.assets.push(candidate); p.page.display.asset_labels.alternate = { alt: 'Configured alternative' };
    p.page.display.asset_labels[source.id] = { alt: 'Configured source', sizes: '100vw' };
    source.responsive_alternatives = [{ surface: source.surface, asset_id: candidate.id }];
    if (difference === 'crop') candidate.crop_policy = 'cover';
    if (difference === 'focal') candidate.focal_point.x = 0.2;
    if (difference === 'aspect') candidate.height = 720;
    if (difference === 'surface') candidate.surface = 'different-surface';
    if (difference === 'reference-surface') source.responsive_alternatives[0]!.surface = 'different-surface';
    const resources = resolveResources(p);
    expect(resources.assets[source.id]?.src_set).toBeUndefined();
    expect(resources.assets[candidate.id]?.crop_policy).toBe(candidate.crop_policy);
    expect(resources.assets[candidate.id]?.focal_point).toEqual(candidate.focal_point);
  });
  it('omits widths srcset without configured sizes while retaining the explicit primary image', () => {
    const p = decodeProductPresentation(wire()); const source = p.assets?.[0]; if (!source || !p.assets) throw new Error('Fixture missing');
    const alternate = { ...source, id: 'small', width: source.width / 2, height: source.height / 2, public_url: '/presentation/small.png', responsive_alternatives: [] };
    p.assets.push(alternate); p.page.display.asset_labels.small = { alt: 'Configured smaller study' };
    p.page.display.asset_labels[source.id] = { alt: 'Configured primary study' };
    source.responsive_alternatives = [{ surface: source.surface, asset_id: alternate.id }];
    const view = render(<AssetImage assetRef={source.id} resources={resolveResources(p)} />);
    const img = view.getByRole('img', { name: 'Configured primary study' });
    expect(img).toHaveAttribute('src', source.public_url); expect(img).not.toHaveAttribute('srcset'); expect(img).not.toHaveAttribute('sizes');
    alternate.public_url = '/presentation/small,extra.png';
    expect(() => resolveResources(p)).toThrow('Unsafe responsive URL');
  });
});

describe('public qualification and navigation boundaries', () => {
  it.each(['https://user:secret@example.test', 'https://example.test/?tenant=private', 'https://example.test/#private', 'ftp://example.test'])('does not use unsafe or ambiguous canonical authority %s', authority => {
    expect(canonicalPresentationHref(authority, '/apps/example')).toBeUndefined();
  });
  it('does not accept a bundle or root projection as an app detail response even when diagnostics match the request', () => {
    const input = publicConfig('/apps/example').presentation; input.mode = 'bundle';
    expect(() => resolvePublicPresentation(input, { route: '/apps/example', locale: '', variant: '' }, '/')).toThrow('Detail route did not resolve an app');
  });
  it('requires an explicit fallback diagnostic when the owner changes the requested locale', () => {
    const input = publicConfig('/', 'en').presentation; const request = { route: '/', locale: 'fr', variant: '' };
    expect(() => resolvePublicPresentation(input, request, '/')).toThrow('Unexplained presentation locale mismatch');
    if (!input.diagnostics) throw new Error('Fixture missing'); input.diagnostics.fallback = true;
    const result = resolvePublicPresentation(input, request, '/');
    expect(result.presentation.page.locale).toBe('en'); expect(result.presentation.diagnostics.fallback).toBe(true);
  });
  it.each(['release', 'digest', 'closure'] as const)('rejects a public asset with unqualified %s evidence', invalid => {
    const input = wire(); const d = input.diagnostics; if (!d || !input.page) throw new Error('Fixture missing');
    Object.assign(d, { preview: false, requestedRoute: '/', resolvedRoute: '/', requestedVariant: '', resolvedVariant: 'control', resolvedRevision: 'revision', blockDigest: 'digest', locale: input.page.locale, assetReleaseRefs: ['release'] });
    for (const asset of input.assets) { asset.releaseRef = 'release'; asset.contentHash = 'a'.repeat(64); }
    const asset = input.assets[0]; if (!asset) throw new Error('Fixture missing');
    if (invalid === 'release') asset.releaseRef = '';
    if (invalid === 'digest') asset.contentHash = 'unverified';
    if (invalid === 'closure') d.assetReleaseRefs = [];
    expect(() => resolvePublicPresentation(input, { route: '/', locale: '', variant: '' }, '/')).toThrow('Unqualified public asset');
  });
  it('ignores unconfigured owner observations but rejects duplicate keys for a configured action', () => {
    const input = publicConfig().presentation; const p = decodeProductPresentation(input);
    const closing = p.page.blocks[0]; if (closing?.kind !== 'closing-action') throw new Error('Fixture missing');
    const action = closing.content.actions[0]; if (!action) throw new Error('Fixture missing');
    const owner = { $typeName: 'vrooli.landing_page_business_suite.v1.shared.ResolvedPresentationAction' as const, key: actionKey(action), status: 'ready', href: '/owner-checkout', reason: '', appKey: '', planRef: 'plan' };
    input.actions = [{ ...owner, key: 'not-configured', href: '/private-action' }, owner];
    expect(resolvePublicActions(input, p, '/proxy')).toEqual({ [actionKey(action)]: { status: 'ready', href: '/proxy/owner-checkout' } });
    input.actions.push(owner);
    expect(() => resolvePublicActions(input, p, '/proxy')).toThrow('Duplicate owner action');
  });
  it('uses a qualified owner anchor and preserves an explicit unavailable observation instead of re-enabling local navigation', () => {
    const input = publicConfig().presentation; const p = decodeProductPresentation(input);
    const action = { kind: 'anchor' as const, label: 'Configured section', accessible_label: 'Open configured section', target: '#closing' };
    p.page.display.shell.header_action = action;
    const owner = { $typeName: 'vrooli.landing_page_business_suite.v1.shared.ResolvedPresentationAction' as const, key: actionKey(action), status: 'ready', href: '#closing', reason: '', appKey: '', planRef: '' };
    input.actions = [owner];
    const view = render(<PresentationPage presentation={p} resolvedActions={resolvePublicActions(input, p, '/proxy')} />);
    expect(view.getByRole('link', { name: 'Open configured section' })).toHaveAttribute('href', '#closing');
    owner.status = 'unavailable'; owner.reason = 'Configured section is not available';
    view.rerender(<PresentationPage presentation={p} resolvedActions={resolvePublicActions(input, p, '/proxy')} />);
    expect(view.queryByRole('link', { name: 'Open configured section' })).toBeNull();
    const button = view.getByRole('button', { name: 'Open configured section' });
    expect(button).toBeDisabled(); expect(button).toHaveAccessibleDescription('Configured section is not available');
  });
  it('keeps explicit locale and variant on configured footer navigation but never adds them to external URLs', () => {
    const p = decodeProductPresentation(publicConfig().presentation);
    p.page.blocks.push({ id: 'footer', kind: 'footer', version: 1, variant: 'defined', content: { label: 'Configured links', links: [
      { label: 'Home', accessible_label: 'Configured home', target: '/proxy' },
      { label: 'Detail', accessible_label: 'Configured detail', target: '/apps/example#details' },
      { label: 'Launch', accessible_label: 'Configured launch', target: 'https://owner.example/open?token=opaque' },
    ] } });
    const result = withPresentationBase(p, '/proxy/', { locale: 'fr', variant: 'review' });
    const footer = result.page.blocks.find(b => b.kind === 'footer');
    expect(footer?.content.links.map(link => link.target)).toEqual(['/proxy?locale=fr&variant=review', '/proxy/apps/example?locale=fr&variant=review#details', 'https://owner.example/open?token=opaque']);
    expect(p.page.blocks.find(b => b.kind === 'footer')?.content.links[0]?.target).toBe('/proxy');
  });
});
