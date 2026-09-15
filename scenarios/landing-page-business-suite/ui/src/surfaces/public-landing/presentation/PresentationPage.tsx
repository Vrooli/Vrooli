import { Fragment, useId, useRef, useState, type CSSProperties } from 'react';
import type { Presentation, ResolvedActions } from './types';
import { ActionLink, ProductMark } from './primitives';
import { renderBlock } from './registry';
import { assertRendererContract } from './contract';
import { resolveResources } from './resolvedResources';
import './presentation.css';
import { safeHref } from './links';
import type { ResolvedPricing } from './commerce';

export interface PresentationPageProps {
  presentation: Presentation;
  resolvedActions?: ResolvedActions;
  resolvedPricing?: ResolvedPricing;
}

/** Pure integration boundary. The parent owns transport, SEO and commerce joins. */
export function PresentationPage({ presentation, resolvedActions, resolvedPricing }: PresentationPageProps) {
  const [menuOpen, setMenuOpen] = useState(false);
  const menu = useRef<HTMLButtonElement>(null);
  const id = useId();
  const page = presentation.page;
  const resources = resolveResources(presentation);
  if (presentation.schema_version !== 1 || !['signal', 'studio'].includes(page.theme.variant)) throw new Error('Unsupported presentation schema or theme');
  assertRendererContract(presentation);
  const shell = resources.shell;
  const colors = [page.theme.primary, page.theme.background, page.theme.accent];
  if (colors.some(color => !/^#[0-9a-f]{6}$/i.test(color))) throw new Error('Invalid presentation theme token');
  const theme = { '--ink': page.theme.primary, '--paper': page.theme.background, '--accent': page.theme.accent } as CSSProperties;
  // An embedded private preview belongs to the editor's landmark, not a second main.
  const Content = presentation.diagnostics.preview ? 'div' : 'main';
  return <div className={`presentation-page theme-${page.theme.variant} ${presentation.scope === 'bundle' ? 'bundle-page' : 'app-page'}`} lang={page.locale} style={theme} data-presentation-mode={presentation.mode}
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-boolean-literal-compare -- Capture boundary emits literal booleans even for malformed runtime input.
    data-presentation-preview={presentation.diagnostics.preview === true} data-presentation-fallback={presentation.diagnostics.fallback === true}>
    <a className="skip-link" href={`#${id}-main`}>{shell.skip_label}</a>
    {presentation.diagnostics.preview && <div className="private-notice">{shell.preview_label}</div>}
    <header className={`site-header wrap ${menuOpen ? 'menu-open' : ''}`} onKeyDown={event => { if (event.key === 'Escape' && menuOpen) { setMenuOpen(false); menu.current?.focus(); } }}>
      <a className="brand" href={safeHref(shell.brand_target)}><ProductMark kind={shell.brand_mark} /><span>{shell.brand_name}</span>{shell.brand_subtitle && <><span className="brand-divider" /><span className="brand-subtitle">{shell.brand_subtitle}</span></>}</a>
      <button ref={menu} type="button" className="menu-toggle" aria-label={shell.menu_label} aria-expanded={menuOpen} aria-controls={`${id}-nav`} onClick={() => { setMenuOpen(!menuOpen); }}><span /><span /></button>
      <nav id={`${id}-nav`} aria-label={page.navigation.label} onClick={() => { setMenuOpen(false); }}>{page.navigation.items.map((item, index) => <a key={index} href={safeHref(item.target)} aria-label={item.accessible_label}>{item.label}</a>)}{shell.header_action && <ActionLink action={shell.header_action} resolvedActions={resolvedActions} reason={shell.unavailable_reason} className="button-nav" />}</nav>
    </header>
    <Content id={`${id}-main`} tabIndex={-1}>{page.blocks.map(block => <Fragment key={block.id}>{renderBlock(block, { presentation, resources, resolvedActions, resolvedPricing })}</Fragment>)}</Content>
    <footer className="site-footer wrap" aria-label={page.footer.label}><div><a className="brand" href={safeHref(shell.footer_brand_target)}><ProductMark kind={shell.footer_brand_mark} /><span>{shell.footer_brand_name}</span></a><p>{shell.footer_tagline}</p></div><div className="footer-links">{page.footer.links.map((item, index) => <a key={index} href={safeHref(item.target)} aria-label={item.accessible_label}>{item.label}</a>)}</div><div className="footer-fine"><span>{shell.copyright}</span><span>{shell.footer_note}</span></div></footer>
  </div>;
}
