import { useState, useCallback } from 'react'
import { AlertTriangle, CheckCircle2, RefreshCw, AlertCircle, FileSearch, ExternalLink } from 'lucide-react'
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card'
import { Button } from '../ui/button'
import { Badge } from '../ui/badge'
import { Separator } from '../ui/separator'
import { buildApiUrl } from '../../utils/apiClient'
import { flattenMissingTemplateSections } from '../../utils/prdStructure'
import type { Violation, DraftValidationResult } from '../../types'
import { IssueCategoryCard } from '../issues'

interface PRDValidationPanelProps {
  draftId: string
  /** Operational targets without requirements linkage */
  orphanedTargetsCount?: number
  /** Requirements without operational target linkage */
  unmatchedRequirementsCount?: number
  /** Auto-validation state from useAutoValidation hook */
  validationResult?: DraftValidationResult | null
  validating?: boolean
  error?: string | null
  lastValidatedAt?: Date | null
  /** Manual validation trigger */
  onManualValidate?: () => Promise<void>
  className?: string
}

/**
 * PRD Validation Panel
 *
 * Displays real-time validation results from scenario-auditor integration.
 * Shows structure compliance, missing sections, and linking issues.
 *
 * Can work in two modes:
 * 1. Standalone (manages own validation state)
 * 2. Connected to useAutoValidation hook (receives validation state as props)
 */
