import { Plus } from 'lucide-react';
import { MetricsModeProvider } from '../../../shared/hooks/useMetrics';
import { Button } from '../../../shared/ui/button';
import type {
  ContentSection,
  LandingConfigResponse,
  LandingSection,
} from '../../../shared/api';
import type { VariantContext } from '../controllers/sectionEditorController';
import { STYLING_CONFIG, getVariantGuidance } from '../services/section.service';

/**
 * Variant Section Timeline - Shows all sections for the current variant
 */
interface VariantSectionTimelineProps {
  variantName: string;
  sections: LandingSection[];
  loading: boolean;
  error: string | null;
  currentSectionId: number | null;
  currentSectionType: ContentSection['section_type'];
  onNavigateSection: (section: LandingSection) => void;
  onAddSection: () => void;
  onReorderSection: (section: LandingSection, direction: 'up' | 'down') => void;
  reorderingSectionId: number | null;
  reorderError: string | null;
}

export function VariantSectionTimeline({
  variantName,
  sections,
  loading,
  error,
  currentSectionId,
  currentSectionType,
  onNavigateSection,
  onAddSection,
  onReorderSection,
  reorderingSectionId,
  reorderError,
}: VariantSectionTimelineProps) {
  return (
    <div className="bg-white/5 border border-white/10 rounded-xl p-6 space-y-4" data-testid="variant-section-timeline">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Variant Timeline</p>
          <h2 className="text-lg font-semibold text-white">Sections for {variantName}</h2>
          <p className="text-xs text-slate-500">Jump directly to any section without leaving the editor.</p>
        </div>
        <Button variant="outline" size="sm" className="gap-2" onClick={onAddSection}>
          <Plus className="h-4 w-4" />
          New Section
        </Button>
      </div>
      {loading && <p className="text-sm text-slate-400">Loading sections...</p>}
      {error && (
        <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-200">
          {error}
        </div>
      )}
      {!loading && !error && sections.length === 0 && (
        <p className="text-sm text-slate-400">
          This variant has no sections yet. Use the button above to create the first one.
        </p>
      )}
      {!loading && !error && sections.length > 0 && (
        <div className="space-y-2">
          {sections.map((section) => {
            const isActive = currentSectionId
              ? section.id === currentSectionId
              : section.section_type === currentSectionType;
            const badge = section.enabled === false ? 'Disabled' : 'Enabled';
            const isFirst = sections[0]?.id === section.id;
            const isLast = sections[sections.length - 1]?.id === section.id;
            return (
              <div
                key={`${section.section_type}-${section.id ?? section.order}`}
                role="button"
                tabIndex={0}
                onClick={() => onNavigateSection(section)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' || event.key === ' ') {
                    event.preventDefault();
                    onNavigateSection(section);
                  }
                }}
                className={`rounded-xl border px-4 py-3 text-left transition-colors ${
                  isActive ? 'border-white/40 bg-white/10' : 'border-white/10 hover:border-white/30'
                }`}
              >
                <div className="flex items-center justify-between gap-4">
                  <div>
                    <div className="text-xs text-slate-500">#{section.order ?? '-'}</div>
                    <div className="text-sm font-medium capitalize text-white">{section.section_type}</div>
                    <div className="text-[11px] uppercase tracking-wide text-slate-500">{badge}</div>
                  </div>
                  <div className="flex flex-wrap gap-2 text-xs text-slate-400">
                    {section.id && (
                      <>
                        <button
                          type="button"
                          className="rounded-full border border-white/20 px-3 py-1 hover:border-white/40 disabled:opacity-50"
                          onClick={(event) => {
                            event.stopPropagation();
                            onReorderSection(section, 'up');
                          }}
                          disabled={reorderingSectionId !== null || isFirst}
                        >
                          Move up
                        </button>
                        <button
                          type="button"
                          className="rounded-full border border-white/20 px-3 py-1 hover:border-white/40 disabled:opacity-50"
                          onClick={(event) => {
                            event.stopPropagation();
                            onReorderSection(section, 'down');
                          }}
                          disabled={reorderingSectionId !== null || isLast}
                        >
                          Move down
                        </button>
                      </>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
      {reorderError && (
        <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-200">
          {reorderError}
        </div>
      )}
    </div>
  );
}

/**
 * Variant Context Card - Shows variant axes and guidelines
 */
interface VariantContextCardProps {
  context: VariantContext | null;
  error: string | null;
  loading: boolean;
}

export function VariantContextCard({ context, error, loading }: VariantContextCardProps) {
  if (!context && !error && !loading) {
    return null;
  }

  return (
    <div className="bg-white/5 border border-white/10 rounded-xl p-6 space-y-4" data-testid="variant-context-card">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h2 className="text-lg font-semibold">Variant Context</h2>
          <p className="text-sm text-slate-400">
            Align copy with the selected persona, JTBD, and conversion style pulled from variant_space.json.
          </p>
        </div>
        {context?.variant && (
          <span className="text-xs uppercase tracking-wide text-slate-500 bg-slate-900/60 px-3 py-1 rounded-full">
            {context.variant.name}
          </span>
        )}
      </div>

      {loading && (
        <div className="text-slate-400 text-sm">Loading variant guidance...</div>
      )}

      {error && (
        <div className="text-sm text-red-400">
          {error}
        </div>
      )}

      {context && (
        <div className="space-y-4">
          {context.axes.map((axis) => (
            <div key={axis.axisId} className="border-l-2 border-purple-500/40 pl-4">
              <div className="flex items-center justify-between text-xs uppercase text-slate-500 mb-1">
                <span>{axis.axisLabel}</span>
                {axis.axisNote && <span className="text-slate-600">{axis.axisNote}</span>}
              </div>
              <div className="text-lg font-semibold text-white">
                {axis.selectionLabel || 'Not selected'}
              </div>
              {axis.selectionDescription && (
                <p className="text-sm text-slate-400 mt-1">{axis.selectionDescription}</p>
              )}
              {axis.agentHints && axis.agentHints.length > 0 && (
                <ul className="mt-2 text-sm text-slate-400 space-y-1 list-disc list-inside">
                  {axis.agentHints.map((hint, index) => (
                    <li key={index}>{hint}</li>
                  ))}
                </ul>
              )}
            </div>
          ))}

          {(context.variantSpace.agentGuidelines && context.variantSpace.agentGuidelines.length > 0) && (
            <div className="rounded-lg border border-white/10 bg-slate-900/60 p-4 text-sm text-slate-300 space-y-2">
              <div className="font-medium text-slate-200">Agent Guidelines</div>
              <ul className="list-disc list-inside space-y-1">
                {context.variantSpace.agentGuidelines.map((guideline, index) => (
                  <li key={index}>{guideline}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

/**
 * Styling Guardrails Card - Shows brand styling guidance
 */
interface StylingGuardrailsCardProps {
  variantSlug?: string;
}

export function StylingGuardrailsCard({ variantSlug }: StylingGuardrailsCardProps) {
  const variantGuidance = getVariantGuidance(variantSlug);

  return (
    <div className="bg-white/5 border border-white/10 rounded-xl p-6 space-y-4" data-testid="styling-guardrails-card">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h2 className="text-lg font-semibold">Styling & Tone Guardrails</h2>
          <p className="text-sm text-slate-400">
            Pulled from styling.json so CTA, palette, and messaging stay aligned.
          </p>
        </div>
        <span className="text-xs uppercase tracking-wide text-slate-500 bg-slate-900/60 px-3 py-1 rounded-full">
          {STYLING_CONFIG.brand?.product_name ?? 'Brand'}
        </span>
      </div>

      {STYLING_CONFIG.tone?.voice && (
        <p className="text-sm text-slate-300">
          Voice: <span className="text-white font-medium">{STYLING_CONFIG.tone.voice}</span>
        </p>
      )}

      {STYLING_CONFIG.tone?.keywords && (
        <div className="flex flex-wrap gap-2">
          {STYLING_CONFIG.tone.keywords.map((keyword) => (
            <span key={keyword} className="text-xs uppercase tracking-wide bg-purple-500/20 text-purple-200 px-2 py-1 rounded-full">
              {keyword}
            </span>
          ))}
        </div>
      )}

      {STYLING_CONFIG.usage_notes && STYLING_CONFIG.usage_notes.length > 0 && (
        <div className="space-y-2">
          <div className="text-xs uppercase text-slate-500">Usage Notes</div>
          <ul className="list-disc list-inside text-sm text-slate-300 space-y-1">
            {STYLING_CONFIG.usage_notes.map((note, index) => (
              <li key={index}>{note}</li>
            ))}
          </ul>
        </div>
      )}

      <div className="rounded-lg border border-white/10 bg-slate-900/60 p-4 space-y-2">
        <div className="text-xs uppercase text-slate-500">Variant CTA Guidance</div>
        {variantGuidance.promise && (
          <p className="text-base text-white font-semibold">{variantGuidance.promise}</p>
        )}
        <div className="text-sm text-slate-300 space-y-1">
          {variantGuidance.primary_cta && <div>Primary CTA: {variantGuidance.primary_cta}</div>}
          {variantGuidance.secondary_cta && <div>Secondary CTA: {variantGuidance.secondary_cta}</div>}
        </div>
        {variantGuidance.notes && variantGuidance.notes.length > 0 && (
          <ul className="list-disc list-inside text-sm text-slate-400 space-y-1">
            {variantGuidance.notes.map((note, index) => (
              <li key={index}>{note}</li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

/**
 * Preview Panel - Renders a single section preview
 */
type PreviewRenderer = (params: {
  content: Record<string, unknown>;
  sectionType: ContentSection['section_type'];
  config: LandingConfigResponse | null;
}) => JSX.Element | null;

interface PreviewPanelProps {
  title: string;
  variantLabel: string;
  renderer?: PreviewRenderer;
  content: Record<string, unknown>;
  sectionType: ContentSection['section_type'];
  config: LandingConfigResponse | null;
  sectionEnabled: boolean;
  missingSectionMessage: string;
}

export function PreviewPanel({
  title,
  variantLabel,
  renderer,
  content,
  sectionType,
  config,
  sectionEnabled,
  missingSectionMessage,
}: PreviewPanelProps) {
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between text-xs text-slate-500">
        <span className="font-semibold text-white">{title}</span>
        <span className="text-[11px] text-slate-400">{variantLabel}</span>
      </div>
      <div className="rounded-2xl border border-white/10 bg-slate-950/90 p-4">
        {!sectionEnabled && (
          <div className="mb-4 px-3 py-2 bg-amber-500/20 border border-amber-500/30 rounded text-amber-300 text-sm">
            Section is currently disabled
          </div>
        )}
        <MetricsModeProvider mode="preview">
          <div className="relative rounded-[28px] border border-white/10 bg-[#07090F] shadow-[0_10px_50px_rgba(7,9,15,0.8)]">
            <div className="max-h-[720px] overflow-y-auto rounded-[28px]">
              {renderer ? (
                renderer({
                  content,
                  sectionType,
                  config,
                })
              ) : (
                <div className="p-8 text-center text-sm text-slate-400">{missingSectionMessage}</div>
              )}
            </div>
          </div>
        </MetricsModeProvider>
      </div>
    </div>
  );
}
