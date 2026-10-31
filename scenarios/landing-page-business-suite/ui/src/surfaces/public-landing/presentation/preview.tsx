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
  const [selected, setSelected] = useState('0');
  // A metadata-rich synthetic catalog exercising the full plan design: four
  // tiers per interval, a highlighted tier, and one-time credit top-ups.
  const tiers = [
    { tier: 'solo', name: 'Solo', month: 2900, year: 23900, subtitle: 'For one founder shipping alone', cta: 'Start with Solo', features: ['Every suite app on desktop and web', 'Linux, macOS, and Windows installers', 'All releases and updates included', 'Email support'] },
    { tier: 'pro', name: 'Pro', month: 7900, year: 64800, subtitle: 'For founders running real daily volume', cta: 'Choose Pro', badge: 'Most popular', highlight: true, features: ['Everything in Solo', 'Usage limits sized for production work', 'Priority support', 'Cancel anytime'] },
    { tier: 'studio', name: 'Studio', month: 19900, year: 166800, subtitle: 'For teams shipping client work every week', cta: 'Choose Studio', features: ['Everything in Pro', 'Studio-scale usage limits', 'Priority support', 'Cancel anytime'] },
    { tier: 'business', name: 'Business', month: 49900, year: 418800, subtitle: 'For a company running the suite at scale', cta: 'Scale with Business', features: ['Everything in Studio', 'Highest usage limits across the suite', 'Priority support with onboarding help', 'Cancel anytime'] },
  ];
  const plan = (t: typeof tiers[number], interval: 'month' | 'year') => ({
    ...ownerPricing.monthly[0]!, plan_name: t.name, plan_tier: t.tier, billing_interval: interval,
    amount_cents: interval === 'month' ? t.month : t.year, stripe_price_id: `price-${t.tier}-${interval}`,
    metadata: { subtitle: t.subtitle, cta_label: t.cta, badge: t.badge, highlight: t.highlight, features: t.features },
  });
  const topup = (amount: number) => ({
    ...ownerPricing.monthly[0]!, plan_name: `$${amount / 100} credits`, plan_tier: 'credits', billing_interval: 'one_time' as const,
    amount_cents: amount, stripe_price_id: `price-credits-${amount}`, kind: 'credits_topup' as const,
    metadata: { subtitle: 'One-time purchase. Credits never expire.' },
  });
  const reviewPricing = {
    ...ownerPricing,
    monthly: tiers.map(t => plan(t, 'month')),
    yearly: tiers.map(t => plan(t, 'year')),
    credit_topups: [topup(1000), topup(5000), topup(10000)],
  };
  const refs = [...reviewPricing.monthly, ...reviewPricing.yearly, ...reviewPricing.credit_topups].map(p => p.stripe_price_id);
  const block: Block<'pricing'> = {
    id: 'review-pricing', kind: 'pricing', version: 1, variant: 'compact',
    content: {
      heading: 'Simple pricing for everything.', description: 'Synthetic owner catalog for isolated visual review.',
      plan_refs: refs,
      actions: refs.map(ref => ({ kind: 'purchase' as const, label: reviewPricing.monthly.concat(reviewPricing.yearly, reviewPricing.credit_topups).find(p => p.stripe_price_id === ref)?.metadata?.cta_label as string ?? 'Buy credits', accessible_label: ref, plan_ref: ref })),
    },
  };
  const presentation = parseProductPresentation(JSON.stringify(fixture.presentation));
  presentation.page.blocks = [block];
  return <main className="presentation-page download-page theme-signal" lang="en"><div className="wrap">
    <DownloadChooser title="Example application" description="Configured download description for this application." options={downloadOptions} selected={selected} onSelect={setSelected} state={{ status: 'idle' }} disabledReason="Private review: downloads are unavailable." unavailableReason="Configured unavailable" detectedPlatform="linux" />
    <section className="pricing"><h2>{block.content.heading}</h2><p>{block.content.description}</p><PricingCards block={block} prices={resolvePricing(presentation, reviewPricing)} reason="Private review" locale="en" /></section>
  </div></main>;
}
const query = new URLSearchParams(window.location.search);
const video = videoFixture(query.get('provider') === 'vimeo' ? 'vimeo' : 'youtube', query.get('layout') === 'split' ? 'split' : 'stacked');
if (query.get('poster') === 'missing' && video.presentation.assets?.[0]) video.presentation.assets[0].public_url = '/presentation/missing-poster.png';
createRoot(mount).render(query.get('editor') === '1' ? <Suspense fallback={null}><EditorReview /></Suspense> : query.get('commerce') === '1' ? <CommerceReview /> : <PresentationPage presentation={query.get('video') === '1' ? video.presentation : parseProductPresentation(JSON.stringify(fixture.presentation))} />);