export function PRDValidationPanel({
  draftId,
  orphanedTargetsCount = 0,
  unmatchedRequirementsCount = 0,
  validationResult: externalValidationResult,
  validating: externalValidating,
  error: externalError,
  lastValidatedAt: externalLastValidatedAt,
  onManualValidate,
  className,
}: PRDValidationPanelProps) {
  const [internalValidationResult, setInternalValidationResult] = useState<DraftValidationResult | null>(null)
  const [internalValidating, setInternalValidating] = useState(false)
  const [internalError, setInternalError] = useState<string | null>(null)
  const [internalLastValidatedAt, setInternalLastValidatedAt] = useState<Date | null>(null)

  // Use external state if provided, otherwise use internal state
  const validationResult = externalValidationResult !== undefined ? externalValidationResult : internalValidationResult
  const validating = externalValidating !== undefined ? externalValidating : internalValidating
  const error = externalError !== undefined ? externalError : internalError
  const lastValidatedAt = externalLastValidatedAt !== undefined ? externalLastValidatedAt : internalLastValidatedAt

  const handleValidate = useCallback(async () => {
    // If external validation is provided, use that
    if (onManualValidate) {
      await onManualValidate()
      return
    }

    // Otherwise, do internal validation
    setInternalValidating(true)
    setInternalError(null)

    try {
      const response = await fetch(buildApiUrl(`/drafts/${draftId}/validate`), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ use_cache: false }),
      })

      if (!response.ok) {
        throw new Error(`Validation failed: ${response.statusText}`)
      }

      const data = await response.json()
      setInternalValidationResult(data)
      setInternalLastValidatedAt(new Date())
    } catch (err) {
      setInternalError(err instanceof Error ? err.message : 'Validation error')
    } finally {
      setInternalValidating(false)
    }
  }, [draftId, onManualValidate])

  const templateComplianceV2 = validationResult?.template_compliance_v2
  const legacyTemplateCompliance = validationResult?.template_compliance
  const hasTemplateCompliance = Boolean(templateComplianceV2 || legacyTemplateCompliance)
  const completionPercent = templateComplianceV2
    ? Math.round(templateComplianceV2.structure_score)
    : legacyTemplateCompliance
      ? Math.round(legacyTemplateCompliance.compliance_percent)
      : 0
  const isFullyComplete = templateComplianceV2
    ? templateComplianceV2.is_fully_compliant
    : legacyTemplateCompliance?.is_compliant ?? false
  const hasLinkageIssues = orphanedTargetsCount > 0 || unmatchedRequirementsCount > 0
  // Ensure violations is an array (API may return object on errors)
  const violations = Array.isArray(validationResult?.violations)
    ? validationResult.violations
    : []
  const hasViolations = violations.length > 0

  const totalSections = templateComplianceV2
    ? templateComplianceV2.required_sections
    : legacyTemplateCompliance
      ? legacyTemplateCompliance.compliant_sections.length + legacyTemplateCompliance.missing_sections.length
      : 0
  const completedSections = templateComplianceV2
    ? templateComplianceV2.completed_sections
    : legacyTemplateCompliance?.compliant_sections.length ?? 0
  const missingSections = templateComplianceV2
    ? flattenMissingTemplateSections(templateComplianceV2)
    : legacyTemplateCompliance?.missing_sections ?? []
  const extraSections = templateComplianceV2?.unexpected_sections ?? []

  const violationsByLine = violations.reduce((acc, violation) => {
    const line = violation.line ?? 0
    if (!acc[line]) {
      acc[line] = []
    }
    acc[line].push(violation)
    return acc
  }, {} as Record<number, Violation[]>)

  return (
    <Card className={className}>
      <CardHeader className="flex flex-row items-center justify-between pb-3">
        <CardTitle className="text-lg flex items-center gap-2">
          <FileSearch size={18} className="text-violet-600" />
          PRD Structure Validation
        </CardTitle>
        <Button
          size="sm"
          variant="outline"
          onClick={handleValidate}
          disabled={validating}
        >
          {validating ? (
            <>
              <RefreshCw size={14} className="mr-1 animate-spin" />
              Validating...
            </>
          ) : (
            <>
              <RefreshCw size={14} className="mr-1" />
              Validate
            </>
          )}
        </Button>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Completion Summary */}
        <div className="rounded-lg border bg-slate-50 p-4 space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">Template Compliance</span>
            <span className="text-sm font-semibold text-violet-600">{completionPercent}%</span>
          </div>
          <div className="h-2 w-full rounded-full bg-slate-200">
            <div
              className={`h-full rounded-full transition-all ${
                isFullyComplete ? 'bg-green-500' : 'bg-violet-500'
              }`}
              style={{ width: `${completionPercent}%` }}
            />
          </div>
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <span>
              {hasTemplateCompliance ? `${completedSections} / ${totalSections} sections` : 'Click Validate to check'}
            </span>
            {hasTemplateCompliance && (
              isFullyComplete ? (
                <span className="text-green-600 font-medium flex items-center gap-1">
                  <CheckCircle2 size={12} />
                  Complete
                </span>
              ) : (
                <span className="text-amber-600 font-medium flex items-center gap-1">
                  <AlertCircle size={12} />
                  {missingSections.length > 0 && (
                    <>
                      {missingSections.length} missing
                      {extraSections.length > 0 && ' · '}
                    </>
                  )}
                  {extraSections.length > 0 && `${extraSections.length} unexpected`}
                </span>
              )
            )}
          </div>
        </div>

        {/* Missing Template Sections */}
        {missingSections.length > 0 && (
          <IssueCategoryCard
            title="Missing template sections"
            icon={<AlertTriangle size={16} />}
            tone="warning"
            items={missingSections}
            maxVisible={6}
          />
        )}

        {extraSections.length > 0 && (
          <IssueCategoryCard
            title="Unexpected PRD sections"
            icon={<AlertTriangle size={16} />}
            tone="info"
            description="These headings are not part of the canonical template and should be removed or migrated."
            items={extraSections}
            maxVisible={6}
          />
        )}

        {hasLinkageIssues && (
          <IssueCategoryCard
            title="Requirements linkage issues"
            icon={<AlertTriangle size={16} />}
            tone="warning"
            items={[
              orphanedTargetsCount > 0
                ? `${orphanedTargetsCount} operational target${orphanedTargetsCount === 1 ? '' : 's'} without requirements`
                : null,
              unmatchedRequirementsCount > 0
                ? `${unmatchedRequirementsCount} requirement${unmatchedRequirementsCount === 1 ? '' : 's'} missing operational targets`
                : null,
            ].filter(Boolean) as string[]}
          />
        )}

        {/* Validation Results */}
        {error && (
          <div className="rounded-lg border border-red-300 bg-red-50 p-3 text-sm text-red-900">
            <div className="flex items-center gap-2 font-medium mb-1">
              <AlertCircle size={16} />
              Validation Error
            </div>
            <p className="text-xs">{error}</p>
          </div>
        )}

        {validationResult && !hasViolations && (
          <div className="rounded-lg border border-green-300 bg-green-50 p-3 text-sm text-green-900">
            <div className="flex items-center gap-2 font-medium">
              <CheckCircle2 size={16} />
              No structure violations detected
            </div>
            {lastValidatedAt && (
              <p className="text-xs mt-1 text-green-700">
                Last validated: {lastValidatedAt.toLocaleTimeString()}
              </p>
            )}
          </div>
        )}

        {hasViolations && (
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">Structure Violations</span>
              <Badge variant="destructive">
                {validationResult?.summary?.errors ?? violations.length}
              </Badge>
            </div>
            <Separator />
            <div className="space-y-2 max-h-[300px] overflow-y-auto">
              {Object.entries(violationsByLine).map(([lineStr, violations]) => {
                const line = parseInt(lineStr, 10)
                return (
                  <div
                    key={line}
                    className="rounded-md border border-red-200 bg-red-50/50 p-3 space-y-2"
                  >
                    {line > 0 && (
                      <div className="text-xs font-mono text-red-700 flex items-center gap-2">
                        <span className="font-semibold">Line {line}</span>
                        <Badge variant="secondary" className="text-xs">
                          {violations.length} issue{violations.length > 1 ? 's' : ''}
                        </Badge>
                      </div>
                    )}
                    {violations.map((violation, idx) => (
                      <div key={idx} className="space-y-1">
                        <div className="flex items-start gap-2">
                          <Badge
                            variant={
                              violation.severity === 'error'
                                ? 'destructive'
                                : violation.severity === 'warning'
                                ? 'warning'
                                : 'secondary'
        }

                            className="text-xs shrink-0"
                          >
                            {violation.severity ?? 'info'}
                          </Badge>
                          <span className="text-sm text-red-900 flex-1">
                            {violation.message || violation.description || 'Structure violation'}
                          </span>
                        </div>
                        {violation.rule && (
                          <p className="text-xs text-red-700 font-mono pl-[72px]">
                            Rule: {violation.rule}
                          </p>
                        )}
                      </div>
                    ))}
                  </div>
                )
              })}
            </div>
            {lastValidatedAt && (
              <p className="text-xs text-muted-foreground">
                Last validated: {lastValidatedAt.toLocaleTimeString()}
              </p>
            )}
          </div>
        )}

        {/* Help Link */}
        <div className="pt-2 border-t">
          <a
            href="https://github.com/Vrooli/vrooli/blob/main/scripts/scenarios/templates/react-vite/PRD.md"
            target="_blank"
            rel="noopener noreferrer"
            className="text-xs text-primary hover:underline flex items-center gap-1"
          >
            <ExternalLink size={12} />
            View PRD Template Standard
          </a>
        </div>
      </CardContent>
    </Card>
  )
}
