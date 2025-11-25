import { useState } from 'react'
import { Save, X, Loader2 } from 'lucide-react'
import type { OperationalTarget } from '../../types'
import { Card, CardHeader, CardTitle, CardContent } from '../ui/card'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { Textarea } from '../ui/textarea'
import { Label } from '../ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select'
import toast from 'react-hot-toast'
import { buildApiUrl } from '../../utils/apiClient'

interface TargetEditPanelProps {
  target: OperationalTarget
  entityType: string
  entityName: string
  onSave: (updated: OperationalTarget) => void
  onCancel: () => void
}

export function TargetEditPanel({
  target,
  entityType,
  entityName,
  onSave,
  onCancel
}: TargetEditPanelProps) {
  const [saving, setSaving] = useState(false)
  const [formData, setFormData] = useState({
    title: target.title,
    notes: target.notes || '',
    status: target.status || 'pending',
    criticality: target.criticality || 'P1',
    category: target.category || 'Must Have',
  })
  const [requirementIdsInput, setRequirementIdsInput] = useState(
    target.linked_requirement_ids?.join(', ') || ''
  )

  const handleSave = async () => {
    setSaving(true)
    try {
      // Parse comma-separated requirement IDs
      const linkedReqIds = requirementIdsInput
        .split(',')
        .map((id) => id.trim())
        .filter((id) => id.length > 0)

      const payload = {
        ...formData,
        linked_requirement_ids: linkedReqIds.length > 0 ? linkedReqIds : undefined,
      }

      const response = await fetch(
        buildApiUrl(`/catalog/${entityType}/${entityName}/targets/${target.id}`),
        {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload),
        }
      )

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || `Failed to save target: ${response.statusText}`)
      }

      const result = await response.json()
      toast.success(`Target updated successfully. Draft ID: ${result.draft_id}`)
      onSave({ ...target, ...formData, linked_requirement_ids: linkedReqIds })
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to save target')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">Edit Operational Target</CardTitle>
          <div className="flex gap-2">
            <Button
              size="sm"
              variant="outline"
              onClick={onCancel}
              disabled={saving}
            >
              <X size={14} className="mr-1" />
              Cancel
            </Button>
            <Button
              size="sm"
              onClick={handleSave}
              disabled={saving}
            >
              {saving ? (
                <>
                  <Loader2 size={14} className="mr-1 animate-spin" />
                  Saving...
                </>
              ) : (
                <>
                  <Save size={14} className="mr-1" />
                  Save
                </>
              )}
            </Button>
          </div>
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        <div className="rounded-lg border border-blue-200 bg-blue-50 p-3">
          <p className="text-xs text-blue-900 font-medium mb-1">Target ID (Read-Only)</p>
          <p className="text-xs font-mono text-blue-800">{target.id}</p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="edit-title">Title</Label>
          <Input
            id="edit-title"
            value={formData.title}
            onChange={(e) => setFormData({ ...formData, title: e.target.value })}
            placeholder="Target title"
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="edit-category">Category</Label>
            <Select value={formData.category} onValueChange={(value) => setFormData({ ...formData, category: value })}>
              <SelectTrigger id="edit-category">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="Must Have">Must Have (P0)</SelectItem>
                <SelectItem value="Should Have">Should Have (P1)</SelectItem>
                <SelectItem value="Nice to Have">Nice to Have (P2)</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="edit-criticality">Criticality</Label>
            <Select value={formData.criticality} onValueChange={(value) => setFormData({ ...formData, criticality: value })}>
              <SelectTrigger id="edit-criticality">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="P0">P0 - Critical</SelectItem>
                <SelectItem value="P1">P1 - High</SelectItem>
                <SelectItem value="P2">P2 - Medium</SelectItem>
                <SelectItem value="P3">P3 - Low</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="edit-status">Status</Label>
          <Select value={formData.status} onValueChange={(value) => setFormData({ ...formData, status: value as 'pending' | 'complete' })}>
            <SelectTrigger id="edit-status">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="pending">Pending</SelectItem>
              <SelectItem value="complete">Complete</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <Label htmlFor="edit-notes">Notes</Label>
          <Textarea
            id="edit-notes"
            value={formData.notes}
            onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
            placeholder="Detailed notes about this operational target"
            rows={4}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="edit-req-ids">Linked Requirement IDs</Label>
          <Input
            id="edit-req-ids"
            value={requirementIdsInput}
            onChange={(e) => setRequirementIdsInput(e.target.value)}
            placeholder="e.g., REQ-001, REQ-002, REQ-003"
          />
          <p className="text-xs text-muted-foreground">Comma-separated list of requirement IDs</p>
        </div>

        <div className="rounded-lg border border-amber-200 bg-amber-50 p-3">
          <p className="text-xs text-amber-900 font-medium mb-1">⚠️ Important</p>
          <p className="text-xs text-amber-800">
            Changes will create or update a draft PRD. Publish the draft to make the changes permanent.
          </p>
        </div>
      </CardContent>
    </Card>
  )
}
