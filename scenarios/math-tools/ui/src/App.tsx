import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import toast, { Toaster } from 'react-hot-toast'
import { Activity, Calculator, BarChart3, FunctionSquare, Gauge, Infinity, Network, ShieldCheck } from 'lucide-react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  ApiSettings,
  DEFAULT_API_BASE_URL,
  DEFAULT_API_TOKEN,
  HealthResponse,
  buildApiSettings,
  fetchHealth,
} from '@/lib/api'
import { formatTimestamp } from '@/lib/utils'
import { BasicCalculationForm } from '@/components/forms/BasicCalculationForm'
import { MatrixOperationForm } from '@/components/forms/MatrixOperationForm'
import { NumberTheoryForm } from '@/components/forms/NumberTheoryForm'
import { CalculusForm } from '@/components/forms/CalculusForm'
import { StatisticsForm } from '@/components/forms/StatisticsForm'
import { EquationSolverForm } from '@/components/forms/EquationSolverForm'
import { OptimizationForm } from '@/components/forms/OptimizationForm'
import { ForecastForm } from '@/components/forms/ForecastForm'
import { ModelManager } from '@/components/forms/ModelManager'

const SETTINGS_STORAGE_KEY = 'math-tools-ui::settings'

type StoredSettings = {
  baseUrl: string
  token: string
}

function loadPersistedSettings(): ApiSettings {
  if (typeof window === 'undefined') {
    return buildApiSettings()
  }

  try {
    const raw = localStorage.getItem(SETTINGS_STORAGE_KEY)
    if (!raw) {
      return buildApiSettings()
    }
    const parsed = JSON.parse(raw) as StoredSettings
    return buildApiSettings(parsed)
  } catch (error) {
    console.warn('[math-tools-ui] Failed to read stored settings', error)
    return buildApiSettings()
  }
}

function persistSettings(settings: ApiSettings) {
  if (typeof window === 'undefined') {
    return
  }
  try {
    localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(settings))
  } catch (error) {
    console.warn('[math-tools-ui] Unable to persist settings', error)
  }
}

