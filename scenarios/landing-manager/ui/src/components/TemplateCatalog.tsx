import { memo } from 'react';
import { AlertCircle, HelpCircle, RefreshCcw } from 'lucide-react';
import { Tooltip } from './Tooltip';
import { TemplateCardSkeleton } from './TemplateCardSkeleton';
import { TemplateCard } from './TemplateCard';
import { TemplateEmptyState } from './TemplateEmptyState';
import { TemplateDetailsPanel } from './TemplateDetailsPanel';
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

export const TemplateCatalog = memo(function TemplateCatalog({
  templates,
  selectedTemplate,
  loadingTemplates,
  templatesError,
  selectedId,
  onSelectTemplate,
  onRefreshTemplates,
}: TemplateCatalogProps) {
  const refreshTemplates = async (errorMessage = 'Failed to load templates') => {
    try {
      onRefreshTemplates(templates, null, true);
      const tpl = await listTemplates();
      onRefreshTemplates(tpl, null, false);
      if (!selectedId && tpl[0]?.id) {
        onSelectTemplate(tpl[0].id);
      }
    } catch (err) {
      onRefreshTemplates(templates, err instanceof Error ? err.message : errorMessage, false);
    }
  };

  const handleRefresh = () => refreshTemplates('Failed to refresh templates');
  const handleTryAgain = () => refreshTemplates();

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
          <TemplateEmptyState onRetry={handleTryAgain} />
        )}
        {!loadingTemplates &&
          templates?.filter(Boolean).map((tpl) => (
            <TemplateCard
              key={tpl.id}
              template={tpl}
              isSelected={selectedTemplate?.id === tpl.id}
              onSelect={onSelectTemplate}
            />
          ))}
      </div>

      {selectedTemplate && <TemplateDetailsPanel template={selectedTemplate} />}
    </section>
  );
});
