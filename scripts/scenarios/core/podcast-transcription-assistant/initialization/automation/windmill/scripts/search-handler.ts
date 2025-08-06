// Search Handler - Manages semantic search functionality
// This script interfaces with the n8n semantic search workflow

interface SearchRequest {
  query: string;
  limit?: number;
  minSimilarity?: number;
  dateFrom?: string;
  dateTo?: string;
  minDuration?: number;
  maxDuration?: number;
  sessionId?: string;
  userIdentifier?: string;
}

interface SearchResult {
  transcriptionId: string;
  filename: string;
  similarity: number;
  snippet: string;
  duration: number;
  confidence: number;
  language: string;
  createdAt: string;
  similarityFormatted: string;
  durationFormatted: string;
  viewUrl: string;
  downloadUrl: string;
}

interface SearchResponse {
  success: boolean;
  message: string;
  query: string;
  resultsCount: number;
  processingTime: number;
  maxSimilarity: number;
  results: SearchResult[];
  searchParams: {
    limit: number;
    minSimilarity: number;
    embeddingModel: string;
  };
  suggestions: string[];
}

// Debounced search state
let searchTimeout: NodeJS.Timeout | null = null;
let searchHistory: string[] = [];
const MAX_HISTORY = 10;

/**
 * Debounced search function for real-time search
 */
export async function debouncedSearch(query: string, delay: number = 500): Promise<void> {
  if (searchTimeout) {
    clearTimeout(searchTimeout);
  }
  
  searchTimeout = setTimeout(async () => {
    if (query.trim().length >= 2) {
      try {
        await executeSearch({ query: query.trim() });
      } catch (error) {
        console.error('Debounced search error:', error);
      }
    }
  }, delay);
}

/**
 * Main search function that calls the n8n semantic search workflow
 */
export async function executeSearch(searchRequest: SearchRequest): Promise<SearchResponse> {
  try {
    // Validate search query
    const validationError = validateSearchQuery(searchRequest.query);
    if (validationError) {
      throw new Error(validationError);
    }
    
    // Prepare request payload for n8n semantic search workflow
    const payload = {
      query: searchRequest.query.trim(),
      limit: Math.min(searchRequest.limit || 20, 100),
      minSimilarity: searchRequest.minSimilarity || 0.7,
      sessionId: searchRequest.sessionId || generateSessionId(),
      userIdentifier: searchRequest.userIdentifier || 'windmill-search',
      embeddingModel: 'nomic-embed-text',
      
      // Optional filters
      dateFrom: searchRequest.dateFrom,
      dateTo: searchRequest.dateTo,
      minDuration: searchRequest.minDuration,
      maxDuration: searchRequest.maxDuration
    };
    
    // Call n8n semantic search workflow
    const response = await fetch('http://localhost:5678/webhook/semantic-search', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(payload)
    });
    
    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`Search failed: ${response.status} - ${errorText}`);
    }
    
    const result = await response.json() as SearchResponse;
    
    // Add to search history
    addToSearchHistory(searchRequest.query);
    
    // Update UI state with results
    updateSearchResults(result);
    
    return result;
    
  } catch (error) {
    console.error('Search error:', error);
    
    // Return error response
    const errorResponse: SearchResponse = {
      success: false,
      message: error instanceof Error ? error.message : 'Search failed',
      query: searchRequest.query,
      resultsCount: 0,
      processingTime: 0,
      maxSimilarity: 0,
      results: [],
      searchParams: {
        limit: searchRequest.limit || 20,
        minSimilarity: searchRequest.minSimilarity || 0.7,
        embeddingModel: 'nomic-embed-text'
      },
      suggestions: [
        'Check your internet connection',
        'Try a different search query',
        'Make sure the search service is running'
      ]
    };
    
    updateSearchResults(errorResponse);
    throw error;
  }
}

/**
 * Advanced search with filters
 */
export async function advancedSearch(params: {
  query: string;
  dateRange?: { from: string; to: string };
  durationRange?: { min: number; max: number };
  language?: string;
  minConfidence?: number;
  sortBy?: 'relevance' | 'date' | 'duration';
}): Promise<SearchResponse> {
  
  const searchRequest: SearchRequest = {
    query: params.query,
    dateFrom: params.dateRange?.from,
    dateTo: params.dateRange?.to,
    minDuration: params.durationRange?.min,
    maxDuration: params.durationRange?.max,
    limit: 50 // Higher limit for advanced search
  };
  
  const results = await executeSearch(searchRequest);
  
  // Apply additional client-side filters
  if (params.language || params.minConfidence || params.sortBy) {
    results.results = filterAndSortResults(results.results, {
      language: params.language,
      minConfidence: params.minConfidence,
      sortBy: params.sortBy
    });
    results.resultsCount = results.results.length;
  }
  
  return results;
}

/**
 * Get search suggestions based on query
 */
export async function getSearchSuggestions(partialQuery: string): Promise<string[]> {
  if (partialQuery.length < 2) {
    return searchHistory.slice(-5); // Return recent searches
  }
  
  // In a real implementation, this could use a suggestion API
  // For now, return history-based suggestions
  const suggestions = searchHistory
    .filter(query => query.toLowerCase().includes(partialQuery.toLowerCase()))
    .slice(-5);
  
  // Add some common search patterns
  const commonPatterns = [
    `${partialQuery} summary`,
    `${partialQuery} key points`,
    `${partialQuery} analysis`,
    `${partialQuery} discussion`
  ];
  
  return [...suggestions, ...commonPatterns].slice(0, 8);
}

