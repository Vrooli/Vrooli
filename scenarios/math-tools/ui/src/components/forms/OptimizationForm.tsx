import { useState } from 'react'
import { Goal, Loader2 } from 'lucide-react'
import toast from 'react-hot-toast'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { safeJsonParse, stringify } from '@/lib/utils'
import { ApiSettings, OptimizeResponse, optimize } from '@/lib/api'

interface OptimizationFormProps {
  settings: ApiSettings
}

export function OptimizationForm({ settings }: OptimizationFormProps) {
  const [objectiveFunction, setObjectiveFunction] = useState('x^2 + y^2')
  const [variables, setVariables] = useState('x, y')
  const [optimizationType, setOptimizationType] = useState<'minimize' | 'maximize'>('minimize')
  const [algorithm, setAlgorithm] = useState('gradient_descent')
  const [boundsInput, setBoundsInput] = useState('{"x": [-5, 5], "y": [-5, 5]}')
  const [tolerance, setTolerance] = useState('0.0001')
  const [maxIterations, setMaxIterations] = useState('500')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [response, setResponse] = useState<OptimizeResponse | null>(null)

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const parsedVariables = variables
      .split(/[\s,]+/u)
      .map(value => value.trim())
      .filter(Boolean)

    if (parsedVariables.length === 0) {
      toast.error('Provide at least one variable')
      return
    }

    let parsedBounds: Record<string, [number, number]> | undefined
    if (boundsInput.trim()) {
      parsedBounds = safeJsonParse<Record<string, [number, number]>>(boundsInput)
      if (!parsedBounds) {
        toast.error('Bounds must be valid JSON, e.g. {"x":[-5,5]}')
        return
      }
    }

    try {
      setIsSubmitting(true)
      const result = await optimize(
        {
          objective_function: objectiveFunction,
          variables: parsedVariables,
          optimization_type: optimizationType,
          algorithm,
          options: {
            bounds: parsedBounds,
            tolerance: Number.parseFloat(tolerance) || 1e-6,
            max_iterations: Number.parseInt(maxIterations, 10) || 500,
          },
        },
        settings,
      )
      setResponse(result)
      toast.success('Optimization complete')
    } catch (error) {
      console.error(error)
      toast.error(error instanceof Error ? error.message : 'Optimization failed')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Card className="h-full">
      <CardHeader className="flex flex-col gap-3">
        <Badge variant="secondary" className="w-fit gap-1">
          <Goal className="h-4 w-4" /> Optimization
        </Badge>
        <CardTitle>Optimization Engine</CardTitle>
        <CardDescription>Find optimal solutions with gradient-based search and bounds awareness.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid gap-2">
            <Label htmlFor="objective">Objective Function</Label>
            <Textarea
              id="objective"
              value={objectiveFunction}
              onChange={event => setObjectiveFunction(event.target.value)}
              spellCheck={false}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="variables">Variables</Label>
            <Input
              id="variables"
              value={variables}
              onChange={event => setVariables(event.target.value)}
              placeholder="x, y"
            />
          </div>

          <div className="grid gap-3 md:grid-cols-3">
            <div className="grid gap-2">
              <Label htmlFor="optimization-type">Optimization Type</Label>
              <Select value={optimizationType} onValueChange={value => setOptimizationType(value as 'minimize' | 'maximize')}>
                <SelectTrigger id="optimization-type">
                  <SelectValue placeholder="Select" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="minimize">Minimize</SelectItem>
                  <SelectItem value="maximize">Maximize</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="algorithm">Algorithm</Label>
              <Input
                id="algorithm"
                value={algorithm}
                onChange={event => setAlgorithm(event.target.value)}
                placeholder="gradient_descent"
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="tolerance">Tolerance</Label>
              <Input id="tolerance" value={tolerance} onChange={event => setTolerance(event.target.value)} />
            </div>
          </div>

          <div className="grid gap-2 md:grid-cols-2">
            <div className="grid gap-2">
              <Label htmlFor="max-iterations">Max Iterations</Label>
              <Input
                id="max-iterations"
                value={maxIterations}
                onChange={event => setMaxIterations(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="bounds">Variable Bounds (JSON)</Label>
              <Textarea
                id="bounds"
                value={boundsInput}
                onChange={event => setBoundsInput(event.target.value)}
                spellCheck={false}
              />
            </div>
          </div>

          <Button type="submit" className="w-full" disabled={isSubmitting}>
            {isSubmitting ? (
              <span className="flex items-center gap-2">
                <Loader2 className="h-4 w-4 animate-spin" />
                Optimizing…
              </span>
            ) : (
              'Run Optimization'
            )}
          </Button>
        </form>

        {response && (
          <div className="mt-6 space-y-4">
            <div className="grid gap-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm md:grid-cols-2">
              <div className="flex flex-col">
                <span className="text-xs uppercase tracking-wide text-muted-foreground">Status</span>
                <span className="text-base font-semibold text-foreground">{response.status}</span>
              </div>
              <div className="flex flex-col">
                <span className="text-xs uppercase tracking-wide text-muted-foreground">Iterations</span>
                <span className="text-base font-semibold text-foreground">{response.iterations}</span>
              </div>
              <div className="flex flex-col">
                <span className="text-xs uppercase tracking-wide text-muted-foreground">Optimal Value</span>
                <span className="text-base font-semibold text-emerald-400">{response.optimal_value.toFixed(4)}</span>
              </div>
              <div className="flex flex-col">
                <span className="text-xs uppercase tracking-wide text-muted-foreground">Algorithm</span>
                <span className="text-base font-semibold text-foreground">{response.algorithm_used}</span>
              </div>
            </div>

            <div className="space-y-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm">
              <div className="text-xs uppercase tracking-wide text-muted-foreground">Optimal Solution</div>
              <pre className="max-h-52 overflow-auto rounded-md bg-slate-950/70 p-3 text-xs">
                {stringify(response.optimal_solution)}
              </pre>
            </div>

            {response.sensitivity_analysis && (
              <div className="space-y-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm">
                <div className="text-xs uppercase tracking-wide text-muted-foreground">Sensitivity Analysis</div>
                <pre className="max-h-52 overflow-auto rounded-md bg-slate-950/70 p-3 text-xs">
                  {stringify(response.sensitivity_analysis)}
                </pre>
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
