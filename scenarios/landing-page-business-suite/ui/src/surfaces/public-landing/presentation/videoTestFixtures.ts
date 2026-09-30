/** Synthetic configured video for isolated tests/dev only, never a released demo. */
import { decodeProductPresentation } from './decode';
import { publicConfig } from './publicTestFixtures';
import type { Block, DemoPlayback } from './types';
import { create } from '@bufbuild/protobuf';
import { PresentationBlockSchema, ResolvedPresentationAssetSchema } from '@vrooli/proto-types/landing-page-business-suite/v1/shared/product_presentation_pb';

export function videoFixture(provider: DemoPlayback['provider'] = 'youtube', layout: DemoPlayback['layout'] = 'stacked') {
  const wire = publicConfig().presentation;
  if (!wire) throw new Error('Missing public fixture');
  const presentation = decodeProductPresentation(wire);
  presentation.diagnostics.preview = true; presentation.diagnostics.noindex = true; presentation.diagnostics.no_store = true;
  const block: Block<'product-demo'> = { id: 'video-demo', kind: 'product-demo', version: 1, variant: 'recorded', content: {
    heading: 'Configured video demonstration', description: 'Synthetic video integration fixture. Not evidence of provider playback.',
    renderer_ref: 'video', poster_ref: 'poster', alt_text: 'Configured demonstration player',
    playback: { provider, external_url: provider === 'youtube' ? 'https://www.youtube.com/watch?v=dQw4w9WgXcQ' : 'https://vimeo.com/76979871', layout,
      play_label: 'Load configured player', caption: 'Configured caption remains visible.', unavailable_label: 'Configured player is unavailable.' },
  } };
  presentation.page.blocks = [block];
  presentation.assets = [{ id: 'poster', public_url: '/presentation/survey-relief.png', width: 1536, height: 1024, mime: 'image/png',
    release_ref: 'test-only-release', content_hash: 'a'.repeat(64), surface: 'test-poster', crop_policy: 'contain', focal_point: { x: 0.5, y: 0.5 },
    provenance: { provider: 'test', job_ref: '', candidate_ref: '' }, overlay_regions: [] }];
  presentation.page.display.asset_labels.poster = { alt: 'Configured demonstration poster', sizes: '(min-width: 1000px) 60vw, 100vw' };
  return { presentation, block };
}

/** Generated wire fixture exercises the real typed content/asset decoder. */
export function videoWireFixture() {
  const wire = publicConfig().presentation;
  if (!wire?.page?.display || !wire.diagnostics) throw new Error('Missing wire fixture');
  const { block, presentation } = videoFixture(); const c = block.content; const p = c.playback; const asset = presentation.assets?.[0];
  if (!p || !asset) throw new Error('Missing video fixture');
  wire.page.blocks = [create(PresentationBlockSchema, { id: block.id, kind: block.kind, variant: block.variant, version: block.version,
    content: { value: { case: 'productDemo', value: { heading: c.heading, description: c.description, rendererRef: c.renderer_ref, posterRef: c.poster_ref, altText: c.alt_text,
      playback: { provider: p.provider, externalUrl: p.external_url, layout: p.layout, playLabel: p.play_label, caption: p.caption, unavailableLabel: p.unavailable_label } } } },
  })];
  wire.assets = [create(ResolvedPresentationAssetSchema, { id: asset.id, publicUrl: asset.public_url, mime: asset.mime, width: asset.width, height: asset.height,
    releaseRef: asset.release_ref, contentHash: asset.content_hash, surface: asset.surface, cropPolicy: asset.crop_policy, focalPoint: asset.focal_point, provenance: { provider: 'test' } })];
  wire.page.display.assetLabels.poster = { $typeName: 'vrooli.landing_page_business_suite.v1.shared.PresentationAssetLabel', alt: 'Configured poster', sizes: '100vw' };
  Object.assign(wire.diagnostics, { preview: true, noindex: true, noStore: true });
  return wire;
}
