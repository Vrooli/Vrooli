import { Component, type ReactNode } from 'react';
import { decodeProductPresentation, type ResolvedProductPresentation } from '../../../shared/api/productPresentation';
import { PresentationPage, type Presentation } from '../../public-landing/presentation';

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

function MappedPreview({ value }: { value: ResolvedProductPresentation }) {
  let presentation: Presentation;
  try {
    if (!value.diagnostics?.preview || !value.diagnostics.noindex || !value.diagnostics.noStore) return <PreviewFailure />;
    presentation = decodeProductPresentation(value);
  } catch { return <PreviewFailure />; }
  // No document, review fixture, or owner transaction crosses this boundary.
  return <PresentationPage presentation={presentation} />;
}

export function RevisionPreview({ value }: { value: ResolvedProductPresentation }) {
  return <section aria-label="Private revision preview" className="overflow-hidden rounded-xl border border-white/10 bg-slate-900">
    <div className="border-b border-white/10 p-5">
      <h2 className="text-lg font-semibold text-white">Private revision preview</h2>
      <p className="text-sm text-slate-300">Administrator-only, noindex, no-store. Transaction actions are not connected in this preview.</p>
      <p className="mt-2 break-all font-mono text-xs text-slate-400">Revision: {value.diagnostics?.resolvedRevision}</p>
    </div>
    <PreviewBoundary key={`${value.diagnostics?.resolvedRevision}:${value.diagnostics?.blockDigest}`}><MappedPreview value={value} /></PreviewBoundary>
  </section>;
}
