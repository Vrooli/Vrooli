import { useState } from 'react'
import { FileText, AlertCircle, CheckCircle2, Link2, TestTube, ExternalLink, AlertTriangle, Code, Edit } from 'lucide-react'
import { Link as RouterLink } from 'react-router-dom'
import type { RequirementRecord } from '../../types'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../ui/card'
import { Badge } from '../ui/badge'
import { Button } from '../ui/button'
import { Separator } from '../ui/separator'
import { RequirementEditPanel } from './RequirementEditPanel'

interface RequirementDetailPanelProps {
  requirement: RequirementRecord | null
  entityType?: string
  entityName?: string
  onRequirementUpdate?: (updated: RequirementRecord) => void
}

export function RequirementDetailPanel({ requirement, entityType, entityName, onRequirementUpdate }: RequirementDetailPanelProps) {
  const [isEditing, setIsEditing] = useState(false)
  if (!requirement) {
    return (
      <Card className="border-dashed">
        <CardContent className="flex flex-col items-center justify-center py-16 text-center">
          <FileText size={64} className="mb-4 text-slate-200" />
          <p className="text-muted-foreground">Select a requirement from the tree to view details</p>
        </CardContent>
      </Card>
    )
  }

  const handleSave = (updated: RequirementRecord) => {
    setIsEditing(false)
    onRequirementUpdate?.(updated)
  }

  const handleCancel = () => {
    setIsEditing(false)
  }

  // Show edit panel if editing
  if (isEditing && entityType && entityName) {
    return (
      <RequirementEditPanel
        requirement={requirement}
        entityType={entityType}
        entityName={entityName}
        onSave={handleSave}
        onCancel={handleCancel}
      />
    )
  }

  const statusIcon =
    requirement.status === 'complete' ? (
      <CheckCircle2 size={16} className="text-green-600" />
    ) : (
      <AlertCircle size={16} className="text-amber-600" />
    )

  const criticalityColor = requirement.criticality === 'P0' ? 'destructive' : requirement.criticality === 'P1' ? 'warning' : 'secondary'

  const hasLinkedTargets = requirement.linked_operational_target_ids && requirement.linked_operational_target_ids.length > 0
  const hasTestFiles = requirement.test_files && requirement.test_files.length > 0
  const hasPRDIssue = !!requirement.prd_ref_issue

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between gap-4">
          <div className="flex-1">
            <CardTitle className="text-xl">{requirement.title}</CardTitle>
            <CardDescription className="mt-2 flex items-center gap-2">
              <Badge variant="secondary">{requirement.id}</Badge>
              {requirement.category && <Badge variant="outline">{requirement.category}</Badge>}
            </CardDescription>
          </div>
          <div className="flex flex-col gap-2">
            <Badge variant={requirement.status === 'complete' ? 'success' : 'warning'} className="gap-1">
              {statusIcon}
              {requirement.status || 'pending'}
            </Badge>
            {requirement.criticality && <Badge variant={criticalityColor}>{requirement.criticality}</Badge>}
            {entityType && entityName && (
              <Button variant="outline" size="sm" onClick={() => setIsEditing(true)} className="gap-1">
                <Edit size={14} />
                Edit
              </Button>
            )}
          </div>
        </div>
      </CardHeader>

      <CardContent className="space-y-6">
        {requirement.description && (
          <div>
            <h3 className="text-sm font-semibold text-slate-700 mb-2">Description</h3>
            <p className="text-sm text-slate-600 leading-relaxed">{requirement.description}</p>
          </div>
        )}

        {requirement.prd_ref && (
          <>
            <Separator />
            <div>
              <h3 className="text-sm font-semibold text-slate-700 mb-2 flex items-center gap-2">
                <Link2 size={16} />
                PRD Reference
              </h3>
              <div className="space-y-2">
                <p className="text-sm text-blue-600 font-mono">{requirement.prd_ref}</p>
                {hasPRDIssue && (
                  <div className="rounded-lg border border-amber-200 bg-amber-50 p-3">
                    <div className="flex items-start gap-2">
                      <AlertTriangle size={16} className="mt-0.5 flex-shrink-0 text-amber-600" />
                      <div className="flex-1">
                        <p className="text-sm font-medium text-amber-900 mb-1">PRD Reference Issue</p>
                        <p className="text-sm text-amber-800">{requirement.prd_ref_issue?.message}</p>
                        {requirement.prd_ref_issue?.suggestions && requirement.prd_ref_issue.suggestions.length > 0 && (
                          <div className="mt-2">
                            <p className="text-xs font-medium text-amber-900 mb-1">Suggestions:</p>
                            <ul className="space-y-1">
                              {requirement.prd_ref_issue.suggestions.map((suggestion, idx) => (
                                <li key={idx} className="text-xs text-amber-800 font-mono">• {suggestion}</li>
                              ))}
                            </ul>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </div>
          </>
        )}

        {hasLinkedTargets && (
          <>
            <Separator />
            <div>
              <h3 className="text-sm font-semibold text-slate-700 mb-3 flex items-center gap-2">
                <Link2 size={16} />
                Linked Operational Targets ({requirement.linked_operational_target_ids!.length})
              </h3>
              <div className="space-y-2">
                <div className="flex flex-wrap gap-2">
                  {requirement.linked_operational_target_ids!.map((targetId) => (
                    <Badge key={targetId} variant="outline" className="font-mono text-xs">
                      {targetId}
                    </Badge>
                  ))}
                </div>
                {entityType && entityName && (
                  <Button variant="outline" size="sm" asChild className="w-full">
                    <RouterLink to={`/targets/${entityType}/${entityName}`} className="flex items-center justify-center gap-2">
                      <ExternalLink size={14} />
                      View Operational Targets
                    </RouterLink>
                  </Button>
                )}
              </div>
            </div>
          </>
        )}

        {hasTestFiles && (
          <>
            <Separator />
            <div>
              <h3 className="text-sm font-semibold text-slate-700 mb-3 flex items-center gap-2">
                <Code size={16} />
                Test Files ({requirement.test_files!.length})
              </h3>
              <div className="space-y-2">
                {requirement.test_files!.map((testFile, idx) => (
                  <div key={idx} className="rounded-lg border border-blue-200 bg-blue-50 p-3">
                    <p className="text-sm font-mono text-blue-900 mb-2">{testFile.file_path}</p>
                    <div className="flex items-center gap-4 text-xs text-blue-700">
                      <span>Lines: {testFile.lines.join(', ')}</span>
                      {testFile.test_names.length > 0 && (
                        <span>Tests: {testFile.test_names.length}</span>
                      )}
                    </div>
                    {testFile.test_names.length > 0 && (
                      <div className="mt-2 space-y-1">
                        {testFile.test_names.slice(0, 3).map((name, i) => (
                          <p key={i} className="text-xs font-mono text-blue-800">→ {name}</p>
                        ))}
                        {testFile.test_names.length > 3 && (
                          <p className="text-xs text-blue-600">... and {testFile.test_names.length - 3} more</p>
                        )}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          </>
        )}

        {requirement.validation && requirement.validation.length > 0 && (
          <>
            <Separator />
            <div>
              <h3 className="text-sm font-semibold text-slate-700 mb-3 flex items-center gap-2">
                <TestTube size={16} />
                Test Coverage & Validation ({requirement.validation.length})
              </h3>
              <div className="space-y-3">
                {requirement.validation.map((val, idx) => {
                  const isTestValidation = val.type === 'test'
                  const isImplemented = val.status === 'implemented'

                  return (
                    <div
                      key={idx}
                      className={`rounded-lg border p-3 ${
                        isTestValidation
                          ? isImplemented
                            ? 'border-green-200 bg-green-50'
                            : 'border-amber-200 bg-amber-50'
                          : 'border-slate-200 bg-slate-50'
                      }`}
                    >
                      <div className="flex items-center gap-2 mb-2">
                        {isTestValidation && <TestTube size={14} className="text-green-600" />}
                        <Badge
                          variant={isTestValidation ? 'secondary' : 'outline'}
                          className="text-xs"
                        >
                          {val.type}
                        </Badge>
                        <Badge
                          variant={isImplemented ? 'success' : 'warning'}
                          className="text-xs"
                        >
                          {val.status || 'pending'}
                        </Badge>
                        {val.phase && (
                          <Badge variant="outline" className="text-xs">
                            {val.phase} phase
                          </Badge>
                        )}
                      </div>
                      {val.ref && (
                        <div className="mb-2">
                          <p className="text-xs font-semibold text-slate-600 mb-1">
                            {isTestValidation ? 'Test File:' : 'Reference:'}
                          </p>
                          <p className="text-sm font-mono text-blue-600 break-all">{val.ref}</p>
                        </div>
                      )}
                      {val.notes && (
                        <div>
                          <p className="text-xs font-semibold text-slate-600 mb-1">Notes:</p>
                          <p className="text-xs text-slate-700 leading-relaxed">{val.notes}</p>
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>
              <div className="mt-3 p-3 rounded-lg bg-blue-50 border border-blue-200">
                <p className="text-xs text-blue-800">
                  <strong>💡 Tip:</strong> Test files marked as "implemented" provide automated validation for this requirement.
                  Pending tests indicate coverage gaps that should be addressed.
                </p>
              </div>
            </div>
          </>
        )}

        {requirement.file_path && (
          <>
            <Separator />
            <div>
              <h3 className="text-sm font-semibold text-slate-700 mb-2">File Location</h3>
              <p className="text-xs font-mono text-muted-foreground">{requirement.file_path}</p>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}
