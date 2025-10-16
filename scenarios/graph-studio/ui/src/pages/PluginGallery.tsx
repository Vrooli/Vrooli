import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api, PluginCollection, PluginSummary } from '../api/client'
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
  Puzzle,
  Shapes,
  Workflow,
  Network,
  Cpu,
  Sparkles,
  LucideIcon,
} from 'lucide-react'

const iconCatalog: Record<string, LucideIcon> = {
  analytics: Shapes,
  automation: Workflow,
  integrations: Network,
  ai: Sparkles,
  default: Puzzle,
  compute: Cpu,
}

function getIcon(category: string | undefined) {
  if (!category) {
    return iconCatalog.default
  }
  const normalized = category.toLowerCase()
  return iconCatalog[normalized] || iconCatalog.default
}

function PluginGallery() {
  const { data: plugins, isLoading } = useQuery<PluginCollection>({
    queryKey: ['plugins'],
    queryFn: () => api.getPlugins(),
  })

  const pluginsByCategory = useMemo<Record<string, PluginSummary[]>>(() => {
    if (!plugins?.data) return {}
    return plugins.data.reduce((acc: Record<string, PluginSummary[]>, plugin) => {
      const category = plugin.category || 'Uncategorized'
      if (!acc[category]) {
        acc[category] = []
      }
      acc[category].push(plugin)
      return acc
    }, {})
  }, [plugins])

  if (isLoading) {
    return <LoadingSpinner message="Loading plugins..." />
  }

  return (
    <div className="space-y-8">
      <div className="space-y-2">
        <h1 className="text-3xl font-semibold tracking-tight text-foreground">
          Plugin gallery
        </h1>
        <p className="text-sm text-muted-foreground md:max-w-3xl">
          Extend Graph Studio with components for analytics, knowledge graphs, workflows, and bespoke transformations. Each plugin is vetted for security and interoperability.
        </p>
      </div>

      {(Object.entries(pluginsByCategory) as [string, PluginSummary[]][]).map(([category, categoryPlugins]) => {
        const Icon = getIcon(category)
        return (
          <Card key={category} className="border border-border/70 bg-card/80">
            <CardHeader className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
              <div className="flex items-center gap-3">
                <div className="flex h-11 w-11 items-center justify-center rounded-full bg-primary/10 text-primary">
                  <Icon className="h-5 w-5" />
                </div>
                <div>
                  <CardTitle className="capitalize">{category}</CardTitle>
                  <CardDescription>
                    {categoryPlugins.length} plugin{categoryPlugins.length === 1 ? '' : 's'} ready to integrate.
                  </CardDescription>
                </div>
              </div>
              <Button variant="outline" size="sm" className="gap-2">
                Browse documentation
              </Button>
            </CardHeader>
            <Separator className="bg-border/60" />
            <CardContent className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              {categoryPlugins.map((plugin) => (
                <Card
                  key={plugin.id}
                  className="relative overflow-hidden border border-border/60 bg-background/70 shadow-sm transition-shadow hover:shadow-glow"
                >
                  <CardContent className="flex h-full flex-col gap-4 p-5">
                    <div className="flex items-start justify-between gap-3">
                      <div className="space-y-1">
                        <h3 className="text-lg font-semibold text-foreground">{plugin.name}</h3>
                        <p className="text-xs text-muted-foreground">{plugin.description}</p>
                      </div>
                      <Badge variant={plugin.enabled ? 'accent' : 'outline'}>
                        {plugin.enabled ? 'Live' : 'Coming soon'}
                      </Badge>
                    </div>
                    <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
                      <span className="font-medium uppercase tracking-wide">Formats:</span>
                      {plugin.formats?.map((format: string) => (
                        <Badge key={format} variant="secondary" className="uppercase">
                          {format}
                        </Badge>
                      ))}
                    </div>
                    <div className="mt-auto flex items-center justify-between gap-2">
                      <div className="text-xs text-muted-foreground">
                        v{plugin.version || '1.0.0'} • Maintainer: {plugin.owner || 'Graph Studio'}
                      </div>
                      <Button
                        size="sm"
                        variant={plugin.enabled ? 'default' : 'outline'}
                        disabled={!plugin.enabled}
                        className="gap-2"
                      >
                        {plugin.enabled ? 'Create graph' : 'Preview'}
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </CardContent>
          </Card>
        )
      })}
    </div>
  )
}

export default PluginGallery
