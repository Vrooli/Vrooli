import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api, GraphCollection, GraphSummary, PluginCollection, StatsSnapshot } from '../api/client'
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
import {
  Activity,
  Users2,
  RefreshCw,
  GitBranch,
  Brain,
  Share2,
  Workflow,
  Upload,
  ArrowRight,
} from 'lucide-react'

function Dashboard() {
  const { data: stats, isLoading: statsLoading } = useQuery<StatsSnapshot>({
    queryKey: ['stats'],
    queryFn: () => api.getStats(),
  })

  const { data: recentGraphs, isLoading: graphsLoading } = useQuery<GraphCollection>({
    queryKey: ['recentGraphs'],
    queryFn: () => api.getGraphs({ limit: 6 }),
  })

  const { data: plugins } = useQuery<PluginCollection>({
    queryKey: ['plugins'],
    queryFn: () => api.getPlugins(),
  })

  if (statsLoading || graphsLoading) {
    return <LoadingSpinner message="Loading dashboard..." />
  }

  const statsCards = [
    {
      label: 'Total Graphs',
      value: stats?.totalGraphs ?? 0,
      icon: Activity,
    },
    {
      label: 'Active Plugins',
      value: plugins?.data?.length ?? plugins?.total ?? 0,
      icon: Share2,
    },
    {
      label: 'Conversions Today',
      value: stats?.conversionsToday ?? 0,
      icon: RefreshCw,
    },
    {
      label: 'Active Users',
      value: stats?.activeUsers ?? 1,
      icon: Users2,
    },
  ]

  const actions = [
    {
      title: 'Generate Mind Map',
      description: 'Transform raw ideas into connected structures.',
      icon: Brain,
    },
    {
      title: 'Design BPMN Diagram',
      description: 'Model end-to-end business processes with guardrails.',
      icon: Workflow,
    },
    {
      title: 'Build Network Graph',
      description: 'Reveal relationships from complex datasets.',
      icon: GitBranch,
    },
    {
      title: 'Import Existing Graph',
      description: 'Start from JSON, CSV, or GraphML inputs.',
      icon: Upload,
    },
  ]

  return (
    <div className="space-y-8">
      <div className="space-y-3">
        <div className="inline-flex items-center rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary shadow-sm">
          Intelligent graph orchestration
        </div>
        <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
          <div className="space-y-2">
            <h1 className="text-3xl font-semibold tracking-tight text-foreground md:text-4xl">
              Welcome back to Graph Studio
            </h1>
            <p className="max-w-2xl text-sm text-muted-foreground md:text-base">
              Validate, visualize, and operationalize graph data across formats with guided AI workflows and precision tooling.
            </p>
          </div>
          <Button className="gap-2" asChild>
            <Link to="/graphs">
              Browse graph library
              <ArrowRight className="h-4 w-4" />
            </Link>
          </Button>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        {statsCards.map(({ label, value, icon: Icon }) => (
          <Card key={label} className="group">
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardDescription>{label}</CardDescription>
              <div className="rounded-full bg-primary/10 p-2 text-primary transition-colors group-hover:bg-primary group-hover:text-primary-foreground">
                <Icon className="h-4 w-4" />
              </div>
            </CardHeader>
            <CardContent>
              <p className="text-3xl font-semibold tracking-tight text-foreground">{value}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="grid gap-6 lg:grid-cols-[2fr,1fr]">
        <Card className="border-dashed border-primary/20">
          <CardHeader className="flex flex-row items-center justify-between">
            <div>
              <CardTitle>Recent graphs</CardTitle>
              <CardDescription>Continue iterating on your latest work.</CardDescription>
            </div>
            <Button variant="ghost" className="gap-2" asChild>
              <Link to="/graphs">
                View all
                <ArrowRight className="h-4 w-4" />
              </Link>
            </Button>
          </CardHeader>
          <CardContent className="grid gap-4 md:grid-cols-2">
            {recentGraphs?.data?.length === 0 ? (
              <div className="flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-border/60 bg-muted/40 p-6 text-center">
                <p className="text-sm font-medium text-muted-foreground">
                  No graphs yet — your creative canvas is ready.
                </p>
                <Button className="gap-2" variant="default">
                  Start a new graph
                </Button>
              </div>
            ) : (
              recentGraphs?.data?.map((graph: GraphSummary) => (
                <Link
                  key={graph.id}
                  to={`/graphs/${graph.id}`}
                  className="group rounded-xl border border-border/60 bg-card/80 p-5 shadow-sm transition-all hover:-translate-y-1 hover:border-primary/60 hover:shadow-glow"
                >
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <h3 className="text-lg font-semibold text-foreground">{graph.name}</h3>
                      <p className="text-xs uppercase tracking-wide text-primary/80">{graph.type}</p>
                    </div>
                    <Badge variant="outline" className="bg-primary/5 capitalize">
                      Updated {new Date(graph.updated_at).toLocaleDateString()}
                    </Badge>
                  </div>
                  <div className="mt-4 flex h-32 items-center justify-center rounded-lg border border-dashed border-border/70 bg-gradient-to-br from-primary/5 via-accent/5 to-secondary/20">
                    <svg viewBox="0 0 220 150" className="h-24 w-auto text-primary/80" fill="none">
                      <rect x="18" y="24" width="60" height="40" rx="8" className="fill-primary/10" />
                      <rect x="140" y="24" width="60" height="40" rx="8" className="fill-secondary/40" />
                      <rect x="80" y="84" width="60" height="40" rx="8" className="fill-accent/30" />
                      <path
                        d="M50 64L110 84M170 64L110 84M50 44H170"
                        stroke="currentColor"
                        strokeWidth="3"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        opacity="0.55"
                      />
                    </svg>
                  </div>
                  <div className="mt-4 flex items-center justify-between text-xs text-muted-foreground">
                    <span>Click to open and continue</span>
                    <span className="inline-flex items-center gap-1 text-primary">
                      Resume
                      <ArrowRight className="h-3 w-3" />
                    </span>
                  </div>
                </Link>
              ))
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Quick actions</CardTitle>
            <CardDescription>Bootstrap a new visualization workflow.</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            {actions.map(({ title, description, icon: Icon }) => (
              <button
                key={title}
                type="button"
                className="group flex w-full items-start gap-3 rounded-lg border border-transparent bg-muted/40 px-4 py-3 text-left transition hover:border-primary/40 hover:bg-primary/5"
              >
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary transition-all group-hover:bg-primary group-hover:text-primary-foreground">
                  <Icon className="h-4 w-4" />
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-semibold text-foreground">{title}</p>
                  <p className="text-xs text-muted-foreground">{description}</p>
                </div>
              </button>
            ))}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

export default Dashboard
