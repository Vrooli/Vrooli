import { Component, type ReactNode } from 'react';
import { decodeProductPresentation, type ResolvedProductPresentation } from '../../../shared/api/productPresentation';
import { PresentationPage, type Presentation } from '../../public-landing/presentation';
import { withPresentationBase, resolveNavigationActions } from '../../public-landing/presentation/publicIntegration';

function PreviewFailure() {
  return <p role="alert" className="p-6 text-amber-200">This revision could not be rendered. Check its configured display mapping and supported block variants.</p>;
}

class PreviewBoundary extends Component<{ children: ReactNode }, { failed: boolean }> {
  state = { failed: false };
  static getDerivedStateFromError() { return { failed: true }; }
  render() {
    return this.state.failed
      ? <PreviewFailure />
      : this.props.children;
  }
}

function MappedPreview({ value, linkBase }: { value: ResolvedProductPresentation; linkBase: string }) {
  let presentation: Presentation;
  try {
    if (!value.diagnostics?.preview || !value.diagnostics.noindex || !value.diagnostics.noStore) return <PreviewFailure />;
    presentation = decodeProductPresentation(value);
  } catch { return <PreviewFailure />; }
  // No document, review fixture, or owner transaction crosses this boundary.
  return <PresentationPage presentation={withPresentationBase(presentation, linkBase)} resolvedActions={resolveNavigationActions(presentation, linkBase)} />;
}

export function RevisionPreview({ value, linkBase = '', ephemeral = false }: { value: ResolvedProductPresentation; linkBase?: string; ephemeral?: boolean }) {
  const label = ephemeral ? 'Unsaved document preview' : 'Private revision preview';
  return <section aria-label={label} className="presentation-preview-surface overflow-hidden rounded-xl border border-white/10 bg-slate-900">
    <div className="border-b border-white/10 p-5">
      <h2 className="text-lg font-semibold text-white">{label}</h2>
      <p className="text-sm text-slate-300">Administrator-only, noindex, no-store. Transaction actions are not connected in this preview.</p>
      <p className="mt-2 break-all font-mono text-xs text-slate-400">{ephemeral ? 'Document identity (not saved)' : 'Revision'}: {value.diagnostics?.resolvedRevision}</p>
    </div>
    <PreviewBoundary key={`${value.diagnostics?.resolvedRevision ?? ''}:${value.diagnostics?.blockDigest ?? ''}`}><MappedPreview value={value} linkBase={linkBase} /></PreviewBoundary>
  </section>;
}