export default function App() {
  const [settings, setSettings] = useState<ApiSettings>(() => loadPersistedSettings())
  const [apiBaseInput, setApiBaseInput] = useState(settings.baseUrl)
  const [apiTokenInput, setApiTokenInput] = useState(settings.token)

  const healthQueryKey = useMemo(() => ['math-tools', 'health', settings.baseUrl, settings.token], [settings])

  const curlCommand = useMemo(
    () =>
      [
        'curl -X POST \\',
        `  -H "Authorization: Bearer ${settings.token}"` + ' \\',
        '  -H "Content-Type: application/json" \\',
        '  -d "{\\"operation\\":\\"add\\",\\"data\\":[2,4,8]}" \\',
        `${settings.baseUrl}/api/v1/math/calculate`,
      ].join('\n'),
    [settings],
  )

  const { data: health, isLoading: isLoadingHealth, isError: isHealthError, refetch: refetchHealth } = useQuery<HealthResponse>({
    queryKey: healthQueryKey,
    queryFn: () => fetchHealth(settings),
    refetchInterval: 60_000,
    refetchOnWindowFocus: false,
  })

  useEffect(() => {
    persistSettings(settings)
  }, [settings])

  function handleApplySettings(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const normalized = buildApiSettings({ baseUrl: apiBaseInput, token: apiTokenInput })
    setSettings(normalized)
    toast.success('Connection settings updated')
    refetchHealth()
  }

  return (
    <div className="min-h-screen bg-gradient-to-b from-slate-950 via-slate-950 to-slate-900 text-slate-100">
      <Toaster position="top-right" toastOptions={{ style: { background: '#0f172a', color: '#f8fafc' } }} />
      <header className="px-6 pb-12 pt-10 md:px-12">
        <div className="mx-auto max-w-6xl">
          <div className="flex flex-col gap-6 md:flex-row md:items-center md:justify-between">
            <div className="space-y-4">
              <Badge variant="secondary" className="w-fit gap-2">
                <Infinity className="h-4 w-4" /> Math Tools 1.0
              </Badge>
              <h1 className="text-4xl font-bold tracking-tight md:text-5xl">Enterprise-Grade Mathematical Intelligence</h1>
              <p className="max-w-2xl text-base text-slate-300 md:text-lg">
                Run advanced calculations, statistics, optimization, and forecasting from a single industrial-strength
                interface. Every operation uses the Math Tools API, so workflows you validate here can ship directly into
                production scenarios.
              </p>
              <div className="flex flex-wrap gap-3 text-sm text-slate-400">
                <span className="inline-flex items-center gap-2 rounded-full bg-slate-900/80 px-3 py-1">
                  <Calculator className="h-4 w-4" /> 40+ Mathematical Operators
                </span>
                <span className="inline-flex items-center gap-2 rounded-full bg-slate-900/80 px-3 py-1">
                  <FunctionSquare className="h-4 w-4" /> Numerical, Linear Algebra & Calculus
                </span>
                <span className="inline-flex items-center gap-2 rounded-full bg-slate-900/80 px-3 py-1">
                  <BarChart3 className="h-4 w-4" /> Time-Series & Regression Analytics
                </span>
              </div>
            </div>
            <Card className="w-full max-w-sm border-slate-800 bg-slate-900/70 shadow-brand">
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-lg">
                  <Network className="h-5 w-5 text-sky-400" /> API Connection
                </CardTitle>
                <CardDescription>Configure how the UI connects to the Math Tools API runtime.</CardDescription>
              </CardHeader>
              <CardContent>
                <form className="space-y-4" onSubmit={handleApplySettings}>
                  <div className="grid gap-2">
                    <Label htmlFor="api-base">API Base URL</Label>
                    <Input
                      id="api-base"
                      value={apiBaseInput}
                      onChange={event => setApiBaseInput(event.target.value)}
                      placeholder={DEFAULT_API_BASE_URL}
                    />
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="api-token">Bearer Token</Label>
                    <Input
                      id="api-token"
                      value={apiTokenInput}
                      onChange={event => setApiTokenInput(event.target.value)}
                      placeholder={DEFAULT_API_TOKEN}
                    />
                  </div>
                  <Button type="submit" className="w-full">
                    Apply Settings
                  </Button>
                </form>
              </CardContent>
            </Card>
          </div>
        </div>
      </header>

      <main className="relative z-10 pb-16">
        <section className="mx-auto max-w-6xl px-6 md:px-12">
          <div className="grid gap-6 md:grid-cols-2">
            <Card className="border-slate-800 bg-slate-900/70">
              <CardHeader className="flex flex-row items-center justify-between">
                <div>
                  <CardTitle className="flex items-center gap-2 text-xl">
                    <ShieldCheck className="h-5 w-5 text-emerald-400" /> Service Health
                  </CardTitle>
                  <CardDescription>Heartbeat and runtime metadata from the Math Tools API.</CardDescription>
                </div>
                <Button variant="secondary" size="sm" onClick={() => refetchHealth()}>
                  Refresh
                </Button>
              </CardHeader>
              <CardContent className="space-y-3 text-sm">
                {isLoadingHealth && <p className="text-muted-foreground">Checking service availability…</p>}
                {isHealthError && <p className="text-rose-400">Unable to reach the API using current credentials.</p>}
                {health && (
                  <div className="grid gap-3">
                    <div className="flex items-center justify-between">
                      <span className="text-muted-foreground">Status</span>
                      <span className="font-semibold capitalize text-emerald-400">{health.status}</span>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-muted-foreground">Version</span>
                      <span className="font-semibold text-foreground">{health.version}</span>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-muted-foreground">Database</span>
                      <span className="font-semibold text-foreground">{health.database ?? '—'}</span>
                    </div>
                    <div className="flex items-center justify-between">
                      <span className="text-muted-foreground">Last Checked</span>
                      <span className="font-semibold text-foreground">{formatTimestamp(health.timestamp * 1000)}</span>
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>

            <Card className="border-slate-800 bg-slate-900/70">
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-xl">
                  <Gauge className="h-5 w-5 text-sky-400" /> Quick Start
                </CardTitle>
                <CardDescription>Essential details to keep your workflow anchored.</CardDescription>
              </CardHeader>
              <CardContent className="space-y-3 text-sm text-slate-300">
                <div>
                  <span className="text-muted-foreground">API Base</span>
                  <p className="font-mono text-sm text-foreground">{settings.baseUrl}</p>
                </div>
                <div>
                  <span className="text-muted-foreground">Token</span>
                  <p className="font-mono text-xs text-foreground">{settings.token}</p>
                </div>
                <div>
                  <span className="text-muted-foreground">CLI Mirror</span>
                  <pre className="mt-1 rounded-md bg-slate-950/70 p-3 text-xs text-emerald-300">{curlCommand}</pre>
                </div>
              </CardContent>
            </Card>
          </div>
        </section>

        <section className="mx-auto mt-12 max-w-6xl px-6 md:px-12">
          <Tabs defaultValue="calculators" className="space-y-8">
            <TabsList>
              <TabsTrigger value="calculators">Core Calculators</TabsTrigger>
              <TabsTrigger value="analysis">Statistics & Forecasting</TabsTrigger>
              <TabsTrigger value="optimization">Optimization & Solver</TabsTrigger>
              <TabsTrigger value="models">Model Library</TabsTrigger>
            </TabsList>

            <TabsContent value="calculators">
              <div className="grid gap-6 lg:grid-cols-2">
                <BasicCalculationForm settings={settings} />
                <MatrixOperationForm settings={settings} />
                <NumberTheoryForm settings={settings} />
                <CalculusForm settings={settings} />
              </div>
            </TabsContent>

            <TabsContent value="analysis">
              <div className="grid gap-6 lg:grid-cols-2">
                <StatisticsForm settings={settings} />
                <ForecastForm settings={settings} />
              </div>
            </TabsContent>

            <TabsContent value="optimization">
              <div className="grid gap-6 lg:grid-cols-2">
                <EquationSolverForm settings={settings} />
                <OptimizationForm settings={settings} />
              </div>
            </TabsContent>

            <TabsContent value="models">
              <ModelManager settings={settings} />
            </TabsContent>
          </Tabs>
        </section>
      </main>

      <footer className="border-t border-slate-900/60 bg-slate-950/80 py-8">
        <div className="mx-auto flex max-w-6xl flex-col gap-3 px-6 text-sm text-slate-500 md:flex-row md:items-center md:justify-between md:px-12">
          <span>© {new Date().getFullYear()} Vrooli · Math Tools Scenario</span>
          <span className="flex items-center gap-2">
            <Activity className="h-4 w-4" /> Powered by Go + React + Shadcn UI
          </span>
        </div>
      </footer>
    </div>
  )
}
