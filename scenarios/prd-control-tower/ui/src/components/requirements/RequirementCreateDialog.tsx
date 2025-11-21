import { useState } from 'react'
import { Plus, Loader2 } from 'lucide-react'
import toast from 'react-hot-toast'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { Textarea } from '../ui/textarea'
import { Label } from '../ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '../ui/dialog'
import { buildApiUrl } from '../../utils/apiClient'
import type { RequirementGroup } from '../../types'

interface RequirementCreateDialogProps {
  entityType: string
  entityName: string
  groups: RequirementGroup[]
  onSuccess: () => void
}

export function RequirementCreateDialog({ entityType, entityName, groups, onSuccess }: RequirementCreateDialogProps) {
  const [open, setOpen] = useState(false)
  const [creating, setCreating] = useState(false)
  const [formData, setFormData] = useState({
    group_id: '',
    id: '',
    title: '',
    description: '',
    status: 'pending',
    criticality: 'P1',
    prd_ref: '',
    category: '',
  })

  // Flatten groups for selection
  const flattenGroups = (groups: RequirementGroup[], prefix = ''): Array<{ id: string; name: string }> => {
    const result: Array<{ id: string; name: string }> = []
    for (const group of groups) {
      const displayName = prefix ? `${prefix} > ${group.name}` : group.name
      result.push({ id: group.id, name: displayName })
      if (group.children && group.children.length > 0) {
        result.push(...flattenGroups(group.children, displayName))
      }
    }
    return result
  }

  const allGroups = flattenGroups(groups)

  const handleCreate = async () => {
    if (!formData.group_id || !formData.id || !formData.title) {
      toast.error('Group, ID, and Title are required')
      return
    }

    setCreating(true)
    try {
      const response = await fetch(buildApiUrl(`/catalog/${entityType}/${entityName}/requirements`), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData),
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || `Failed to create requirement: ${response.statusText}`)
      }

      toast.success('Requirement created successfully')
      setOpen(false)
      setFormData({
        group_id: '',
        id: '',
        title: '',
        description: '',
        status: 'pending',
        criticality: 'P1',
        prd_ref: '',
        category: '',
      })
      onSuccess()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to create requirement')
    } finally {
      setCreating(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button className="gap-2">
          <Plus size={16} />
          Add Requirement
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Create New Requirement</DialogTitle>
          <DialogDescription>
            Add a new requirement to the selected group. Make sure the ID is unique.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="create-group">Group *</Label>
            <Select value={formData.group_id} onValueChange={(value) => setFormData({ ...formData, group_id: value })}>
              <SelectTrigger id="create-group">
                <SelectValue>
                  {formData.group_id ? allGroups.find(g => g.id === formData.group_id)?.name : 'Select a group...'}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                {allGroups.map((group) => (
                  <SelectItem key={group.id} value={group.id}>
                    {group.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="create-id">Requirement ID *</Label>
            <Input
              id="create-id"
              value={formData.id}
              onChange={(e) => setFormData({ ...formData, id: e.target.value })}
              placeholder="e.g., REQ-001"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="create-title">Title *</Label>
            <Input
              id="create-title"
              value={formData.title}
              onChange={(e) => setFormData({ ...formData, title: e.target.value })}
              placeholder="Brief description of the requirement"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="create-status">Status</Label>
              <Select value={formData.status} onValueChange={(value) => setFormData({ ...formData, status: value })}>
                <SelectTrigger id="create-status">
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
              <Label htmlFor="create-criticality">Criticality</Label>
              <Select
                value={formData.criticality}
                onValueChange={(value) => setFormData({ ...formData, criticality: value })}
              >
                <SelectTrigger id="create-criticality">
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
            <Label htmlFor="create-category">Category</Label>
            <Input
              id="create-category"
              value={formData.category}
              onChange={(e) => setFormData({ ...formData, category: e.target.value })}
              placeholder="e.g., foundation, integration, validation"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="create-prd-ref">PRD Reference</Label>
            <Input
              id="create-prd-ref"
              value={formData.prd_ref}
              onChange={(e) => setFormData({ ...formData, prd_ref: e.target.value })}
              placeholder="Operational Targets > P0 > OT-P0-001"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="create-description">Description</Label>
            <Textarea
              id="create-description"
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              placeholder="Detailed description of this requirement"
              rows={4}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)} disabled={creating}>
            Cancel
          </Button>
          <Button onClick={handleCreate} disabled={creating}>
            {creating ? (
              <>
                <Loader2 size={16} className="mr-2 animate-spin" />
                Creating...
              </>
            ) : (
              <>
                <Plus size={16} className="mr-2" />
                Create Requirement
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
