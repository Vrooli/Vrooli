import { useState, useEffect, useMemo, useCallback } from 'react'
import { Link2, AlertTriangle, CheckCircle2, X, Save, RefreshCw, ChevronRight, Code, FileText } from 'lucide-react'
import { Button } from '../ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card'
import { Badge } from '../ui/badge'
import { Input } from '../ui/input'
import { Textarea } from '../ui/textarea'
import { Separator } from '../ui/separator'
import toast from 'react-hot-toast'
import type { RequirementGroup, RequirementRecord } from '../../types'
import { buildApiUrl } from '../../utils/apiClient'
import { IssueCategoryCard } from '../issues'

interface OperationalTarget {
  id: string
  entity_type: string
  entity_name: string
  category: string
  criticality: string
  title: string
  notes: string
  status: string
  path: string
  linked_requirement_ids: string[]
}

interface OperationalTargetsEditorProps {
  draftId: string
  requirements: RequirementGroup[] | null
}

/**
 * OperationalTargetsEditor
 *
 * Allows editing operational targets directly in a draft, including:
 * - Viewing all targets from the Operational Targets section
 * - Editing title, notes, status
 * - Linking targets to requirements from requirements/ folder
 * - Validating that P0/P1 targets have linkages
 * - Saving changes back to draft markdown
 */
