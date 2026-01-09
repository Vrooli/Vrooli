import { useMemo, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Brain, Loader2, Plus, Trash2 } from 'lucide-react'
import toast from 'react-hot-toast'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { formatTimestamp, safeJsonParse, stringify } from '@/lib/utils'
import { ApiSettings, ModelSummary, createModel, deleteModel, listModels } from '@/lib/api'

interface ModelManagerProps {
  settings: ApiSettings
}

const MODEL_TYPES = ['linear_regression', 'polynomial', 'optimization', 'differential_equation'] as const

export function ModelManager({ settings }: ModelManagerProps) {
  const queryClient = useQueryClient()
  const queryKey = useMemo(() => ['math-tools', 'models', settings.baseUrl, settings.token], [settings])

  const { data, isLoading, isError, refetch, isFetching } = useQuery<ModelSummary[]>({
    queryKey,
    queryFn: () => listModels(settings),
    refetchOnWindowFocus: false,
  })

  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [name, setName] = useState('Quadratic Curve Fit')
  const [modelType, setModelType] = useState<(typeof MODEL_TYPES)[number]>('linear_regression')
  const [formula, setFormula] = useState('y = ax^2 + bx + c')
  const [parameters, setParameters] = useState('{"a": 1.2, "b": -3.4, "c": 2.2}')
  const [isSubmitting, setIsSubmitting] = useState(false)

  async function handleCreate(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const parsedParameters = parameters.trim() ? safeJsonParse(parameters) : {}

    if (parameters.trim() && parsedParameters === undefined) {
      toast.error('Parameters must be valid JSON')
      return
    }

    try {
      setIsSubmitting(true)
      await createModel(
        {
          name,
          model_type: modelType,
          formula,
          parameters: parsedParameters,
        },
        settings,
      )
      toast.success('Model registered')
      setIsDialogOpen(false)
      setName('Quadratic Curve Fit')
      setFormula('y = ax^2 + bx + c')
      setParameters('{"a": 1.2, "b": -3.4, "c": 2.2}')
      queryClient.invalidateQueries({ queryKey })
    } catch (error) {
      console.error(error)
      toast.error(error instanceof Error ? error.message : 'Failed to create model')
    } finally {
      setIsSubmitting(false)
    }
  }

  async function handleDelete(id: string) {
    try {
      await deleteModel(id, settings)
      toast.success('Model removed')
      refetch()
    } catch (error) {
      console.error(error)
      toast.error(error instanceof Error ? error.message : 'Failed to delete model')
    }
  }

  const models = data ?? []

  return (
    <Card className="h-full">
      <CardHeader className="flex flex-col gap-3">
        <Badge variant="secondary" className="w-fit gap-1">
          <Brain className="h-4 w-4" /> Model Registry
        </Badge>
        <CardTitle>Curated Models Library</CardTitle>
        <CardDescription>Register reusable analytical models that the Math Tools API can execute on demand.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div className="text-sm text-muted-foreground">
            {isFetching ? 'Refreshing models…' : `${models.length} models available`}
          </div>
          <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
            <DialogTrigger asChild>
              <Button size="sm" className="gap-2">
                <Plus className="h-4 w-4" /> New Model
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Add Mathematical Model</DialogTitle>
              </DialogHeader>
              <form onSubmit={handleCreate} className="space-y-4">
                <div className="grid gap-2">
                  <Label htmlFor="model-name">Model Name</Label>
                  <Input
                    id="model-name"
                    value={name}
                    onChange={event => setName(event.target.value)}
                    placeholder="Quadratic Curve Fit"
                    required
                  />
                </div>

                <div className="grid gap-2">
                  <Label htmlFor="model-type">Model Type</Label>
                  <Input
                    id="model-type"
                    value={modelType}
                    onChange={event => setModelType(event.target.value as typeof modelType)}
                    list="model-types"
                    required
                  />
                  <datalist id="model-types">
                    {MODEL_TYPES.map(type => (
                      <option key={type} value={type} />
                    ))}
                  </datalist>
                </div>

                <div className="grid gap-2">
                  <Label htmlFor="model-formula">Formula</Label>
                  <Textarea
                    id="model-formula"
                    value={formula}
                    onChange={event => setFormula(event.target.value)}
                    rows={3}
                    spellCheck={false}
                  />
                </div>

                <div className="grid gap-2">
                  <Label htmlFor="model-parameters">Parameters (JSON)</Label>
                  <Textarea
                    id="model-parameters"
                    value={parameters}
                    onChange={event => setParameters(event.target.value)}
                    rows={4}
                    spellCheck={false}
                  />
                </div>

                <DialogFooter>
                  <Button type="submit" disabled={isSubmitting} className="gap-2">
                    {isSubmitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                    Save Model
                  </Button>
                </DialogFooter>
              </form>
            </DialogContent>
          </Dialog>
        </div>

        <div className="space-y-3">
          {isLoading && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" /> Loading models…
            </div>
          )}
          {isError && <p className="text-sm text-rose-400">Unable to load models from the persistence store.</p>}
          {!isLoading && !isError && models.length === 0 && (
            <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-6 text-center text-sm text-muted-foreground">
              No models saved yet. Create your first analytical model to unlock reusable insights.
            </div>
          )}
          {models.map(model => (
            <div
              key={model.id}
              className="flex flex-col gap-3 rounded-lg border border-slate-800 bg-slate-900/60 p-4 text-sm md:flex-row md:items-center md:justify-between"
            >
              <div className="space-y-1">
                <div className="flex items-center gap-3">
                  <span className="text-base font-semibold text-foreground">{model.name}</span>
                  <Badge variant="outline" className="capitalize">
                    {model.model_type.replace(/_/g, ' ')}
                  </Badge>
                </div>
                <div className="text-xs text-muted-foreground">Created {formatTimestamp(model.created_at)}</div>
              </div>
              <div className="flex items-center gap-2">
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="text-rose-300 hover:text-rose-200"
                  onClick={() => handleDelete(model.id)}
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            </div>
          ))}
        </div>

        {models.length > 0 && (
          <details className="rounded-lg border border-slate-800 bg-slate-900/50">
            <summary className="cursor-pointer select-none px-4 py-3 text-sm font-semibold text-foreground">
              API Payload Reference
            </summary>
            <pre className="max-h-60 overflow-auto bg-slate-950/70 p-4 text-xs text-slate-200">
              {stringify(
                {
                  endpoint: '/api/v1/models',
                  payload: {
                    name: 'Model name',
                    model_type: 'linear_regression',
                    formula: 'y = ax + b',
                    parameters: { a: 1.2, b: -0.4 },
                  },
                },
                '—',
              )}
            </pre>
          </details>
        )}
      </CardContent>
    </Card>
  )
}
