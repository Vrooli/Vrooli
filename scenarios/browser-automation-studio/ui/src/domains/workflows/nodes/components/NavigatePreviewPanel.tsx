import { useCallback, useEffect, useMemo, useState } from 'react';
import { RefreshCw } from 'lucide-react';
import { getConfig } from '@/config';
import { useScenarioStore } from '@stores/scenarioStore';
import { scenariosClient } from '@/api/scenarios';

interface Props {
  destinationType: string;
  url: string;
  scenario: string;
  scenarioPath: string;
  viewport?: { width: number; height: number };
  onChange: (values: Record<string, unknown>) => void;
  onScreenshot: (values: Record<string, unknown>) => void;
}

/** Scenario chooser and live destination preview used by the shared Navigate ActionNode. */
export default function NavigatePreviewPanel({ destinationType, url, scenario, scenarioPath, viewport, onChange, onScreenshot }: Props) {
  const [scenarios, fetchScenarios] = useScenarioStore((state) => [state.scenarios, state.fetchScenarios]);
  const [open, setOpen] = useState(true);
  const [loading, setLoading] = useState(false);
  const [image, setImage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [target, setTarget] = useState<string | null>(null);
  const [consoleLogs, setConsoleLogs] = useState<Array<{ level?: string; message?: string }>>([]);
  const [showConsole, setShowConsole] = useState(false);
  const [updatedAt, setUpdatedAt] = useState<string | null>(null);
  const [scenarioSearch, setScenarioSearch] = useState('');
  const [scenariosLoading, setScenariosLoading] = useState(false);
  const matchingScenarios = scenarios.filter((item) => `${item.name} ${item.description}`.toLowerCase().includes(scenarioSearch.trim().toLowerCase()));

  const refreshScenarios = useCallback(() => {
    setScenariosLoading(true);
    void Promise.resolve(fetchScenarios()).finally(() => setScenariosLoading(false));
  }, [fetchScenarios]);
  useEffect(() => { if (destinationType.toUpperCase().endsWith('SCENARIO')) refreshScenarios(); }, [destinationType, refreshScenarios]);

  const resolveTarget = useCallback(async () => {
    if (!destinationType.toUpperCase().endsWith('SCENARIO')) return url.trim();
    if (!scenario.trim()) return '';
    const info = await scenariosClient.getPort({ name: scenario.trim() });
    const base = info.url || (info.port > 0 ? `http://localhost:${info.port}` : '');
    if (!base) return '';
    return scenarioPath.trim() ? new URL(scenarioPath.trim(), base.endsWith('/') ? base : `${base}/`).toString() : base;
  }, [destinationType, scenario, scenarioPath, url]);

  const fetchPreview = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const targetUrl = await resolveTarget();
      if (!targetUrl) throw new Error(destinationType.toUpperCase().endsWith('SCENARIO') ? 'Select an app to preview' : 'Enter a URL to preview');
      setTarget(targetUrl);
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/preview-screenshot`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ url: targetUrl, viewport }) });
      if (!response.ok) throw new Error((await response.text()) || 'Failed to take preview screenshot');
      const payload = await response.json();
      if (!payload?.screenshot) throw new Error('No screenshot data received');
      const logs = Array.isArray(payload.consoleLogs) ? payload.consoleLogs : [];
      setImage(payload.screenshot); setConsoleLogs(logs); setUpdatedAt(new Date().toISOString());
      onScreenshot({ previewScreenshot: payload.screenshot, previewScreenshotCapturedAt: new Date().toISOString(), previewScreenshotSourceUrl: targetUrl });
    } catch (cause) {
      const message = cause instanceof Error ? cause.message : 'Failed to take preview screenshot';
      setError(message); setImage(null); setConsoleLogs([]);
    } finally { setLoading(false); }
  }, [destinationType, onScreenshot, resolveTarget, viewport]);

  useEffect(() => { if (destinationType.toUpperCase().endsWith('SCENARIO') ? scenario.trim() : url.trim().length >= 5) void fetchPreview(); }, [destinationType, scenario, scenarioPath, url]);

  const aspectRatio = useMemo(() => viewport && viewport.width > 0 && viewport.height > 0 ? `${viewport.width} / ${viewport.height}` : '16 / 9', [viewport]);
  const isScenario = destinationType.toUpperCase().endsWith('SCENARIO');

  return <section className="mt-3 overflow-hidden rounded-lg border border-gray-800 bg-flow-bg/60" aria-label="Navigation preview">
    <div className="flex items-center gap-2 border-b border-gray-800 px-3 py-2 text-xs text-gray-300">
      <button type="button" onClick={() => setOpen((value) => !value)} aria-expanded={open}>Preview {open ? '▾' : '▸'}</button>
      <span className="min-w-0 flex-1 truncate text-[10px] text-gray-500">{target || (isScenario ? scenario : url)}</span>
      {updatedAt && <span className="text-[10px] text-gray-500">Updated {new Date(updatedAt).toLocaleTimeString()}</span>}
      <button type="button" onClick={() => void fetchPreview()} disabled={loading} aria-label="Refresh navigation preview" className="rounded p-1 hover:bg-gray-700 disabled:opacity-50"><RefreshCw size={13} className={loading ? 'animate-spin' : ''} /></button>
    </div>
    {open && <div className="space-y-2 p-3">
      {isScenario && <>
        <label className="block text-[11px] text-gray-400">Find an app<input aria-label="Search apps" value={scenarioSearch} onChange={(event) => setScenarioSearch(event.target.value)} placeholder="Search available apps" className="mt-1 w-full rounded border border-gray-700 bg-flow-bg px-2 py-1 text-xs" /></label>
        <div className="flex items-center gap-2"><label className="min-w-0 flex-1 text-[11px] text-gray-400">App<select aria-label="Scenario app" value={scenario} onChange={(event) => onChange({ scenario: event.target.value, destinationType: 'NAVIGATE_DESTINATION_TYPE_SCENARIO', url: undefined })} className="mt-1 w-full rounded border border-gray-700 bg-flow-bg px-2 py-1 text-xs"><option value="">Select an app</option>{matchingScenarios.map((item) => <option key={item.name} value={item.name}>{item.name}</option>)}</select></label><button type="button" onClick={refreshScenarios} disabled={scenariosLoading} className="mt-4 rounded border border-gray-700 px-2 py-1 text-xs disabled:opacity-50">{scenariosLoading ? 'Loading…' : 'Refresh apps'}</button></div>
        <label className="block text-[11px] text-gray-400">Path<input aria-label="Scenario path" value={scenarioPath} onChange={(event) => onChange({ scenarioPath: event.target.value, destinationType: 'NAVIGATE_DESTINATION_TYPE_SCENARIO' })} placeholder="/path" className="mt-1 w-full rounded border border-gray-700 bg-flow-bg px-2 py-1 text-xs" /></label>
      </>}
      {image ? <div className="relative overflow-hidden rounded bg-black/40" style={{ aspectRatio, maxHeight: 240 }}><img src={image} alt="Navigation target preview" className="h-full w-full object-contain" onError={() => { setImage(null); setError('Failed to load preview image'); }} /></div> : <div className="flex min-h-24 items-center justify-center rounded bg-black/30 text-xs text-gray-500">{loading ? 'Loading preview…' : error || 'Enter a destination to preview'}</div>}
      {error && image && <p role="alert" className="text-xs text-red-400">{error}</p>}
      {consoleLogs.length > 0 && <><button type="button" onClick={() => setShowConsole((value) => !value)} className="text-[11px] text-gray-400">{showConsole ? 'Hide' : 'Show'} console ({consoleLogs.length})</button>{showConsole && <div className="max-h-28 overflow-auto rounded bg-black/40 p-2 font-mono text-[10px] text-gray-300">{consoleLogs.map((entry, index) => <p key={`${index}-${entry.message}`}>{entry.level ? `[${entry.level}] ` : ''}{entry.message}</p>)}</div>}</>}
    </div>}
  </section>;
}
