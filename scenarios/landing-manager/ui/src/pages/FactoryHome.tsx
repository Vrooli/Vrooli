import { useEffect, useMemo, useState } from 'react';
import { CheckCircle, Loader2, RefreshCcw, Rocket, AlertCircle, ExternalLink } from 'lucide-react';
import {
  generateScenario,
  getTemplate,
  listTemplates,
  listGeneratedScenarios,
  customizeScenario,
  type GenerationResult,
  type Template,
  type GeneratedScenario,
  type CustomizeResult,
} from '../lib/api';

function slugify(input: string) {
  return input
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/(^-|-$)/g, '')
    .slice(0, 60);
}

export default function FactoryHome() {
  const [templates, setTemplates] = useState<Template[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [loadingTemplates, setLoadingTemplates] = useState(true);
  const [templatesError, setTemplatesError] = useState<string | null>(null);
  const [generated, setGenerated] = useState<GeneratedScenario[]>([]);
  const [loadingGenerated, setLoadingGenerated] = useState(true);
  const [generatedError, setGeneratedError] = useState<string | null>(null);

  const [name, setName] = useState('demo-landing');
  const [slug, setSlug] = useState('demo-landing');
  const [generating, setGenerating] = useState(false);
  const [lastResult, setLastResult] = useState<{ dryRun: boolean; result: GenerationResult } | null>(null);
  const [generateError, setGenerateError] = useState<string | null>(null);

  const [customizeSlug, setCustomizeSlug] = useState('demo-landing');
  const [customizeBrief, setCustomizeBrief] = useState('');
  const [customizeAssets, setCustomizeAssets] = useState('');
  const [customizing, setCustomizing] = useState(false);
  const [customizeError, setCustomizeError] = useState<string | null>(null);
  const [customizeResult, setCustomizeResult] = useState<CustomizeResult | null>(null);

  const selectedTemplate = useMemo(
    () => templates.find((t) => t.id === selectedId) ?? templates[0],
    [selectedId, templates],
  );

  useEffect(() => {
    async function loadTemplates() {
      try {
        setLoadingTemplates(true);
        const tpl = await listTemplates();
        setTemplates(tpl);
        setSelectedId((id) => id ?? tpl[0]?.id ?? null);
        setTemplatesError(null);

        // Hydrate the selected template with full metadata if needed
        if (tpl[0]?.id) {
          const full = await getTemplate(tpl[0].id);
          setTemplates([full, ...tpl.slice(1)]);
        }
      } catch (err) {
        setTemplatesError(err instanceof Error ? err.message : 'Failed to load templates');
      } finally {
        setLoadingTemplates(false);
      }
    }

    loadTemplates();
  }, []);

  useEffect(() => {
    async function loadGenerated() {
      try {
        setLoadingGenerated(true);
        const scenarios = await listGeneratedScenarios();
        setGenerated(scenarios);
        setGeneratedError(null);
      } catch (err) {
        setGeneratedError(err instanceof Error ? err.message : 'Failed to load generated scenarios');
      } finally {
        setLoadingGenerated(false);
      }
    }

    loadGenerated();
  }, []);

  const handleNameChange = (value: string) => {
    setName(value);
    const nextSlug = slugify(value);
    setSlug((prev) => {
      const prevAuto = slugify(name);
      if (!prev || prev === prevAuto) {
        return nextSlug;
      }
      return prev;
    });
  };

  const handleGenerate = async (dryRun: boolean) => {
    if (!selectedTemplate) return;
    if (!name.trim() || !slug.trim()) {
      setGenerateError('Name and slug are required');
      return;
    }

    try {
      setGenerating(true);
      setGenerateError(null);
      const result = await generateScenario(selectedTemplate.id, name.trim(), slug.trim(), { dry_run: dryRun });
      setLastResult({ dryRun, result });
      // Refresh generated scenarios list if we actually wrote files
      if (!dryRun) {
        const scenarios = await listGeneratedScenarios();
        setGenerated(scenarios);
      }
    } catch (err) {
      setGenerateError(err instanceof Error ? err.message : 'Generation failed');
    } finally {
      setGenerating(false);
    }
  };

  const handleCustomize = async () => {
    if (!customizeSlug.trim() || !customizeBrief.trim()) {
      setCustomizeError('Scenario slug and a brief are required');
      return;
    }

    try {
      setCustomizeError(null);
      setCustomizeResult(null);
      setCustomizing(true);
      const assets = customizeAssets
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);
      const result = await customizeScenario(customizeSlug.trim(), customizeBrief.trim(), assets, true);
      setCustomizeResult(result);
    } catch (err) {
      setCustomizeError(err instanceof Error ? err.message : 'Failed to file customization request');
    } finally {
      setCustomizing(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50">
      <div className="max-w-6xl mx-auto px-6 py-16 space-y-12">
        <header className="space-y-4">
          <p className="inline-flex items-center rounded-full bg-emerald-500/10 px-3 py-1 text-xs font-semibold text-emerald-300 border border-emerald-500/30">
            Landing Manager · Factory
          </p>
          <h1 className="text-4xl font-bold">Generate landing-page scenarios, not a landing page.</h1>
          <p className="text-slate-300 max-w-3xl">
            This scenario is the factory. Use it to inspect templates and spin up new landing scenarios. The public landing and admin
            portal live inside the generated scenario, not here.
          </p>
        </header>

        <div className="grid gap-6 md:grid-cols-3">
          <div className="rounded-2xl border border-white/10 bg-white/5 p-6 space-y-3">
            <div className="text-sm text-slate-400">Templates</div>
            <div className="flex items-center gap-2 text-xl font-semibold">
              {loadingTemplates ? <Loader2 className="h-5 w-5 animate-spin text-emerald-300" /> : <CheckCircle className="h-5 w-5 text-emerald-300" />}
              {templates.length} available
            </div>
            <p className="text-sm text-slate-300">
              Select a template to view metadata, then dry-run or generate a new landing scenario. Previews run from the generated
              scenario at <code className="px-1 py-0.5 rounded bg-slate-900">http://localhost:&lt;UI_PORT&gt;/</code> and{' '}
              <code className="px-1 py-0.5 rounded bg-slate-900">/admin</code>.
            </p>
          </div>

          <div className="rounded-2xl border border-white/10 bg-white/5 p-6 space-y-3">
            <div className="text-sm text-slate-400">Generation</div>
            <div className="text-xl font-semibold">CLI + UI</div>
            <p className="text-sm text-slate-300">
              Use the UI buttons below or the CLI to create scenarios with dry-run planning.
            </p>
            <div className="space-y-1 text-xs text-slate-200">
              <code className="block rounded-lg bg-slate-900 px-3 py-2 border border-white/10">
                landing-manager generate saas-landing-page --name "{name || 'demo'}" --slug "{slug || 'demo'}" --dry-run
              </code>
              <code className="block rounded-lg bg-slate-900 px-3 py-2 border border-white/10">
                landing-manager generate saas-landing-page --name "{name || 'demo'}" --slug "{slug || 'demo'}"
              </code>
            </div>
          </div>

          <div className="rounded-2xl border border-amber-500/30 bg-amber-500/10 p-6 space-y-3">
            <div className="text-sm text-amber-200/80">Important</div>
            <div className="text-xl font-semibold text-amber-100">Admin portal lives in the template</div>
            <p className="text-sm text-amber-100/90">
              Keep this UI focused on template selection and scenario creation. Admin, analytics, metrics, A/B testing, and Stripe
              run only inside generated scenarios.
            </p>
          </div>
        </div>

        <div className="rounded-2xl border border-white/10 bg-white/5 p-6 space-y-6">
          <div className="flex items-center justify-between">
            <h2 className="text-2xl font-semibold">Template Catalog</h2>
            <button
              className="inline-flex items-center gap-2 text-sm text-slate-300 hover:text-white transition-colors"
              onClick={async () => {
                try {
                  setLoadingTemplates(true);
                  setTemplatesError(null);
                  const tpl = await listTemplates();
                  setTemplates(tpl);
                  setSelectedId((id) => id ?? tpl[0]?.id ?? null);
                } catch (err) {
                  setTemplatesError(err instanceof Error ? err.message : 'Failed to refresh templates');
                } finally {
                  setLoadingTemplates(false);
                }
              }}
              disabled={loadingTemplates}
            >
              <RefreshCcw className={`h-4 w-4 ${loadingTemplates ? 'animate-spin' : ''}`} />
              Refresh
            </button>
          </div>

          {templatesError && (
            <div className="rounded-xl border border-red-500/30 bg-red-500/10 p-4 text-red-200 text-sm flex items-start gap-3">
              <AlertCircle className="h-4 w-4 mt-0.5" />
              <div>{templatesError}</div>
            </div>
          )}

          <div className="grid md:grid-cols-2 gap-4">
            {loadingTemplates && (
              <div className="col-span-2 flex items-center justify-center py-8 text-slate-400">
                <Loader2 className="h-5 w-5 animate-spin mr-2" />
                Loading templates...
              </div>
            )}
            {!loadingTemplates &&
              templates.map((tpl) => (
                <button
                  key={tpl.id}
                  onClick={() => setSelectedId(tpl.id)}
                  className={`rounded-xl border p-4 text-left transition-all ${
                    selectedTemplate?.id === tpl.id ? 'border-emerald-400/60 bg-emerald-500/5' : 'border-white/10 bg-white/5 hover:border-white/20'
                  }`}
                >
                  <div className="flex items-center justify-between gap-2">
                    <div>
                      <div className="text-sm text-slate-400">Template</div>
                      <div className="text-lg font-semibold">{tpl.name}</div>
                    </div>
                    <span className="text-xs text-slate-400 border border-white/10 rounded-full px-2 py-1">v{tpl.version}</span>
                  </div>
                  <p className="text-sm text-slate-300 mt-2 line-clamp-3">{tpl.description}</p>
                </button>
              ))}
          </div>

          {selectedTemplate && (
            <div className="rounded-xl border border-white/10 bg-slate-900/40 p-5 space-y-3">
              <div className="flex items-center justify-between">
                <div>
                  <div className="text-sm text-slate-400">Selected Template</div>
                  <div className="text-xl font-semibold">{selectedTemplate.name}</div>
                </div>
                <div className="text-xs text-slate-400">ID: {selectedTemplate.id}</div>
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
                      ? selectedTemplate.metrics_hooks.map((hook, idx) => <div key={idx}>• {hook.name || hook.id}</div>)
                      : 'page_view, scroll_depth, click, form_submit, conversion'}
                  </div>
                </div>
                <div className="rounded-lg border border-white/10 bg-white/5 p-3">
                  <div className="text-xs text-slate-400 uppercase tracking-wide mb-1">Customization Schema</div>
                  <div className="text-sm text-slate-200 space-y-1">
                    {(selectedTemplate.customization_schema && Object.keys(selectedTemplate.customization_schema).length > 0)
                      ? Object.keys(selectedTemplate.customization_schema).map((key) => <div key={key}>• {key}</div>)
                      : 'Branding, SEO, Stripe, sections'}
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>

        <div className="rounded-2xl border border-white/10 bg-white/5 p-6 space-y-6">
          <div className="flex items-center justify-between">
            <h2 className="text-2xl font-semibold">Generate a landing scenario</h2>
            {generating && <Loader2 className="h-5 w-5 animate-spin text-emerald-300" />}
          </div>

          <div className="grid md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-slate-300 mb-1">Name</label>
              <input
                value={name}
                onChange={(e) => handleNameChange(e.target.value)}
                className="w-full rounded-lg border border-white/10 bg-slate-900 px-3 py-2 text-sm focus:border-emerald-400 outline-none"
                placeholder="Vrooli Pro Landing"
              />
            </div>
            <div>
              <label className="block text-sm text-slate-300 mb-1">Slug</label>
              <input
                value={slug}
                onChange={(e) => setSlug(slugify(e.target.value))}
                className="w-full rounded-lg border border-white/10 bg-slate-900 px-3 py-2 text-sm focus:border-emerald-400 outline-none"
                placeholder="vrooli-pro"
              />
            </div>
          </div>

          <div className="flex flex-wrap gap-3">
            <button
              onClick={() => handleGenerate(true)}
              className="inline-flex items-center gap-2 rounded-lg border border-white/15 bg-white/5 px-4 py-2 text-sm hover:border-emerald-300 transition-colors disabled:opacity-60"
              disabled={generating || !selectedTemplate}
            >
              {generating ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCcw className="h-4 w-4" />}
              Dry-run (plan only)
            </button>
            <button
              onClick={() => handleGenerate(false)}
              className="inline-flex items-center gap-2 rounded-lg border border-emerald-500/40 bg-emerald-500/10 px-4 py-2 text-sm font-semibold hover:border-emerald-300 hover:bg-emerald-500/20 transition-colors disabled:opacity-60"
              disabled={generating || !selectedTemplate}
            >
              {generating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Rocket className="h-4 w-4" />}
              Generate now
            </button>
          </div>

          {generateError && (
            <div className="rounded-xl border border-red-500/30 bg-red-500/10 p-4 text-red-200 text-sm flex items-start gap-3">
              <AlertCircle className="h-4 w-4 mt-0.5" />
              <div>{generateError}</div>
            </div>
          )}

          {lastResult && (
            <div className="rounded-xl border border-white/10 bg-slate-900/50 p-5 space-y-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <CheckCircle className="h-5 w-5 text-emerald-300" />
                  <div className="text-sm text-slate-300">
                    {lastResult.dryRun ? 'Dry-run plan generated' : 'Scenario generated'}
                  </div>
                </div>
                <span className="text-xs text-slate-400">Status: {lastResult.result.status}</span>
              </div>

              <div className="grid md:grid-cols-2 gap-4 text-sm text-slate-200">
                <div className="space-y-1">
                  <div className="text-slate-400 text-xs uppercase">Scenario</div>
                  <div className="font-semibold">{lastResult.result.name}</div>
                  <div className="text-slate-400 text-xs">Slug: {lastResult.result.scenario_id}</div>
                  <div className="text-slate-400 text-xs">Template: {lastResult.result.template}</div>
                </div>
                <div className="space-y-1">
                  <div className="text-slate-400 text-xs uppercase">Output path</div>
                  <div className="font-mono text-emerald-200 break-all">{lastResult.result.path}</div>
                </div>
              </div>

              {lastResult.result.plan?.paths && (
                <div className="space-y-2">
                  <div className="text-xs text-slate-400 uppercase">Planned paths</div>
                  <div className="text-xs text-slate-200 bg-slate-900 rounded-lg border border-white/10 p-3 grid gap-1">
                    {lastResult.result.plan.paths.map((p) => (
                      <div key={p} className="font-mono">{p}</div>
                    ))}
                  </div>
                </div>
              )}

              {lastResult.result.next_steps && (
                <div className="space-y-1">
                  <div className="text-xs text-slate-400 uppercase">Next steps</div>
                  <ul className="list-disc list-inside text-sm text-slate-200 space-y-1">
                    {lastResult.result.next_steps.map((step, idx) => (
                      <li key={idx}>{step}</li>
                    ))}
                  </ul>
                </div>
              )}

              {!lastResult.dryRun && (
                <div className="space-y-2 text-xs text-slate-300">
                  <div className="flex items-center gap-2">
                    <ExternalLink className="h-4 w-4 text-emerald-300" />
                    Move the folder to <code className="px-1 py-0.5 rounded bg-slate-900">/scenarios/{lastResult.result.scenario_id}</code> then run <code className="px-1 py-0.5 rounded bg-slate-900">make start</code>.
                  </div>
                  <div className="space-y-1">
                    <div>After start:</div>
                    <div className="grid md:grid-cols-2 gap-1 font-mono">
                      <span>Public: http://localhost:${'{'}UI_PORT{'}'}/</span>
                      <span>Admin: http://localhost:${'{'}UI_PORT{'}'}/admin</span>
                    </div>
                    <div>Resolve UI_PORT via <code className="px-1 py-0.5 rounded bg-slate-900">vrooli scenario port {lastResult.result.scenario_id} UI_PORT</code>.</div>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        <div className="rounded-2xl border border-white/10 bg-white/5 p-6 space-y-6">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-2xl font-semibold">Agent customization (app-issue-tracker)</h2>
              <p className="text-sm text-slate-300 max-w-2xl">
                Files a rich issue in app-issue-tracker and auto-triggers an investigation agent with your brief and assets. Use a generated scenario slug.
              </p>
            </div>
            {customizing && <Loader2 className="h-5 w-5 animate-spin text-emerald-300" />}
          </div>

          <div className="grid md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-slate-300 mb-1">Scenario slug</label>
              <input
                value={customizeSlug}
                onChange={(e) => setCustomizeSlug(slugify(e.target.value))}
                className="w-full rounded-lg border border-white/10 bg-slate-900 px-3 py-2 text-sm focus:border-emerald-400 outline-none"
                placeholder="vrooli-pro"
              />
            </div>
            <div>
              <label className="block text-sm text-slate-300 mb-1">Assets (comma-separated paths/URLs)</label>
              <input
                value={customizeAssets}
                onChange={(e) => setCustomizeAssets(e.target.value)}
                className="w-full rounded-lg border border-white/10 bg-slate-900 px-3 py-2 text-sm focus:border-emerald-400 outline-none"
                placeholder="assets/logo.svg, screenshots/hero.png"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm text-slate-300 mb-1">Brief</label>
            <textarea
              value={customizeBrief}
              onChange={(e) => setCustomizeBrief(e.target.value)}
              className="w-full rounded-lg border border-white/10 bg-slate-900 px-3 py-2 text-sm focus:border-emerald-400 outline-none min-h-[120px]"
              placeholder="Goals, audience, value prop, tone, CTA, success metrics..."
            />
          </div>

          <div className="flex flex-wrap gap-3">
            <button
              onClick={handleCustomize}
              className="inline-flex items-center gap-2 rounded-lg border border-emerald-500/40 bg-emerald-500/10 px-4 py-2 text-sm font-semibold hover:border-emerald-300 hover:bg-emerald-500/20 transition-colors disabled:opacity-60"
              disabled={customizing}
            >
              {customizing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Rocket className="h-4 w-4" />}
              File issue & trigger agent
            </button>
            <button
              onClick={() => {
                setCustomizeSlug('');
                setCustomizeBrief('');
                setCustomizeAssets('');
                setCustomizeError(null);
                setCustomizeResult(null);
              }}
              className="inline-flex items-center gap-2 rounded-lg border border-white/15 bg-white/5 px-4 py-2 text-sm hover:border-white/25 transition-colors"
            >
              <RefreshCcw className="h-4 w-4" />
              Reset form
            </button>
          </div>

          {customizeError && (
            <div className="rounded-xl border border-red-500/30 bg-red-500/10 p-4 text-red-200 text-sm flex items-start gap-3">
              <AlertCircle className="h-4 w-4 mt-0.5" />
              <div>{customizeError}</div>
            </div>
          )}

          {customizeResult && (
            <div className="rounded-xl border border-white/10 bg-slate-900/50 p-4 space-y-2 text-sm text-slate-200">
              <div className="flex items-center gap-2">
                <CheckCircle className="h-4 w-4 text-emerald-300" />
                <div>Issue filed and agent queued</div>
              </div>
              <div className="grid md:grid-cols-2 gap-2 text-xs text-slate-300">
                <div>Issue ID: {customizeResult.issue_id || 'unknown'}</div>
                <div>Agent: {customizeResult.agent || 'auto'}</div>
                <div>Run ID: {customizeResult.run_id || 'pending'}</div>
                <div>Status: {customizeResult.status}</div>
              </div>
              {customizeResult.tracker_url && (
                <div className="text-xs text-slate-300">
                  Tracker API: <code className="px-1 py-0.5 rounded bg-slate-900">{customizeResult.tracker_url}</code>
                </div>
              )}
              {customizeResult.message && <div className="text-xs text-slate-400">{customizeResult.message}</div>}
            </div>
          )}
        </div>

        <div className="rounded-2xl border border-white/10 bg-white/5 p-6 space-y-6">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-2xl font-semibold">Generated scenarios</h2>
              <p className="text-sm text-slate-300">Folders under <code className="px-1 py-0.5 rounded bg-slate-900">generated/</code> ready to move into <code className="px-1 py-0.5 rounded bg-slate-900">/scenarios/&lt;slug&gt;</code>.</p>
            </div>
            <button
              className="inline-flex items-center gap-2 text-sm text-slate-300 hover:text-white transition-colors"
              onClick={async () => {
                try {
                  setLoadingGenerated(true);
                  setGeneratedError(null);
                  const scenarios = await listGeneratedScenarios();
                  setGenerated(scenarios);
                } catch (err) {
                  setGeneratedError(err instanceof Error ? err.message : 'Failed to refresh generated scenarios');
                } finally {
                  setLoadingGenerated(false);
                }
              }}
              disabled={loadingGenerated}
            >
              <RefreshCcw className={`h-4 w-4 ${loadingGenerated ? 'animate-spin' : ''}`} />
              Refresh
            </button>
          </div>

          {generatedError && (
            <div className="rounded-xl border border-red-500/30 bg-red-500/10 p-4 text-red-200 text-sm flex items-start gap-3">
              <AlertCircle className="h-4 w-4 mt-0.5" />
              <div>{generatedError}</div>
            </div>
          )}

          {loadingGenerated && (
            <div className="flex items-center gap-2 text-slate-300 text-sm">
              <Loader2 className="h-4 w-4 animate-spin" />
              Loading generated scenarios...
            </div>
          )}

          {!loadingGenerated && generated.length === 0 && !generatedError && (
            <div className="rounded-xl border border-white/10 bg-slate-900/40 p-4 text-sm text-slate-200">
              No generated scenarios yet. Run a generation without <code className="px-1 py-0.5 rounded bg-slate-900">--dry-run</code> to populate this list.
            </div>
          )}

          {!loadingGenerated && generated.length > 0 && (
            <div className="grid gap-3">
              {generated.map((scenario) => (
                <div key={scenario.scenario_id} className="rounded-xl border border-white/10 bg-slate-900/40 p-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="text-xs text-slate-400 uppercase tracking-wide">Scenario</div>
                      <div className="text-lg font-semibold">{scenario.name}</div>
                      <div className="text-xs text-slate-400">Slug: {scenario.scenario_id}</div>
                    </div>
                    <span className="text-xs text-slate-400 border border-white/10 rounded-full px-2 py-1">{scenario.status}</span>
                  </div>
                  <div className="text-xs text-slate-400 mt-2">
                    Path:{' '}
                    <code className="px-1 py-0.5 rounded bg-slate-900 break-all border border-white/10">
                      {scenario.path}
                    </code>
                  </div>
                  <div className="flex flex-wrap gap-3 text-xs text-slate-300 mt-2">
                    {scenario.template_id && <span>Template: {scenario.template_id} {scenario.template_version ? `(v${scenario.template_version})` : ''}</span>}
                    {scenario.generated_at && <span>Generated: {scenario.generated_at}</span>}
                  </div>
                  <div className="text-xs text-slate-300 mt-3 space-y-1">
                    <div>Next: move to <code className="px-1 py-0.5 rounded bg-slate-900">/scenarios/{scenario.scenario_id}</code>, then run <code className="px-1 py-0.5 rounded bg-slate-900">make start</code>.</div>
                    <div className="font-mono text-slate-200 bg-slate-900/60 border border-white/10 rounded p-2">
                      mv {scenario.path} /home/matthalloran8/Vrooli/scenarios/{scenario.scenario_id} && cd /home/matthalloran8/Vrooli/scenarios/{scenario.scenario_id} && make start
                    </div>
                    <div className="grid md:grid-cols-2 gap-1">
                      <span>Public: http://localhost:${'{'}UI_PORT{'}'}/</span>
                      <span>Admin: http://localhost:${'{'}UI_PORT{'}'}/admin</span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
