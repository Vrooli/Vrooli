import { useState, useCallback, useEffect, useRef } from 'react'
import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'

const rawExplicitBase =
  typeof import.meta !== 'undefined' && import.meta.env && typeof import.meta.env.VITE_API_BASE_URL === 'string'
    ? import.meta.env.VITE_API_BASE_URL.trim()
    : ''

const rawFallbackPort =
  typeof import.meta !== 'undefined' && import.meta.env && typeof import.meta.env.VITE_API_PORT === 'string'
    ? import.meta.env.VITE_API_PORT.trim()
    : ''

const explicitApiBase = rawExplicitBase.length > 0 ? rawExplicitBase : undefined
const fallbackApiPort = rawFallbackPort.length > 0 ? rawFallbackPort : '15000'

const apiOrigin = resolveApiBase({
  explicitUrl: explicitApiBase,
  defaultPort: fallbackApiPort,
  appendSuffix: false,
})

const apiBase = buildApiUrl('/api', {
  baseUrl: apiOrigin,
  appendSuffix: false,
})

const createEndpointUrl = (endpoint) =>
  buildApiUrl(endpoint, {
    baseUrl: apiBase,
    appendSuffix: false,
  })

/**
 * Custom hook for Symbol Search API interactions
 * Handles character search, filtering, pagination, and metadata
 */
export function useSymbolSearch() {
  const [state, setState] = useState({
    characters: [],
    loading: false,
    error: null,
    total: 0,
    queryTime: 0,
    hasMore: false,
    categories: [],
    blocks: []
  })

  // Track current request to cancel outdated ones
  const currentRequestRef = useRef(null)
  const searchAbortControllerRef = useRef(null)

  // API request helper with abort support
  const apiRequest = useCallback(async (endpoint, options = {}) => {
    const {
      signal,
      headers = {},
      ...requestInit
    } = options

    try {
      const response = await fetch(createEndpointUrl(endpoint), {
        ...requestInit,
        signal,
        headers: {
          'Content-Type': 'application/json',
          ...headers
        }
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Network error' }))
        throw new Error(errorData.error || `HTTP ${response.status}`)
      }

      return await response.json()
    } catch (error) {
      if (error.name === 'AbortError') {
        throw error // Re-throw abort errors
      }
      throw new Error(error.message || 'Request failed')
    }
  }, [])

  // Search characters with filters and pagination
  const searchCharacters = useCallback(async (params = {}) => {
    const {
      query = '',
      category = '',
      block = '',
      unicodeVersion = '',
      limit = 100,
      offset = 0
    } = params

    // Generate request ID to handle race conditions
    const requestId = Date.now()
    currentRequestRef.current = requestId

    // Cancel previous search request
    if (searchAbortControllerRef.current) {
      searchAbortControllerRef.current.abort()
    }

    const controller = new AbortController()
    searchAbortControllerRef.current = controller

    setState(prev => ({
      ...prev,
      loading: true,
      error: null,
      // Reset characters if this is a new search (offset 0)
      characters: offset === 0 ? [] : prev.characters
    }))

    try {
      const queryParams = new URLSearchParams()
      if (query) queryParams.set('q', query)
      if (category) queryParams.set('category', category)
      if (block) queryParams.set('block', block)
      if (unicodeVersion) queryParams.set('unicode_version', unicodeVersion)
      queryParams.set('limit', limit.toString())
      queryParams.set('offset', offset.toString())

      const data = await apiRequest(`/search?${queryParams.toString()}`, {
        signal: controller.signal
      })

      // Check if this is still the current request
      if (currentRequestRef.current !== requestId) {
        return // Ignore outdated responses
      }

      setState(prev => ({
        ...prev,
        characters: offset === 0 ? data.characters : [...prev.characters, ...data.characters],
        loading: false,
        total: data.total,
        queryTime: data.query_time_ms,
        hasMore: data.characters.length === limit && (offset + limit) < data.total,
        error: null
      }))
    } catch (error) {
      if (error?.name === 'AbortError') {
        return // Ignore aborted requests
      }

      if (currentRequestRef.current === requestId) {
        setState(prev => ({
          ...prev,
          loading: false,
          error: error.message,
          hasMore: false
        }))
      }
    } finally {
      if (searchAbortControllerRef.current === controller) {
        searchAbortControllerRef.current = null
      }
    }
  }, [apiRequest])

  // Load more characters (for infinite scrolling)
  const loadMore = useCallback(async (params) => {
    if (state.loading || !state.hasMore) return

    await searchCharacters({
      ...params,
      offset: state.characters.length
    })
  }, [state.loading, state.hasMore, state.characters.length, searchCharacters])

  // Get character details
  const getCharacterDetail = useCallback(async (codepoint) => {
    try {
      const encodedCodepoint = encodeURIComponent(codepoint)
      return await apiRequest(`/character/${encodedCodepoint}`)
    } catch (error) {
      console.error('Failed to get character detail:', error)
      throw error
    }
  }, [apiRequest])

  // Load categories and blocks metadata
  const loadMetadata = useCallback(async () => {
    try {
      const [categoriesResponse, blocksResponse] = await Promise.all([
        apiRequest('/categories'),
        apiRequest('/blocks')
      ])

      setState(prev => ({
        ...prev,
        categories: categoriesResponse.categories || [],
        blocks: blocksResponse.blocks || []
      }))
    } catch (error) {
      if (error?.name === 'AbortError') {
        return
      }
      console.error('Failed to load metadata:', error)
      // Don't set error state for metadata failures
    }
  }, [apiRequest])

  // Load categories and blocks on component mount
  useEffect(() => {
    loadMetadata()
  }, [loadMetadata])

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (searchAbortControllerRef.current) {
        searchAbortControllerRef.current.abort()
      }
    }
  }, [])

  return {
    // State
    characters: state.characters,
    loading: state.loading,
    error: state.error,
    total: state.total,
    queryTime: state.queryTime,
    hasMore: state.hasMore,
    categories: state.categories,
    blocks: state.blocks,

    // Actions
    searchCharacters,
    loadMore,
    getCharacterDetail,
    loadMetadata
  }
}
