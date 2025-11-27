import { Loader2, CheckCircle, FileText, AlertCircle, HelpCircle } from 'lucide-react';
import { Tooltip } from './Tooltip';
import type { Template } from '../lib/api';

interface StatsOverviewProps {
  templates: Template[];
  loadingTemplates: boolean;
  selectedTemplate: Template | null;
}

export function StatsOverview({ templates, loadingTemplates, selectedTemplate }: StatsOverviewProps) {
  return (
    <div className="grid gap-4 sm:gap-6 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-3" role="region" aria-label="Overview statistics">
      <div className="rounded-2xl border border-white/10 bg-white/5 p-4 sm:p-6 xl:p-7 space-y-3" role="article">
        <div className="flex items-center justify-between">
          <div className="text-sm font-medium text-slate-400">Templates Available</div>
          <Tooltip content="Pre-built templates with different landing page configurations. Each includes a full stack: React UI, Go API, PostgreSQL schema, and Stripe integration.">
            <HelpCircle className="h-4 w-4 text-slate-500 hover:text-slate-400 transition-colors" />
          </Tooltip>
        </div>
        <div className="flex items-center gap-2 text-2xl xl:text-3xl font-bold" aria-live="polite">
          {loadingTemplates ? (
            <Loader2 className="h-5 w-5 xl:h-6 xl:w-6 animate-spin text-emerald-300" aria-label="Loading templates" />
          ) : (
            <CheckCircle className="h-5 w-5 xl:h-6 xl:w-6 text-emerald-300" aria-label="Templates loaded successfully" />
          )}
          <span>{templates.length}</span>
        </div>
        <p className="text-xs sm:text-sm xl:text-base text-slate-400 leading-relaxed">
          Browse templates below to see features, sections, and customization options
        </p>
      </div>

      <div className="rounded-2xl border border-white/10 bg-white/5 p-4 sm:p-6 xl:p-7 space-y-3" role="article">
        <div className="flex items-center justify-between">
          <div className="text-sm font-medium text-slate-400">Generation Methods</div>
          <Tooltip content="Create scenarios via the UI form below or using CLI commands. Both support dry-run mode for planning before generation.">
            <HelpCircle className="h-4 w-4 text-slate-500 hover:text-slate-400 transition-colors" />
          </Tooltip>
        </div>
        <div className="flex items-center gap-2 text-2xl xl:text-3xl font-bold">
          <FileText className="h-5 w-5 xl:h-6 xl:w-6 text-blue-400" aria-hidden="true" />
          <span>CLI + UI</span>
        </div>
        <div className="space-y-2">
          <p className="text-xs sm:text-sm xl:text-base text-slate-400">
            Use buttons below or CLI:
          </p>
          <code className="hidden sm:block text-[10px] lg:text-xs xl:text-sm rounded-lg bg-slate-900 px-2 py-1.5 xl:px-3 xl:py-2 border border-white/10 overflow-x-auto whitespace-nowrap">
            landing-manager generate {selectedTemplate?.id || 'template-id'} --name "..." --slug "..."
          </code>
        </div>
      </div>

      <div className="rounded-2xl border border-amber-500/30 bg-amber-500/10 p-4 sm:p-6 xl:p-7 space-y-3 sm:col-span-2 lg:col-span-1" role="alert" aria-live="polite">
        <div className="flex items-center gap-2">
          <AlertCircle className="h-4 w-4 xl:h-5 xl:w-5 text-amber-400 flex-shrink-0" aria-hidden="true" />
          <div className="text-sm xl:text-base font-medium text-amber-200">Important Note</div>
        </div>
        <p className="text-sm sm:text-base xl:text-lg font-semibold text-amber-100">Runtime lives in generated scenarios</p>
        <p className="text-xs sm:text-sm xl:text-base text-amber-100/90 leading-relaxed">
          This factory creates scenarios. The admin portal, analytics, A/B testing, and Stripe integration run inside each generated scenario, not here.
        </p>
      </div>
    </div>
  );
}
