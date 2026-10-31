import { useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { UNSAFE_NavigationContext, useHref, useNavigate, useParams } from 'react-router-dom';
import { AdminLayout } from '../components/AdminLayout';
import { listVariants } from '../../../shared/api/variants';
import type { Variant } from '../../../shared/api/types';
import { PresentationEditorRoute } from './PresentationEditorRoute';
import { presentationSystemUi } from '../../public-landing/presentation/systemUi';
import { updateMetaTags } from '../../../shared/lib/seo';
import { rememberVariantSession } from '../../../shared/lib/adminExperience';

/** BrowserRouter-compatible guard scoped to this editor, not global history patches. */
function DirtyNavigation({ dirty, children }: { dirty: boolean; children: ReactNode }) {
  const context = useContext(UNSAFE_NavigationContext);
  const approvedPop = useRef(false);
  const dirtyRef = useRef(dirty);
  dirtyRef.current = dirty;
  const allow = useCallback(() => !dirtyRef.current || window.confirm(presentationSystemUi.leave), []);
  const guarded = useMemo(() => ({ ...context, navigator: {
    ...context.navigator,
    push: (...args: Parameters<typeof context.navigator.push>) => { if (allow()) context.navigator.push(...args); },
    replace: (...args: Parameters<typeof context.navigator.replace>) => { if (allow()) context.navigator.replace(...args); },
    go: (delta: number) => { if (allow()) { approvedPop.current = true; context.navigator.go(delta); } },
  } }), [allow, context]);
  useEffect(() => {
    const index = () => {
      const state: unknown = window.history.state;
      return state && typeof state === 'object' && 'idx' in state && typeof state.idx === 'number' ? state.idx : undefined;
    };
    let acceptedIndex = index();
    let restoring = false;
    const pop = (event: PopStateEvent) => {
      const next = index();
      if (restoring) { restoring = false; event.stopImmediatePropagation(); return; }
      if (approvedPop.current || allow()) { approvedPop.current = false; acceptedIndex = next; return; }
      if (next !== undefined && acceptedIndex !== undefined && next !== acceptedIndex) {
        event.stopImmediatePropagation(); restoring = true; window.history.go(acceptedIndex - next);
      }
    };
    window.addEventListener('popstate', pop, true);
    return () => { window.removeEventListener('popstate', pop, true); };
  }, [allow, dirty]);
  return <UNSAFE_NavigationContext.Provider value={guarded}>{children}</UNSAFE_NavigationContext.Provider>;
}

export function PresentationAdminPage() {
  const [dirty, setDirty] = useState(false);
  useEffect(() => { updateMetaTags({ title: 'Presentation editor', noindex: true }); document.querySelector('link[rel="canonical"]')?.remove(); }, []);
  return <DirtyNavigation dirty={dirty}><EditorSelection onDirtyChange={setDirty} dirty={dirty} /></DirtyNavigation>;
}

function EditorSelection({ onDirtyChange, dirty }: { onDirtyChange: (dirty: boolean) => void; dirty: boolean }) {
  const { variantSlug } = useParams();
  const navigate = useNavigate();
  const linkBase = useHref('/');
  const [variants, setVariants] = useState<Variant[]>();
  const [error, setError] = useState(false);
  useEffect(() => {
    let active = true;
    void listVariants().then(result => { if (active) setVariants(result.variants); }).catch(() => { if (active) setError(true); });
    return () => { active = false; };
  }, []);
  const selected = variantSlug && variants?.some(v => v.slug === variantSlug);
  useEffect(() => {
    if (selected && variantSlug) {
      // Preserve existing private resume data without creating visitor identities.
      try { rememberVariantSession({ slug: variantSlug, surface: 'section' }); } catch { /* blocked local storage must not prevent editing */ }
    }
  }, [selected, variantSlug]);
  return <AdminLayout maxWidth="full" beforeLogout={() => { const leave = !dirty || window.confirm(presentationSystemUi.leave); if (leave) onDirtyChange(false); return leave; }}>
    <section className="mb-6 space-y-2" aria-label="Presentation variant selection">
      <label htmlFor="presentation-variant">Presentation variant</label>
      <select id="presentation-variant" className="block max-w-full rounded border border-white/20 bg-slate-900 px-3 py-2" value={selected ? variantSlug : ''} disabled={!variants} onChange={event => { navigate(event.target.value ? `/admin/presentation/${encodeURIComponent(event.target.value)}` : '/admin/presentation'); }}>
        <option value="">Choose a variant</option>
        {variants?.map(variant => <option key={variant.slug} value={variant.slug}>{variant.name || variant.slug}</option>)}
      </select>
      {error ? <p role="alert">Variants could not be loaded. Reload to try again.</p> : !variants ? <p role="status">Loading variants…</p> : variantSlug && !selected ? <p role="alert">The selected variant is unavailable.</p> : null}
    </section>
    {selected && variantSlug && <PresentationEditorRoute key={variantSlug} variantSlug={variantSlug} onDirtyChange={onDirtyChange} linkBase={linkBase} />}
  </AdminLayout>;
}
