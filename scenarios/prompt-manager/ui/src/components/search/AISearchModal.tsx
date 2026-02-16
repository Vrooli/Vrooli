/**
 * AI Search Modal - Shows AI-powered search results.
 *
 * Features:
 * - Displays semantic search results with similarity scores
 * - Shows search method used (AI or text fallback)
 * - Click result to navigate to skill
 */

import { useState, useEffect, useCallback } from 'react'
import { createPortal } from 'react-dom'
import { X, Sparkles, Search, AlertCircle, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { AISearchResponse, AISearchResult } from '@/lib/schemas'
import { aiSearch } from '@/services/skillService'
import { AISearchStatusPanel } from '@/components/shared/AISearchStatusPanel'

interface AISearchModalProps {
  isOpen: boolean
  onClose: () => void
  initialQuery: string
  onSelectSkill: (skillId: string) => void
}

export function AISearchModal({
  isOpen,
  onClose,
  initialQuery,
  onSelectSkill,
}: AISearchModalProps) {
  const [query, setQuery] = useState(initialQuery)
  const [results, setResults] = useState<AISearchResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const performSearch = useCallback(async (searchQuery: string) => {
    if (!searchQuery.trim()) {
      setResults(null)
      return
    }

    setLoading(true)
    setError(null)

    try {
      const response = await aiSearch(searchQuery, 10)
      setResults(response)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed')
    } finally {
      setLoading(false)
    }
  }, [])

  // Run search when modal opens or query changes
  useEffect(() => {
    if (isOpen && query.trim()) {
      const debounce = setTimeout(() => {
        void performSearch(query)
      }, 300)
      return () => clearTimeout(debounce)
    }
  }, [isOpen, query, performSearch])

  // Update query only when the initialQuery prop changes
  useEffect(() => {
    setQuery(initialQuery)
  }, [initialQuery])

  const handleSelectResult = (result: AISearchResult) => {
    onSelectSkill(result.id)
    onClose()
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      onClose()
    }
  }

  if (!isOpen) return null
  if (typeof document === 'undefined') return null

  return createPortal(
    <div
      className="fixed inset-0 flex items-start justify-center pt-[10vh] bg-black/50"
      style={{ zIndex: 2147483647 }}
      onClick={onClose}
      onKeyDown={handleKeyDown}
      role="dialog"
      aria-modal="true"
      aria-label="AI Search"
    >
      <div
        className="w-full max-w-2xl bg-card border border-border rounded-xl shadow-xl overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-border bg-muted/30">
          <Sparkles className="h-5 w-5 text-primary" />
          <span className="text-sm font-medium text-foreground">AI Search</span>
          <span className="text-xs text-muted-foreground">
            Semantic search powered by embeddings
          </span>
          <button
            type="button"
            onClick={onClose}
            className="ml-auto p-1.5 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Search input */}
        <div className="px-4 py-3 border-b border-border">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Describe what you're looking for..."
              autoFocus
              className={cn(
                'w-full pl-10 pr-4 py-2.5 text-sm',
                'bg-background border border-border rounded-lg',
                'text-foreground placeholder:text-muted-foreground',
                'focus:outline-none focus:ring-2 focus:ring-primary'
              )}
            />
          </div>
        </div>

        {/* Status */}
        <div className="px-4 py-2 border-b border-border bg-muted/20">
          <AISearchStatusPanel active={isOpen} />
        </div>

        {/* Results */}
        <div className="max-h-[60vh] overflow-y-auto">
          {loading && (
            <div className="flex items-center justify-center py-12 gap-3 text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin" />
              <span className="text-sm">Searching...</span>
            </div>
          )}

          {error && (
            <div className="flex items-center gap-3 px-4 py-6 text-destructive">
              <AlertCircle className="h-5 w-5 flex-shrink-0" />
              <span className="text-sm">{error}</span>
            </div>
          )}

          {!loading && !error && results && (
            <>
              {/* Method indicator */}
              <div className="px-4 py-2 text-xs text-muted-foreground border-b border-border">
                {results.total} result{results.total !== 1 ? 's' : ''} found
                {results.method === 'text' && (
                  <span className="ml-1 text-amber-500">
                    (using text search - AI unavailable)
                  </span>
                )}
              </div>

              {results.results.length === 0 ? (
                <div className="px-4 py-8 text-center text-muted-foreground">
                  <p className="text-sm">No skills match your search.</p>
                  <p className="text-xs mt-1">Try different keywords or a more general description.</p>
                </div>
              ) : (
                <ul className="divide-y divide-border">
                  {results.results.map((result) => (
                    <li key={result.id}>
                      <button
                        type="button"
                        onClick={() => handleSelectResult(result)}
                        className={cn(
                          'w-full px-4 py-3 text-left',
                          'hover:bg-muted/50 transition-colors',
                          'focus:outline-none focus:bg-muted/50'
                        )}
                      >
                        <div className="flex items-start justify-between gap-3">
                          <div className="min-w-0 flex-1">
                            <div className="flex items-center gap-2">
                              <span className="font-medium text-foreground truncate">
                                {result.name}
                              </span>
                              <span className="text-xs text-muted-foreground flex-shrink-0">
                                {result.folder}/
                              </span>
                            </div>
                            {result.description && (
                              <p className="mt-1 text-xs text-muted-foreground line-clamp-2">
                                {result.description}
                              </p>
                            )}
                            {result.tags.length > 0 && (
                              <div className="mt-1.5 flex flex-wrap gap-1">
                                {result.tags.slice(0, 5).map((tag) => (
                                  <span
                                    key={tag}
                                    className="px-1.5 py-0.5 text-[10px] bg-muted text-muted-foreground rounded"
                                  >
                                    {tag}
                                  </span>
                                ))}
                                {result.tags.length > 5 && (
                                  <span className="text-[10px] text-muted-foreground">
                                    +{result.tags.length - 5}
                                  </span>
                                )}
                              </div>
                            )}
                          </div>
                          <div className="flex-shrink-0">
                            <span
                              className={cn(
                                'inline-flex items-center px-2 py-1 text-xs font-medium rounded-full',
                                result.scorePercent >= 70
                                  ? 'bg-green-500/20 text-green-400'
                                  : result.scorePercent >= 50
                                  ? 'bg-yellow-500/20 text-yellow-400'
                                  : 'bg-muted text-muted-foreground'
                              )}
                            >
                              {result.scorePercent}%
                            </span>
                          </div>
                        </div>
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </>
          )}

          {!loading && !error && !results && query.trim() === '' && (
            <div className="px-4 py-8 text-center text-muted-foreground">
              <Sparkles className="h-8 w-8 mx-auto mb-3 opacity-50" />
              <p className="text-sm">Enter a query to search with AI</p>
              <p className="text-xs mt-1">
                Describe what you're looking for in natural language
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  , document.body)
}
