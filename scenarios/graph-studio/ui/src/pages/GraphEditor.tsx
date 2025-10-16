import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api, GraphSummary } from '../api/client'
import LoadingSpinner from '../components/LoadingSpinner'
import { Badge } from '../components/ui/badge.tsx'
import { Button } from '../components/ui/button.tsx'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '../components/ui/card.tsx'
import { Separator } from '../components/ui/separator.tsx'
import {
  Save,
  Download,
  ShieldCheck,
  PlayCircle,
  Calendar,
  Tag,
  PackageSearch,
} from 'lucide-react'

function GraphEditor() {
  const { id } = useParams<{ id: string }>()

  const { data: graph, isLoading } = useQuery<GraphSummary>({
    queryKey: ['graph', id],
    queryFn: () => api.getGraph(id!),
    enabled: !!id,
  })

  if (isLoading) {
    return <LoadingSpinner message="Loading graph..." />
  }

  if (!graph) {
    return (
      <Card className="border border-dashed border-destructive/40 bg-destructive/10">
        <CardHeader>
          <CardTitle>Graph not found</CardTitle>
          <CardDescription>
            The requested model is unavailable. It may have been removed or you may not have access yet.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button variant="outline" size="sm" className="gap-2">
            Back to graph library
          </Button>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight text-foreground">
            {graph.name}
          </h1>
          <p className="text-sm text-muted-foreground">
            {graph.summary || 'Deep dive into the live graph structure, tweak nodes, and ship updates with confidence.'}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="ghost" className="gap-2">
            <Save className="h-4 w-4" />
            Save draft
          </Button>
          <Button variant="ghost" className="gap-2">
            <Download className="h-4 w-4" />
            Export
          </Button>
          <Button variant="ghost" className="gap-2">
            <ShieldCheck className="h-4 w-4" />
            Validate
          </Button>
          <Button className="gap-2">
            <PlayCircle className="h-4 w-4" />
            Convert
          </Button>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-[320px,1fr]">
        <Card className="h-fit">
          <CardHeader>
            <CardTitle>Graph details</CardTitle>
            <CardDescription>Metadata and diagnostics at a glance.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4 text-sm text-muted-foreground">
            <div className="space-y-1">
              <p className="text-xs font-semibold uppercase text-primary/80">Type</p>
              <div className="flex items-center gap-2 text-foreground">
                <PackageSearch className="h-4 w-4" />
                {graph.type}
              </div>
            </div>
            <Separator className="bg-border/70" />
            <div className="space-y-1">
              <p className="text-xs font-semibold uppercase text-primary/80">Version</p>
              <span className="text-foreground">{graph.version || '1.0'}</span>
            </div>
            <div className="space-y-1">
              <p className="text-xs font-semibold uppercase text-primary/80">Created</p>
              <div className="flex items-center gap-2">
                <Calendar className="h-4 w-4" />
                {new Date(graph.created_at).toLocaleDateString()}
              </div>
            </div>
            <div className="space-y-2">
              <p className="text-xs font-semibold uppercase text-primary/80">Tags</p>
              <div className="flex flex-wrap gap-2">
                {graph.tags?.length ? (
                  graph.tags.map((tag) => (
                    <Badge key={tag} variant="secondary" className="capitalize">
                      <Tag className="mr-1 h-3 w-3" />
                      {tag}
                    </Badge>
                  ))
                ) : (
                  <Badge variant="outline">No tags yet</Badge>
                )}
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="overflow-hidden">
          <CardHeader>
            <CardTitle>Canvas preview</CardTitle>
            <CardDescription>
              Interact with the live graph canvas, structure nodes, and simulate flows before deploying updates.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="flex min-h-[360px] items-center justify-center rounded-xl border border-dashed border-border/60 bg-gradient-to-br from-primary/5 via-secondary/5 to-accent/10 p-6">
              <svg viewBox="0 0 800 600" className="h-full w-full max-w-3xl text-primary" fill="none">
                <rect x="100" y="100" width="150" height="80" rx="16" className="fill-primary/15" stroke="currentColor" strokeWidth="3" />
                <text x="175" y="145" textAnchor="middle" className="fill-foreground text-base font-semibold">
                  Start
                </text>

                <rect x="550" y="100" width="150" height="80" rx="16" className="fill-secondary/30" stroke="currentColor" strokeWidth="3" />
                <text x="625" y="145" textAnchor="middle" className="fill-foreground text-base font-semibold">
                  Outcome
                </text>

                <rect x="325" y="300" width="150" height="80" rx="16" className="fill-accent/30" stroke="currentColor" strokeWidth="3" />
                <text x="400" y="345" textAnchor="middle" className="fill-foreground text-base font-semibold">
                  Process
                </text>

                <path d="M240 140 L325 340" stroke="currentColor" strokeWidth="3" strokeLinecap="round" />
                <path d="M475 340 L560 140" stroke="currentColor" strokeWidth="3" strokeLinecap="round" />
                <path d="M255 140 H545" stroke="currentColor" strokeWidth="3" strokeLinecap="round" />
              </svg>
            </div>
            <div className="rounded-lg border border-border/60 bg-muted/40 px-4 py-3 text-sm text-muted-foreground">
              Realtime collaborative editing, AI-assisted rule validation, and export pipelines will light up here soon.
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

export default GraphEditor
