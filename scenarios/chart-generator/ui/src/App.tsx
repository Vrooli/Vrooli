import { useEffect, useMemo, useRef, useState } from 'react';
import { ArrowUpRight, BarChart3, DownloadIcon } from 'lucide-react';

import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

import type { ChartDatum, ChartStyleId, ChartType } from './lib/chart-engine';
import { ChartEngine } from './lib/chart-engine';
import { chartTypeSamples, sampleDataMap, type SampleDataKey } from './lib/sample-data';
import { downloadDataUrl, downloadFile } from './lib/utils';
import { ChartPreview } from './components/chart/chart-preview';
import { ChartTypeSelector } from './components/chart/chart-type-selector';
import { DataPanel } from './components/chart/data-panel';
import { StyleSelector } from './components/chart/style-selector';
import { Button } from './components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from './components/ui/card';
import { ScrollArea } from './components/ui/scroll-area';
import { Tabs, TabsContent, TabsList, TabsTrigger } from './components/ui/tabs';
import { useToast } from './components/ui/use-toast';

type ExportFormat = 'png' | 'svg';

declare global {
  interface Window {
    __chartGeneratorBridgeInitialized?: boolean;
  }
}

const formatLabel: Record<ExportFormat, string> = {
  png: 'PNG image',
  svg: 'SVG vector',
};

function initialiseIframeBridge() {
  if (typeof window === 'undefined' || window.__chartGeneratorBridgeInitialized) {
    return;
  }

  if (window.parent !== window) {
    let parentOrigin: string | undefined;
    try {
      if (document.referrer) {
        parentOrigin = new URL(document.referrer).origin;
      }
    } catch (error) {
      console.warn('[chart-generator/ui] Unable to detect parent origin', error);
    }

    initIframeBridgeChild({ parentOrigin, appId: 'chart-generator' });
    window.__chartGeneratorBridgeInitialized = true;
  }
}

