import { useCallback, useEffect, useMemo, useState, type ComponentType, type FormEvent, type SVGProps } from 'react';
import {
  Activity,
  AlertCircle,
  Database,
  Layers,
  Lightbulb,
  Network,
  RefreshCw,
  Sparkles,
  FileText,
  Info,
} from 'lucide-react';

import {
  apiClient,
  type CreateSchemaPayload,
  type InputType,
  type JobSummary,
  type ProcessDataPayload,
  type ProcessedRecord,
  type SchemaDetails,
  type SchemaSummary,
} from '@/lib/api';
import {
  cn,
  computeConfidenceVariant,
  formatDate,
  percent,
  safeJsonParse,
  stringify,
} from '@/lib/utils';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Progress } from '@/components/ui/progress';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Textarea } from '@/components/ui/textarea';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';

type AnnouncementTone = 'info' | 'success' | 'error';

type HealthState = {
  status: 'checking' | 'healthy' | 'degraded' | 'offline';
  dependencies: Array<{
    name: string;
    status?: string;
    latency?: number;
  }>;
  message?: string;
  checkedAt?: string;
};

const DEFAULT_SCHEMA_DEFINITION = `{
  "type": "object",
  "properties": {
    "title": { "type": "string", "description": "Human readable label" },
    "summary": { "type": "string", "description": "Short synopsis of the ingested content" },
    "confidence": { "type": "number", "minimum": 0, "maximum": 1 }
  },
  "required": ["title", "summary"]
}`;

const DEFAULT_EXAMPLE_DATA = `{
  "title": "Quarterly earnings call",
  "summary": "Highlights profit growth, risk flags, and upcoming commitments.",
  "confidence": 0.93
}`;

const INITIAL_CREATE_FORM = {
  name: '',
  description: '',
  schemaDefinition: DEFAULT_SCHEMA_DEFINITION,
  exampleData: DEFAULT_EXAMPLE_DATA,
};

const INITIAL_PROCESS_FORM = {
  schemaId: '',
  inputType: 'text' as InputType,
  inputData: '',
};

const STATUS_TITLES: Record<HealthState['status'], string> = {
  checking: 'Verifying API health…',
  healthy: 'API healthy',
  degraded: 'API degraded',
  offline: 'API unreachable',
};

const STATUS_BADGE: Record<HealthState['status'], 'outline' | 'success' | 'warning' | 'danger'> = {
  checking: 'outline',
  healthy: 'success',
  degraded: 'warning',
  offline: 'danger',
};

const PROCESS_STATUS_VARIANTS: Record<string, 'default' | 'success' | 'warning' | 'danger'> = {
  completed: 'success',
  running: 'warning',
  queued: 'outline',
  failed: 'danger',
};

function metricSkeleton(): JSX.Element {
  return (
    <div className="flex flex-col gap-2 rounded-xl border border-slate-800/60 bg-slate-900/30 p-4">
      <div className="h-4 w-28 animate-pulse rounded bg-slate-800/80" />
      <div className="h-8 w-20 animate-pulse rounded bg-slate-800/80" />
    </div>
  );
}

