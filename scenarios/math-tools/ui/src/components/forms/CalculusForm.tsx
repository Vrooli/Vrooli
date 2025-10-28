import { useState } from 'react'
import { Sigma, Loader2 } from 'lucide-react'
import toast from 'react-hot-toast'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { parseNumberList, stringify } from '@/lib/utils'
import { ApiSettings, CalculationResponse, performCalculation } from '@/lib/api'

interface CalculusFormProps {
  settings: ApiSettings
}

const CALCULUS_OPERATIONS = [
  { value: 'derivative', label: 'Derivative (f′)' },
  { value: 'integral', label: 'Definite Integral' },
  { value: 'partial_derivative', label: 'Partial Derivative' },
  { value: 'double_integral', label: 'Double Integral' },
]

const OPERATION_HELP = {
  derivative: 'Provide a single value for the evaluation point (e.g. x = 3).',
  integral: 'Provide lower and upper bounds (e.g. 0, 5).',
  partial_derivative: 'Provide variable value (e.g. x, y). Currently uses sample function f(x, y) = x² + y².',
  double_integral: 'Provide lower/upper bounds for x and y (e.g. 0, 1, 0, 2).',
} as const

export function CalculusForm({ settings }: CalculusFormProps) {
  const [operation, setOperation] = useState<string>('derivative')
  const [dataInput, setDataInput] = useState<string>('3')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [response, setResponse] = useState<CalculationResponse | null>(null)

  function getPlaceholder() {
    switch (operation) {
      case 'integral':
        return 'Lower, Upper bounds (e.g. 0, 5)'
      case 'partial_derivative':
        return 'Variable values (e.g. 1.5, 2.2)'
      case 'double_integral':
        return 'x₀, x₁, y₀, y₁ (e.g. 0, 1, 0, 2)'
      default:
        return 'Evaluation point (e.g. 3)'
    }
  }

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const data = parseNumberList(dataInput)

    if (data.length === 0) {
      toast.error('Provide at least one numeric value')
      return
    }

    try {
      setIsSubmitting(true)
      const result = await performCalculation(
        {
          operation,
          data,
        },
        settings,
      )
      setResponse(result)
      toast.success('Calculus operation complete')
    } catch (error) {
      console.error(error)
      toast.error(error instanceof Error ? error.message : 'Calculus operation failed')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Card className="h-full">
      <CardHeader className="flex flex-col gap-3">
        <Badge variant="secondary" className="w-fit gap-1">
          <Sigma className="h-4 w-4" /> Calculus
        </Badge>
        <CardTitle>Calculus Playground</CardTitle>
        <CardDescription>{OPERATION_HELP[operation as keyof typeof OPERATION_HELP]}</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid gap-2">
            <Label htmlFor="calculus-operation">Operation</Label>
            <Select
              value={operation}
              onValueChange={value => {
                setOperation(value)
                setResponse(null)
                switch (value) {
                  case 'integral':
                    setDataInput('0, 5')
                    break
                  case 'partial_derivative':
                    setDataInput('1.5, 2.2')
                    break
                  case 'double_integral':
                    setDataInput('0, 1, 0, 2')
                    break
                  default:
                    setDataInput('3')
                }
              }}
            >
              <SelectTrigger id="calculus-operation">
                <SelectValue placeholder="Choose calculus operation" />
              </SelectTrigger>
              <SelectContent>
                {CALCULUS_OPERATIONS.map(option => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="calculus-data">Inputs</Label>
            <Input
              id="calculus-data"
              value={dataInput}
              onChange={event => setDataInput(event.target.value)}
              placeholder={getPlaceholder()}
              autoComplete="off"
            />
          </div>

          <Button type="submit" className="w-full" disabled={isSubmitting}>
            {isSubmitting ? (
              <span className="flex items-center gap-2">
                <Loader2 className="h-4 w-4 animate-spin" />
                Computing…
              </span>
            ) : (
              'Compute'
            )}
          </Button>
        </form>

        {response && (
          <div className="mt-6 space-y-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm">
            <div className="flex items-center justify-between text-xs uppercase tracking-wide text-muted-foreground">
              <span>{response.operation}</span>
              <span>{response.execution_time_ms} ms</span>
            </div>
            <pre className="max-h-52 overflow-auto rounded-md bg-slate-950/70 p-3 text-xs">
              {stringify(response.result)}
            </pre>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