/**
 * Get search history for the current session
 */
export function getSearchHistory(): string[] {
  return [...searchHistory].reverse(); // Most recent first
}

/**
 * Clear search history
 */
export function clearSearchHistory(): void {
  searchHistory = [];
}

/**
 * Save a search query for later use
 */
export async function saveSearch(query: string, name: string): Promise<{ success: boolean; message: string }> {
  try {
    const savedSearch = {
      name: name,
      query: query,
      savedAt: new Date().toISOString()
    };
    
    // In a real implementation, this would save to the database
    // For now, we'll save to localStorage
    const savedSearches = JSON.parse(localStorage.getItem('savedSearches') || '[]');
    savedSearches.push(savedSearch);
    localStorage.setItem('savedSearches', JSON.stringify(savedSearches));
    
    return { success: true, message: 'Search saved successfully' };
    
  } catch (error) {
    console.error('Failed to save search:', error);
    return { success: false, message: 'Failed to save search' };
  }
}

/**
 * Get saved searches
 */
export function getSavedSearches(): Array<{ name: string; query: string; savedAt: string }> {
  try {
    return JSON.parse(localStorage.getItem('savedSearches') || '[]');
  } catch {
    return [];
  }
}

/**
 * Export search results to various formats
 */
export async function exportSearchResults(results: SearchResult[], format: 'csv' | 'json' | 'txt'): Promise<Blob> {
  let content: string;
  let mimeType: string;
  
  switch (format) {
    case 'csv':
      content = convertToCSV(results);
      mimeType = 'text/csv';
      break;
    case 'json':
      content = JSON.stringify(results, null, 2);
      mimeType = 'application/json';
      break;
    case 'txt':
      content = convertToText(results);
      mimeType = 'text/plain';
      break;
    default:
      throw new Error(`Unsupported export format: ${format}`);
  }
  
  return new Blob([content], { type: mimeType });
}

// Private helper functions

/**
 * Validates search query
 */
function validateSearchQuery(query: string): string | null {
  if (!query || query.trim().length === 0) {
    return 'Search query cannot be empty';
  }
  
  if (query.trim().length < 2) {
    return 'Search query must be at least 2 characters long';
  }
  
  if (query.length > 500) {
    return 'Search query must be less than 500 characters';
  }
  
  return null;
}

/**
 * Generates a session ID for tracking
 */
function generateSessionId(): string {
  return `search-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

/**
 * Adds query to search history
 */
function addToSearchHistory(query: string): void {
  const trimmedQuery = query.trim();
  
  // Remove duplicate if exists
  const index = searchHistory.indexOf(trimmedQuery);
  if (index > -1) {
    searchHistory.splice(index, 1);
  }
  
  // Add to beginning
  searchHistory.unshift(trimmedQuery);
  
  // Keep only recent searches
  if (searchHistory.length > MAX_HISTORY) {
    searchHistory = searchHistory.slice(0, MAX_HISTORY);
  }
}

/**
 * Updates the UI with search results
 */
function updateSearchResults(results: SearchResponse): void {
  // This would integrate with Windmill's state management
  // For now, we'll log the results
  console.log('Search results updated:', {
    query: results.query,
    count: results.resultsCount,
    processingTime: results.processingTime
  });
  
  // In a real Windmill integration, you would update the UI state here
  // Example: setState({ searchResults: results.results, searchQuery: results.query })
}

/**
 * Filters and sorts search results
 */
function filterAndSortResults(results: SearchResult[], filters: {
  language?: string;
  minConfidence?: number;
  sortBy?: 'relevance' | 'date' | 'duration';
}): SearchResult[] {
  
  let filtered = [...results];
  
  // Apply language filter
  if (filters.language) {
    filtered = filtered.filter(r => r.language === filters.language);
  }
  
  // Apply confidence filter
  if (filters.minConfidence) {
    filtered = filtered.filter(r => r.confidence >= filters.minConfidence!);
  }
  
  // Apply sorting
  if (filters.sortBy) {
    switch (filters.sortBy) {
      case 'date':
        filtered.sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
        break;
      case 'duration':
        filtered.sort((a, b) => b.duration - a.duration);
        break;
      case 'relevance':
      default:
        // Already sorted by relevance (similarity) from the search API
        break;
    }
  }
  
  return filtered;
}

/**
 * Converts search results to CSV format
 */
function convertToCSV(results: SearchResult[]): string {
  const headers = ['Filename', 'Similarity', 'Duration', 'Language', 'Created At', 'Preview'];
  const rows = results.map(r => [
    r.filename,
    r.similarityFormatted,
    r.durationFormatted,
    r.language,
    new Date(r.createdAt).toLocaleString(),
    r.snippet.replace(/"/g, '""') // Escape quotes
  ]);
  
  const csvContent = [
    headers.join(','),
    ...rows.map(row => row.map(cell => `"${cell}"`).join(','))
  ].join('\n');
  
  return csvContent;
}

/**
 * Converts search results to plain text format
 */
function convertToText(results: SearchResult[]): string {
  let content = `Search Results\n${'='.repeat(50)}\n\n`;
  
  results.forEach((result, index) => {
    content += `${index + 1}. ${result.filename}\n`;
    content += `   Similarity: ${result.similarityFormatted}\n`;
    content += `   Duration: ${result.durationFormatted}\n`;
    content += `   Language: ${result.language}\n`;
    content += `   Created: ${new Date(result.createdAt).toLocaleString()}\n`;
    content += `   Preview: ${result.snippet}\n\n`;
  });
  
  return content;
}