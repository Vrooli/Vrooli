import { useEffect, useState } from 'react'
import { FileText, Plus, X, Loader2 } from 'lucide-react'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { buildApiUrl } from '../../utils/apiClient'
import type { CatalogEntry } from '../../types'

const MAX_REFERENCE_PRDS = 10

const PRD_SECTIONS = [
  { value: 'full', label: 'Full PRD' },
  { value: 'overview', label: 'Overview' },
  { value: 'operational-targets', label: 'Operational Targets' },
  { value: 'tech-direction', label: 'Tech Direction' },
  { value: 'dependencies', label: 'Dependencies & Launch Plan' },
  { value: 'ux-branding', label: 'UX & Branding' },
  { value: 'appendix', label: 'Appendix' },
] as const

interface SelectedPRD {
  name: string
  displayName: string
  section: string
  content: string
}

interface ReferencePRDSelectorProps {
  selectedPRDs: Array<{ name: string; content: string }>
  onSelectionChange: (prds: Array<{ name: string; content: string }>) => void
}

export function ReferencePRDSelector({ selectedPRDs, onSelectionChange }: ReferencePRDSelectorProps) {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [availablePRDs, setAvailablePRDs] = useState<CatalogEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [tempSelected, setTempSelected] = useState<Map<string, SelectedPRD>>(new Map())

  useEffect(() => {
    if (dialogOpen) {
      loadAvailablePRDs()
    }
  }, [dialogOpen])

  const loadAvailablePRDs = async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await fetch(buildApiUrl('/catalog'))
      if (!response.ok) {
        throw new Error('Failed to load catalog')
      }
      const data = await response.json() as { entries: CatalogEntry[] }
      // Filter to only PRDs that exist
      const prdsOnly = data.entries.filter(entry => entry.has_prd)
      setAvailablePRDs(prdsOnly)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load PRDs')
    } finally {
      setLoading(false)
    }
  }

  const handleTogglePRD = async (entry: CatalogEntry, section: string) => {
    const key = `${entry.type}/${entry.name}:${section}`

    if (tempSelected.has(key)) {
      // Remove from selection
      const newSelected = new Map(tempSelected)
      newSelected.delete(key)
      setTempSelected(newSelected)
    } else {
      // Add to selection (if under limit)
      if (tempSelected.size >= MAX_REFERENCE_PRDS) {
        setError(`Maximum ${MAX_REFERENCE_PRDS} reference PRDs allowed`)
        return
      }

      // Fetch the PRD content
      try {
        const response = await fetch(buildApiUrl(`/catalog/${entry.type}/${entry.name}`))
        if (!response.ok) {
          throw new Error('Failed to load PRD content')
        }
        const data = await response.json() as { content: string }

        let content = data.content
        // If not "full", extract only the requested section
        if (section !== 'full') {
          // TODO: Extract specific section from content
          // For now, include full content with a note
          content = `[${section} section from ${entry.name}]\n\n${content}`
        }

        const newSelected = new Map(tempSelected)
        newSelected.set(key, {
          name: `${entry.type}/${entry.name}:${section}`,
          displayName: `${entry.name} (${section})`,
          section,
          content,
        })
        setTempSelected(newSelected)
        setError(null)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load PRD')
      }
    }
  }

  const handleApplySelection = () => {
    const selectedArray = Array.from(tempSelected.values()).map(prd => ({
      name: prd.displayName,
      content: prd.content,
    }))
    onSelectionChange(selectedArray)
    setDialogOpen(false)
  }

  const handleOpenDialog = () => {
    // Initialize temp selection with current selection
    const initialSelection = new Map<string, SelectedPRD>()
    selectedPRDs.forEach(prd => {
      initialSelection.set(prd.name, {
        name: prd.name,
        displayName: prd.name,
        section: 'full',
        content: prd.content,
      })
    })
    setTempSelected(initialSelection)
    setDialogOpen(true)
  }

  const filteredPRDs = availablePRDs.filter(entry =>
    entry.name.toLowerCase().includes(searchQuery.toLowerCase())
  )

  // Generate summary text
  const getSummaryText = () => {
    if (selectedPRDs.length === 0) {
      return 'No reference PRDs selected'
    }

    const sectionCounts = new Map<string, number>()
    selectedPRDs.forEach(prd => {
      const section = prd.name.includes('(') ? prd.name.split('(')[1].split(')')[0] : 'full'
      sectionCounts.set(section, (sectionCounts.get(section) || 0) + 1)
    })

    const parts: string[] = []
    sectionCounts.forEach((count, section) => {
      if (section === 'full') {
        parts.push(`${count} Full PRD${count > 1 ? 's' : ''}`)
      } else {
        parts.push(`${section} from ${count} PRD${count > 1 ? 's' : ''}`)
      }
    })

    return parts.join(', ')
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-slate-700">Reference PRDs (optional)</span>
        <Button
          variant="outline"
          size="sm"
          onClick={handleOpenDialog}
          className="gap-2"
        >
          <Plus size={14} />
          Select PRDs ({selectedPRDs.length}/{MAX_REFERENCE_PRDS})
        </Button>
      </div>

      <p className="text-xs text-slate-600">
        {getSummaryText()}
      </p>

      {selectedPRDs.length > 0 && (
        <div className="space-y-2">
          {selectedPRDs.map((prd, index) => (
            <div key={index} className="flex items-center gap-2 rounded-lg border border-slate-200 bg-white p-2">
              <FileText size={14} className="text-slate-500" />
              <span className="flex-1 text-sm text-slate-700 truncate">{prd.name}</span>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  onSelectionChange(selectedPRDs.filter((_, i) => i !== index))
                }}
                className="h-6 w-6 p-0 text-red-600 hover:text-red-700 hover:bg-red-50"
              >
                <X size={14} />
              </Button>
            </div>
          ))}
        </div>
      )}

      {/* Selection Dialog */}
      {dialogOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/80 px-4" role="dialog" aria-modal>
          <div className="w-full max-w-3xl rounded-2xl border bg-white p-6 shadow-2xl max-h-[80vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-xl font-semibold text-slate-900">Select Reference PRDs</h3>
              <Button variant="ghost" size="sm" onClick={() => setDialogOpen(false)}>
                <X size={18} />
              </Button>
            </div>

            <p className="text-sm text-slate-600 mb-4">
              Select up to {MAX_REFERENCE_PRDS} PRDs and specify which sections to include. The AI will use these as examples for style and structure.
            </p>

            <Input
              placeholder="Search PRDs..."
              value={searchQuery}
              onChange={e => setSearchQuery(e.target.value)}
              className="mb-4"
            />

            {error && (
              <div className="mb-4 rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800">
                {error}
              </div>
            )}

            {loading ? (
              <div className="flex items-center justify-center py-8">
                <Loader2 size={24} className="animate-spin text-slate-400" />
              </div>
            ) : (
              <div className="space-y-3 mb-4 max-h-[400px] overflow-y-auto">
                {filteredPRDs.map(entry => (
                  <div key={`${entry.type}/${entry.name}`} className="rounded-lg border border-slate-200 p-3">
                    <div className="font-medium text-slate-900 mb-2">
                      {entry.name}
                      <span className="ml-2 text-xs text-slate-500">({entry.type})</span>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      {PRD_SECTIONS.map(section => {
                        const key = `${entry.type}/${entry.name}:${section.value}`
                        const isSelected = tempSelected.has(key)
                        return (
                          <button
                            key={section.value}
                            onClick={() => handleTogglePRD(entry, section.value)}
                            className={`px-3 py-1 text-xs rounded-full border transition ${
                              isSelected
                                ? 'bg-blue-100 border-blue-300 text-blue-900'
                                : 'bg-white border-slate-300 text-slate-700 hover:border-blue-300 hover:bg-blue-50'
                            }`}
                          >
                            {section.label}
                          </button>
                        )
                      })}
                    </div>
                  </div>
                ))}
              </div>
            )}

            <div className="flex items-center justify-between gap-3 pt-4 border-t">
              <span className="text-sm text-slate-600">
                {tempSelected.size} of {MAX_REFERENCE_PRDS} selected
              </span>
              <div className="flex gap-2">
                <Button variant="outline" onClick={() => setDialogOpen(false)}>
                  Cancel
                </Button>
                <Button onClick={handleApplySelection}>
                  Apply Selection
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
