import { CheckCircle2, AlertTriangle, AlertCircle, Info } from 'lucide-react'
import type { StructureSummary } from '../../utils/prdStructure'
import { Badge } from '../ui/badge'

interface StructureChecklistProps {
  summary: StructureSummary
}

export function StructureChecklist({ summary }: StructureChecklistProps) {
  const { missingRequired, missingRecommended, completenessPercent, presentSections } = summary

  const hasAllRequired = missingRequired.length === 0
  const hasAllRecommended = missingRecommended.length === 0
  const isComplete = hasAllRequired && hasAllRecommended

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between text-sm">
        <span className="font-medium text-slate-700">Template compliance</span>
        <div className="flex items-center gap-2">
          <span className={`text-sm font-semibold ${completenessPercent >= 90 ? 'text-emerald-600' : completenessPercent >= 70 ? 'text-amber-600' : 'text-red-600'}`}>
            {completenessPercent}%
          </span>
          <Badge variant={completenessPercent >= 90 ? 'success' : completenessPercent >= 70 ? 'warning' : 'destructive'} className="text-xs">
            {presentSections.length}/{summary.totalRequired}
          </Badge>
        </div>
      </div>

      <div className="space-y-3 text-sm">
        {isComplete ? (
          <p className="flex items-center gap-2 text-emerald-600 font-medium">
            <CheckCircle2 size={16} className="shrink-0" />
            Fully compliant with PRD template
          </p>
        ) : (
          <>
            {/* Required sections */}
            {missingRequired.length > 0 && (
              <div className="border-l-4 border-red-400 pl-3 py-1">
                <p className="flex items-center gap-2 text-red-700 font-medium mb-2">
                  <AlertCircle size={16} className="shrink-0" />
                  Missing {missingRequired.length} required section{missingRequired.length === 1 ? '' : 's'}
                </p>
                <ul className="space-y-1 text-red-800">
                  {missingRequired.map(section => (
                    <li key={section.label} className="flex items-start gap-2">
                      <span className="text-red-400 mt-1">•</span>
                      <span>{section.label}</span>
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {/* Recommended sections */}
            {missingRecommended.length > 0 && (
              <div className="border-l-4 border-amber-400 pl-3 py-1">
                <p className="flex items-center gap-2 text-amber-700 font-medium mb-2">
                  <AlertTriangle size={16} className="shrink-0" />
                  Missing {missingRecommended.length} recommended section{missingRecommended.length === 1 ? '' : 's'}
                </p>
                <ul className="space-y-1 text-amber-800">
                  {missingRecommended.map(section => (
                    <li key={section.label} className="flex items-start gap-2">
                      <span className="text-amber-400 mt-1">•</span>
                      <span>{section.label}</span>
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {hasAllRequired && missingRecommended.length > 0 && (
              <p className="flex items-start gap-2 text-blue-600 text-xs italic">
                <Info size={14} className="shrink-0 mt-0.5" />
                <span>All required sections present. Add recommended sections for full compliance.</span>
              </p>
            )}
          </>
        )}
      </div>
    </div>
  )
}
