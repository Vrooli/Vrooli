import { useState } from 'react'
import { Save, X, Loader2 } from 'lucide-react'
import type { RequirementRecord } from '../../types'
import { Card, CardHeader, CardTitle, CardContent } from '../ui/card'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { Textarea } from '../ui/textarea'
import { Label } from '../ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select'
import toast from 'react-hot-toast'
import { buildApiUrl } from '../../utils/apiClient'

interface RequirementEditPanelProps {
  requirement: RequirementRecord
  entityType: string
  entityName: string
  onSave: (updated: RequirementRecord) => void
  onCancel: () => void
}

export function RequirementEditPanel({
  requirement,
  entityType,
  entityName,
  onSave,
  onCancel
}: RequirementEditPanelProps) {
  const [saving, setSaving] = useState(false)
  const [formData, setFormData] = useState({
    title: requirement.title,
    description: requirement.description || '',
    status: requirement.status || 'pending',
    criticality: requirement.criticality || 'P1',
    prd_ref: requirement.prd_ref || '',
    category: requirement.category || '',
  })

  const handleSave = async () => {
    setSaving(true)
    try {
      // TODO: Implement actual API endpoint
      // For now, this is a placeholder showing the structure
      const response = await fetch(
        buildApiUrl(`/catalog/${entityType}/${entityName}/requirements/${requirement.id}`),
        {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(formData),
        }
      )

      if (!response.ok) {
        throw new Error(`Failed to save requirement: ${response.statusText}`)
      }

      await response.json()
      toast.success('Requirement updated successfully')
      onSave({ ...requirement, ...formData })
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to save requirement')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">Edit Requirement</CardTitle>
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
          <p className="text-xs text-blue-900 font-medium mb-1">Note: Requirements Editor (Read-Only ID)</p>
          <p className="text-xs font-mono text-blue-800">{requirement.id}</p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="edit-title">Title</Label>
          <Input
            id="edit-title"
            value={formData.title}
            onChange={(e) => setFormData({ ...formData, title: e.target.value })}
            placeholder="Requirement title"
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="edit-status">Status</Label>
            <Select value={formData.status} onValueChange={(value) => setFormData({ ...formData, status: value })}>
              <SelectTrigger id="edit-status">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="pending">Pending</SelectItem>
                <SelectItem value="in_progress">In Progress</SelectItem>
                <SelectItem value="complete">Complete</SelectItem>
                <SelectItem value="blocked">Blocked</SelectItem>
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
          <Label htmlFor="edit-category">Category</Label>
          <Input
            id="edit-category"
            value={formData.category}
            onChange={(e) => setFormData({ ...formData, category: e.target.value })}
            placeholder="e.g., foundation, integration, validation"
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="edit-prd-ref">PRD Reference</Label>
          <Input
            id="edit-prd-ref"
            value={formData.prd_ref}
            onChange={(e) => setFormData({ ...formData, prd_ref: e.target.value })}
            placeholder="Operational Targets > P0 > OT-P0-001"
          />
          <p className="text-xs text-muted-foreground">
            Reference to the PRD section this requirement validates
          </p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="edit-description">Description</Label>
          <Textarea
            id="edit-description"
            value={formData.description}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            placeholder="Detailed description of this requirement"
            rows={4}
          />
        </div>

        <div className="rounded-lg border border-amber-200 bg-amber-50 p-3">
          <p className="text-xs text-amber-900 font-medium mb-1">⚠️ Important</p>
          <p className="text-xs text-amber-800">
            Changes will be saved to the requirements JSON file. Make sure they align with the actual implementation and tests.
          </p>
        </div>
      </CardContent>
    </Card>
  )
}
