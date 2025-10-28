import { useState } from 'react'
import { Braces, Loader2 } from 'lucide-react'
import toast from 'react-hot-toast'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { stringify } from '@/lib/utils'
import { ApiSettings, SolveResponse, solveEquation } from '@/lib/api'

interface EquationSolverFormProps {
  settings: ApiSettings
}

export function EquationSolverForm({ settings }: EquationSolverFormProps) {
  const [equation, setEquation] = useState<string>('x^2 - 4 = 0')
  const [variables, setVariables] = useState<string>('x')
  const [method, setMethod] = useState<string>('numerical')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [response, setResponse] = useState<SolveResponse | null>(null)

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const parsedVariables = variables
      .split(/[\s,]+/u)
      .map(value => value.trim())
      .filter(Boolean)

    if (parsedVariables.length === 0) {
      toast.error('Provide at least one variable name')
      return
    }

    try {
      setIsSubmitting(true)
      const result = await solveEquation(
        {
          equations: equation,
          variables: parsedVariables,
          method,
          options: {
            tolerance: 1e-6,
            max_iterations: 1000,
          },
        },
        settings,
      )
      setResponse(result)
      toast.success('Equation solved')
    } catch (error) {
      console.error(error)
      toast.error(error instanceof Error ? error.message : 'Equation solving failed')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Card className="h-full">
      <CardHeader className="flex flex-col gap-3">
        <Badge variant="secondary" className="w-fit gap-1">
          <Braces className="h-4 w-4" /> Equation Solver
        </Badge>
        <CardTitle>Non-linear Equation Solver</CardTitle>
        <CardDescription>Use Newton-Raphson and analytical shortcuts to solve equations quickly.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid gap-2">
            <Label htmlFor="equation-expression">Equation</Label>
            <Input
              id="equation-expression"
              value={equation}
              onChange={event => setEquation(event.target.value)}
              placeholder="x^2 - 4 = 0"
              autoComplete="off"
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="equation-variables">Variables</Label>
            <Input
              id="equation-variables"
              value={variables}
              onChange={event => setVariables(event.target.value)}
              placeholder="x, y"
              autoComplete="off"
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="equation-method">Method</Label>
            <Input
              id="equation-method"
              value={method}
              onChange={event => setMethod(event.target.value)}
              placeholder="numerical"
              autoComplete="off"
            />
          </div>

          <Button type="submit" className="w-full" disabled={isSubmitting}>
            {isSubmitting ? (
              <span className="flex items-center gap-2">
                <Loader2 className="h-4 w-4 animate-spin" />
                Solving…
              </span>
            ) : (
              'Solve Equation'
            )}
          </Button>
        </form>

        {response && (
          <div className="mt-6 space-y-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm">
            <div className="grid gap-2 text-xs uppercase tracking-wide text-muted-foreground">
              <div className="flex justify-between">
                <span>Method</span>
                <span>{response.method_used}</span>
              </div>
              <div className="flex justify-between">
                <span>Solution Type</span>
                <span>{response.solution_type}</span>
              </div>
              <div className="flex justify-between">
                <span>Iterations</span>
                <span>{response.convergence_info.iterations}</span>
              </div>
            </div>
            <pre className="max-h-52 overflow-auto rounded-md bg-slate-950/70 p-3 text-xs">
              {stringify(response.solutions)}
            </pre>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
