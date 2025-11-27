import { AlertCircle, CheckCircle, FileText, HelpCircle, RefreshCcw } from 'lucide-react';
import { Tooltip } from './Tooltip';
import { TemplateCardSkeleton } from './TemplateCardSkeleton';
import { listTemplates, type Template } from '../lib/api';

interface TemplateCatalogProps {
  templates: Template[];
  selectedTemplate: Template | null;
  loadingTemplates: boolean;
  templatesError: string | null;
  selectedId: string | null;
  onSelectTemplate: (id: string) => void;
  onRefreshTemplates: (templates: Template[], error: string | null, loading: boolean) => void;
}

export function TemplateCatalog({
  templates,
  selectedTemplate,
  loadingTemplates,
  templatesError,
  selectedId,
  onSelectTemplate,
  onRefreshTemplates,
}: TemplateCatalogProps) {
  const handleRefresh = async () => {
    try {
      onRefreshTemplates(templates, null, true);
      const tpl = await listTemplates();
      onRefreshTemplates(tpl, null, false);
      if (!selectedId && tpl[0]?.id) {
        onSelectTemplate(tpl[0].id);
      }
    } catch (err) {
      onRefreshTemplates(templates, err instanceof Error ? err.message : 'Failed to refresh templates', false);
    }
  };

  const handleTryAgain = async () => {
    try {
      onRefreshTemplates(templates, null, true);
      const tpl = await listTemplates();
      onRefreshTemplates(tpl, null, false);
      if (!selectedId && tpl[0]?.id) {
        onSelectTemplate(tpl[0].id);
      }
    } catch (err) {
      onRefreshTemplates(templates, err instanceof Error ? err.message : 'Failed to load templates', false);
    }
  };

  return (
    <section
      className="rounded-2xl border border-white/10 bg-white/5 p-4 sm:p-6 space-y-4 sm:space-y-6 relative overflow-hidden"
      data-testid="template-catalog"
      aria-labelledby="template-catalog-heading"
    >
      <div className="absolute top-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-emerald-500/20 to-transparent"></div>
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
        <div className="flex items-center gap-2">
          <h2 id="template-catalog-heading" className="text-xl sm:text-2xl font-semibold">
            Template Catalog
          </h2>
          <Tooltip content="Choose a template as the starting point for your landing page. Each template includes pre-configured sections, metrics tracking, and Stripe integration.">
            <HelpCircle className="h-4 w-4 text-slate-500 hover:text-slate-400 transition-colors" />
          </Tooltip>
        </div>
        <button
          data-testid="refresh-templates-button"
          className="group touch-target inline-flex items-center gap-2 px-3 py-2 text-sm text-slate-300 hover:text-white hover:bg-white/5 rounded-lg transition-all duration-250 hover:scale-105 active:scale-95 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:ring-offset-2 focus:ring-offset-slate-950 disabled:opacity-50 disabled:cursor-not-allowed"
          onClick={handleRefresh}
          disabled={loadingTemplates}
          aria-label="Refresh template list"
        >
          <RefreshCcw
            className={`h-4 w-4 transition-transform duration-300 ${loadingTemplates ? 'animate-spin' : 'group-hover:rotate-180'}`}
            aria-hidden="true"
          />
          <span className="hidden sm:inline">Refresh</span>
        </button>
      </div>

      {templatesError && (
        <div className="rounded-xl border border-red-500/30 bg-red-500/10 p-4 text-red-200 text-sm flex items-start gap-3">
          <AlertCircle className="h-4 w-4 mt-0.5" />
          <div>{templatesError}</div>
        </div>
      )}

      <div
        className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3 sm:gap-4"
        data-testid="template-grid"
        role="list"
        aria-label="Available templates"
      >
        {loadingTemplates && (
          <>
            <TemplateCardSkeleton />
            <TemplateCardSkeleton />
          </>
        )}
        {!loadingTemplates && templates.length === 0 && !templatesError && (
          <div
            className="col-span-full rounded-xl border border-white/10 bg-slate-900/40 p-8 sm:p-12 text-center space-y-4"
            role="status"
          >
            <div className="flex justify-center">
              <FileText className="h-12 w-12 text-slate-600" aria-hidden="true" />
            </div>
            <div className="space-y-2">
              <h3 className="text-lg font-semibold text-slate-300">No templates available</h3>
              <p className="text-sm text-slate-400 max-w-md mx-auto">
                Templates will appear here once they're loaded. Try refreshing or check your API connection.
              </p>
            </div>
            <button
              onClick={handleTryAgain}
              className="inline-flex items-center gap-2 px-4 py-2 text-sm bg-emerald-500/10 border border-emerald-500/30 text-emerald-300 rounded-lg hover:bg-emerald-500/20 transition-colors focus:outline-none focus:ring-2 focus:ring-emerald-500"
            >
              <RefreshCcw className="h-4 w-4" aria-hidden="true" />
              Try again
            </button>
          </div>
        )}
        {!loadingTemplates &&
          templates?.filter(Boolean).map((tpl) => (
            <button
              key={tpl.id}
              role="listitem"
              data-testid={`template-card-${tpl.id}`}
              onClick={() => onSelectTemplate(tpl.id)}
              className={`rounded-xl border p-4 sm:p-5 text-left transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:ring-offset-2 focus:ring-offset-slate-950 ${
                selectedTemplate?.id === tpl.id
                  ? 'border-emerald-400/60 bg-emerald-500/5 shadow-lg shadow-emerald-500/10 scale-[1.02]'
                  : 'border-white/10 bg-white/5 hover:border-white/20 hover:bg-white/10 hover:scale-[1.01]'
              }`}
              aria-current={selectedTemplate?.id === tpl.id ? 'true' : 'false'}
              aria-label={`Select ${tpl.name} template, version ${tpl.version}${selectedTemplate?.id === tpl.id ? ' (currently selected)' : ''}`}
            >
              <div className="flex items-start justify-between gap-3">
                <div className="flex-1 min-w-0">
                  <div className="text-xs font-medium text-slate-400 uppercase tracking-wide mb-1">Template</div>
                  <div className="text-base sm:text-lg font-semibold truncate" data-testid={`template-name-${tpl.id}`}>
                    {tpl.name}
                  </div>
                </div>
                <span
                  className="flex-shrink-0 text-xs text-slate-400 border border-white/10 rounded-full px-2 py-1"
                  data-testid={`template-version-${tpl.id}`}
                >
                  v{tpl.version}
                </span>
              </div>
              <p className="text-xs sm:text-sm text-slate-300 mt-2 line-clamp-2 leading-relaxed">{tpl.description}</p>
              {selectedTemplate?.id === tpl.id && (
                <div className="mt-3 pt-3 border-t border-emerald-500/20">
                  <span className="inline-flex items-center gap-1.5 text-xs font-medium text-emerald-300">
                    <CheckCircle className="h-3.5 w-3.5" aria-hidden="true" />
                    Selected
                  </span>
                </div>
              )}
            </button>
          ))}
      </div>

      {selectedTemplate && (
        <div className="rounded-xl border border-white/10 bg-slate-900/40 p-5 space-y-3" data-testid="selected-template-details">
          <div className="flex items-center justify-between">
            <div>
              <div className="text-sm text-slate-400">Selected Template</div>
              <div className="text-xl font-semibold" data-testid="selected-template-name">
                {selectedTemplate.name}
              </div>
            </div>
            <div className="text-xs text-slate-400" data-testid="selected-template-id">
              ID: {selectedTemplate.id}
            </div>
          </div>
          <p className="text-sm text-slate-300">{selectedTemplate.description}</p>

          <div className="grid md:grid-cols-3 gap-4">
            <div className="rounded-lg border border-white/10 bg-white/5 p-3">
              <div className="text-xs text-slate-400 uppercase tracking-wide mb-1">Sections</div>
              <div className="text-sm text-slate-200 space-y-1">
                {selectedTemplate.sections
                  ? Object.keys(selectedTemplate.sections).map((key) => <div key={key}>• {key}</div>)
                  : 'Defined in template payload'}
              </div>
            </div>
            <div className="rounded-lg border border-white/10 bg-white/5 p-3">
              <div className="text-xs text-slate-400 uppercase tracking-wide mb-1">Metrics Hooks</div>
              <div className="text-sm text-slate-200 space-y-1">
                {selectedTemplate.metrics_hooks?.length
                  ? selectedTemplate.metrics_hooks.map((hook, idx) => (
                      <div key={idx}>
                        • {(hook as { name?: string; id?: string }).name || (hook as { name?: string; id?: string }).id || 'hook'}
                      </div>
                    ))
                  : 'page_view, scroll_depth, click, form_submit, conversion'}
              </div>
            </div>
            <div className="rounded-lg border border-white/10 bg-white/5 p-3">
              <div className="text-xs text-slate-400 uppercase tracking-wide mb-1">Customization Schema</div>
              <div className="text-sm text-slate-200 space-y-1">
                {selectedTemplate.customization_schema && Object.keys(selectedTemplate.customization_schema).length > 0
                  ? Object.keys(selectedTemplate.customization_schema).map((key) => <div key={key}>• {key}</div>)
                  : 'Branding, SEO, Stripe, sections'}
              </div>
            </div>
          </div>
        </div>
      )}
    </section>
  );
}
