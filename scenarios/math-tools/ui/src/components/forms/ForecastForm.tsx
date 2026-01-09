import { useMemo, useState } from 'react'
import { LineChart, Loader2 } from 'lucide-react'
import toast from 'react-hot-toast'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Checkbox } from '@/components/ui/checkbox'
import { Badge } from '@/components/ui/badge'
import { parseNumberList, stringify } from '@/lib/utils'
import { ApiSettings, ForecastResponse, forecast } from '@/lib/api'

interface ForecastFormProps {
  settings: ApiSettings
}

const METHODS = [
  { value: 'linear_trend', label: 'Linear Trend' },
  { value: 'exponential_smoothing', label: 'Exponential Smoothing' },
  { value: 'moving_average', label: 'Moving Average' },
]

export function ForecastForm({ settings }: ForecastFormProps) {
  const [seriesInput, setSeriesInput] = useState<string>('100,102,98,105,110,108,112,115,118,120')
  const [horizon, setHorizon] = useState<string>('5')
  const [method, setMethod] = useState<string>('linear_trend')
  const [includeIntervals, setIncludeIntervals] = useState<boolean>(true)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [response, setResponse] = useState<ForecastResponse | null>(null)

  const summaryMetrics = useMemo(() => {
    if (!response?.model_metrics) {
      return []
    }
    return Object.entries(response.model_metrics).map(([key, value]) => ({
      label: key.replace(/_/g, ' '),
      value,
    }))
  }, [response])

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const data = parseNumberList(seriesInput)

    if (data.length < 3) {
      toast.error('Provide at least three data points for forecasting')
      return
    }

    try {
      setIsSubmitting(true)
      const result = await forecast(
        {
          time_series: data,
          forecast_horizon: Number.parseInt(horizon, 10) || 5,
          method,
          options: {
            confidence_intervals: includeIntervals,
            seasonality: false,
            validation_split: 0.2,
          },
        },
        settings,
      )
      setResponse(result)
      toast.success('Forecast ready')
    } catch (error) {
      console.error(error)
      toast.error(error instanceof Error ? error.message : 'Forecasting failed')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Card className="h-full">
      <CardHeader className="flex flex-col gap-3">
        <Badge variant="secondary" className="w-fit gap-1">
          <LineChart className="h-4 w-4" /> Forecasting
        </Badge>
        <CardTitle>Time-Series Forecasting</CardTitle>
        <CardDescription>Predict future values with configurable statistical forecasting methods.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid gap-2">
            <Label htmlFor="forecast-series">Historical Series</Label>
            <Input
              id="forecast-series"
              value={seriesInput}
              onChange={event => setSeriesInput(event.target.value)}
              placeholder="Comma-separated values"
            />
          </div>

          <div className="grid gap-3 md:grid-cols-3">
            <div className="grid gap-2">
              <Label htmlFor="forecast-horizon">Horizon</Label>
              <Input
                id="forecast-horizon"
                value={horizon}
                onChange={event => setHorizon(event.target.value)}
                placeholder="5"
              />
            </div>
            <div className="grid gap-2 md:col-span-2">
              <Label htmlFor="forecast-method">Method</Label>
              <Select value={method} onValueChange={setMethod}>
                <SelectTrigger id="forecast-method">
                  <SelectValue placeholder="Select method" />
                </SelectTrigger>
                <SelectContent>
                  {METHODS.map(item => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <label className="flex items-center gap-3 text-sm text-slate-200">
            <Checkbox checked={includeIntervals} onCheckedChange={checked => setIncludeIntervals(Boolean(checked))} />
            Include 95% confidence interval bands
          </label>

          <Button type="submit" className="w-full" disabled={isSubmitting}>
            {isSubmitting ? (
              <span className="flex items-center gap-2">
                <Loader2 className="h-4 w-4 animate-spin" />
                Forecasting…
              </span>
            ) : (
              'Generate Forecast'
            )}
          </Button>
        </form>

        {response && (
          <div className="mt-6 space-y-4">
            <div className="grid gap-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm md:grid-cols-2">
              {summaryMetrics.map(metric => (
                <div key={metric.label} className="flex flex-col">
                  <span className="text-xs uppercase tracking-wide text-muted-foreground">{metric.label}</span>
                  <span className="text-base font-semibold text-foreground">{metric.value.toFixed(3)}</span>
                </div>
              ))}
            </div>

            <div className="space-y-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm">
              <div className="text-xs uppercase tracking-wide text-muted-foreground">Forecasted Values</div>
              <pre className="max-h-52 overflow-auto rounded-md bg-slate-950/70 p-3 text-xs">
                {stringify(response.forecast)}
              </pre>
            </div>

            {includeIntervals && response.confidence_intervals && (
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm">
                  <div className="text-xs uppercase tracking-wide text-muted-foreground">Lower Bounds</div>
                  <pre className="max-h-52 overflow-auto rounded-md bg-slate-950/70 p-3 text-xs">
                    {stringify(response.confidence_intervals.lower)}
                  </pre>
                </div>
                <div className="space-y-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm">
                  <div className="text-xs uppercase tracking-wide text-muted-foreground">Upper Bounds</div>
                  <pre className="max-h-52 overflow-auto rounded-md bg-slate-950/70 p-3 text-xs">
                    {stringify(response.confidence_intervals.upper)}
                  </pre>
                </div>
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
