import type { Presentation } from './types';
import { safeHref } from './links';

const text = (value: string | undefined): void => {
  if (!value?.trim()) throw new Error('Missing configured display copy');
};
/** Validate only resolved references, never infer membership or substitute content. */
export function assertResourceClosure(p: Presentation): void {
  const d = p.page.display;
  const fixtures = new Map((p.fixtures ?? []).map(f => [f.id, f]));
  const assets = new Map((p.assets ?? []).map(a => [a.id, a]));
  const blocks = new Map(p.page.blocks.map(b => [b.id, b]));
  const profiles = new Map((p.spotlights ?? []).map(a => [a.app_key, a]));
  const capabilities = new Set((p.capabilities ?? []).map(c => c.id));
  if (fixtures.size !== (p.fixtures ?? []).length || assets.size !== (p.assets ?? []).length || profiles.size !== (p.spotlights ?? []).length) throw new Error('Duplicate resolved resource');
  const fixture = (id: string): void => { if (!fixtures.has(id)) throw new Error('Unresolved fixture'); };
  const asset = (id: string): void => { if (!assets.has(id)) throw new Error('Unresolved asset'); };
  const profile = (id: string): void => { if (!profiles.has(id)) throw new Error('Unresolved app profile'); };
  const capability = (id: string): void => { if (!capabilities.has(id)) throw new Error('Unresolved capability'); };
  const exhibit = (fixtureRef: string | undefined, assetRef: string | undefined): void => {
    if (Boolean(fixtureRef) === Boolean(assetRef)) throw new Error('Exhibit requires exactly one fixture or asset');
    if (fixtureRef) fixture(fixtureRef);
    if (assetRef) asset(assetRef);
  };
  for (const label of [d.shell.brand_name, d.shell.skip_label, d.shell.menu_label, d.shell.footer_brand_name, d.shell.unavailable_reason, d.shell.preview_label, p.page.locale, p.page.title, p.page.navigation.label, p.page.footer.label]) text(label);
  for (const target of [d.shell.brand_target, d.shell.footer_brand_target]) if (!safeHref(target)) throw new Error('Unsafe display target');
  for (const link of [...p.page.navigation.items, ...p.page.footer.links]) {
    text(link.label); text(link.accessible_label);
    if (!safeHref(link.target)) throw new Error('Unsafe navigation target');
  }
  for (const key of p.selected_app_keys) profile(key);
  for (const [id, a] of Object.entries(d.apps)) { profile(id); exhibit(a.fixture_ref, a.visual_ref); text(a.detail_label); }
  for (const [id, decoration] of Object.entries(d.fixture_display)) {
    fixture(id);
    if (fixtures.get(id)?.kind === 'workspace') {
      if (!('tabs_label' in decoration)) throw new Error('Missing workspace display labels');
      for (const label of [decoration.avatar, decoration.time, decoration.tabs_label, decoration.terminal_label, decoration.messages_label]) text(label);
    }
  }
  for (const id of Object.keys(d.asset_labels)) asset(id);
  for (const [id, b] of Object.entries(d.blocks)) {
    const block = blocks.get(id);
    if (!block) throw new Error('Unresolved display block');
    if (b.fixture_ref) { if (block.kind !== 'device-story') throw new Error('Invalid display fixture slot'); fixture(b.fixture_ref); }
    for (const [key, ref] of Object.entries(b.hero_fixture_refs ?? {})) {
      if (block.kind !== 'bundle-hero' || !block.content.hero_items.some(item => item.app_key === key)) throw new Error('Unresolved hero fixture app');
      fixture(ref);
    }
    for (const [key, anchor] of Object.entries(b.anchors ?? {})) {
      capability(key);
      if (!anchor.startsWith('#') || !blocks.has(anchor.slice(1))) throw new Error('Unresolved display anchor');
    }
  }
  for (const f of fixtures.values()) {
    if (f.kind !== 'workflow' && !d.fixture_display[f.id]) throw new Error('Missing fixture display');
    if (f.kind === 'backdrop') f.backdrop.asset_refs.forEach(asset);
  }
  for (const a of assets.values()) {
    if (!safeHref(a.public_url) || a.public_url?.startsWith('#') || !a.width || !a.height || a.width < 0 || a.height < 0) throw new Error('Invalid released asset');
    if (!d.asset_labels[a.id]) throw new Error('Missing asset display label');
    for (const alternative of a.responsive_alternatives ?? []) asset(alternative.asset_id);
  }
  for (const block of p.page.blocks) {
    const b = d.blocks[block.id];
    switch (block.kind) {
      case 'product-hero': exhibit(block.content.fixture_ref, block.content.visual_ref); text(block.content.accessibility_label); break;
      case 'bundle-hero':
        for (const item of block.content.hero_items) {
          profile(item.app_key); exhibit(b?.hero_fixture_refs?.[item.app_key], item.visual_ref); text(item.detail_label);
          if (!d.apps[item.app_key]) throw new Error('Missing hero app display');
        }
        break;
      case 'device-story': exhibit(b?.fixture_ref, block.content.visual_ref); text(block.content.alt_text); break;
      case 'product-demo': {
        if (block.variant === 'recorded') {
          const poster = assets.get(block.content.poster_ref ?? '');
          if (!poster || poster.mime !== 'image/png' || !poster.release_ref || !poster.content_hash || !poster.public_url?.startsWith('/') || poster.public_url.startsWith('//')) throw new Error('Recorded demo requires a released same-origin poster');
        } else {
          fixture(block.content.fixture_ref ?? '');
          if (fixtures.get(block.content.fixture_ref ?? '')?.kind !== block.content.renderer_ref) throw new Error('Demo renderer and fixture kind differ');
        }
        break;
      }
      case 'app-spotlights': for (const key of block.content.app_keys) { profile(key); if (!d.apps[key]) throw new Error('Missing app display'); } break;
      case 'product-story': for (const item of block.content.items) if (item.visual_ref) asset(item.visual_ref); break;
      case 'closing-action': if (block.content.visual_ref) asset(block.content.visual_ref); break;
      case 'capability-strip': block.content.items.forEach(item => { capability(item.capability_id); }); break;
      case 'capability-roadmap': block.content.capability_ids.forEach(capability); break;
      case 'voice-story':
        block.content.capability_ids.forEach(capability);
        block.content.features.forEach(item => { capability(item.capability_id); });
        for (const label of [block.content.input_label, block.content.transcript, block.content.summary_label, block.content.summary_title, block.content.output_label, block.content.demo_note, ...block.content.summary_items]) text(label);
        break;
      case 'artifact-explorer':
        capability(block.content.capability_id); text(b?.accessibility_label);
        for (const example of block.content.examples) {
          text(example.label); text(example.filename); text(example.alt_text);
          if (example.asset_ref) asset(example.asset_ref);
          if (example.source.media_ref) asset(example.source.media_ref);
          for (const frame of example.frames ?? []) { asset(frame.asset_ref); text(frame.label); text(frame.time); }
        }
        break;
      case 'pricing': case 'faq': case 'footer': break;
    }
  }
}
