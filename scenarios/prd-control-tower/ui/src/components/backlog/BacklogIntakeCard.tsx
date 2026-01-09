import { PlusCircle, Sparkles } from 'lucide-react'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../ui/card'
import { Button } from '../ui/button'
import { Textarea } from '../ui/textarea'
import { Badge } from '../ui/badge'
import { selectors } from '../../consts/selectors'

interface BacklogIntakeCardProps {
  rawInput: string
  ideaCount: number
  onChange: (value: string) => void
  onClear: () => void
}

export function BacklogIntakeCard({ rawInput, ideaCount, onChange, onClear }: BacklogIntakeCardProps) {
  const exampleText = `Examples (paste your notes here):

- Scenario blueprint generator
- API testing automation tool
- [resource] OpenAI integration
- Customer feedback analyzer
1. Portfolio management dashboard
2. Resource usage tracker`

  return (
    <Card data-testid={selectors.backlog.intakeCard} className="h-full border-2 border-violet-200 bg-gradient-to-br from-white to-violet-50/30 shadow-md hover:shadow-lg transition-shadow">
      <CardHeader className="flex flex-row items-start justify-between gap-4 pb-4">
        <div className="space-y-3 flex-1">
          <div className="flex flex-col sm:flex-row sm:items-center gap-2">
            <CardTitle className="text-xl sm:text-2xl font-bold text-slate-900 leading-tight">💡 Idea Intake</CardTitle>
            <Badge variant="outline" className="gap-1.5 border-violet-300 bg-violet-100 text-violet-700 font-semibold shadow-sm w-fit">
              <Sparkles size={13} />
              <span>Auto-detect</span>
            </Badge>
          </div>
          <CardDescription className="text-sm sm:text-base text-slate-600 leading-relaxed">
            Paste freeform text from your notes. We'll automatically detect and structure your ideas.
          </CardDescription>
          <details className="group">
            <summary className="cursor-pointer text-xs font-semibold text-violet-600 hover:text-violet-700 flex items-center gap-1.5 w-fit">
              <span className="group-open:rotate-90 transition-transform">▶</span>
              <span>What happens automatically?</span>
            </summary>
            <ul className="grid grid-cols-1 sm:grid-cols-2 gap-1.5 text-sm text-slate-600 mt-2 pl-3">
              <li className="flex items-center gap-2">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-green-100 text-xs font-bold text-green-700 shrink-0">✓</span>
                <span>Split by newlines</span>
              </li>
              <li className="flex items-center gap-2">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-green-100 text-xs font-bold text-green-700 shrink-0">✓</span>
                <span>Remove bullets & numbering</span>
              </li>
              <li className="flex items-center gap-2">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-green-100 text-xs font-bold text-green-700 shrink-0">✓</span>
                <span>Detect scenario vs resource</span>
              </li>
              <li className="flex items-center gap-2">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-green-100 text-xs font-bold text-green-700 shrink-0">✓</span>
                <span>Generate clean slugs</span>
              </li>
            </ul>
          </details>
          <div className="rounded-xl border-2 border-violet-200 bg-gradient-to-br from-white to-violet-50 p-3.5 shadow-sm">
            <p className="text-xs font-bold text-violet-900 mb-2 flex items-center gap-1.5">
              <span className="text-base">💡</span>
              <span>Pro Tips</span>
            </p>
            <ul className="space-y-1.5 text-xs text-violet-800 leading-relaxed">
              <li className="flex items-start gap-1.5">
                <span className="text-violet-500 mt-0.5">•</span>
                <span>Prefix with <code className="rounded bg-violet-100 px-1.5 py-0.5 font-semibold">[resource]</code> to force resource type</span>
              </li>
              <li className="flex items-start gap-1.5">
                <span className="text-violet-500 mt-0.5">•</span>
                <span>Include keywords like "manager", "builder", "dashboard" for scenarios</span>
              </li>
              <li className="flex items-start gap-1.5">
                <span className="text-violet-500 mt-0.5">•</span>
                <span>Include "integration", "api wrapper", "connector" for resources</span>
              </li>
            </ul>
          </div>
        </div>
        <span className="hidden sm:flex rounded-2xl bg-gradient-to-br from-violet-100 to-violet-200 p-3 text-violet-600 shadow-sm shrink-0">
          <PlusCircle size={22} strokeWidth={2.5} />
        </span>
      </CardHeader>
      <CardContent className="space-y-4 pt-4">
        <div className="relative">
          <Textarea
            data-testid={selectors.backlog.textArea}
            className="min-h-[280px] font-mono text-sm border-2 border-slate-200 focus-visible:border-violet-400 focus-visible:ring-4 focus-visible:ring-violet-100 transition-all shadow-sm"
            placeholder={exampleText}
            value={rawInput}
            onChange={(event) => onChange(event.target.value)}
            aria-label="Paste your ideas here"
          />
          {ideaCount > 0 && (
            <div className="absolute bottom-3 right-3 rounded-full bg-gradient-to-r from-green-100 to-emerald-100 px-3.5 py-1.5 text-sm font-bold text-green-700 shadow-md border border-green-200 animate-in fade-in slide-in-from-bottom-2 duration-300">
              {ideaCount} idea{ideaCount === 1 ? '' : 's'} ready ✓
            </div>
          )}
        </div>
        <div className="flex items-center justify-between gap-3 text-sm">
          <div className="flex gap-2 flex-1">
            {ideaCount === 0 && rawInput.length > 0 && (
              <div className="flex items-center gap-2 text-amber-700 bg-amber-50 px-3 py-1.5 rounded-lg border border-amber-200">
                <span className="text-base">⚠</span>
                <span className="font-medium">No valid ideas detected</span>
              </div>
            )}
          </div>
          {rawInput && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-8 gap-1.5 text-violet-600 hover:bg-violet-50 hover:text-violet-700 font-semibold transition-colors"
              onClick={onClear}
            >
              <span>Clear</span>
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
