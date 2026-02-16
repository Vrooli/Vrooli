/**
 * AI Search Modal - Shows AI-powered search results.
 *
 * Features:
 * - Entity type tabs (Skills, Agents, Teams)
 * - Displays semantic search results with similarity scores
 * - Shows search method used (AI or text fallback)
 * - Click result to navigate to entity
 */

import { useState, useEffect, useCallback } from 'react'
import { createPortal } from 'react-dom'
import { X, Sparkles, Search, AlertCircle, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { AISearchResponse, AISearchResult, AIAgentSearchResponse, AIAgentSearchResult, AITeamSearchResponse, AITeamSearchResult } from '@/lib/schemas'
import { api } from '@/lib/api'
import { AISearchStatusPanel } from '@/components/shared/AISearchStatusPanel'

type EntityType = 'skills' | 'agents' | 'teams'

interface AISearchModalProps {
  isOpen: boolean
  onClose: () => void
  initialQuery: string
  onSelectSkill: (skillId: string) => void
  onSelectAgent?: (agentId: string) => void
  onSelectTeam?: (teamId: string) => void
  initialEntityType?: EntityType
}

export function AISearchModal({
  isOpen,
  onClose,
  initialQuery,
  onSelectSkill,
  onSelectAgent,
  onSelectTeam,
  initialEntityType = 'skills',
}: AISearchModalProps) {
  const [query, setQuery] = useState(initialQuery)
  const [entityType, setEntityType] = useState<EntityType>(initialEntityType)
  const [skillResults, setSkillResults] = useState<AISearchResponse | null>(null)
  const [agentResults, setAgentResults] = useState<AIAgentSearchResponse | null>(null)
  const [teamResults, setTeamResults] = useState<AITeamSearchResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const performSearch = useCallback(async (searchQuery: string, type: EntityType) => {
    if (!searchQuery.trim()) {
      setSkillResults(null)
      setAgentResults(null)
      setTeamResults(null)
      return
    }

    setLoading(true)
    setError(null)

    try {
      switch (type) {
        case 'skills': {
          const response = await api.aiSearch(searchQuery, 10)
          setSkillResults(response)
          break
        }
        case 'agents': {
          const response = await api.aiSearchAgents(searchQuery, 10)
          setAgentResults(response)
          break
        }
        case 'teams': {
          const response = await api.aiSearchTeams(searchQuery, 10)
          setTeamResults(response)
          break
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed')
    } finally {
      setLoading(false)
    }
  }, [])

  // Run search when modal opens, query changes, or entity type changes
  useEffect(() => {
    if (isOpen && query.trim()) {
      const debounce = setTimeout(() => {
        void performSearch(query, entityType)
      }, 300)
      return () => clearTimeout(debounce)
    }
  }, [isOpen, query, entityType, performSearch])

  // Update query only when the initialQuery prop changes
  useEffect(() => {
    setQuery(initialQuery)
  }, [initialQuery])

  // Update entity type when prop changes
  useEffect(() => {
    setEntityType(initialEntityType)
  }, [initialEntityType])

  const handleSelectSkillResult = (result: AISearchResult) => {
    onSelectSkill(result.id)
    onClose()
  }

  const handleSelectAgentResult = (result: AIAgentSearchResult) => {
    onSelectAgent?.(result.id)
    onClose()
  }

  const handleSelectTeamResult = (result: AITeamSearchResult) => {
    onSelectTeam?.(result.id)
    onClose()
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      onClose()
    }
  }

  const currentResults = entityType === 'skills' ? skillResults : entityType === 'agents' ? agentResults : teamResults
  const currentTotal = currentResults?.total ?? 0
  const currentMethod = currentResults?.method ?? 'ai'

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

        {/* Entity type tabs */}
        <div className="flex border-b border-border bg-muted/20">
          {(['skills', 'agents', 'teams'] as const).map((type) => (
            <button
              key={type}
              type="button"
              onClick={() => setEntityType(type)}
              className={cn(
                'flex-1 px-4 py-2 text-sm font-medium transition-colors',
                entityType === type
                  ? 'text-primary border-b-2 border-primary bg-background'
                  : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'
              )}
            >
              {type.charAt(0).toUpperCase() + type.slice(1)}
            </button>
          ))}
        </div>

        {/* Search input */}
        <div className="px-4 py-3 border-b border-border">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={`Search ${entityType}...`}
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

          {!loading && !error && currentResults && (
            <>
              {/* Method indicator */}
              <div className="px-4 py-2 text-xs text-muted-foreground border-b border-border">
                {currentTotal} result{currentTotal !== 1 ? 's' : ''} found
                {currentMethod === 'text' && (
                  <span className="ml-1 text-amber-500">
                    (using text search - AI unavailable)
                  </span>
                )}
              </div>

              {currentTotal === 0 ? (
                <div className="px-4 py-8 text-center text-muted-foreground">
                  <p className="text-sm">No {entityType} match your search.</p>
                  <p className="text-xs mt-1">Try different keywords or a more general description.</p>
                </div>
              ) : (
                <ul className="divide-y divide-border">
                  {entityType === 'skills' && skillResults?.results.map((result) => (
                    <li key={result.id}>
                      <button
                        type="button"
                        onClick={() => handleSelectSkillResult(result)}
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
                          <ScoreBadge scorePercent={result.scorePercent} />
                        </div>
                      </button>
                    </li>
                  ))}

                  {entityType === 'agents' && agentResults?.results.map((result) => (
                    <li key={result.id}>
                      <button
                        type="button"
                        onClick={() => handleSelectAgentResult(result)}
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
                                {result.displayName}
                              </span>
                              <span className="text-xs text-muted-foreground flex-shrink-0">
                                {result.status}
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
                              </div>
                            )}
                          </div>
                          <ScoreBadge scorePercent={result.scorePercent} />
                        </div>
                      </button>
                    </li>
                  ))}

                  {entityType === 'teams' && teamResults?.results.map((result) => (
                    <li key={result.id}>
                      <button
                        type="button"
                        onClick={() => handleSelectTeamResult(result)}
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
                                {result.displayName}
                              </span>
                              <span className="text-xs text-muted-foreground flex-shrink-0">
                                {result.enabled ? 'enabled' : 'disabled'} · {result.memberCount} member{result.memberCount !== 1 ? 's' : ''}
                              </span>
                            </div>
                            {result.mission && (
                              <p className="mt-1 text-xs text-muted-foreground line-clamp-2">
                                {result.mission}
                              </p>
                            )}
                          </div>
                          <ScoreBadge scorePercent={result.scorePercent} />
                        </div>
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </>
          )}

          {!loading && !error && !currentResults && query.trim() === '' && (
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

function ScoreBadge({ scorePercent }: { scorePercent: number }) {
  return (
    <div className="flex-shrink-0">
      <span
        className={cn(
          'inline-flex items-center px-2 py-1 text-xs font-medium rounded-full',
          scorePercent >= 70
            ? 'bg-green-500/20 text-green-400'
            : scorePercent >= 50
            ? 'bg-yellow-500/20 text-yellow-400'
            : 'bg-muted text-muted-foreground'
        )}
      >
        {scorePercent}%
      </span>
    </div>
  )
}
