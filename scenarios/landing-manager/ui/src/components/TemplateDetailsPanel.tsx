import { type Template } from '../lib/api';

interface TemplateDetailsPanelProps {
  template: Template;
}

interface MetricHook {
  name?: string;
  id?: string;
}

function renderSections(sections: Record<string, unknown> | undefined) {
  if (!sections) return 'Defined in template payload';
  return Object.keys(sections).map((key) => <div key={key}>• {key}</div>);
}

function renderMetricsHooks(hooks: MetricHook[] | undefined) {
  if (!hooks?.length) {
    return 'page_view, scroll_depth, click, form_submit, conversion';
  }
  return hooks.map((hook, idx) => (
    <div key={idx}>• {hook.name || hook.id || 'hook'}</div>
  ));
}

function renderCustomizationSchema(schema: Record<string, unknown> | undefined) {
  if (!schema || Object.keys(schema).length === 0) {
    return 'Branding, SEO, Stripe, sections';
  }
  return Object.keys(schema).map((key) => <div key={key}>• {key}</div>);
}

export function TemplateDetailsPanel({ template }: TemplateDetailsPanelProps) {
  return (
    <div className="rounded-xl border border-white/10 bg-slate-900/40 p-5 space-y-3" data-testid="selected-template-details">
      <div className="flex items-center justify-between">
        <div>
          <div className="text-sm text-slate-400">Selected Template</div>
          <div className="text-xl font-semibold" data-testid="selected-template-name">
            {template.name}
          </div>
        </div>
        <div className="text-xs text-slate-400" data-testid="selected-template-id">
          ID: {template.id}
        </div>
      </div>
      <p className="text-sm text-slate-300">{template.description}</p>

      <div className="grid md:grid-cols-3 gap-4">
        <div className="rounded-lg border border-white/10 bg-white/5 p-3">
          <div className="text-xs text-slate-400 uppercase tracking-wide mb-1">Sections</div>
          <div className="text-sm text-slate-200 space-y-1">
            {renderSections(template.sections as Record<string, unknown>)}
          </div>
        </div>
        <div className="rounded-lg border border-white/10 bg-white/5 p-3">
          <div className="text-xs text-slate-400 uppercase tracking-wide mb-1">Metrics Hooks</div>
          <div className="text-sm text-slate-200 space-y-1">
            {renderMetricsHooks(template.metrics_hooks as MetricHook[])}
          </div>
        </div>
        <div className="rounded-lg border border-white/10 bg-white/5 p-3">
          <div className="text-xs text-slate-400 uppercase tracking-wide mb-1">Customization Schema</div>
          <div className="text-sm text-slate-200 space-y-1">
            {renderCustomizationSchema(template.customization_schema as Record<string, unknown>)}
          </div>
        </div>
      </div>
    </div>
  );
}
