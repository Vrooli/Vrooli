import { useState } from 'react'
import { Target, CheckCircle2, Circle, Link2, FileText, Edit, Trash2 } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import toast from 'react-hot-toast'
import type { OperationalTarget } from '../../types'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../ui/card'
import { Badge } from '../ui/badge'
import { Button } from '../ui/button'
import { Separator } from '../ui/separator'
import { TargetEditPanel } from './TargetEditPanel'
import { buildApiUrl } from '../../utils/apiClient'

interface TargetDetailPanelProps {
  target: OperationalTarget | null
  entityType?: string
  entityName?: string
  onTargetUpdate?: (updated: OperationalTarget) => void
  onTargetDelete?: (id: string) => void
}

export function TargetDetailPanel({ target, entityType, entityName, onTargetUpdate, onTargetDelete }: TargetDetailPanelProps) {
  const [isEditing, setIsEditing] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const navigate = useNavigate()

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

  const handleSave = (updated: OperationalTarget) => {
    setIsEditing(false)
    onTargetUpdate?.(updated)
  }

  const handleCancel = () => {
    setIsEditing(false)
  }

  const handleDelete = async () => {
    if (!entityType || !entityName) return

    const confirmed = window.confirm(`Are you sure you want to delete target "${target.title}"? This action cannot be undone.`)
    if (!confirmed) return

    setIsDeleting(true)
    try {
      const response = await fetch(
        buildApiUrl(`/catalog/${entityType}/${entityName}/targets/${target.id}`),
        { method: 'DELETE' }
      )

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || `Failed to delete target: ${response.statusText}`)
      }

      toast.success('Target deleted successfully')
      onTargetDelete?.(target.id)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to delete target')
    } finally {
      setIsDeleting(false)
    }
  }

  // Show edit panel if editing
  if (isEditing && entityType && entityName) {
    return (
      <TargetEditPanel
        target={target}
        entityType={entityType}
        entityName={entityName}
        onSave={handleSave}
        onCancel={handleCancel}
      />
    )
  }

  const statusIcon = target.status === 'complete' ? <CheckCircle2 size={16} className="text-green-600" /> : <Circle size={16} className="text-slate-400" />

  const criticalityColor = target.criticality === 'P0' ? 'destructive' : target.criticality === 'P1' ? 'warning' : 'secondary'

  const handleNavigateToRequirement = (reqId: string) => {
    if (!entityType || !entityName) return
    navigate(`/scenario/${entityType}/${entityName}?tab=requirements&requirement=${reqId}`)
  }

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
          <div className="flex flex-col gap-2">
            <Badge variant={target.status === 'complete' ? 'success' : 'warning'} className="gap-1">
              {statusIcon}
              {target.status}
            </Badge>
            {entityType && entityName && (
              <>
                <Button variant="outline" size="sm" onClick={() => setIsEditing(true)} className="gap-1">
                  <Edit size={14} />
                  Edit
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleDelete}
                  disabled={isDeleting}
                  className="gap-1 border-red-200 text-red-600 hover:bg-red-50 hover:text-red-700"
                >
                  <Trash2 size={14} />
                  {isDeleting ? 'Deleting...' : 'Delete'}
                </Button>
              </>
            )}
          </div>
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
                  <Button
                    key={reqId}
                    variant="outline"
                    size="sm"
                    className="w-full justify-start h-auto px-3 py-2 font-mono text-sm hover:bg-blue-50"
                    onClick={() => handleNavigateToRequirement(reqId)}
                  >
                    {reqId}
                  </Button>
                ))}
                <p className="text-xs text-muted-foreground">Click requirement IDs to view in Requirements tab</p>
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
