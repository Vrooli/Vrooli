import { useState } from 'react'
import { Grid3x3, Loader2 } from 'lucide-react'
import toast from 'react-hot-toast'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { safeJsonParse, stringify } from '@/lib/utils'
import { ApiSettings, CalculationResponse, performCalculation } from '@/lib/api'

interface MatrixOperationFormProps {
  settings: ApiSettings
}

const MATRIX_OPERATIONS = [
  { value: 'matrix_multiply', label: 'Matrix Multiplication' },
  { value: 'matrix_inverse', label: 'Matrix Inverse' },
  { value: 'matrix_determinant', label: 'Determinant' },
  { value: 'matrix_transpose', label: 'Transpose' },
]

const DEFAULT_MATRIX_A = `[[1, 2],\n [3, 4]]`
const DEFAULT_MATRIX_B = `[[5, 6],\n [7, 8]]`

export function MatrixOperationForm({ settings }: MatrixOperationFormProps) {
  const [operation, setOperation] = useState<string>('matrix_multiply')
  const [matrixAInput, setMatrixAInput] = useState<string>(DEFAULT_MATRIX_A)
  const [matrixBInput, setMatrixBInput] = useState<string>(DEFAULT_MATRIX_B)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [response, setResponse] = useState<CalculationResponse | null>(null)

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const matrixA = safeJsonParse<number[][]>(matrixAInput)
    if (!matrixA) {
      toast.error('Matrix A must be valid JSON (e.g. [[1,2],[3,4]])')
      return
    }

    const requiresMatrixB = operation === 'matrix_multiply'
    const matrixB = requiresMatrixB ? safeJsonParse<number[][]>(matrixBInput) : undefined

    if (requiresMatrixB && !matrixB) {
      toast.error('Matrix B is required for multiplication and must be valid JSON')
      return
    }

    try {
      setIsSubmitting(true)
      const result = await performCalculation(
        {
          operation,
          matrix_a: matrixA,
          ...(matrixB ? { matrix_b: matrixB } : {}),
        },
        settings,
      )
      setResponse(result)
      toast.success('Matrix operation completed')
    } catch (error) {
      console.error(error)
      toast.error(error instanceof Error ? error.message : 'Matrix operation failed')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Card className="h-full">
      <CardHeader className="flex flex-col gap-3">
        <Badge variant="secondary" className="w-fit gap-1">
          <Grid3x3 className="h-4 w-4" /> Linear Algebra
        </Badge>
        <CardTitle>Matrix Operations</CardTitle>
        <CardDescription>Work with matrix math for transformations and systems of equations.</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid gap-2">
            <Label htmlFor="matrix-operation">Operation</Label>
            <Select value={operation} onValueChange={setOperation}>
              <SelectTrigger id="matrix-operation">
                <SelectValue placeholder="Choose matrix operation" />
              </SelectTrigger>
              <SelectContent>
                {MATRIX_OPERATIONS.map(option => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="matrix-a">Matrix A</Label>
            <Textarea
              id="matrix-a"
              value={matrixAInput}
              onChange={event => setMatrixAInput(event.target.value)}
              spellCheck={false}
            />
          </div>

          {operation === 'matrix_multiply' && (
            <div className="grid gap-2">
              <Label htmlFor="matrix-b">Matrix B</Label>
              <Textarea
                id="matrix-b"
                value={matrixBInput}
                onChange={event => setMatrixBInput(event.target.value)}
                spellCheck={false}
              />
            </div>
          )}

          <Button type="submit" className="w-full" disabled={isSubmitting}>
            {isSubmitting ? (
              <span className="flex items-center gap-2">
                <Loader2 className="h-4 w-4 animate-spin" />
                Solving…
              </span>
            ) : (
              'Run Matrix Operation'
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
