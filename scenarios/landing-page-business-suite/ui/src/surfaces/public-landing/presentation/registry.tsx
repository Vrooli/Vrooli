/* eslint-disable react-refresh/only-export-components -- finite dispatch module; PresentationPage owns the refresh boundary. */
import type { ReactNode } from 'react';
import type { Block, BlockKind, Presentation, ResolvedActions, Action } from './types';
import type { PresentationResources } from './resources';
import { ActionLink, Arrow, AssetImage, BrandLogo, Heading, Intro } from './primitives';
import { Phone, Visual } from './exhibits';
import { safeHref } from './links';
import { ArtifactExplorer } from './ArtifactExplorer';
import { PricingCards } from './PricingCards';
import { ProductDemo } from './ProductDemo';
import { ConstellationFigure } from './ConstellationSky';
import { constellationForMark } from './constellations';
import type { ResolvedPricing } from './commerce';

export interface RenderContext { presentation: Presentation; resources: PresentationResources; resolvedActions?: ResolvedActions; resolvedPricing?: ResolvedPricing }
type Registry = { [K in BlockKind]: (block: Block<K>, context: RenderContext) => ReactNode };
function Actions({ actions, context }: { actions: Action[]; context: RenderContext }) {
  return <div className="hero-actions">{actions.map((action, index) => <ActionLink key={index} action={action} resolvedActions={context.resolvedActions} reason={context.resources.shell.unavailable_reason} className={index === 0 ? 'button-primary' : 'button-text'} />)}</div>;
}
function Hero({ block, context }: { block: Block<'product-hero' | 'bundle-hero'>; context: RenderContext }) {
  const { resources, presentation } = context;
  const display = resources.blocks[block.id] ?? {};
  const content = block.content;
  const bundle = block.kind === 'bundle-hero';
  // Decorative: the page's own figure, drawn from real star positions, behind
  // the hero and resolving into the exhibit. Never page-owned content.
  const figure = constellationForMark(resources.shell.brand_mark);
  return <section id={block.id} data-block={block.kind} data-capture-landmark="hero" className={`hero hero-${bundle ? 'editorial' : 'center'}`}>
    {figure && <ConstellationFigure chart={figure} />}
    <div className="hero-copy"><p className="eyebrow"><span />{content.eyebrow}</p><Heading level={1} text={content.title} breaks={display.heading_breaks} /><p className="hero-description">{content.description}</p><Actions actions={content.actions} context={context} />{display.note && <p className="hero-note">{display.note}</p>}</div>
    <figure className="hero-stage" aria-label={content.accessibility_label}>
      {block.kind === 'product-hero' ? <Visual visualRef={block.content.fixture_ref || block.content.visual_ref} resources={resources} eager /> : <>
        {(() => {
          // Two or more product views compose as a hand-held fan of cards; a
          // card scrolls to its app's spotlight, where the full story waits.
          const deck = block.content.hero_items.length > 1 && block.content.hero_items.every(item => item.exhibit_kind === 'product-view');
          return <div className={`bundle-art ${deck ? 'bundle-art-deck' : block.content.hero_items.length !== 2 ? 'bundle-art-grid' : ''}`} data-deck-count={deck ? block.content.hero_items.length : undefined}>
            {block.content.hero_items.map((item, index) => {
              const app = presentation.spotlights?.find(app => app.app_key === item.app_key);
              const fixtureRef = display.hero_fixture_refs?.[item.app_key];
              const visual = fixtureRef ? resources.visuals[fixtureRef] : undefined;
              if (!app) throw new Error(`Unresolved hero app: ${item.app_key}`);
              if (deck) {
                const style = resources.apps[item.app_key];
                return <a className="hero-app-group deck-card" key={item.app_key} data-hero-app-key={item.app_key} data-deck-index={index} href={`#app-${item.app_key}`} aria-label={`${app.name} — ${item.detail_label}`}>
                  <div className="bundle-terminal"><Visual visualRef={fixtureRef ?? item.visual_ref} resources={resources} interactive={false} eager /></div>
                  <span className="deck-card-label"><BrandLogo kind={style?.mark ?? resources.shell.brand_mark} logo={style?.logo} alt={style?.logo_alt} /><b>{app.name}</b><small>{item.detail_label}</small></span>
                </a>;
              }
              return <div className="hero-app-group" key={item.app_key} data-hero-app-key={item.app_key} role="group" aria-label={app.name}>
                {item.exhibit_kind === 'artwork' && visual?.kind === 'backdrop'
                  ? visual.asset_refs.map((asset_ref, index) => <div key={asset_ref} className={`art-print ${index === 0 ? 'print-front' : 'print-back'}`}><AssetImage assetRef={asset_ref} resources={resources} eager />{index === 0 && <div className="hero-print-label"><BrandLogo kind={visual.mark} logo={resources.apps[item.app_key]?.logo} alt={resources.apps[item.app_key]?.logo_alt} />{app.name}</div>}</div>)
                  : <div className="bundle-terminal"><Visual visualRef={fixtureRef ?? item.visual_ref} resources={resources} interactive={false} eager /></div>}
              </div>;
            })}
            {block.content.hero_items.length > 0 && <div className="art-seal" aria-hidden="true"><BrandLogo kind={resources.shell.brand_mark} logo={resources.shell.brand_logo} /></div>}
          </div>;
        })()}
        <nav className="hero-product-key" aria-label={block.content.accessibility_label}>{block.content.hero_items.map(item => {
          const app = presentation.spotlights?.find(app => app.app_key === item.app_key);
          const style = resources.apps[item.app_key];
          if (!app || !style || !safeHref(app.detail_route)) throw new Error(`Unresolved hero detail route: ${item.app_key}`);
          return <a key={item.app_key} href={app.detail_route}><BrandLogo kind={style.mark} logo={style.logo} alt={style.logo_alt} /><span><b>{app.name}</b><small>{item.detail_label}</small></span><Arrow /></a>;
        })}</nav>
      </>}
      {display.accessibility_label && <figcaption>{display.accessibility_label}</figcaption>}
    </figure>
  </section>;
}

