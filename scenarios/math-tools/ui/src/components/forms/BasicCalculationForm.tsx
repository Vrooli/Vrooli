import { useState } from 'react'
import { Calculator, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { parseNumberList, stringify } from '@/lib/utils'
import { ApiSettings, CalculationResponse, performCalculation } from '@/lib/api'
import toast from 'react-hot-toast'

interface BasicCalculationFormProps {
  settings: ApiSettings
}

const BASIC_OPERATIONS = [
  { value: 'add', label: 'Addition (Σ)' },
  { value: 'subtract', label: 'Subtraction' },
  { value: 'multiply', label: 'Multiplication (Π)' },
  { value: 'divide', label: 'Division' },
  { value: 'power', label: 'Power' },
  { value: 'sqrt', label: 'Square Root' },
  { value: 'log', label: 'Logarithm' },
  { value: 'exp', label: 'Exponential' },
]

export function BasicCalculationForm({ settings }: BasicCalculationFormProps) {
  const [operation, setOperation] = useState<string>('add')
  const [dataInput, setDataInput] = useState<string>('2, 4, 6, 8, 10')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [response, setResponse] = useState<CalculationResponse | null>(null)

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const data = parseNumberList(dataInput)

    if (data.length === 0) {
      toast.error('Please provide at least one numeric value')
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
      toast.success('Calculation complete')
    } catch (error) {
      console.error(error)
      toast.error(error instanceof Error ? error.message : 'Failed to execute calculation')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Card className="h-full">
      <CardHeader className="flex flex-col gap-3">
        <Badge variant="secondary" className="w-fit gap-1">
          <Calculator className="h-4 w-4" /> Basic Arithmetic
        </Badge>
        <CardTitle>Run Quick Calculations</CardTitle>
        <CardDescription>Evaluate core arithmetic operations with arbitrary precision.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid gap-2">
            <Label htmlFor="basic-operation">Operation</Label>
            <Select value={operation} onValueChange={setOperation}>
              <SelectTrigger id="basic-operation">
                <SelectValue placeholder="Choose operation" />
              </SelectTrigger>
              <SelectContent>
                {BASIC_OPERATIONS.map(option => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="basic-values">Values</Label>
            <Input
              id="basic-values"
              value={dataInput}
              onChange={event => setDataInput(event.target.value)}
              placeholder="Comma or space separated numbers"
              autoComplete="off"
            />
          </div>

          <Button type="submit" className="w-full" disabled={isSubmitting}>
            {isSubmitting ? (
              <span className="flex items-center gap-2">
                <Loader2 className="h-4 w-4 animate-spin" />
                Calculating…
              </span>
            ) : (
              'Calculate'
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