function parseChartData(raw: string): ChartDatum[] {
  const trimmed = raw.trim();
  if (!trimmed) {
    return [];
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch (error) {
    throw new Error('Data must be valid JSON.');
  }

  if (!Array.isArray(parsed)) {
    throw new Error('Data must be an array of objects.');
  }

  const coerced = parsed.map((entry, index) => {
    if (entry === null || typeof entry !== 'object') {
      throw new Error(`Item ${index + 1} must be an object with x/y values.`);
    }
    return entry as ChartDatum;
  });

  return coerced;
}

const formatJson = (value: ChartDatum[]) => JSON.stringify(value, null, 2);

export default function App() {
  const engine = useMemo(() => new ChartEngine(), []);
  const previewRef = useRef<HTMLDivElement | null>(null);
  const { toast } = useToast();

  const [chartType, setChartType] = useState<ChartType>('bar');
  const [styleId, setStyleId] = useState<ChartStyleId>('professional');
  const initialSample = formatJson(sampleDataMap[chartTypeSamples.bar]);
  const [dataInput, setDataInput] = useState<string>(initialSample);
  const [data, setData] = useState<ChartDatum[]>(sampleDataMap[chartTypeSamples.bar]);
  const [lastValidated, setLastValidated] = useState<Date | null>(new Date());
  const [isValid, setIsValid] = useState<boolean | null>(true);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'charts' | 'styles' | 'data'>('charts');

  useEffect(() => {
    initialiseIframeBridge();
  }, []);

  const validateAndUpdate = (raw: string) => {
    try {
      const parsed = parseChartData(raw);
      setData(parsed);
      setIsValid(true);
      setErrorMessage(null);
      setLastValidated(new Date());
      if (parsed.length === 0) {
        toast({
          title: 'Data cleared',
          description: 'Provide at least one item to render a chart.',
          variant: 'warning',
        });
      } else {
        toast({
          title: 'Data ready',
          description: `Loaded ${parsed.length} data points.`,
          variant: 'success',
        });
      }
    } catch (error) {
      if (error instanceof Error) {
        setIsValid(false);
        setErrorMessage(error.message);
        toast({ title: 'Invalid data', description: error.message, variant: 'destructive' });
      }
    }
  };

  const handleSampleSelect = (sample: SampleDataKey) => {
    const sampleData = sampleDataMap[sample];
    const formatted = formatJson(sampleData);
    setDataInput(formatted);
    setActiveTab('data');
    validateAndUpdate(formatted);
  };

  const handleImport = async (file: File) => {
    const text = await file.text();
    setDataInput(text);
    setActiveTab('data');
    validateAndUpdate(text);
    toast({ title: 'Data imported', description: `Loaded ${file.name}`, variant: 'success' });
  };

  const handleExport = async (format: ExportFormat) => {
    if (!previewRef.current) {
      toast({ title: 'Preview not ready', description: 'Render a chart before exporting.', variant: 'warning' });
      return;
    }

    if (!data || data.length === 0) {
      toast({ title: 'No data to export', description: 'Add data points before exporting.', variant: 'warning' });
      return;
    }

    const result = await engine.exportChart(previewRef.current, format);
    if (!result) {
      toast({ title: 'Export failed', description: 'Unable to export chart. Try again.', variant: 'destructive' });
      return;
    }

    if (format === 'svg') {
      downloadFile(result, `chart-generator.${format}`, 'image/svg+xml');
    } else {
      downloadDataUrl(result, `chart-generator.${format}`);
    }
    toast({ title: 'Export complete', description: `${formatLabel[format]} downloaded.` });
  };

  const handleClear = () => {
    setDataInput('');
    setData([]);
    setIsValid(null);
    setErrorMessage(null);
    setLastValidated(null);
  };

  const handleChartTypeChange = (type: ChartType) => {
    setChartType(type);
    if (!dataInput.trim()) {
      const sample = sampleDataMap[chartTypeSamples[type]];
      const formatted = formatJson(sample);
      setDataInput(formatted);
      validateAndUpdate(formatted);
    }
  };

  useEffect(() => {
    // Ensure we always have data for brand-new sessions
    if (!dataInput.trim()) {
      const fallback = formatJson(sampleDataMap[chartTypeSamples[chartType]]);
      setDataInput(fallback);
      validateAndUpdate(fallback);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="mx-auto flex min-h-screen max-w-7xl flex-col gap-6 px-6 py-8">
      <header className="flex flex-col gap-4 rounded-3xl border border-border bg-white/90 p-6 shadow-lg backdrop-blur">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-3 text-brand-600">
              <BarChart3 className="h-7 w-7" />
              <span className="font-semibold uppercase tracking-wide">Professional Chart Generator</span>
            </div>
            <h1 className="mt-2 text-3xl font-bold text-slate-900">Build investor-ready visuals in minutes</h1>
            <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
              Select a chart type, tune the brand aesthetic with shadcn components, and export polished assets using
              lucide-powered UI affordances.
            </p>
          </div>
          <div className="flex gap-2">
            {(['png', 'svg'] as ExportFormat[]).map((format) => (
              <Button key={format} variant="outline" onClick={() => handleExport(format)}>
                <DownloadIcon className="mr-2 h-4 w-4" /> {format.toUpperCase()} export
              </Button>
            ))}
          </div>
        </div>
      </header>

      <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as typeof activeTab)}>
        <div className="flex flex-wrap items-center justify-between gap-4">
          <TabsList>
            <TabsTrigger value="charts">Chart library</TabsTrigger>
            <TabsTrigger value="styles">Styles</TabsTrigger>
            <TabsTrigger value="data">Data</TabsTrigger>
          </TabsList>
          <Button variant="link" className="text-sm text-muted-foreground" onClick={() => setActiveTab('data')}>
            Data contracts <ArrowUpRight className="ml-1 h-4 w-4" />
          </Button>
        </div>

        <div className="grid gap-6 lg:grid-cols-[340px_minmax(0,1fr)]">
          <ScrollArea className="h-[520px] rounded-2xl border border-border bg-white/80 p-5">
            <TabsContent value="charts" className="m-0">
              <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">Chart types</h2>
              <p className="mt-1 mb-4 text-sm text-muted-foreground">
                Switch between high-impact visualisations optimised for investor decks and executive dashboards.
              </p>
              <ChartTypeSelector value={chartType} onChange={handleChartTypeChange} />
            </TabsContent>

            <TabsContent value="styles" className="m-0">
              <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">Visual style</h2>
              <p className="mt-1 mb-4 text-sm text-muted-foreground">
                lucide-driven affordances and shadcn primitives keep your storytelling consistent across chart types.
              </p>
              <StyleSelector value={styleId} onChange={setStyleId} />
            </TabsContent>

            <TabsContent value="data" className="m-0">
              <DataPanel
                value={dataInput}
                onChange={(value) => {
                  setDataInput(value);
                  setIsValid(null);
                  setErrorMessage(null);
                }}
                onSampleSelect={handleSampleSelect}
                onValidate={() => validateAndUpdate(dataInput)}
                onClear={handleClear}
                onImport={handleImport}
                samples={[chartTypeSamples.bar, 'revenue', 'performance', 'growth', 'correlation']}
                dataPoints={data.length}
                lastUpdated={lastValidated}
                isValid={isValid}
                errors={errorMessage}
              />
            </TabsContent>
          </ScrollArea>

          <div className="space-y-4">
            <Card className="bg-white/95">
              <CardHeader className="pb-3">
                <CardTitle className="text-xl font-semibold text-slate-900">Live preview</CardTitle>
                <CardDescription>
                  React + Vite + TypeScript + shadcn UI render a production-ready experience with lucide icons guiding
                  every interaction.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <ChartPreview engine={engine} chartType={chartType} styleId={styleId} data={data} ref={previewRef} />
              </CardContent>
            </Card>

            <Card className="bg-gradient-to-r from-brand-500/10 via-brand-500/5 to-transparent">
              <CardContent className="flex flex-wrap items-center justify-between gap-4 py-5">
                <div>
                  <p className="text-sm uppercase tracking-wide text-brand-600">Chart summary</p>
                  <p className="mt-1 text-2xl font-bold text-slate-900">{data.length} data points</p>
                  <p className="text-sm text-muted-foreground">
                    {isValid ? 'Validated and ready to export.' : 'Validate your data to unlock export options.'}
                  </p>
                </div>
                <div className="rounded-xl border border-brand-200 bg-white/80 px-5 py-4 text-sm">
                  <p className="font-medium text-slate-900">Configuration</p>
                  <p className="mt-1 text-muted-foreground">
                    <span className="font-medium text-slate-900">{chartType.toUpperCase()}</span> · {styleId}
                  </p>
                  <p className="text-muted-foreground">
                    {lastValidated ? `Last validated ${lastValidated.toLocaleTimeString()}` : 'Not yet validated'}
                  </p>
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      </Tabs>
    </div>
  );
}