export const rendererRegistry: Registry = {
  'product-hero': (block, context) => <Hero block={block} context={context} />,
  'bundle-hero': (block, context) => <Hero block={block} context={context} />,
  'capability-strip': (block, { resources }) => <section id={block.id} data-block={block.kind} className="principles wrap" aria-label={block.content.heading}>{block.content.items.map((item, index) => {
    const target = safeHref(resources.blocks[block.id]?.anchors?.[item.capability_id]);
    const content = <><span className="principle-icon" aria-hidden="true">{['⊞', '◎', '↗'][index % 3]}</span>{item.label}</>;
    return <div key={item.capability_id}>{target ? <a href={target}>{content}</a> : content}</div>;
  })}</section>,
  'product-story': (block, { resources }) => <section id={block.id} data-block={block.kind} className="story wrap"><Intro heading={block.content.heading} body={block.content.body} eyebrow={resources.blocks[block.id]?.eyebrow} breaks={resources.blocks[block.id]?.heading_breaks} /><div className="story-grid">{block.content.items.map((item, index) => <article key={index}><span className="item-number">{String(index + 1).padStart(2, '0')}</span><div className="story-rule" /><h3>{item.title}</h3><p>{item.description}</p>{item.visual_ref && <AssetImage assetRef={item.visual_ref} resources={resources} alt={item.alt_text} />}</article>)}</div></section>,
  'product-demo': (block, { resources, presentation }) => <ProductDemo block={block} resources={resources} revision={presentation.diagnostics.resolved_revision} />,
  'artifact-explorer': (block, { resources }) => <ArtifactExplorer key={block.content.selected_example_id} block={block} resources={resources} />,
  'voice-story': (block, { resources }) => {
    const content = block.content;
    return <section id={block.id} data-block={block.kind} className="voice-story wrap"><div className="voice-copy"><p className="eyebrow">{content.eyebrow}</p><Heading text={content.heading} breaks={resources.blocks[block.id]?.heading_breaks} /><p>{content.body}</p><dl>{content.features.map((feature, index) => <div key={index}><dt>{feature.title}</dt><dd>{feature.description}</dd></div>)}</dl><p className="voice-note">{content.note}</p>{content.provider_qualification !== content.note && <p className="voice-qualification">{content.provider_qualification}</p>}</div>
      <figure className="voice-scene"><div className="voice-input"><p>{content.input_label}</p><div className="voice-wave" aria-hidden="true">{content.waveform.map((height, index) => <i key={index} style={{ height: Math.max(0, Math.min(1, height) * 64) }} />)}</div><blockquote>{content.transcript}</blockquote></div><div className="summary-card"><p className="exhibit-kicker">{content.summary_label}</p><h3>{content.summary_title}</h3><ul>{content.summary_items.map((line, index) => <li key={index}>{line}</li>)}</ul><div className="summary-audio"><span aria-hidden="true">◖))</span>{content.output_label}</div></div><figcaption>{content.demo_note}</figcaption></figure>
    </section>;
  },
  'device-story': (block, { resources }) => {
    const fixture = resources.visuals[resources.blocks[block.id]?.fixture_ref ?? block.content.visual_ref];
    const display = resources.blocks[block.id] ?? {};
    return <section id={block.id} data-block={block.kind} className="mobile-story wrap"><div className="mobile-copy"><p className="eyebrow">{display.eyebrow}</p><Heading text={block.content.heading} breaks={resources.blocks[block.id]?.heading_breaks} /><p>{block.content.description}</p><span className="mobile-note">{display.note}</span></div><figure aria-label={block.content.alt_text}>{fixture?.kind === 'workspace' && block.variant === 'phone' ? <Phone fixture={fixture} /> : <Visual visualRef={display.fixture_ref ?? block.content.visual_ref} resources={resources} />}</figure></section>;
  },
  'capability-roadmap': (block, { presentation, resources }) => <section id={block.id} data-block={block.kind} className="roadmap-story wrap"><Intro heading={block.content.heading} body={block.content.description} eyebrow={resources.blocks[block.id]?.eyebrow} breaks={resources.blocks[block.id]?.heading_breaks} /><div className="roadmap-grid">{block.content.capability_ids.map((id, index) => {
    const capability = presentation.capabilities?.find(capability => capability.id === id);
    if (!capability) throw new Error(`Unresolved capability: ${id}`);
    return <article key={id} data-capability-id={id} data-capability-status={capability.status}><div><span className="roadmap-symbol" aria-hidden="true">{['⌘', '◇', '▯'][index % 3]}</span><span className="roadmap-badge">{capability.status_label}</span></div><h3>{capability.label}</h3>{capability.benefits.map((benefit, i) => <p key={i}>{benefit}</p>)}{capability.constraints?.map((constraint, i) => <p key={i}>{constraint}</p>)}</article>;
  })}</div></section>,
  'app-spotlights': (block, { presentation, resources }) => <section id={block.id} data-block={block.kind} data-capture-landmark="catalog" className="catalog wrap"><Intro heading={block.content.heading} body={resources.blocks[block.id]?.description} eyebrow={resources.blocks[block.id]?.eyebrow} breaks={resources.blocks[block.id]?.heading_breaks} /><div className="catalog-grid">{block.content.app_keys.map((key, index) => {
    const app = presentation.spotlights?.find(app => app.app_key === key);
    const style = resources.apps[key];
    if (!app || !style || !safeHref(app.detail_route)) throw new Error(`Unresolved app spotlight: ${key}`);
    return <article key={key} id={`app-${key}`} className={`product-card ${style.tone}`} data-app-key={key}><a className="product-card-visual" href={app.detail_route} aria-label={style.detail_label}><Visual visualRef={style.fixture_ref ?? style.visual_ref ?? ''} resources={resources} interactive={false} /><span className="visual-arrow" aria-hidden="true">↗</span></a><div className="product-card-copy"><div className="product-name"><BrandLogo kind={style.mark} logo={style.logo} alt={style.logo_alt} /><h3>{app.name}</h3><span className="product-number">{String(index + 1).padStart(2, '0')}</span></div><h4>{app.tagline}</h4><p>{app.description}</p><a className="detail-link" href={app.detail_route}>{style.detail_label}<Arrow /></a></div></article>;
  })}</div></section>,
  'closing-action': (block, context) => <section id={block.id} data-block={block.kind} data-capture-landmark="closing" className="closing wrap">{block.content.visual_ref && <div className="closing-art" aria-hidden="true"><AssetImage assetRef={block.content.visual_ref} resources={context.resources} alt="" /></div>}<div className="closing-content"><p className="eyebrow">{context.resources.blocks[block.id]?.eyebrow}</p><Heading text={block.content.heading} breaks={context.resources.blocks[block.id]?.heading_breaks} /><p>{block.content.description}</p><Actions actions={block.content.actions} context={context} /></div></section>,
  'pricing': (block, context) => <section id={block.id} data-block={block.kind} className="pricing wrap"><Intro heading={block.content.heading} body={block.content.description} /><PricingCards block={block} prices={context.resolvedPricing} actions={context.resolvedActions} reason={context.resources.shell.unavailable_reason} locale={context.presentation.page.locale} /><Actions actions={block.content.actions.filter(action => action.kind !== 'purchase' || !block.content.plan_refs.includes(action.plan_ref ?? ''))} context={context} /></section>,
  'faq': (block) => <section id={block.id} data-block={block.kind} className="faq wrap"><Heading text={block.content.heading} />{block.content.items.map((item, index) => <details key={index}><summary aria-label={item.accessible_label}>{item.question}</summary><p>{item.answer}</p></details>)}</section>,
  'footer': (block) => <section id={block.id} data-block={block.kind} className="footer-block wrap" aria-label={block.content.label}>{block.content.links.map((link, index) => <a key={index} href={safeHref(link.target)} aria-label={link.accessible_label}>{link.label}</a>)}</section>,
};

export function renderBlock(block: Block, context: RenderContext): ReactNode {
  // The checked kind discriminant chooses the matching registry/content pair.
  const render = rendererRegistry[block.kind] as (block: Block, context: RenderContext) => ReactNode;
  return render(block, context);
}
