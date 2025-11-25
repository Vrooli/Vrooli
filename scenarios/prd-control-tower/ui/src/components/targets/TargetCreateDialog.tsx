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

interface TargetCreateDialogProps {
  entityType: string
  entityName: string
  onSuccess: () => void
}

export function TargetCreateDialog({ entityType, entityName, onSuccess }: TargetCreateDialogProps) {
  const [open, setOpen] = useState(false)
  const [creating, setCreating] = useState(false)
  const [formData, setFormData] = useState({
    title: '',
    notes: '',
    status: 'pending',
    criticality: 'P1',
    category: 'Must Have',
    linked_requirement_ids: [] as string[],
  })
  const [requirementIdsInput, setRequirementIdsInput] = useState('')

  const handleCreate = async () => {
    if (!formData.title) {
      toast.error('Title is required')
      return
    }

    setCreating(true)
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

      const response = await fetch(buildApiUrl(`/catalog/${entityType}/${entityName}/targets`), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || `Failed to create target: ${response.statusText}`)
      }

      const result = await response.json()
      toast.success(`Target created successfully. Draft ID: ${result.draft_id}`)
      setOpen(false)
      setFormData({
        title: '',
        notes: '',
        status: 'pending',
        criticality: 'P1',
        category: 'Must Have',
        linked_requirement_ids: [],
      })
      setRequirementIdsInput('')
      onSuccess()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to create target')
    } finally {
      setCreating(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button className="gap-2">
          <Plus size={16} />
          Add Target
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Create New Operational Target</DialogTitle>
          <DialogDescription>
            Add a new operational target. This will create a draft that you can publish later.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="create-title">Title *</Label>
            <Input
              id="create-title"
              value={formData.title}
              onChange={(e) => setFormData({ ...formData, title: e.target.value })}
              placeholder="Brief description of the operational target"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="create-category">Category</Label>
              <Select value={formData.category} onValueChange={(value) => setFormData({ ...formData, category: value })}>
                <SelectTrigger id="create-category">
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
            <Label htmlFor="create-status">Status</Label>
            <Select value={formData.status} onValueChange={(value) => setFormData({ ...formData, status: value as 'pending' | 'complete' })}>
              <SelectTrigger id="create-status">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="pending">Pending</SelectItem>
                <SelectItem value="complete">Complete</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="create-notes">Notes</Label>
            <Textarea
              id="create-notes"
              value={formData.notes}
              onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
              placeholder="Detailed notes about this operational target"
              rows={4}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="create-req-ids">Linked Requirement IDs</Label>
            <Input
              id="create-req-ids"
              value={requirementIdsInput}
              onChange={(e) => setRequirementIdsInput(e.target.value)}
              placeholder="e.g., REQ-001, REQ-002, REQ-003"
            />
            <p className="text-xs text-muted-foreground">Comma-separated list of requirement IDs to link to this target</p>
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
                Create Target
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
