import { useId } from 'react';
import { safeHref } from './links';
import type { Action, ResolvedActions } from './types';
import { actionKey } from './types';
import type { Mark, PresentationResources } from './resources';
import { getProductMarkPath, productMarkDrawing } from './productMarks.js';

export function ProductMark({ kind }: { kind: Mark }) {
  const path = getProductMarkPath(kind);
  if (!path) throw new Error('Unsupported product mark');
  const { viewBox, fill, ...stroke } = productMarkDrawing;
  return <span className={`product-mark mark-${kind}`} aria-hidden="true"><svg viewBox={viewBox} fill={fill}><path d={path} stroke="currentColor" {...stroke} /></svg></span>;
}
/** Renders the real brand image when configured, else the finite SVG mark. */
export function BrandLogo({ kind, logo, alt }: { kind: Mark; logo?: string; alt?: string }) {
  if (logo) {
    // The contract is a same-origin path; reject absolute or scheme-relative URLs.
    const src = safeHref(logo);
    if (!src || !src.startsWith('/')) throw new Error('Unsafe product logo');
    return <span className={`product-mark product-logo mark-${kind}`}><img src={src} alt={alt ?? ''} loading="eager" decoding="async" /></span>;
  }
  return <ProductMark kind={kind} />;
}
export function Arrow() {
  return <svg viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M5 12h14m-6-6 6 6-6 6" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" /></svg>;
}
export function Heading({ text, level = 2, breaks }: { text: string; level?: 1 | 2; breaks?: number[] }) {
  const Tag = level === 1 ? 'h1' : 'h2';
  const offsets = [0, ...(breaks ?? []).filter(offset => offset > 0 && offset < text.length), text.length];
  const lines = breaks ? offsets.slice(0, -1).map((offset, index) => text.slice(offset, offsets[index + 1])) : text.split('\n');
  return <Tag>{lines.map((line, index) => <span key={index} className={index > 0 ? 'accent-line' : undefined}>{line}</span>)}</Tag>;
}
export function Intro({ heading, body, eyebrow, breaks }: { heading: string; body?: string; eyebrow?: string; breaks?: number[] }) {
  return <div className="section-intro">{eyebrow && <p className="eyebrow">{eyebrow}</p>}<div className="intro-row"><Heading text={heading} breaks={breaks} />{body && <p>{body}</p>}</div></div>;
}
export function ActionLink({ action, resolvedActions = {}, reason, className = 'button-primary' }: {
  action: Action; resolvedActions?: ResolvedActions; reason: string; className?: string;
}) {
  const id = useId();
  const owner = resolvedActions[actionKey(action)];
  const local = action.kind === 'anchor' || action.kind === 'app-detail';
  const href = safeHref(owner?.status === 'ready' ? owner.href : local ? action.target : undefined);
  const disabledReason = owner?.status === 'unavailable' ? owner.reason : action.reason || reason;
  const classes = `button ${className}`;
  if (action.kind !== 'unavailable' && owner?.status !== 'unavailable') {
    if (href) return <a className={classes} href={href} aria-label={action.accessible_label}>{action.label}<Arrow /></a>;
    if (owner?.status === 'ready' && owner.onActivate) return <button type="button" className={classes} onClick={owner.onActivate} aria-label={action.accessible_label}>{action.label}<Arrow /></button>;
  }
  return <span className="unavailable-action"><button type="button" className={classes} disabled aria-label={action.accessible_label} aria-describedby={id}>{action.label}<Arrow /></button><small id={id}>{disabledReason}</small></span>;
}
export function AssetImage({ assetRef, resources, eager = false, className, alt }: {
  assetRef: string; resources: PresentationResources; eager?: boolean; className?: string; alt?: string;
}) {
  const asset = resources.assets[assetRef];
  if (!asset || !safeHref(asset.src)) throw new Error(`Unresolved presentation asset: ${assetRef}`);
  return <img className={className} src={asset.src} width={asset.width} height={asset.height}
    // Legacy center is a cover crop; the released focal point owns placement.
    // Inline placement overrides decorative CSS without changing its frame/layout.
    style={{ objectFit: asset.crop_policy === 'contain' ? 'contain' : 'cover', objectPosition: `${String(asset.focal_point.x * 100)}% ${String(asset.focal_point.y * 100)}%` }}
    alt={alt ?? asset.alt} loading={eager ? 'eager' : 'lazy'} decoding="async"
    srcSet={asset.src_set} sizes={asset.sizes} />;
}
