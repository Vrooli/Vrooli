import { PlusCircle } from 'lucide-react'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '../ui/card'
import { Button } from '../ui/button'
import { Textarea } from '../ui/textarea'

interface BacklogIntakeCardProps {
  rawInput: string
  ideaCount: number
  onChange: (value: string) => void
  onClear: () => void
}

export function BacklogIntakeCard({ rawInput, ideaCount, onChange, onClear }: BacklogIntakeCardProps) {
  return (
    <Card className="h-full">
      <CardHeader className="flex flex-row items-start justify-between gap-4">
        <div className="space-y-1">
          <CardTitle className="text-2xl font-semibold">Freeform intake</CardTitle>
          <CardDescription>Drop in any list of scenario or resource ideas. We'll auto-split, clean bullets, and prep slugs.</CardDescription>
        </div>
        <span className="rounded-2xl bg-violet-100 p-3 text-violet-600">
          <PlusCircle size={20} />
        </span>
      </CardHeader>
      <CardContent className="space-y-4">
        <Textarea
          className="min-h-[280px]"
          placeholder={['Example:', '- Scenario blueprint generator', '- [resource] Shared vector cache', '1. Portfolio ops cockpit'].join('\n')}
          value={rawInput}
          onChange={(event) => onChange(event.target.value)}
        />
        <div className="flex items-center justify-between text-sm text-muted-foreground">
          <span>
            {ideaCount} idea{ideaCount === 1 ? '' : 's'} detected
          </span>
          {rawInput && (
            <Button type="button" variant="link" className="h-auto px-0" onClick={onClear}>
              Clear input
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
