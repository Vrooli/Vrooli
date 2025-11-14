import { CheckCircle2, AlertTriangle } from 'lucide-react'
import type { StructureSummary } from '../../utils/prdStructure'

interface StructureChecklistProps {
  summary: StructureSummary
}

export function StructureChecklist({ summary }: StructureChecklistProps) {
  const { missingSections, presentSections, totalRequired } = summary
  const completeness = Math.round(((presentSections.length || 0) / totalRequired) * 100)

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between text-sm">
        <span className="font-medium text-slate-700">Template coverage</span>
        <span className="text-muted-foreground">{completeness}%</span>
      </div>
      <div className="space-y-2 text-sm">
        {missingSections.length === 0 ? (
          <p className="flex items-center gap-2 text-emerald-600">
            <CheckCircle2 size={16} /> Matches the recommended template structure.
          </p>
        ) : (
          <div>
            <p className="flex items-center gap-2 text-amber-700">
              <AlertTriangle size={16} /> Missing {missingSections.length} required section
              {missingSections.length === 1 ? '' : 's'}:
            </p>
            <ul className="mt-2 list-disc pl-5 text-amber-800">
              {missingSections.map(section => (
                <li key={section}>{section}</li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </div>
  )
}