export default function App(): JSX.Element {
  const [health, setHealth] = useState<HealthState>({ status: 'checking', dependencies: [] });
  const [schemas, setSchemas] = useState<SchemaSummary[]>([]);
  const [selectedSchemaId, setSelectedSchemaId] = useState<string | null>(null);
  const [selectedSchemaDetails, setSelectedSchemaDetails] = useState<SchemaDetails | null>(null);
  const [processedRecords, setProcessedRecords] = useState<ProcessedRecord[]>([]);
  const [jobs, setJobs] = useState<JobSummary[]>([]);
  const [loadingHealth, setLoadingHealth] = useState(false);
  const [loadingSchemas, setLoadingSchemas] = useState(false);
  const [loadingData, setLoadingData] = useState(false);
  const [loadingJobs, setLoadingJobs] = useState(false);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [processDialogOpen, setProcessDialogOpen] = useState(false);
  const [createForm, setCreateForm] = useState({ ...INITIAL_CREATE_FORM });
  const [createSubmitting, setCreateSubmitting] = useState(false);
  const [processForm, setProcessForm] = useState({ ...INITIAL_PROCESS_FORM });
  const [processSubmitting, setProcessSubmitting] = useState(false);
  const [announcement, setAnnouncement] = useState<string | null>(null);
  const [announcementTone, setAnnouncementTone] = useState<AnnouncementTone>('info');

  const publishAnnouncement = useCallback((message: string, tone: AnnouncementTone = 'info') => {
    setAnnouncementTone(tone);
    setAnnouncement(message);
  }, []);

  useEffect(() => {
    if (!announcement) {
      return;
    }
    const timer = window.setTimeout(() => setAnnouncement(null), 8000);
    return () => window.clearTimeout(timer);
  }, [announcement]);

  const loadHealth = useCallback(async () => {
    setLoadingHealth(true);
    try {
      const response = await apiClient.health();
      const normalized = response.status?.toLowerCase?.() ?? 'unknown';
      let status: HealthState['status'] = 'healthy';
      if (normalized === 'healthy') {
        status = 'healthy';
      } else if (normalized === 'degraded') {
        status = 'degraded';
      } else if (normalized === 'unhealthy' || normalized === 'error') {
        status = response.readiness ? 'degraded' : 'offline';
      } else {
        status = response.readiness ? 'degraded' : 'offline';
      }

      const dependencies = Object.entries(response.dependencies ?? {}).map(([name, detail]) => ({
        name,
        status: detail?.status,
        latency: typeof detail?.latency_ms === 'number' ? detail.latency_ms : undefined,
      }));

      setHealth({
        status,
        dependencies,
        message: response.status,
        checkedAt: new Date().toISOString(),
      });
    } catch (error) {
      console.error('[data-structurer] Health check failed', error);
      setHealth({ status: 'offline', dependencies: [], message: 'Unable to reach API' });
      publishAnnouncement('Health check failed. Ensure the API container is running.', 'error');
    } finally {
      setLoadingHealth(false);
    }
  }, [publishAnnouncement]);

  const loadJobs = useCallback(async () => {
    setLoadingJobs(true);
    try {
      const response = await apiClient.listJobs();
      setJobs(response.jobs ?? []);
    } catch (error) {
      console.error('[data-structurer] Failed to fetch jobs', error);
      publishAnnouncement('Unable to fetch job activity. Network capture will be limited.', 'error');
    } finally {
      setLoadingJobs(false);
    }
  }, [publishAnnouncement]);

  const loadSchemaBundle = useCallback(async (schemaId: string) => {
    setLoadingData(true);
    try {
      const [details, data] = await Promise.all([
        apiClient.schemaDetails(schemaId),
        apiClient.processedData(schemaId, 10),
      ]);
      setSelectedSchemaDetails(details);
      setProcessedRecords(data.data ?? []);
    } catch (error) {
      console.error('[data-structurer] Failed to load schema context', error);
      publishAnnouncement('Failed to load schema details. Inspect the API logs for specifics.', 'error');
      setSelectedSchemaDetails(null);
      setProcessedRecords([]);
    } finally {
      setLoadingData(false);
    }
  }, [publishAnnouncement]);

  const loadSchemas = useCallback(async () => {
    setLoadingSchemas(true);
    try {
      const response = await apiClient.listSchemas();
      const list = response.schemas ?? [];
      setSchemas(list);
      setProcessForm(prev => ({ ...prev, schemaId: prev.schemaId || list[0]?.id || '' }));
      setSelectedSchemaId(prev => {
        if (prev && list.some(schema => schema.id === prev)) {
          return prev;
        }
        return list[0]?.id ?? null;
      });
    } catch (error) {
      console.error('[data-structurer] Failed to fetch schemas', error);
      publishAnnouncement('Could not load schemas. Validate the database connection.', 'error');
      setSchemas([]);
      setSelectedSchemaId(null);
    } finally {
      setLoadingSchemas(false);
    }
  }, [publishAnnouncement]);

  useEffect(() => {
    void loadHealth();
    void loadSchemas();
    void loadJobs();
  }, [loadHealth, loadSchemas, loadJobs]);

  useEffect(() => {
    if (selectedSchemaId) {
      void loadSchemaBundle(selectedSchemaId);
      setProcessForm(prev => ({ ...prev, schemaId: selectedSchemaId }));
    } else {
      setSelectedSchemaDetails(null);
      setProcessedRecords([]);
    }
  }, [selectedSchemaId, loadSchemaBundle]);

  useEffect(() => {
    const interval = window.setInterval(() => {
      void loadHealth();
      void loadJobs();
      if (selectedSchemaId) {
        void loadSchemaBundle(selectedSchemaId);
      }
    }, 30_000);

    return () => window.clearInterval(interval);
  }, [loadHealth, loadJobs, loadSchemaBundle, selectedSchemaId]);

  const metrics = useMemo(() => {
    const totalSchemas = schemas.length;
    const totalProcessed = schemas.reduce((sum, schema) => sum + (schema.usage_count ?? 0), 0);
    const avgConfidence = totalSchemas
      ? schemas.reduce((sum, schema) => sum + (schema.avg_confidence ?? 0), 0) / totalSchemas
      : 0;
    const activeJobs = jobs.filter(job => ['running', 'queued'].includes(job.status)).length;

    return { totalSchemas, totalProcessed, avgConfidence, activeJobs };
  }, [schemas, jobs]);

  const filteredJobs = useMemo(() => jobs.slice(0, 6), [jobs]);
  const healthBadgeVariant = STATUS_BADGE[health.status];
  const lastChecked = health.checkedAt ? formatDate(health.checkedAt) : '—';

  const metricsLoading = loadingSchemas && schemas.length === 0;

  const handleSelectSchema = useCallback((schemaId: string) => {
    setSelectedSchemaId(schemaId);
  }, []);

  const handleCreateSchema = useCallback(
    async (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      if (createSubmitting) {
        return;
      }

      const trimmedName = createForm.name.trim();
      if (!trimmedName) {
        publishAnnouncement('Schema name is required.', 'error');
        return;
      }

      const parsedDefinition = safeJsonParse<Record<string, unknown>>(createForm.schemaDefinition);
      if (!parsedDefinition || typeof parsedDefinition !== 'object') {
        publishAnnouncement('Schema definition must be valid JSON.', 'error');
        return;
      }

      const exampleData = createForm.exampleData.trim()
        ? safeJsonParse<Record<string, unknown>>(createForm.exampleData)
        : undefined;

      setCreateSubmitting(true);
      try {
        const payload: CreateSchemaPayload = {
          name: trimmedName,
          description: createForm.description.trim(),
          schema_definition: parsedDefinition,
          example_data: exampleData,
        };
        await apiClient.createSchema(payload);
        publishAnnouncement(`Schema “${payload.name}” created successfully.`, 'success');
        setCreateDialogOpen(false);
        setCreateForm({ ...INITIAL_CREATE_FORM });
        await loadSchemas();
      } catch (error) {
        console.error('[data-structurer] Failed to create schema', error);
        const message = error instanceof Error ? error.message : 'Failed to create schema.';
        publishAnnouncement(message, 'error');
      } finally {
        setCreateSubmitting(false);
      }
    },
    [createForm, createSubmitting, loadSchemas, publishAnnouncement],
  );

  const handleProcessData = useCallback(
    async (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      if (processSubmitting) {
        return;
      }

      const schemaId = processForm.schemaId || selectedSchemaId || '';
      if (!schemaId) {
        publishAnnouncement('Select a schema before processing data.', 'error');
        return;
      }

      const trimmedData = processForm.inputData.trim();
      if (!trimmedData) {
        publishAnnouncement('Provide some input data to process.', 'error');
        return;
      }

      const payload: ProcessDataPayload = {
        schema_id: schemaId,
        input_type: processForm.inputType,
        input_data: trimmedData,
      };

      setProcessSubmitting(true);
      try {
        const response = await apiClient.processData(payload);
        if (response.status === 'failed') {
          const detail = response.errors?.join(', ') ?? 'Processing failed.';
          publishAnnouncement(detail, 'error');
        } else {
          publishAnnouncement('Processing pipeline started successfully.', 'success');
        }
        setProcessDialogOpen(false);
        setProcessForm(prev => ({ ...prev, inputData: '' }));
        await Promise.all([loadSchemaBundle(schemaId), loadJobs(), loadSchemas()]);
      } catch (error) {
        console.error('[data-structurer] Failed to process data', error);
        const message = error instanceof Error ? error.message : 'Failed to process data.';
        publishAnnouncement(message, 'error');
      } finally {
        setProcessSubmitting(false);
      }
    },
    [processForm, processSubmitting, selectedSchemaId, loadSchemaBundle, loadJobs, loadSchemas, publishAnnouncement],
  );

  const refreshAll = useCallback(() => {
    void Promise.all([
      loadHealth(),
      loadSchemas(),
      loadJobs(),
      selectedSchemaId ? loadSchemaBundle(selectedSchemaId) : Promise.resolve(),
    ]);
  }, [loadHealth, loadSchemas, loadJobs, loadSchemaBundle, selectedSchemaId]);

  const announcementIcon = announcementTone === 'error' ? (
    <AlertCircle className="h-5 w-5 text-rose-300" aria-hidden="true" />
  ) : announcementTone === 'success' ? (
    <Sparkles className="h-5 w-5 text-emerald-300" aria-hidden="true" />
  ) : (
    <Info className="h-5 w-5 text-sky-300" aria-hidden="true" />
  );

  return (
    <TooltipProvider delayDuration={150}>
      <div className="min-h-screen bg-slate-950 text-slate-100">
        <header className="border-b border-slate-900/80 bg-slate-950/70 backdrop-blur supports-[backdrop-filter]:bg-slate-950/40">
          <div className="container mx-auto flex flex-col gap-6 px-4 py-8 lg:flex-row lg:items-center lg:justify-between">
            <div className="space-y-2">
              <div className="flex items-center gap-3">
                <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-sky-500/20">
                  <Database className="h-6 w-6 text-sky-300" aria-hidden="true" />
                </div>
                <div>
                  <h1 className="text-3xl font-semibold tracking-tight">Data Structurer</h1>
                  <p className="text-sm text-slate-400">
                    Transform unstructured input into governed datasets with schema intelligence.
                  </p>
                </div>
              </div>
              <div className="flex flex-wrap items-center gap-2 text-xs text-slate-500">
                <span>Runtime bridge addressable</span>
                <span aria-hidden="true">•</span>
                <span>{apiClient.baseUrl ? `API base ${apiClient.baseUrl}` : 'Using same-origin API'}</span>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <Badge variant={healthBadgeVariant}>{STATUS_TITLES[health.status]}</Badge>
              <div className="text-xs text-slate-400">Last check: {lastChecked}</div>
              <Button
                type="button"
                variant="secondary"
                className="gap-2"
                onClick={refreshAll}
                disabled={loadingHealth || loadingSchemas || loadingJobs}
              >
                <RefreshCw
                  className={cn('h-4 w-4', {
                    'animate-spin': loadingHealth || loadingSchemas || loadingJobs,
                  })}
                  aria-hidden="true"
                />
                Refresh now
              </Button>
            </div>
          </div>
        </header>

        <main className="container mx-auto space-y-10 px-4 py-10">
          {announcement ? (
            <div
              className={cn(
                'flex items-start gap-3 rounded-xl border p-4 shadow-lg',
                announcementTone === 'error' && 'border-rose-500/50 bg-rose-500/10 text-rose-100',
                announcementTone === 'success' && 'border-emerald-500/40 bg-emerald-500/10 text-emerald-100',
                announcementTone === 'info' && 'border-sky-500/40 bg-slate-900/70 text-slate-100',
              )}
              role={announcementTone === 'error' ? 'alert' : 'status'}
              aria-live="polite"
            >
              {announcementIcon}
              <p className="text-sm leading-relaxed">{announcement}</p>
            </div>
          ) : null}

          <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            {metricsLoading ? (
              <>
                {metricSkeleton()}
                {metricSkeleton()}
                {metricSkeleton()}
                {metricSkeleton()}
              </>
            ) : (
              <>
                <MetricCard
                  icon={Database}
                  label="Active Schemas"
                  value={metrics.totalSchemas.toLocaleString()}
                  helper="Blueprints currently available for structuring data."
                />
                <MetricCard
                  icon={Layers}
                  label="Documents Structured"
                  value={metrics.totalProcessed.toLocaleString()}
                  helper="Aggregated usage count across every schema."
                />
                <MetricCard
                  icon={Sparkles}
                  label="Average Confidence"
                  value={percent(metrics.avgConfidence, '0.0%')}
                  helper="Mean confidence score reported by the API."
                />
                <MetricCard
                  icon={Activity}
                  label="Active Jobs"
                  value={metrics.activeJobs.toString()}
                  helper="Queued or running processing runs."
                />
              </>
            )}
          </section>

          <section className="grid gap-6 lg:grid-cols-[320px_1fr] xl:grid-cols-[360px_1fr]">
            <Card className="h-full">
              <CardHeader>
                <CardTitle>Schema Catalog</CardTitle>
                <CardDescription>
                  Pick a schema to inspect the blueprint, metrics, and recent structured outputs.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex flex-wrap gap-2">
                  <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
                    <DialogTrigger asChild>
                      <Button type="button" className="gap-2">
                        <Sparkles className="h-4 w-4" aria-hidden="true" />
                        New schema
                      </Button>
                    </DialogTrigger>
                    <DialogContent>
                      <DialogHeader>
                        <DialogTitle>Create schema</DialogTitle>
                        <DialogDescription>
                          Describe the fields you expect. JSON schema shapes the API&apos;s validation and UI hints.
                        </DialogDescription>
                      </DialogHeader>
                      <form className="space-y-4" onSubmit={handleCreateSchema}>
                        <div className="space-y-2">
                          <Label htmlFor="schema-name">Schema name</Label>
                          <Input
                            id="schema-name"
                            value={createForm.name}
                            onChange={event => setCreateForm(prev => ({ ...prev, name: event.target.value }))}
                            placeholder="Customer Intake"
                            autoFocus
                          />
                        </div>
                        <div className="space-y-2">
                          <Label htmlFor="schema-description">Description</Label>
                          <Textarea
                            id="schema-description"
                            value={createForm.description}
                            onChange={event => setCreateForm(prev => ({ ...prev, description: event.target.value }))}
                            placeholder="Short human readable explanation"
                            rows={3}
                          />
                        </div>
                        <div className="space-y-2">
                          <div className="flex items-center justify-between">
                            <Label htmlFor="schema-definition">Schema definition (JSON)</Label>
                            <span className="text-xs text-slate-500">Required</span>
                          </div>
                          <Textarea
                            id="schema-definition"
                            value={createForm.schemaDefinition}
                            onChange={event => setCreateForm(prev => ({ ...prev, schemaDefinition: event.target.value }))}
                            className="font-mono"
                            rows={8}
                          />
                        </div>
                        <div className="space-y-2">
                          <div className="flex items-center justify-between">
                            <Label htmlFor="schema-example">Example output (optional)</Label>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <button
                                  type="button"
                                  className="text-xs text-sky-300 underline-offset-2 hover:underline"
                                  onClick={() =>
                                    setCreateForm(prev => ({
                                      ...prev,
                                      exampleData: DEFAULT_EXAMPLE_DATA,
                                    }))
                                  }
                                >
                                  Reset example
                                </button>
                              </TooltipTrigger>
                              <TooltipContent>Used for UI hints and sample payloads.</TooltipContent>
                            </Tooltip>
                          </div>
                          <Textarea
                            id="schema-example"
                            value={createForm.exampleData}
                            onChange={event => setCreateForm(prev => ({ ...prev, exampleData: event.target.value }))}
                            className="font-mono"
                            rows={6}
                          />
                        </div>
                        <DialogFooter>
                          <DialogClose asChild>
                            <Button type="button" variant="secondary">
                              Cancel
                            </Button>
                          </DialogClose>
                          <Button type="submit" disabled={createSubmitting} className="gap-2">
                            {createSubmitting ? (
                              <RefreshCw className="h-4 w-4 animate-spin" aria-hidden="true" />
                            ) : (
                              <Sparkles className="h-4 w-4" aria-hidden="true" />
                            )}
                            Create schema
                          </Button>
                        </DialogFooter>
                      </form>
                    </DialogContent>
                  </Dialog>
                  <Dialog open={processDialogOpen} onOpenChange={setProcessDialogOpen}>
                    <DialogTrigger asChild>
                      <Button type="button" variant="outline" className="gap-2">
                        <FileText className="h-4 w-4" aria-hidden="true" />
                        Process data
                      </Button>
                    </DialogTrigger>
                    <DialogContent>
                      <DialogHeader>
                        <DialogTitle>Submit data for structuring</DialogTitle>
                        <DialogDescription>
                          The API will enqueue the request and surface structured output when the job completes.
                        </DialogDescription>
                      </DialogHeader>
                      <form className="space-y-4" onSubmit={handleProcessData}>
                        <div className="space-y-2">
                          <Label htmlFor="process-schema">Target schema</Label>
                          <select
                            id="process-schema"
                            className="h-11 w-full rounded-lg border border-slate-800 bg-slate-950/70 px-3 text-sm text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                            value={processForm.schemaId || selectedSchemaId || ''}
                            onChange={event =>
                              setProcessForm(prev => ({ ...prev, schemaId: event.target.value }))
                            }
                          >
                            <option value="" disabled>
                              {schemas.length ? 'Choose a schema' : 'No schemas available'}
                            </option>
                            {schemas.map(schema => (
                              <option key={schema.id} value={schema.id}>
                                {schema.name}
                              </option>
                            ))}
                          </select>
                        </div>
                        <div className="space-y-2">
                          <Label>Input type</Label>
                          <div className="grid gap-2 sm:grid-cols-3">
                            {(['text', 'url', 'file'] as InputType[]).map(option => (
                              <button
                                type="button"
                                key={option}
                                onClick={() =>
                                  setProcessForm(prev => ({ ...prev, inputType: option }))
                                }
                                className={cn(
                                  'rounded-lg border px-3 py-2 text-sm capitalize transition',
                                  processForm.inputType === option
                                    ? 'border-sky-500 bg-sky-500/10 text-sky-100'
                                    : 'border-slate-800 bg-slate-950/70 text-slate-300 hover:border-slate-700',
                                )}
                              >
                                {option}
                              </button>
                            ))}
                          </div>
                          {processForm.inputType === 'file' ? (
                            <p className="text-xs text-slate-400">
                              File uploads are planned — supply a URL for now so the automation can fetch your document.
                            </p>
                          ) : null}
                        </div>
                        <div className="space-y-2">
                          <Label htmlFor="process-data">Input</Label>
                          {processForm.inputType === 'text' ? (
                            <Textarea
                              id="process-data"
                              rows={6}
                              value={processForm.inputData}
                              onChange={event =>
                                setProcessForm(prev => ({ ...prev, inputData: event.target.value }))
                              }
                              placeholder="Paste the raw content that needs structuring"
                            />
                          ) : (
                            <Input
                              id="process-data"
                              value={processForm.inputData}
                              onChange={event =>
                                setProcessForm(prev => ({ ...prev, inputData: event.target.value }))
                              }
                              placeholder={
                                processForm.inputType === 'url'
                                  ? 'https://example.com/report.pdf'
                                  : 'Upload support coming soon'
                              }
                              disabled={processForm.inputType === 'file'}
                            />
                          )}
                        </div>
                        <DialogFooter>
                          <DialogClose asChild>
                            <Button type="button" variant="secondary">
                              Cancel
                            </Button>
                          </DialogClose>
                          <Button type="submit" disabled={processSubmitting} className="gap-2">
                            {processSubmitting ? (
                              <RefreshCw className="h-4 w-4 animate-spin" aria-hidden="true" />
                            ) : (
                              <Activity className="h-4 w-4" aria-hidden="true" />
                            )}
                            Start processing
                          </Button>
                        </DialogFooter>
                      </form>
                    </DialogContent>
                  </Dialog>
                </div>
                <ScrollArea className="h-[470px] pr-2">
                  <div className="space-y-3">
                    {loadingSchemas && schemas.length === 0 ? (
                      <div className="space-y-3" aria-live="polite">
                        {[0, 1, 2].map(index => (
                          <div
                            key={index}
                            className="h-20 animate-pulse rounded-lg border border-slate-800/70 bg-slate-900/40"
                          />
                        ))}
                      </div>
                    ) : null}
                    {!loadingSchemas && schemas.length === 0 ? (
                      <div className="rounded-lg border border-dashed border-slate-700 bg-slate-900/30 p-6 text-sm text-slate-400">
                        No schemas yet. Create one to unlock the structuring pipeline.
                      </div>
                    ) : null}
                    {schemas.map(schema => {
                      const isActive = selectedSchemaId === schema.id;
                      return (
                        <button
                          key={schema.id}
                          type="button"
                          onClick={() => handleSelectSchema(schema.id)}
                          className={cn(
                            'w-full rounded-lg border p-4 text-left transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                            isActive
                              ? 'border-sky-500/70 bg-sky-500/10'
                              : 'border-slate-800 bg-slate-950/60 hover:border-slate-700 hover:bg-slate-900/60',
                          )}
                        >
                          <div className="flex items-center justify-between gap-2">
                            <p className="text-base font-semibold text-slate-50">{schema.name}</p>
                            <Badge variant={schema.is_active ? 'success' : 'outline'}>
                              v{schema.version}
                            </Badge>
                          </div>
                          <p className="mt-2 line-clamp-2 text-sm text-slate-400">
                            {schema.description || 'No description provided yet.'}
                          </p>
                          <div className="mt-3 flex flex-wrap items-center gap-3 text-xs text-slate-400">
                            <span>{(schema.usage_count ?? 0).toLocaleString()} runs</span>
                            <span aria-hidden="true">•</span>
                            <span>{percent(schema.avg_confidence, '0%')} avg confidence</span>
                          </div>
                        </button>
                      );
                    })}
                  </div>
                </ScrollArea>
              </CardContent>
            </Card>

            <div className="space-y-6">
              <Card>
                <CardHeader className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
                  <div>
                    <CardTitle>Schema overview</CardTitle>
                    <CardDescription>
                      Detailed blueprint, version history, and metadata for the active schema.
                    </CardDescription>
                  </div>
                  <div className="flex flex-wrap items-center gap-2 text-xs text-slate-400">
                    <Network className="h-4 w-4" aria-hidden="true" />
                    {health.dependencies.length ? (
                      <span>
                        Dependencies: {health.dependencies.map(dep => dep.name).join(', ')}
                      </span>
                    ) : (
                      <span>No dependency telemetry yet</span>
                    )}
                  </div>
                </CardHeader>
                <CardContent className="space-y-6">
                  {loadingData && !selectedSchemaDetails ? (
                    <div className="space-y-4">
                      <div className="h-6 w-56 animate-pulse rounded bg-slate-800/60" />
                      <div className="h-32 animate-pulse rounded-lg bg-slate-900/40" />
                    </div>
                  ) : null}
                  {!loadingData && !selectedSchemaDetails ? (
                    <EmptyState
                      title="Select a schema"
                      description="Pick a schema from the left to inspect its definition and recent activity."
                    />
                  ) : null}
                  {selectedSchemaDetails ? (
                    <div className="space-y-6">
                      <div className="grid gap-4 md:grid-cols-3">
                        <InfoBlock label="Version" value={`v${selectedSchemaDetails.version}`} />
                        <InfoBlock label="Created" value={formatDate(selectedSchemaDetails.created_at)} />
                        <InfoBlock label="Updated" value={formatDate(selectedSchemaDetails.updated_at)} />
                        <InfoBlock
                          label="Usage"
                          value={`${(selectedSchemaDetails.usage_count ?? 0).toLocaleString()} runs`}
                        />
                        <InfoBlock
                          label="Confidence"
                          value={percent(selectedSchemaDetails.avg_confidence, '—')}
                        />
                        <InfoBlock label="Status" value={selectedSchemaDetails.is_active ? 'Active' : 'Disabled'} />
                      </div>
                      <div className="space-y-2">
                        <h3 className="text-sm font-semibold text-slate-200">Schema definition</h3>
                        <pre className="max-h-72 overflow-auto rounded-lg border border-slate-800 bg-slate-950/70 p-4 text-xs text-slate-100">
                          {stringify(selectedSchemaDetails.schema_definition)}
                        </pre>
                      </div>
                      {selectedSchemaDetails.example_data ? (
                        <div className="space-y-2">
                          <h3 className="text-sm font-semibold text-slate-200">Example output</h3>
                          <pre className="max-h-60 overflow-auto rounded-lg border border-slate-800 bg-slate-950/70 p-4 text-xs text-slate-100">
                            {stringify(selectedSchemaDetails.example_data)}
                          </pre>
                        </div>
                      ) : null}
                    </div>
                  ) : null}
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>Recent structured outputs</CardTitle>
                  <CardDescription>
                    Confidence, raw payload, and metadata for the latest processing runs.
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  {loadingData && processedRecords.length === 0 ? (
                    <div className="space-y-4" aria-live="polite">
                      {[0, 1, 2].map(index => (
                        <div
                          key={index}
                          className="h-28 animate-pulse rounded-lg border border-slate-800/70 bg-slate-900/40"
                        />
                      ))}
                    </div>
                  ) : null}
                  {!loadingData && processedRecords.length === 0 ? (
                    <EmptyState
                      title="No structured documents yet"
                      description="Run the processing pipeline to populate this view with fresh, structured data."
                    />
                  ) : null}
                  <div className="space-y-4">
                    {processedRecords.map(record => {
                      const variant = computeConfidenceVariant(record.confidence_score);
                      const statusVariant = PROCESS_STATUS_VARIANTS[record.processing_status ?? 'completed'] ?? 'outline';
                      return (
                        <div
                          key={record.id}
                          className="rounded-xl border border-slate-800 bg-slate-950/70 p-5 shadow-inner shadow-slate-950/30"
                        >
                          <div className="flex flex-wrap items-center justify-between gap-3">
                            <div className="space-y-1">
                              <p className="text-sm font-semibold text-slate-200">
                                {formatDate(record.processed_at ?? record.created_at)}
                              </p>
                              <p className="text-xs text-slate-500">Record ID: {record.id}</p>
                            </div>
                            <Badge variant={statusVariant}>{record.processing_status ?? 'completed'}</Badge>
                          </div>
                          <div className="mt-4 space-y-2">
                            <div className="flex items-center justify-between text-xs text-slate-400">
                              <span>Confidence</span>
                              <span>{percent(record.confidence_score, '—')}</span>
                            </div>
                            <Progress
                              value={Math.min(100, Math.max(0, (record.confidence_score ?? 0) * 100))}
                              className={cn(
                                variant === 'high' && 'bg-emerald-900/40',
                                variant === 'medium' && 'bg-amber-900/40',
                                variant === 'low' && 'bg-rose-900/40',
                              )}
                            />
                          </div>
                          <pre className="mt-4 max-h-64 overflow-auto rounded-lg border border-slate-800 bg-slate-950/80 p-4 text-xs text-slate-100">
                            {stringify(record.structured_data ?? record.metadata ?? {})}
                          </pre>
                          {record.error_message ? (
                            <p className="mt-3 text-xs text-rose-300">{record.error_message}</p>
                          ) : null}
                        </div>
                      );
                    })}
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="flex flex-col gap-2 lg:flex-row lg:items-center lg:justify-between">
                  <div>
                    <CardTitle>Job activity</CardTitle>
                    <CardDescription>Live view of the processing queue.</CardDescription>
                  </div>
                  <Button
                    type="button"
                    variant="ghost"
                    className="gap-2 text-xs text-slate-400 hover:text-slate-100"
                    onClick={() => void loadJobs()}
                  >
                    <RefreshCw className={cn('h-4 w-4', { 'animate-spin': loadingJobs })} aria-hidden="true" />
                    Sync jobs
                  </Button>
                </CardHeader>
                <CardContent>
                  {loadingJobs && filteredJobs.length === 0 ? (
                    <div className="space-y-3">
                      {[0, 1, 2].map(index => (
                        <div key={index} className="h-16 animate-pulse rounded-lg bg-slate-900/40" />
                      ))}
                    </div>
                  ) : null}
                  {!loadingJobs && filteredJobs.length === 0 ? (
                    <EmptyState
                      title="No recent jobs"
                      description="Start a processing run to populate the job timeline."
                    />
                  ) : null}
                  <div className="space-y-3">
                    {filteredJobs.map(job => (
                      <div
                        key={job.id}
                        className="flex flex-col gap-2 rounded-lg border border-slate-800 bg-slate-950/60 p-4 md:flex-row md:items-center md:justify-between"
                      >
                        <div>
                          <p className="text-sm font-semibold text-slate-200">{job.status}</p>
                          <p className="text-xs text-slate-500">Job ID: {job.id}</p>
                        </div>
                        <div className="flex flex-wrap items-center gap-3 text-xs text-slate-400">
                          <span>Schema: {job.schema_id.slice(0, 8)}…</span>
                          {job.progress !== undefined ? (
                            <span>Progress: {Math.round(job.progress)}%</span>
                          ) : null}
                          <span>Updated: {formatDate(job.updated_at ?? job.created_at)}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>

              <Card className="border-sky-500/30 bg-slate-950/60">
                <CardHeader>
                  <div className="flex items-center gap-3">
                    <Lightbulb className="h-5 w-5 text-amber-300" aria-hidden="true" />
                    <div>
                      <CardTitle>Adoption playbook</CardTitle>
                      <CardDescription>Quick wins to get value from Data Structurer immediately.</CardDescription>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <ul className="space-y-3 text-sm text-slate-300">
                    <li className="flex items-start gap-2">
                      <span className="mt-1 h-2 w-2 rounded-full bg-sky-400" aria-hidden="true" />
                      Wire a schema per downstream table to avoid brittle mappings later.
                    </li>
                    <li className="flex items-start gap-2">
                      <span className="mt-1 h-2 w-2 rounded-full bg-sky-400" aria-hidden="true" />
                      Use the confidence tracking to trigger human review when accuracy dips below 70%.
                    </li>
                    <li className="flex items-start gap-2">
                      <span className="mt-1 h-2 w-2 rounded-full bg-sky-400" aria-hidden="true" />
                      Keep schema descriptions crisp; they drive the in-app tooltips and API docs.
                    </li>
                  </ul>
                </CardContent>
                <CardFooter className="flex flex-wrap gap-3 text-xs text-slate-400">
                  <span>Need to debug a run?</span>
                  <a href="/docs" className="inline-flex items-center gap-1 text-sky-300">
                    View scenario docs
                    <span aria-hidden="true">→</span>
                  </a>
                </CardFooter>
              </Card>
            </div>
          </section>
        </main>

        <footer className="border-t border-slate-900/80 bg-slate-950/80 py-4 text-xs text-slate-500">
          <div className="container mx-auto flex flex-col gap-2 px-4 sm:flex-row sm:items-center sm:justify-between">
            <span>Console &amp; network capture ready via Vrooli runtime bridge.</span>
            <span>© {new Date().getFullYear()} Vrooli intelligence stack</span>
          </div>
        </footer>
      </div>
    </TooltipProvider>
  );
}

type MetricCardProps = {
  icon: ComponentType<SVGProps<SVGSVGElement>>;
  label: string;
  value: string;
  helper: string;
};

function MetricCard({ icon: Icon, label, value, helper }: MetricCardProps): JSX.Element {
  return (
    <Card className="border-slate-800/70 bg-slate-950/60">
      <CardContent className="flex flex-col gap-3 p-5">
        <div className="flex items-center justify-between gap-3">
          <p className="text-sm text-slate-400">{label}</p>
          <div className="rounded-lg bg-slate-900/80 p-2">
            <Icon className="h-5 w-5 text-sky-300" aria-hidden="true" />
          </div>
        </div>
        <div className="text-3xl font-semibold tracking-tight text-slate-50">{value}</div>
        <Tooltip>
          <TooltipTrigger className="w-fit text-left text-xs text-slate-500">
            Learn more
          </TooltipTrigger>
          <TooltipContent>{helper}</TooltipContent>
        </Tooltip>
      </CardContent>
    </Card>
  );
}

type InfoBlockProps = {
  label: string;
  value: string;
};

function InfoBlock({ label, value }: InfoBlockProps): JSX.Element {
  return (
    <div className="rounded-lg border border-slate-800 bg-slate-950/70 p-4">
      <p className="text-xs uppercase tracking-wide text-slate-500">{label}</p>
      <p className="mt-2 text-sm font-semibold text-slate-100">{value}</p>
    </div>
  );
}

type EmptyStateProps = {
  title: string;
  description: string;
};

function EmptyState({ title, description }: EmptyStateProps): JSX.Element {
  return (
    <div className="rounded-lg border border-dashed border-slate-800 bg-slate-950/50 p-8 text-center">
      <p className="text-sm font-semibold text-slate-200">{title}</p>
      <p className="mt-2 text-sm text-slate-400">{description}</p>
    </div>
  );
}
