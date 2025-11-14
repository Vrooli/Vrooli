import { Target, CheckCircle2, Circle, Link2, FileText } from 'lucide-react'
import type { OperationalTarget } from '../../types'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../ui/card'
import { Badge } from '../ui/badge'
import { Separator } from '../ui/separator'

interface TargetDetailPanelProps {
  target: OperationalTarget | null
}

export function TargetDetailPanel({ target }: TargetDetailPanelProps) {
  if (!target) {
    return (
      <Card className="border-dashed">
        <CardContent className="flex flex-col items-center justify-center py-16 text-center">
          <Target size={64} className="mb-4 text-slate-200" />
          <p className="text-muted-foreground">Select an operational target to view details</p>
        </CardContent>
      </Card>
    )
  }

  const statusIcon = target.status === 'complete' ? <CheckCircle2 size={16} className="text-green-600" /> : <Circle size={16} className="text-slate-400" />

  const criticalityColor = target.criticality === 'P0' ? 'destructive' : target.criticality === 'P1' ? 'warning' : 'secondary'

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between gap-4">
          <div className="flex-1">
            <CardTitle className="text-xl">{target.title}</CardTitle>
            <CardDescription className="mt-2 flex items-center gap-2">
              <Badge variant="secondary">{target.category}</Badge>
              {target.criticality && <Badge variant={criticalityColor}>{target.criticality}</Badge>}
            </CardDescription>
          </div>
          <Badge variant={target.status === 'complete' ? 'success' : 'warning'} className="gap-1">
            {statusIcon}
            {target.status}
          </Badge>
        </div>
      </CardHeader>

      <CardContent className="space-y-6">
        {target.notes && (
          <div>
            <h3 className="text-sm font-semibold text-slate-700 mb-2">Notes</h3>
            <p className="text-sm text-slate-600 leading-relaxed">{target.notes}</p>
          </div>
        )}

        <div>
          <h3 className="text-sm font-semibold text-slate-700 mb-2">Target ID</h3>
          <p className="text-xs font-mono text-muted-foreground">{target.id}</p>
        </div>

        {target.path && (
          <>
            <Separator />
            <div>
              <h3 className="text-sm font-semibold text-slate-700 mb-2 flex items-center gap-2">
                <FileText size={16} />
                PRD Path
              </h3>
              <p className="text-sm text-blue-600">{target.path}</p>
            </div>
          </>
        )}

        {target.linked_requirement_ids && target.linked_requirement_ids.length > 0 && (
          <>
            <Separator />
            <div>
              <h3 className="text-sm font-semibold text-slate-700 mb-3 flex items-center gap-2">
                <Link2 size={16} />
                Linked Requirements ({target.linked_requirement_ids.length})
              </h3>
              <div className="space-y-2">
                {target.linked_requirement_ids.map((reqId) => (
                  <div key={reqId} className="rounded-lg border bg-slate-50 p-3">
                    <p className="text-sm font-mono text-blue-600">{reqId}</p>
                  </div>
                ))}
              </div>
            </div>
          </>
        )}

        {(!target.linked_requirement_ids || target.linked_requirement_ids.length === 0) && (
          <>
            <Separator />
            <div className="rounded-lg border-2 border-dashed border-amber-300 bg-amber-50 p-4">
              <p className="text-sm font-medium text-amber-900">⚠️ No Linked Requirements</p>
              <p className="text-xs text-amber-700 mt-1">
                This operational target has no requirements linked to it. Consider adding requirements or updating prd_ref fields in the
                requirements registry.
              </p>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}
