import { useState } from 'react'
import type { ChangeEvent } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api, GraphCollection, GraphSummary, PluginCollection, PluginSummary } from '../api/client'
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
import { Input } from '../components/ui/input.tsx'
import { Select } from '../components/ui/select.tsx'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '../components/ui/table.tsx'
import { Plus, Pencil, Trash2, Filter } from 'lucide-react'

function GraphList() {
  const [filter, setFilter] = useState('')
  const [typeFilter, setTypeFilter] = useState('')

  const { data: graphs, isLoading } = useQuery<GraphCollection>({
    queryKey: ['graphs', typeFilter],
    queryFn: () => api.getGraphs({ type: typeFilter || undefined }),
  })

  const { data: plugins } = useQuery<PluginCollection>({
    queryKey: ['plugins'],
    queryFn: () => api.getPlugins(),
  })

  if (isLoading) {
    return <LoadingSpinner message="Loading graphs..." />
  }

  const filteredGraphs = (graphs?.data ?? []).filter((graph: GraphSummary) => {
    const query = filter.toLowerCase()
    return (
      graph.name.toLowerCase().includes(query) ||
      (graph.description?.toLowerCase() ?? '').includes(query)
    )
  })

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
        <div className="space-y-1">
          <h1 className="text-3xl font-semibold tracking-tight text-foreground">
            Graph workspace
          </h1>
          <p className="text-sm text-muted-foreground md:max-w-2xl">
            Browse, refactor, and deploy every graph model in your portfolio with rich metadata, filters, and lightning-fast previews.
          </p>
        </div>
        <Button className="gap-2" asChild>
          <Link to="/graphs/new">
            <Plus className="h-4 w-4" />
            New graph
          </Link>
        </Button>
      </div>

      <Card className="border border-dashed border-primary/20">
        <CardHeader>
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div>
              <CardTitle>Filters</CardTitle>
              <CardDescription>Focus on the graphs that matter right now.</CardDescription>
            </div>
            <Button variant="ghost" size="sm" className="gap-2">
              <Filter className="h-4 w-4" />
              Save view
            </Button>
          </div>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-3">
          <div className="md:col-span-2">
            <Input
              type="search"
              placeholder="Search by name, tag, or description"
              value={filter}
              onChange={(event: ChangeEvent<HTMLInputElement>) => setFilter(event.target.value)}
            />
          </div>
          <Select
            value={typeFilter}
            onChange={(event: ChangeEvent<HTMLSelectElement>) => setTypeFilter(event.target.value)}
          >
            <option value="">All graph types</option>
            {plugins?.data?.map((plugin: PluginSummary) => (
              <option key={plugin.id} value={plugin.id}>
                {plugin.name}
              </option>
            ))}
          </Select>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div>
              <CardTitle>Graph library</CardTitle>
              <CardDescription>
                {filteredGraphs.length} models in scope. Sort by type, updated time, or ownership.
              </CardDescription>
            </div>
            <Button variant="outline" size="sm" className="gap-2">
              Export CSV
            </Button>
          </div>
        </CardHeader>
        <CardContent className="overflow-x-auto">
          {filteredGraphs.length === 0 ? (
            <div className="flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-border/60 bg-muted/40 p-8 text-center">
              <p className="text-sm font-medium text-muted-foreground">
                No graphs match this filter set yet.
              </p>
              <Button variant="default" className="gap-2">
                <Plus className="h-4 w-4" />
                Create graph
              </Button>
            </div>
          ) : (
            <Table className="min-w-full">
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[220px]">Name</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Updated</TableHead>
                  <TableHead className="w-[160px]">Status</TableHead>
                  <TableHead className="w-[160px] text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredGraphs.map((graph) => (
                  <TableRow key={graph.id} className="bg-card/40 hover:bg-primary/5">
                    <TableCell className="font-medium text-foreground">
                      <Link to={`/graphs/${graph.id}`} className="hover:text-primary">
                        {graph.name}
                      </Link>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline" className="capitalize">
                        {graph.type}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {new Date(graph.updated_at).toLocaleDateString()}
                    </TableCell>
                    <TableCell>
                      <Badge variant="secondary" className="capitalize">
                        {graph.status ?? 'Ready'}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-2">
                        <Button variant="ghost" size="sm" className="gap-2" asChild>
                          <Link to={`/graphs/${graph.id}`}>
                            <Pencil className="h-4 w-4" />
                            Edit
                          </Link>
                        </Button>
                        <Button variant="ghost" size="sm" className="gap-2 text-destructive">
                          <Trash2 className="h-4 w-4" />
                          Delete
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export default GraphList
