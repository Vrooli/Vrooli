import { useId, useRef, useState, type ReactNode } from 'react';
import { Link, useHref } from 'react-router-dom';
import { StarField } from '../presentation/ConstellationSky';
import { ActionLink } from '../presentation/primitives';
import { useSitePageMeta, type SitePageMeta } from './siteMeta';
import '../presentation/presentation.css';
import './site.css';
import { useSiteIdentity, type SiteIdentity } from './useSiteIdentity';
import { siteCopy } from './siteCopy';

function Brand({ identity, className = 'brand' }: { identity: SiteIdentity; className?: string }) {
  return <Link className={className} to="/">
    {identity.brandLogo
      ? <span className="product-mark product-logo"><img src={identity.brandLogo} alt="" width={40} height={40} loading="eager" decoding="async" /></span>
      : <span className="site-brand-initial" aria-hidden="true">{identity.brandName.slice(0, 1)}</span>}
    <span>{identity.brandName || <span className="site-skeleton site-skeleton-brand" />}</span>
  </Link>;
}

export function SiteFooter({ identity }: { identity: SiteIdentity }) {
  return <footer className="site-footer site-footer-full wrap" aria-label={siteCopy.footerLabel}>
    <div className="site-footer-brand">
      <Brand identity={identity} />
      {identity.tagline && <p>{identity.tagline}</p>}
    </div>
    <nav className="site-footer-nav" aria-label={siteCopy.footerNavLabel}>
      <p className="site-footer-heading">{siteCopy.company}</p>
      <Link to="/contact">{siteCopy.contact}</Link>
      <Link to="/privacy">{siteCopy.privacy}</Link>
      <Link to="/terms">{siteCopy.terms}</Link>
    </nav>
    <address className="site-footer-contact">
      <p className="site-footer-heading">{siteCopy.reachUs}</p>
      {identity.contactEmail && <a href={`mailto:${identity.contactEmail}`}>{identity.contactEmail}</a>}
      {identity.addressLines.length > 0 && <span>{identity.addressLines.map((line, index) => <span key={index}>{line}</span>)}</span>}
      {!identity.contactEmail && identity.addressLines.length === 0 && <Link to="/contact">{siteCopy.sendMessage}</Link>}
    </address>
    <div className="footer-fine">
      <span>{identity.copyright}</span>
      {identity.businessName && identity.businessName !== identity.brandName && <span>{identity.businessName}</span>}
    </div>
  </footer>;
}

/**
 * Chrome for site-owned pages (contact, legal, thank-you, not-found, sign-in):
 * the landing page's night sky, masthead and footer, around page content.
 */
export function SiteShell({ meta, children, width = 'default', identity: provided, chrome = 'full' }: { meta: SitePageMeta; children: ReactNode; width?: 'default' | 'narrow' | 'reading' | 'wide'; identity?: SiteIdentity; /** 'minimal' keeps the sky and brand but drops navigation and the footer, for focused tasks such as operator sign-in. */ chrome?: 'full' | 'minimal' }) {
  const own = useSiteIdentity();
  const identity = provided ?? own;
  useSitePageMeta(meta, identity);
  const [menuOpen, setMenuOpen] = useState(false);
  const menu = useRef<HTMLButtonElement>(null);
  const id = useId();
  const signIn = useHref('/auth/login');
  return <div className="presentation-page theme-signal site-page intro-replay" data-site-page="">
    <a className="skip-link" href={`#${id}-main`}>{siteCopy.skip}</a>
    <StarField />
    <header className={`site-header wrap ${menuOpen ? 'menu-open' : ''}`} onKeyDown={event => { if (event.key === 'Escape' && menuOpen) { setMenuOpen(false); menu.current?.focus(); } }}>
      <Brand identity={identity} />
      {chrome === 'full' && <button ref={menu} type="button" className="menu-toggle" aria-label={siteCopy.menu} aria-expanded={menuOpen} aria-controls={`${id}-nav`} onClick={() => { setMenuOpen(!menuOpen); }}><span /><span /></button>}
      {chrome === 'full' ? <nav id={`${id}-nav`} aria-label={siteCopy.primaryNavLabel} onClick={() => { setMenuOpen(false); }}>
        <Link to="/">{siteCopy.home}</Link>
        <Link to="/contact">{siteCopy.contact}</Link>
        <a className="nav-signin" href={signIn}>{siteCopy.signIn}</a>
        {identity.headerAction && <ActionLink action={identity.headerAction.action} resolvedActions={identity.headerAction.resolvedActions} reason={identity.headerAction.reason} className="button-nav" />}
      </nav> : <Link className="site-minimal-exit" to="/">{siteCopy.home}</Link>}
    </header>
    <main id={`${id}-main`} tabIndex={-1} className={`site-main site-main-${width} wrap`}>{children}</main>
    {chrome === 'full' ? <SiteFooter identity={identity} /> : <footer className="site-minimal-footer wrap"><span>{identity.copyright}</span></footer>}
  </div>;
}
