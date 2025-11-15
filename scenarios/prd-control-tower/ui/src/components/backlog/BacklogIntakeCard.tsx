import { PlusCircle, Sparkles } from 'lucide-react'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../ui/card'
import { Button } from '../ui/button'
import { Textarea } from '../ui/textarea'
import { Badge } from '../ui/badge'

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
    <Card className="h-full border-2 border-violet-200 bg-gradient-to-br from-white to-violet-50/30">
      <CardHeader className="flex flex-row items-start justify-between gap-4">
        <div className="space-y-2">
          <div className="flex items-center gap-2">
            <CardTitle className="text-2xl font-semibold">💡 Idea Intake</CardTitle>
            <Badge variant="outline" className="gap-1 border-violet-200 bg-violet-50 text-violet-700">
              <Sparkles size={12} />
              Auto-detect
            </Badge>
          </div>
          <CardDescription className="text-base">
            Paste freeform text from your notes. We'll automatically:
          </CardDescription>
          <ul className="space-y-1 text-sm text-muted-foreground">
            <li className="flex items-center gap-2">
              <span className="text-green-600">✓</span> Split by newlines
            </li>
            <li className="flex items-center gap-2">
              <span className="text-green-600">✓</span> Remove bullets and numbering
            </li>
            <li className="flex items-center gap-2">
              <span className="text-green-600">✓</span> Detect if scenario or resource
            </li>
            <li className="flex items-center gap-2">
              <span className="text-green-600">✓</span> Generate clean slugs
            </li>
          </ul>
          <div className="rounded-lg border border-violet-200 bg-white/70 p-3">
            <p className="text-xs font-medium text-violet-900">💡 Pro Tips:</p>
            <ul className="mt-1 space-y-1 text-xs text-violet-700">
              <li>• Prefix with <code className="rounded bg-violet-100 px-1">[resource]</code> to force resource type</li>
              <li>• Include keywords like "manager", "builder", "dashboard" for scenarios</li>
              <li>• Include "integration", "api wrapper", "connector" for resources</li>
            </ul>
          </div>
        </div>
        <span className="rounded-2xl bg-violet-100 p-3 text-violet-600">
          <PlusCircle size={20} />
        </span>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="relative">
          <Textarea
            className="min-h-[280px] font-mono text-sm"
            placeholder={exampleText}
            value={rawInput}
            onChange={(event) => onChange(event.target.value)}
          />
          {ideaCount > 0 && (
            <div className="absolute bottom-3 right-3 rounded-full bg-green-100 px-3 py-1 text-sm font-semibold text-green-700 shadow-sm">
              {ideaCount} idea{ideaCount === 1 ? '' : 's'} ready
            </div>
          )}
        </div>
        <div className="flex items-center justify-between text-sm">
          <div className="flex gap-2">
            {ideaCount === 0 && rawInput.length > 0 && (
              <span className="text-amber-600">⚠ No valid ideas detected</span>
            )}
          </div>
          {rawInput && (
            <Button type="button" variant="link" className="h-auto px-0 text-violet-600" onClick={onClear}>
              Clear input
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
