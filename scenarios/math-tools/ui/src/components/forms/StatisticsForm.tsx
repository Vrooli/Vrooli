import { useMemo, useState } from 'react'
import { BarChart3, Loader2 } from 'lucide-react'
import toast from 'react-hot-toast'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { parseNumberList, stringify } from '@/lib/utils'
import { ApiSettings, StatisticsResponse, runStatistics } from '@/lib/api'

interface StatisticsFormProps {
  settings: ApiSettings
}

const AVAILABLE_ANALYSES = [
  { value: 'descriptive', label: 'Descriptive Statistics (mean, median, std dev)' },
  { value: 'correlation', label: 'Correlation Matrix' },
  { value: 'regression', label: 'Linear Regression' },
]

export function StatisticsForm({ settings }: StatisticsFormProps) {
  const [dataInput, setDataInput] = useState<string>('1,2,3,4,5,6,7,8,9,10')
  const [selectedAnalyses, setSelectedAnalyses] = useState<string[]>(['descriptive'])
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [response, setResponse] = useState<StatisticsResponse | null>(null)

  const statisticsSummary = useMemo(() => {
    if (!response?.results) {
      return null
    }

    const descriptive = response.results['descriptive'] as Record<string, number> | undefined
    if (!descriptive) {
      return null
    }

    return [
      { label: 'Mean', value: descriptive['mean'] },
      { label: 'Median', value: descriptive['median'] },
      { label: 'Std Dev', value: descriptive['std_dev'] },
      { label: 'Variance', value: descriptive['variance'] },
      { label: 'Min', value: descriptive['min'] },
      { label: 'Max', value: descriptive['max'] },
    ]
  }, [response])

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const data = parseNumberList(dataInput)

    if (data.length < 2) {
      toast.error('Statistics requires at least two values')
      return
    }

    if (selectedAnalyses.length === 0) {
      toast.error('Select at least one analysis to run')
      return
    }

    try {
      setIsSubmitting(true)
      const result = await runStatistics(
        {
          data,
          analyses: selectedAnalyses,
        },
        settings,
      )
      setResponse(result)
      toast.success('Statistical analysis complete')
    } catch (error) {
      console.error(error)
      toast.error(error instanceof Error ? error.message : 'Failed to compute statistics')
    } finally {
      setIsSubmitting(false)
    }
  }

  function toggleAnalysis(value: string) {
    setSelectedAnalyses(prev => (prev.includes(value) ? prev.filter(item => item !== value) : [...prev, value]))
  }

  return (
    <Card className="h-full">
      <CardHeader className="flex flex-col gap-3">
        <Badge variant="secondary" className="w-fit gap-1">
          <BarChart3 className="h-4 w-4" /> Statistical Insights
        </Badge>
        <CardTitle>Run Statistical Analyses</CardTitle>
        <CardDescription>Upload datasets and compute descriptive statistics, correlation, or regression.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid gap-2">
            <Label htmlFor="statistics-data">Dataset</Label>
            <Input
              id="statistics-data"
              value={dataInput}
              onChange={event => setDataInput(event.target.value)}
              placeholder="Numbers separated by comma or space"
              autoComplete="off"
            />
          </div>

          <div className="grid gap-2">
            <Label>Analyses</Label>
            <ScrollArea className="h-[120px] w-full rounded-md border border-slate-800 bg-slate-900/40 p-3">
              <div className="grid gap-3">
                {AVAILABLE_ANALYSES.map(analysis => (
                  <label key={analysis.value} className="flex items-center gap-3 text-sm text-slate-200">
                    <Checkbox
                      checked={selectedAnalyses.includes(analysis.value)}
                      onCheckedChange={() => toggleAnalysis(analysis.value)}
                    />
                    <span>{analysis.label}</span>
                  </label>
                ))}
              </div>
            </ScrollArea>
          </div>

          <Button type="submit" className="w-full" disabled={isSubmitting}>
            {isSubmitting ? (
              <span className="flex items-center gap-2">
                <Loader2 className="h-4 w-4 animate-spin" />
                Analysing…
              </span>
            ) : (
              'Run Analysis'
            )}
          </Button>
        </form>

        {response && (
          <div className="mt-6 space-y-4">
            {statisticsSummary && (
              <div className="grid grid-cols-2 gap-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm md:grid-cols-3">
                {statisticsSummary.map(stat => (
                  <div key={stat.label} className="flex flex-col">
                    <span className="text-xs uppercase tracking-wide text-muted-foreground">{stat.label}</span>
                    <span className="text-base font-semibold text-foreground">
                      {Number.isFinite(stat.value) ? stat.value?.toFixed(3) : '—'}
                    </span>
                  </div>
                ))}
              </div>
            )}

            <div className="space-y-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm">
              <div className="flex items-center justify-between text-xs uppercase tracking-wide text-muted-foreground">
                <span>Raw Response</span>
                <span>{response.data_points} data points</span>
              </div>
              <pre className="max-h-52 overflow-auto rounded-md bg-slate-950/70 p-3 text-xs">
                {stringify(response.results)}
              </pre>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
