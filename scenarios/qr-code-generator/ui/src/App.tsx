import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';
import {
  ArrowDownToLine,
  CheckCircle2,
  Copy,
  Layers,
  Loader2,
  PlayCircle,
  PlusCircle,
  QrCode,
  Sparkles,
  Wand2,
  X
} from 'lucide-react';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { cn } from '@/lib/utils';

const APP_ID = 'qr-code-generator';
const DEFAULT_API_PATH = '/api';

type ErrorCorrection = 'L' | 'M' | 'Q' | 'H';

type StatusTone = 'idle' | 'processing' | 'success' | 'error';

interface StatusState {
  message: string;
  tone: StatusTone;
}

interface QrOptions {
  text: string;
  size: number;
  errorCorrection: ErrorCorrection;
  color: string;
  background: string;
}

interface QrResult {
  base64: string;
  text: string;
}

interface BatchItem {
  id: string;
  text: string;
  label: string;
}

interface BatchSummary {
  success: number;
  total: number;
}

interface GenerateResponse {
  success?: boolean;
  data?: string;
  base64?: string;
  format?: string;
  error?: string;
}

interface ConfigResponse {
  apiBase?: string;
  apiUrl?: string;
  apiPath?: string;
}

const DEFAULT_OPTIONS: QrOptions = {
  text: '',
  size: 256,
  errorCorrection: 'M',
  color: '#5effc4',
  background: '#041d14'
};

const ERROR_LEVELS: Array<{ value: ErrorCorrection; label: string }> = [
  { value: 'L', label: 'Low · 7% redundancy' },
  { value: 'M', label: 'Medium · 15% redundancy' },
  { value: 'Q', label: 'Quartile · 25% redundancy' },
  { value: 'H', label: 'High · 30% redundancy' }
];

const SIZE_OPTIONS = [128, 192, 256, 320, 384, 448, 512];

