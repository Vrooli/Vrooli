import { useState } from 'react'
import { Binary, Loader2 } from 'lucide-react'
import toast from 'react-hot-toast'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { parseNumberList, stringify } from '@/lib/utils'
import { ApiSettings, CalculationResponse, performCalculation } from '@/lib/api'

interface NumberTheoryFormProps {
  settings: ApiSettings
}

const NUMBER_THEORY_OPERATIONS = [
  { value: 'prime_factors', label: 'Prime Factorization' },
  { value: 'gcd', label: 'Greatest Common Divisor' },
  { value: 'lcm', label: 'Least Common Multiple' },
]

export function NumberTheoryForm({ settings }: NumberTheoryFormProps) {
  const [operation, setOperation] = useState<string>('prime_factors')
  const [dataInput, setDataInput] = useState<string>('48')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [response, setResponse] = useState<CalculationResponse | null>(null)

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const data = parseNumberList(dataInput)

    if (data.length === 0) {
      toast.error('Provide at least one integer value')
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
      toast.success('Number theory operation complete')
    } catch (error) {
      console.error(error)
      toast.error(error instanceof Error ? error.message : 'Number theory operation failed')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Card className="h-full">
      <CardHeader className="flex flex-col gap-3">
        <Badge variant="secondary" className="w-fit gap-1">
          <Binary className="h-4 w-4" /> Number Theory
        </Badge>
        <CardTitle>Number Theory Toolkit</CardTitle>
        <CardDescription>Run discrete math utilities for primes, divisibility, and multiples.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid gap-2">
            <Label htmlFor="number-theory-operation">Operation</Label>
            <Select value={operation} onValueChange={setOperation}>
              <SelectTrigger id="number-theory-operation">
                <SelectValue placeholder="Choose number theory operation" />
              </SelectTrigger>
              <SelectContent>
                {NUMBER_THEORY_OPERATIONS.map(option => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="number-theory-values">Values</Label>
            <Input
              id="number-theory-values"
              value={dataInput}
              onChange={event => setDataInput(event.target.value)}
              placeholder="Enter integers"
              autoComplete="off"
            />
          </div>

          <Button type="submit" className="w-full" disabled={isSubmitting}>
            {isSubmitting ? (
              <span className="flex items-center gap-2">
                <Loader2 className="h-4 w-4 animate-spin" />
                Running…
              </span>
            ) : (
              'Run Operation'
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
