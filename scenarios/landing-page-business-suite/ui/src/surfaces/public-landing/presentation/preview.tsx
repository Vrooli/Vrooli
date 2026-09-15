/** Explicit standalone development entry; never imported by the application. */
/* eslint-disable react-refresh/only-export-components -- Standalone review mount, not a hot-reloaded application module. */
import { createRoot } from 'react-dom/client';
import { PresentationPage } from './PresentationPage';
import { parseProductPresentation } from './decode';
import signal from './fixtures/signal.json';
import studio from './fixtures/studio.json';
import { lazy, Suspense, useState } from 'react';
import { DownloadChooser } from './DownloadChooser';
import { PricingCards } from './PricingCards';
import { downloadOptions, ownerPricing } from './commerceFixtures';
import { resolvePricing } from './commerce';
import type { Block } from './types';
import { videoFixture } from './videoTestFixtures';
const EditorReview = lazy(() => import('../../admin-portal/presentation/editorReview').then(module => ({ default: module.EditorReview })));
const fixture = new URLSearchParams(window.location.search).get('design') === 'studio' ? studio : signal;
const mount = document.getElementById('presentation-preview');
if (!mount) throw new Error('Missing presentation preview mount');
function CommerceReview() {
  const [selected, setSelected] = useState('');
  const block: Block<'pricing'> = { id: 'review-pricing', kind: 'pricing', version: 1, variant: 'compact', content: { heading: 'Configured plans', description: 'Synthetic owner catalog for isolated visual review.', plan_refs: ['price-month', 'price-year'], actions: [] } };
  const presentation = parseProductPresentation(JSON.stringify(fixture.presentation));
  presentation.page.blocks = [block];
  return <main className="presentation-page download-page theme-signal" lang="en"><div className="wrap">
    <DownloadChooser title="Example application" description="Configured download description for this application." options={downloadOptions} selected={selected} onSelect={setSelected} state={{ status: 'idle' }} disabledReason="Private review: downloads are unavailable." unavailableReason="Configured unavailable" />
    <section className="pricing"><h2>{block.content.heading}</h2><p>{block.content.description}</p><PricingCards block={block} prices={resolvePricing(presentation, ownerPricing)} reason="Private review" locale="en" /></section>
  </div></main>;
}
const query = new URLSearchParams(window.location.search);
const video = videoFixture(query.get('provider') === 'vimeo' ? 'vimeo' : 'youtube', query.get('layout') === 'split' ? 'split' : 'stacked');
if (query.get('poster') === 'missing' && video.presentation.assets?.[0]) video.presentation.assets[0].public_url = '/presentation/missing-poster.png';
createRoot(mount).render(query.get('editor') === '1' ? <Suspense fallback={null}><EditorReview /></Suspense> : query.get('commerce') === '1' ? <CommerceReview /> : <PresentationPage presentation={query.get('video') === '1' ? video.presentation : parseProductPresentation(JSON.stringify(fixture.presentation))} />);