function App() {
  const [options, setOptions] = useState<QrOptions>(DEFAULT_OPTIONS);
  const [apiBase, setApiBase] = useState<string>(DEFAULT_API_PATH);
  const [qrResult, setQrResult] = useState<QrResult | null>(null);
  const [status, setStatus] = useState<StatusState>({ message: 'System ready', tone: 'idle' });
  const [batchItems, setBatchItems] = useState<BatchItem[]>([]);
  const [batchSummary, setBatchSummary] = useState<BatchSummary | null>(null);
  const [isGenerating, setIsGenerating] = useState(false);
  const [isBatchProcessing, setIsBatchProcessing] = useState(false);

  const statusResetHandle = useRef<ReturnType<typeof setTimeout> | null>(null);
  const bridgeRef = useRef<ReturnType<typeof initIframeBridgeChild> | null>(null);

  useEffect(() => {
    function setupBridge() {
      if (typeof window === 'undefined' || window.parent === window) {
        return;
      }

      if (bridgeRef.current) {
        bridgeRef.current.notify();
        return;
      }

      let parentOrigin: string | undefined;
      try {
        if (document.referrer) {
          parentOrigin = new URL(document.referrer).origin;
        }
      } catch (error) {
        console.warn('[QR UI] Unable to determine parent origin for iframe bridge', error);
      }

      const logLevels = ['log', 'info', 'warn', 'error', 'debug'];

      try {
        bridgeRef.current = initIframeBridgeChild({
          parentOrigin,
          appId: APP_ID,
          captureLogs: {
            enabled: true,
            streaming: true,
            bufferSize: 400,
            levels: logLevels
          },
          captureNetwork: {
            enabled: true,
            streaming: true,
            bufferSize: 200
          }
        });

        window.__qrCodeGeneratorBridgeInitialized = true;
      } catch (error) {
        console.error('[QR UI] Failed to initialize iframe bridge', error);
        bridgeRef.current = null;
      }
    }

    setupBridge();

    const handlePageShow = () => setupBridge();
    const handleVisibility = () => {
      if (!document.hidden) {
        bridgeRef.current?.notify?.();
      }
    };

    window.addEventListener('pageshow', handlePageShow);
    document.addEventListener('visibilitychange', handleVisibility);

    return () => {
      window.removeEventListener('pageshow', handlePageShow);
      document.removeEventListener('visibilitychange', handleVisibility);
      if (statusResetHandle.current) {
        clearTimeout(statusResetHandle.current);
        statusResetHandle.current = null;
      }
    };
  }, []);

  useEffect(() => {
    let isMounted = true;

    async function hydrate() {
      try {
        const config = await fetchConfig();
        if (!isMounted) {
          return;
        }
        const resolved = resolveApiBase(config);
        setApiBase(resolved);
      } catch (error) {
        console.warn('[QR UI] Failed to hydrate API base, falling back to default', error);
        if (isMounted) {
          setApiBase(DEFAULT_API_PATH);
        }
      }
    }

    hydrate();

    return () => {
      isMounted = false;
    };
  }, []);

  const setStatusMessage = useCallback(
    (message: string, tone: StatusTone) => {
      setStatus({ message, tone });

      if (statusResetHandle.current) {
        clearTimeout(statusResetHandle.current);
        statusResetHandle.current = null;
      }

      if (tone === 'processing') {
        return;
      }

      statusResetHandle.current = window.setTimeout(() => {
        setStatus({ message: 'System ready', tone: 'idle' });
        statusResetHandle.current = null;
      }, 4200);
    },
    []
  );

  const handleGenerate = useCallback(async () => {
    const text = options.text.trim();
    if (!text) {
      setStatusMessage('Error: Enter text before generating', 'error');
      return;
    }

    setIsGenerating(true);
    setStatusMessage('Generating QR code…', 'processing');

    const payload = {
      text,
      size: options.size,
      color: options.color,
      background: options.background,
      errorCorrection: options.errorCorrection,
      format: 'png'
    };

    try {
      const response = await fetch(buildApiUrl(apiBase, '/generate'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });

      const data: GenerateResponse = await response.json();

      if (!response.ok || !data?.success) {
        const errorMessage = data?.error || `Request failed with status ${response.status}`;
        throw new Error(errorMessage);
      }

      const base64 = extractBase64Payload(data);
      if (!base64) {
        throw new Error('Invalid QR payload received');
      }

      setQrResult({ base64, text });
      setStatusMessage('QR code generated', 'success');
    } catch (error) {
      console.error('[QR UI] Generate error', error);
      setStatusMessage(`Error: ${(error as Error).message}`, 'error');
    } finally {
      setIsGenerating(false);
    }
  }, [apiBase, options, setStatusMessage]);

  const handleDownload = useCallback(() => {
    if (!qrResult?.base64) {
      setStatusMessage('Generate a code first', 'error');
      return;
    }

    const link = document.createElement('a');
    link.href = `data:image/png;base64,${qrResult.base64}`;
    link.download = buildDownloadFilename(qrResult.text);
    document.body.appendChild(link);
    link.click();
    link.remove();

    setStatusMessage('Download started', 'success');
  }, [qrResult, setStatusMessage]);

  const handleCopyText = useCallback(async () => {
    const text = options.text.trim();
    if (!text) {
      setStatusMessage('Nothing to copy', 'error');
      return;
    }

    try {
      await navigator.clipboard.writeText(text);
      setStatusMessage('Copied to clipboard', 'success');
    } catch (error) {
      console.warn('[QR UI] Clipboard error', error);
      setStatusMessage('Copy failed', 'error');
    }
  }, [options.text, setStatusMessage]);

  const handleAddBatchItem = useCallback(() => {
    setBatchItems((items) => [
      ...items,
      {
        id: crypto.randomUUID?.() ?? `${Date.now()}-${Math.random()}`,
        text: '',
        label: ''
      }
    ]);
  }, []);

  const handleBatchItemChange = useCallback((id: string, key: 'text' | 'label', value: string) => {
    setBatchItems((items) =>
      items.map((item) => (item.id === id ? { ...item, [key]: value } : item))
    );
  }, []);

  const handleRemoveBatchItem = useCallback((id: string) => {
    setBatchItems((items) => items.filter((item) => item.id !== id));
  }, []);

  const handleProcessBatch = useCallback(async () => {
    const entries = batchItems
      .map((item, index) => ({
        text: item.text.trim(),
        label: item.label.trim() || `QR_${index + 1}`
      }))
      .filter((item) => item.text.length > 0);

    if (entries.length === 0) {
      setStatusMessage('Error: No valid items in batch', 'error');
      setBatchSummary(null);
      return;
    }

    setIsBatchProcessing(true);
    setStatusMessage(`Processing ${entries.length} ${entries.length === 1 ? 'item' : 'items'}…`, 'processing');

    const payload = {
      items: entries,
      options: {
        size: options.size,
        color: options.color,
        background: options.background,
        errorCorrection: options.errorCorrection,
        format: 'png'
      }
    };

    try {
      const response = await fetch(buildApiUrl(apiBase, '/batch'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });

      const data = (await response.json()) as { success?: boolean; results?: GenerateResponse[]; error?: string };

      if (!response.ok || !data?.success) {
        throw new Error(data?.error || `Request failed with status ${response.status}`);
      }

      const results = Array.isArray(data.results) ? data.results : [];
      const successCount = results.filter((item) => item?.success).length;
      const total = results.length || entries.length;
      const failureCount = total - successCount;

      setBatchSummary({ success: successCount, total });

      const message =
        failureCount > 0
          ? `Batch complete with warnings: ${successCount}/${total} generated`
          : `Batch complete: ${successCount}/${total} generated`;

      setStatusMessage(message, failureCount > 0 ? 'error' : 'success');
    } catch (error) {
      console.error('[QR UI] Batch error', error);
      setBatchSummary(null);
      setStatusMessage(`Error: ${(error as Error).message}`, 'error');
    } finally {
      setIsBatchProcessing(false);
    }
  }, [apiBase, batchItems, options, setStatusMessage]);

  const batchCount = batchItems.length;

  const colorSwatches = useMemo(
    () => [
      { label: 'Foreground', value: options.color },
      { label: 'Background', value: options.background }
    ],
    [options.color, options.background]
  );

  return (
    <div className="crt animate-flicker">
      <a href="#main" className="skip-link">
        Skip to main content
      </a>

      <div className="relative z-10 mx-auto flex max-w-6xl flex-col gap-8 px-4 pb-20 pt-12 sm:px-6 lg:px-10">
        <header className="flex flex-col items-center gap-4 text-center">
          <div className="relative flex items-center gap-4 rounded-full border border-panelBorder/40 bg-black/40 px-5 py-2 text-sm uppercase tracking-[0.3em] text-foreground/70 shadow-glow">
            <Sparkles className="h-4 w-4 text-accent" aria-hidden="true" />
            Quantum Retro Lab
          </div>

          <h1 className="font-display text-4xl uppercase tracking-[0.6em] text-foreground drop-shadow-sm sm:text-5xl">
            QR Code Generator
          </h1>
          <p className="max-w-2xl text-sm uppercase tracking-[0.32em] text-foreground/65">
            Craft resilient codes with a neon-etched interface engineered for rapid iteration, accessibility, and mobile-first flow.
          </p>
        </header>

        <main id="main" className="space-y-8">
          <section className="grid gap-6 lg:grid-cols-[minmax(0,1.05fr)_minmax(0,1fr)]">
            <article className="retro-panel">
              <div className="card-glare" aria-hidden="true" />
              <div className="space-y-6">
                <header className="flex items-center gap-3">
                  <div className="flex h-12 w-12 items-center justify-center rounded-2xl border border-panelBorder/60 bg-black/40 shadow-glow">
                    <QrCode className="h-6 w-6 text-accent" aria-hidden="true" />
                  </div>
                  <div>
                    <h2 className="font-display text-2xl uppercase tracking-[0.4em] text-foreground">
                      Encode Payload
                    </h2>
                    <p className="text-xs uppercase tracking-[0.28em] text-foreground/55">
                      Input the content and tune fidelity controls
                    </p>
                  </div>
                </header>

                <div className="space-y-5">
                  <div>
                    <Label htmlFor="qr-text">Payload</Label>
                    <Textarea
                      id="qr-text"
                      value={options.text}
                      onChange={(event) =>
                        setOptions((prev) => ({ ...prev, text: event.target.value }))
                      }
                      placeholder="Drop URLs, raw text, or commands."
                      aria-describedby="payload-help"
                    />
                    <p id="payload-help" className="mt-2 text-xs text-foreground/55">
                      We never store your content. Processing happens locally against the Vrooli API service.
                    </p>
                  </div>

                  <div className="grid gap-5 md:grid-cols-2">
                    <div className="space-y-2">
                      <Label htmlFor="qr-size">Size</Label>
                      <Select
                        value={String(options.size)}
                        onValueChange={(value) =>
                          setOptions((prev) => ({ ...prev, size: Number.parseInt(value, 10) }))
                        }
                      >
                        <SelectTrigger id="qr-size">
                          <SelectValue placeholder="Select size" />
                        </SelectTrigger>
                        <SelectContent>
                          {SIZE_OPTIONS.map((size) => (
                            <SelectItem key={size} value={String(size)}>
                              {size}px
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>

                    <div className="space-y-2">
                      <Label htmlFor="qr-correction">Error Correction</Label>
                      <Select
                        value={options.errorCorrection}
                        onValueChange={(value: ErrorCorrection) =>
                          setOptions((prev) => ({ ...prev, errorCorrection: value }))
                        }
                      >
                        <SelectTrigger id="qr-correction">
                          <SelectValue placeholder="Choose level" />
                        </SelectTrigger>
                        <SelectContent>
                          {ERROR_LEVELS.map((level) => (
                            <SelectItem key={level.value} value={level.value}>
                              {level.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  </div>

                  <div className="grid gap-5 sm:grid-cols-2">
                    {colorSwatches.map((swatch) => (
                      <div key={swatch.label} className="space-y-2">
                        <Label>{swatch.label}</Label>
                        <div className="flex items-center gap-3">
                          <Input
                            type="color"
                            className="h-12 w-16 cursor-pointer rounded-2xl border border-panelBorder/40 bg-black/60 p-1"
                            aria-label={`${swatch.label} color`}
                            value={swatch.value}
                            onChange={(event) =>
                              setOptions((prev) => ({
                                ...prev,
                                [swatch.label === 'Foreground' ? 'color' : 'background']:
                                  event.target.value
                              }))
                            }
                          />
                          <div className="flex flex-col">
                            <span className="text-xs font-semibold uppercase tracking-[0.28em] text-foreground/55">
                              HEX
                            </span>
                            <span className="font-mono text-sm tracking-[0.18em] text-foreground">
                              {swatch.value.toUpperCase()}
                            </span>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>

                  <div className="flex flex-wrap items-center gap-3">
                    <Button onClick={handleGenerate} disabled={isGenerating}>
                      {isGenerating ? (
                        <Loader2 className="mr-3 h-4 w-4 animate-spin" aria-hidden="true" />
                      ) : (
                        <Wand2 className="mr-3 h-4 w-4" aria-hidden="true" />
                      )}
                      Generate
                    </Button>
                    <Button
                      variant="secondary"
                      onClick={handleDownload}
                      disabled={!qrResult || isGenerating}
                    >
                      <ArrowDownToLine className="mr-3 h-4 w-4" aria-hidden="true" /> Download
                    </Button>
                    <Button
                      variant="ghost"
                      onClick={handleCopyText}
                      disabled={options.text.trim().length === 0}
                    >
                      <Copy className="mr-3 h-4 w-4" aria-hidden="true" /> Copy Payload
                    </Button>
                  </div>
                </div>
              </div>
            </article>

            <article className="retro-panel flex flex-col">
              <div className="card-glare" aria-hidden="true" />
              <header className="flex flex-col gap-2">
                <h2 className="font-display text-2xl uppercase tracking-[0.4em] text-foreground">
                  Live Preview
                </h2>
                <p className="text-xs uppercase tracking-[0.28em] text-foreground/55">
                  Responsive viewport keeps readability locked in across devices
                </p>
              </header>

              <div
                className={cn(
                  'retro-grid mt-6 flex flex-1 items-center justify-center rounded-2xl border border-panelBorder/60 bg-black/40 p-6 text-center shadow-inner',
                  'min-h-[280px]'
                )}
                aria-live="polite"
                aria-busy={isGenerating}
                aria-label={qrResult?.text ? `QR code preview for ${qrResult.text}` : 'QR code preview'}
              >
                {qrResult ? (
                  <img
                    src={`data:image/png;base64,${qrResult.base64}`}
                    alt={qrResult.text ? `Generated QR code for ${qrResult.text}` : 'Generated QR code'}
                    className="max-h-[320px] w-full max-w-[320px] rounded-3xl border border-panelBorder/40 bg-black/60 p-4 shadow-glow"
                    style={{ imageRendering: 'pixelated' }}
                  />
                ) : (
                  <div className="flex flex-col items-center gap-4 text-center text-sm uppercase tracking-[0.28em] text-foreground/55">
                    <Layers className="h-12 w-12 text-accent/60" aria-hidden="true" />
                    Waiting for your first render
                  </div>
                )}
              </div>

              {batchSummary && (
                <div className="mt-6 rounded-2xl border border-panelBorder/40 bg-black/35 p-4 text-sm uppercase tracking-[0.24em] text-foreground/70">
                  <div className="flex items-center gap-3 text-accent">
                    <CheckCircle2 className="h-4 w-4" aria-hidden="true" /> Batch processed
                  </div>
                  <p className="mt-3 text-xs text-foreground/55">
                    {batchSummary.success} of {batchSummary.total} items rendered successfully.
                  </p>
                </div>
              )}
            </article>
          </section>

          <section className="retro-panel">
            <div className="card-glare" aria-hidden="true" />
            <div className="flex flex-col gap-4">
              <header className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <h2 className="font-display text-2xl uppercase tracking-[0.4em] text-foreground">
                    Batch Lab
                  </h2>
                  <p className="text-xs uppercase tracking-[0.28em] text-foreground/55">
                    Queue multiple payloads and process them in one sweep.
                  </p>
                </div>
                <div className="flex items-center gap-3">
                  <Badge tone={batchCount > 0 ? 'processing' : 'idle'}>
                    {batchCount} queued
                  </Badge>
                  <Button variant="secondary" onClick={handleAddBatchItem}>
                    <PlusCircle className="mr-3 h-4 w-4" aria-hidden="true" /> Add Item
                  </Button>
                </div>
              </header>

              <div className="max-h-[360px] space-y-4 overflow-y-auto pr-2">
                {batchItems.length === 0 ? (
                  <div className="flex items-center justify-between rounded-2xl border border-dashed border-panelBorder/40 bg-black/35 px-4 py-6 text-xs uppercase tracking-[0.28em] text-foreground/45">
                    <span>Nothing queued yet — add payloads above.</span>
                  </div>
                ) : (
                  batchItems.map((item, index) => (
                    <div
                      key={item.id}
                      className="relative rounded-2xl border border-panelBorder/50 bg-black/45 p-4 shadow-[0_0_30px_rgba(94,255,196,0.08)]"
                    >
                      <div className="absolute right-3 top-3">
                        <Button
                          aria-label={`Remove batch item ${index + 1}`}
                          variant="ghost"
                          size="icon"
                          onClick={() => handleRemoveBatchItem(item.id)}
                        >
                          <X className="h-4 w-4" aria-hidden="true" />
                        </Button>
                      </div>

                      <div className="text-xs uppercase tracking-[0.24em] text-foreground/55">Item {index + 1}</div>
                      <div className="mt-3 grid gap-4 md:grid-cols-2">
                        <div>
                          <Label htmlFor={`batch-text-${item.id}`}>Payload</Label>
                          <Textarea
                            id={`batch-text-${item.id}`}
                            value={item.text}
                            onChange={(event) => handleBatchItemChange(item.id, 'text', event.target.value)}
                            placeholder="https://vrooli.app"
                            rows={3}
                          />
                        </div>
                        <div>
                          <Label htmlFor={`batch-label-${item.id}`}>Label</Label>
                          <Input
                            id={`batch-label-${item.id}`}
                            value={item.label}
                            onChange={(event) => handleBatchItemChange(item.id, 'label', event.target.value)}
                            placeholder="QR_01"
                          />
                        </div>
                      </div>
                    </div>
                  ))
                )}
              </div>

              <div className="flex flex-wrap items-center justify-end gap-3">
                <Button
                  variant="secondary"
                  onClick={handleProcessBatch}
                  disabled={batchItems.length === 0 || isBatchProcessing}
                >
                  {isBatchProcessing ? (
                    <Loader2 className="mr-3 h-4 w-4 animate-spin" aria-hidden="true" />
                  ) : (
                    <PlayCircle className="mr-3 h-4 w-4" aria-hidden="true" />
                  )}
                  Process Batch
                </Button>
              </div>
            </div>
          </section>
        </main>

        <footer className="rounded-3xl border border-panelBorder/40 bg-black/35 p-5">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <Badge tone={status.tone} aria-live="polite" aria-atomic="true">
              {status.message}
            </Badge>
            <p className="text-xs uppercase tracking-[0.24em] text-foreground/45">
              API base: {apiBase}
            </p>
          </div>
        </footer>
      </div>
    </div>
  );
}

async function fetchConfig(): Promise<ConfigResponse> {
  try {
    const response = await fetch('/config', { cache: 'no-store' });
    if (!response.ok) {
      throw new Error(`Config request failed with status ${response.status}`);
    }
    return (await response.json()) as ConfigResponse;
  } catch (error) {
    console.warn('[QR UI] Falling back to default API base', error);
    return {};
  }
}

function resolveApiBase(config: ConfigResponse = {}): string {
  const hints = extractConfigApiHints(config);

  if (typeof window === 'undefined') {
    return hints.absolute || hints.relative || DEFAULT_API_PATH;
  }

  const proxyBase = resolveProxyBase();
  if (proxyBase) {
    if (hints.absolute) {
      return hints.absolute;
    }
    const proxyPath = hints.relative || hints.path || DEFAULT_API_PATH;
    return joinUrl(proxyBase, proxyPath);
  }

  const pathBase = resolvePathDerivedProxyBase();

  if (hints.absolute) {
    return hints.absolute;
  }

  if (hints.relative) {
    if (pathBase) {
      return joinUrl(pathBase, hints.relative);
    }

    const originFromLocation = window.location?.origin;
    if (originFromLocation) {
      return joinUrl(originFromLocation, hints.relative);
    }

    return hints.relative;
  }

  const envApiUrl = typeof window.ENV?.API_URL === 'string' ? window.ENV.API_URL.trim() : '';
  if (envApiUrl) {
    return joinUrl(envApiUrl, DEFAULT_API_PATH);
  }

  if (pathBase) {
    return joinUrl(pathBase, DEFAULT_API_PATH);
  }

  const origin = window.location?.origin;
  if (origin) {
    return joinUrl(origin, DEFAULT_API_PATH);
  }

  return DEFAULT_API_PATH;
}

function extractConfigApiHints(config: ConfigResponse = {}) {
  const apiBaseRaw = typeof config.apiBase === 'string' ? config.apiBase.trim() : '';
  const apiUrlRaw = typeof config.apiUrl === 'string' ? config.apiUrl.trim() : '';
  const apiPathRaw = typeof config.apiPath === 'string' ? config.apiPath.trim() : '';

  const absoluteCandidate = [apiBaseRaw, apiUrlRaw].find((value) => value && isAbsoluteUrl(value));
  const absolute = absoluteCandidate ? stripTrailingSlash(absoluteCandidate) : '';

  const relativeCandidate = [apiPathRaw, apiBaseRaw].find((value) => value && !isAbsoluteUrl(value));
  const relative = relativeCandidate ? ensureLeadingSlash(stripTrailingSlash(relativeCandidate)) : '';

  const defaultPath = relative || (apiPathRaw ? ensureLeadingSlash(stripTrailingSlash(apiPathRaw)) : DEFAULT_API_PATH);

  return {
    absolute,
    relative,
    path: defaultPath
  };
}

function resolveProxyBase(): string | undefined {
  if (typeof window === 'undefined') {
    return undefined;
  }

  const info: any = window.__APP_MONITOR_PROXY_INFO__ || window.__APP_MONITOR_PROXY_INDEX__;
  if (!info) {
    return undefined;
  }

  const candidate = pickProxyEndpoint(info);
  if (!candidate) {
    return undefined;
  }

  if (/^https?:\/\//i.test(candidate)) {
    return stripTrailingSlash(candidate);
  }

  const origin = window.location?.origin;
  if (!origin) {
    return undefined;
  }

  return stripTrailingSlash(joinUrl(origin, ensureLeadingSlash(candidate)));
}

function pickProxyEndpoint(info: any): string | undefined {
  const candidates: string[] = [];

  const append = (value: any) => {
    if (!value) {
      return;
    }
    if (typeof value === 'string') {
      const trimmed = value.trim();
      if (trimmed) {
        candidates.push(trimmed);
      }
      return;
    }
    if (typeof value === 'object') {
      append(value.url);
      append(value.path);
      append(value.target);
    }
  };

  append(info);

  if (info && typeof info === 'object') {
    append(info.primary);
    if (Array.isArray(info.endpoints)) {
      info.endpoints.forEach(append);
    }
  }

  return candidates.find(Boolean);
}

function resolvePathDerivedProxyBase(): string {
  if (typeof window === 'undefined') {
    return '';
  }

  const { origin, pathname } = window.location || {};
  if (!origin || typeof pathname !== 'string') {
    return '';
  }

  const proxyIndex = pathname.indexOf('/proxy');
  if (proxyIndex === -1) {
    return '';
  }

  const basePath = pathname.slice(0, proxyIndex + '/proxy'.length);
  return stripTrailingSlash(`${origin}${basePath}`);
}

function isAbsoluteUrl(value?: string): boolean {
  return typeof value === 'string' && /^https?:\/\//i.test(value);
}

function ensureLeadingSlash(value?: string): string {
  if (!value) {
    return '/';
  }
  return value.startsWith('/') ? value : `/${value}`;
}

function stripTrailingSlash(value?: string): string {
  if (!value) {
    return '';
  }
  return value.replace(/\/+$/, '');
}

function joinUrl(base: string, next: string): string {
  const normalizedBase = stripTrailingSlash(base || '');
  const normalizedNext = ensureLeadingSlash(next || '/');
  if (!normalizedBase) {
    return normalizedNext;
  }
  return `${normalizedBase}${normalizedNext}`;
}

function buildApiUrl(base: string, path: string): string {
  const normalizedBase = stripTrailingSlash(base || '');
  const normalizedPath = ensureLeadingSlash(path || '/');
  if (!normalizedBase || normalizedBase === '.') {
    return normalizedPath;
  }
  if (/^https?:\/\//i.test(normalizedBase)) {
    return `${normalizedBase}${normalizedPath}`;
  }
  return `${ensureLeadingSlash(normalizedBase)}${normalizedPath}`;
}

function extractBase64Payload(data: GenerateResponse): string {
  if (!data) {
    return '';
  }
  if (typeof data.data === 'string' && data.data.trim()) {
    return data.data.trim();
  }
  if (typeof data.base64 === 'string' && data.base64.trim()) {
    return data.base64.trim();
  }
  return '';
}

function buildDownloadFilename(text: string): string {
  const sanitized = text
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
  const timestamp = Date.now();
  return sanitized ? `qr-code-${sanitized}-${timestamp}.png` : `qr-code-${timestamp}.png`;
}

export default App;