export function OperationalTargetsEditor({
  draftId,
  requirements,
}: OperationalTargetsEditorProps) {
  const [targets, setTargets] = useState<OperationalTarget[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editForm, setEditForm] = useState<Partial<OperationalTarget>>({})

  // Flatten requirements for autocomplete
  const flatRequirements = useMemo(() => {
    if (!requirements) return []
    const result: Array<{ id: string; title: string; prd_ref?: string; criticality?: string }> = []
    const walk = (groups: RequirementGroup[]) => {
      for (const group of groups) {
        for (const req of group.requirements) {
          result.push({
            id: req.id,
            title: req.title,
            prd_ref: req.prd_ref,
            criticality: req.criticality,
          })
        }
        if (group.children) {
          walk(group.children)
        }
      }
    }
    walk(requirements)
    return result
  }, [requirements])

  // Build requirement map for tree visualization
  const requirementMap = useMemo(() => {
    if (!requirements) return new Map<string, RequirementRecord>()
    const map = new Map<string, RequirementRecord>()
    const walk = (groups: RequirementGroup[]) => {
      for (const group of groups) {
        for (const req of group.requirements) {
          map.set(req.id, req)
        }
        if (group.children) {
          walk(group.children)
        }
      }
    }
    walk(requirements)
    return map
  }, [requirements])

  const fetchTargets = useCallback(async () => {
    setLoading(true)
    try {
      const response = await fetch(buildApiUrl(`/drafts/${draftId}/targets`))
      if (!response.ok) {
        throw new Error(`Failed to fetch targets: ${response.statusText}`)
      }
      const data = await response.json()
      setTargets(data.targets ?? [])
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to load targets')
    } finally {
      setLoading(false)
    }
  }, [draftId])

  // Load targets from draft
  useEffect(() => {
    fetchTargets()
  }, [fetchTargets])

  const handleSaveTargets = async () => {
    setSaving(true)
    try {
      const payload = {
        targets: targets.map(t => ({
          id: t.id,
          title: t.title,
          notes: t.notes,
          status: t.status,
          linked_requirement_ids: t.linked_requirement_ids,
        })),
      }

      const response = await fetch(buildApiUrl(`/drafts/${draftId}/targets`), {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })

      if (!response.ok) {
        throw new Error(`Failed to save targets: ${response.statusText}`)
      }

      const result = await response.json()
      toast.success(result.message || 'Targets updated successfully')
      setEditingId(null)
      await fetchTargets() // Refresh to get updated IDs/state
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to save targets')
    } finally {
      setSaving(false)
    }
  }

  const handleEdit = (target: OperationalTarget) => {
    setEditingId(target.id)
    setEditForm(target)
  }

  const handleCancelEdit = () => {
    setEditingId(null)
    setEditForm({})
  }

  const handleSaveEdit = () => {
    if (!editingId || !editForm) return
    setTargets(prev =>
      prev.map(t =>
        t.id === editingId
          ? { ...t, ...editForm }
          : t
      )
    )
    setEditingId(null)
    setEditForm({})
  }

  const handleToggleRequirement = (targetId: string, reqId: string) => {
    setTargets(prev =>
      prev.map(t => {
        if (t.id !== targetId) return t
        const linked = t.linked_requirement_ids || []
        const hasReq = linked.includes(reqId)
        return {
          ...t,
          linked_requirement_ids: hasReq
            ? linked.filter(id => id !== reqId)
            : [...linked, reqId],
        }
      })
    )
  }

  const handleToggleStatus = (targetId: string) => {
    setTargets(prev =>
      prev.map(t =>
        t.id === targetId
          ? { ...t, status: t.status === 'complete' ? 'pending' : 'complete' }
          : t
      )
    )
  }

  // Validation: find critical targets without linkages
  const orphanedCritical = targets.filter(
    t => (t.criticality === 'P0' || t.criticality === 'P1') && (!t.linked_requirement_ids || t.linked_requirement_ids.length === 0)
  )

  if (loading) {
    return (
      <Card>
        <CardContent className="py-6 text-center text-muted-foreground">
          <RefreshCw className="inline animate-spin mr-2" size={16} />
          Loading operational targets...
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-4">
      {/* Validation warnings */}
      {orphanedCritical.length > 0 && (
        <IssueCategoryCard
          title={`⚠️ ${orphanedCritical.length} critical target${orphanedCritical.length === 1 ? '' : 's'} without requirements`}
          icon={<AlertTriangle size={16} />}
          tone="warning"
          description="P0 and P1 targets must be linked to requirements from the requirements/ folder before publishing."
          items={orphanedCritical.map((t) => `${t.title} (${t.criticality})`)}
          maxVisible={6}
        />
      )}

      {/* Targets list */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>Operational Targets ({targets.length})</CardTitle>
          <div className="flex gap-2">
            <Button size="sm" variant="outline" onClick={fetchTargets} disabled={loading}>
              <RefreshCw size={14} className="mr-1" />
              Refresh
            </Button>
            <Button size="sm" onClick={handleSaveTargets} disabled={saving || targets.length === 0}>
              <Save size={14} className="mr-1" />
              {saving ? 'Saving...' : 'Save Changes'}
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-3">
          {targets.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-4">
              No operational targets found in the Operational Targets section.
            </p>
          ) : (
            targets.map(target => (
              <Card key={target.id} className="border shadow-sm">
                <CardContent className="pt-4 space-y-3">
                  {/* Title row */}
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex-1">
                      {editingId === target.id ? (
                        <Input
                          value={editForm.title || ''}
                          onChange={e => setEditForm({ ...editForm, title: e.target.value })}
                          className="font-semibold"
                          placeholder="Target title"
                        />
                      ) : (
                        <div className="flex items-center gap-2">
                          <button
                            onClick={() => handleToggleStatus(target.id)}
                            className="flex items-center gap-2"
                          >
                            {target.status === 'complete' ? (
                              <CheckCircle2 size={18} className="text-emerald-600" />
                            ) : (
                              <div className="w-4 h-4 border-2 border-gray-400 rounded" />
                            )}
                          </button>
                          <span className="font-semibold">{target.title}</span>
                        </div>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      <Badge variant={target.criticality === 'P0' ? 'destructive' : target.criticality === 'P1' ? 'warning' : 'default'}>
                        {target.criticality || 'P2'}
                      </Badge>
                      <Badge variant={target.status === 'complete' ? 'success' : 'secondary'}>
                        {target.status}
                      </Badge>
                      {editingId === target.id ? (
                        <>
                          <Button size="sm" variant="ghost" onClick={handleSaveEdit}>
                            <CheckCircle2 size={14} />
                          </Button>
                          <Button size="sm" variant="ghost" onClick={handleCancelEdit}>
                            <X size={14} />
                          </Button>
                        </>
                      ) : (
                        <Button size="sm" variant="ghost" onClick={() => handleEdit(target)}>
                          Edit
                        </Button>
                      )}
                    </div>
                  </div>

                  {/* Notes */}
                  {editingId === target.id ? (
                    <Textarea
                      value={editForm.notes || ''}
                      onChange={e => setEditForm({ ...editForm, notes: e.target.value })}
                      placeholder="Implementation notes (optional)"
                      rows={2}
                    />
                  ) : (
                    target.notes && (
                      <p className="text-sm text-muted-foreground italic">
                        {target.notes}
                      </p>
                    )
                  )}

                  {/* Linked requirements */}
                  <div className="space-y-2">
                    <div className="flex items-center gap-2 text-sm font-medium">
                      <Link2 size={14} />
                      Linked Requirements ({target.linked_requirement_ids?.length || 0})
                    </div>
                    <div className="flex flex-wrap gap-2">
                      {flatRequirements.map(req => {
                        const isLinked = target.linked_requirement_ids?.includes(req.id)
                        return (
                          <button
                            key={req.id}
                            onClick={() => handleToggleRequirement(target.id, req.id)}
                            className={`px-2 py-1 text-xs rounded border transition-colors ${
                              isLinked
                                ? 'bg-blue-100 border-blue-400 text-blue-900'
                                : 'bg-white border-gray-300 text-gray-700 hover:border-gray-400'
                            }`}
                          >
                            {isLinked && <CheckCircle2 size={12} className="inline mr-1" />}
                            {req.id}
                          </button>
                        )
                      })}
                      {flatRequirements.length === 0 && (
                        <span className="text-xs text-muted-foreground">
                          No requirements available in requirements/ folder
                        </span>
                      )}
                    </div>
                  </div>

                  {/* Linkage Tree Visualization */}
                  {target.linked_requirement_ids && target.linked_requirement_ids.length > 0 && (
                    <>
                      <Separator />
                      <div className="space-y-2">
                        <p className="text-sm font-medium text-slate-700">Linkage Tree</p>
                        <div className="rounded-lg border bg-slate-50 p-3 space-y-2">
                          {target.linked_requirement_ids.map(reqId => {
                            const req = requirementMap.get(reqId)
                            if (!req) return null

                            return (
                              <div key={reqId} className="space-y-1">
                                {/* Requirement Node */}
                                <div className="flex items-start gap-2 text-xs">
                                  <ChevronRight size={14} className="mt-0.5 text-blue-600 shrink-0" />
                                  <div className="flex-1 space-y-1">
                                    <div className="flex items-center gap-2">
                                      <Badge variant="secondary" className="text-xs font-mono">{req.id}</Badge>
                                      {req.criticality && (
                                        <Badge variant={req.criticality === 'P0' ? 'destructive' : 'default'} className="text-xs">
                                          {req.criticality}
                                        </Badge>
                                      )}
                                      <Badge variant={req.status === 'complete' ? 'success' : 'warning'} className="text-xs">
                                        {req.status || 'pending'}
                                      </Badge>
                                    </div>
                                    <p className="text-slate-700 font-medium">{req.title}</p>
                                    {req.prd_ref && (
                                      <p className="text-blue-600 font-mono text-xs">PRD: {req.prd_ref}</p>
                                    )}

                                    {/* Validation/Test References */}
                                    {req.validation && req.validation.length > 0 && (
                                      <div className="ml-4 mt-2 space-y-1 border-l-2 border-green-300 pl-2">
                                        <div className="flex items-center gap-1 text-green-700">
                                          <Code size={12} />
                                          <span className="font-medium">Tests ({req.validation.length})</span>
                                        </div>
                                        {req.validation.map((val, idx) => (
                                          <div key={idx} className="text-xs bg-white rounded px-2 py-1 border border-green-200">
                                            <div className="flex items-center gap-2 mb-0.5">
                                              <Badge variant={val.type === 'test' ? 'secondary' : 'outline'} className="text-xs">
                                                {val.type}
                                              </Badge>
                                              <Badge variant={val.status === 'implemented' ? 'success' : 'warning'} className="text-xs">
                                                {val.status}
                                              </Badge>
                                              <span className="text-muted-foreground">{val.phase}</span>
                                            </div>
                                            {val.ref && <p className="font-mono text-blue-600">{val.ref}</p>}
                                            {val.notes && <p className="text-slate-600 mt-1">{val.notes}</p>}
                                          </div>
                                        ))}
                                      </div>
                                    )}

                                    {/* Test Files (alternative to validation) */}
                                    {req.test_files && req.test_files.length > 0 && !req.validation && (
                                      <div className="ml-4 mt-2 space-y-1 border-l-2 border-purple-300 pl-2">
                                        <div className="flex items-center gap-1 text-purple-700">
                                          <FileText size={12} />
                                          <span className="font-medium">Test Files ({req.test_files.length})</span>
                                        </div>
                                        {req.test_files.map((testFile, idx) => (
                                          <div key={idx} className="text-xs bg-white rounded px-2 py-1 border border-purple-200">
                                            <p className="font-mono text-purple-600">{testFile.file_path}</p>
                                            <p className="text-muted-foreground">Lines: {testFile.lines.join(', ')}</p>
                                            {testFile.test_names.length > 0 && (
                                              <p className="text-slate-600">Tests: {testFile.test_names.slice(0, 2).join(', ')}
                                                {testFile.test_names.length > 2 && ` +${testFile.test_names.length - 2} more`}
                                              </p>
                                            )}
                                          </div>
                                        ))}
                                      </div>
                                    )}
                                  </div>
                                </div>
                              </div>
                            )
                          })}
                        </div>
                      </div>
                    </>
                  )}
                </CardContent>
              </Card>
            ))
          )}
        </CardContent>
      </Card>
    </div>
  )
}
