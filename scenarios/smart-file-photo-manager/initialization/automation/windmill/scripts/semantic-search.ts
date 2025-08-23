/**
 * Smart File Photo Manager - Semantic Search Script
 * Performs semantic search across files using embeddings and hybrid search techniques
 */

import { Resource } from "windmill-client";

interface OllamaResource {
  base_url: string;
  api_key?: string;
}

interface QdrantResource {
  url: string;
  api_key?: string;
}

interface PostgresResource {
  host: string;
  port: number;
  database: string;
  username: string;
  password: string;
}

interface SemanticSearchInput {
  query: string;
  searchType?: 'files' | 'content' | 'images' | 'hybrid';
  limit?: number;
  minScore?: number;
  filters?: SearchFilters;
  userId?: string;
  rerank?: boolean;
}

interface SearchFilters {
  fileTypes?: string[];
  folderPaths?: string[];
  dateRange?: { start: string; end: string };
  tags?: string[];
  sizeRange?: { min: number; max: number };
  hasDescription?: boolean;
}

interface SemanticSearchResult {
  results: SearchResultItem[];
  totalResults: number;
  searchTime: number;
  query: string;
  searchType: string;
  suggestions?: string[];
}

interface SearchResultItem {
  id: string;
  filename: string;
  folderPath: string;
  fileType: string;
  size: number;
  uploadedAt: string;
  description?: string;
  tags: string[];
  score: number;
  snippet?: string;
  thumbnailUrl?: string;
  matchType: 'semantic' | 'keyword' | 'hybrid';
  highlights?: string[];
}

export async function main(
  input: SemanticSearchInput,
  ollama: Resource<"ollama"> = "f/file_manager/ollama",
  qdrant: Resource<"qdrant"> = "f/file_manager/qdrant",
  postgres: Resource<"postgres"> = "f/file_manager/postgres"
): Promise<SemanticSearchResult> {
  
  console.log(`Performing semantic search:`, input);
  const startTime = Date.now();
  
  try {
    const searchType = input.searchType || 'hybrid';
    const limit = Math.min(input.limit || 20, 100);
    const minScore = input.minScore || 0.6;
    
    let results: SearchResultItem[] = [];
    
    // Store search query for learning
    await storeSearchQuery(postgres as PostgresResource, input.query, searchType, input.userId);
    
    switch (searchType) {
      case 'files':
        results = await searchFiles(
          ollama as OllamaResource,
          qdrant as QdrantResource,
          postgres as PostgresResource,
          input.query,
          limit,
          minScore,
          input.filters
        );
        break;
        
      case 'content':
        results = await searchContent(
          ollama as OllamaResource,
          qdrant as QdrantResource,
          postgres as PostgresResource,
          input.query,
          limit,
          minScore,
          input.filters
        );
        break;
        
      case 'images':
        results = await searchImages(
          ollama as OllamaResource,
          qdrant as QdrantResource,
          postgres as PostgresResource,
          input.query,
          limit,
          minScore,
          input.filters
        );
        break;
        
      case 'hybrid':
      default:
        results = await hybridSearch(
          ollama as OllamaResource,
          qdrant as QdrantResource,
          postgres as PostgresResource,
          input.query,
          limit,
          minScore,
          input.filters
        );
        break;
    }
    
    // Rerank results if requested
    if (input.rerank && results.length > 1) {
      results = await rerankResults(ollama as OllamaResource, input.query, results);
    }
    
    // Generate search suggestions
    const suggestions = await generateSearchSuggestions(
      postgres as PostgresResource,
      input.query,
      results
    );
    
    const searchTime = Date.now() - startTime;
    
    const result: SemanticSearchResult = {
      results,
      totalResults: results.length,
      searchTime,
      query: input.query,
      searchType,
      suggestions
    };
    
    console.log(`Search completed: ${results.length} results in ${searchTime}ms`);
    return result;
    
  } catch (error) {
    console.error('Error in semantic search:', error);
    throw error;
  }
}

async function searchFiles(
  ollama: OllamaResource,
  qdrant: QdrantResource,
  postgres: PostgresResource,
  query: string,
  limit: number,
  minScore: number,
  filters?: SearchFilters
): Promise<SearchResultItem[]> {
  
  // Generate query embedding
  const queryEmbedding = await generateQueryEmbedding(ollama, query);
  
  // Search in file embeddings collection
  const vectorResults = await searchVectorDatabase(
    qdrant,
    'file_embeddings',
    queryEmbedding,
    limit * 2, // Get more results for filtering
    minScore,
    filters
  );
  
  // Get detailed file information
  const results = await enrichWithFileDetails(postgres, vectorResults, 'semantic');
  
  // Apply additional filters
  const filteredResults = applyFilters(results, filters);
  
  return filteredResults.slice(0, limit);
}

async function searchContent(
  ollama: OllamaResource,
  qdrant: QdrantResource,
  postgres: PostgresResource,
  query: string,
  limit: number,
  minScore: number,
  filters?: SearchFilters
): Promise<SearchResultItem[]> {
  
  // Generate query embedding
  const queryEmbedding = await generateQueryEmbedding(ollama, query);
  
  // Search in content chunks collection
  const vectorResults = await searchVectorDatabase(
    qdrant,
    'content_chunks',
    queryEmbedding,
    limit * 3, // Get more chunks to consolidate by file
    minScore,
    filters
  );
  
  // Group chunks by file and get file details
  const fileIds = [...new Set(vectorResults.map(r => r.payload.file_id))];
  const results = await getFilesByIds(postgres, fileIds);
  
  // Add snippets from matching chunks
  return results.map(result => ({
    ...result,
    matchType: 'semantic' as const,
    snippet: extractSnippet(vectorResults, result.id, query)
  })).slice(0, limit);
}

async function searchImages(
  ollama: OllamaResource,
  qdrant: QdrantResource,
  postgres: PostgresResource,
  query: string,
  limit: number,
  minScore: number,
  filters?: SearchFilters
): Promise<SearchResultItem[]> {
  
  // Generate query embedding
  const queryEmbedding = await generateQueryEmbedding(ollama, query);
  
  // Search in image embeddings collection
  const vectorResults = await searchVectorDatabase(
    qdrant,
    'image_embeddings',
    queryEmbedding,
    limit * 2,
    minScore,
    { ...filters, fileTypes: ['image'] }
  );
  
  // Get detailed file information
  const results = await enrichWithFileDetails(postgres, vectorResults, 'semantic');
  
  // Generate thumbnails URLs
  return results.map(result => ({
    ...result,
    thumbnailUrl: `/api/files/${result.id}/thumbnail`
  })).slice(0, limit);
}

async function hybridSearch(
  ollama: OllamaResource,
  qdrant: QdrantResource,
  postgres: PostgresResource,
  query: string,
  limit: number,
  minScore: number,
  filters?: SearchFilters
): Promise<SearchResultItem[]> {
  
  // Perform both semantic and keyword searches
  const [semanticResults, keywordResults] = await Promise.all([
    searchFiles(ollama, qdrant, postgres, query, limit, minScore, filters),
    keywordSearch(postgres, query, limit, filters)
  ]);
  
  // Combine and deduplicate results
  const combinedResults = new Map<string, SearchResultItem>();
  
  // Add semantic results with weight
  for (const result of semanticResults) {
    combinedResults.set(result.id, {
      ...result,
      score: result.score * 0.7, // Weight semantic results
      matchType: 'semantic'
    });
  }
  
  // Add keyword results with weight, combine scores if duplicate
  for (const result of keywordResults) {
    if (combinedResults.has(result.id)) {
      const existing = combinedResults.get(result.id)!;
      combinedResults.set(result.id, {
        ...existing,
        score: existing.score + (result.score * 0.3), // Weight keyword results
        matchType: 'hybrid'
      });
    } else {
      combinedResults.set(result.id, {
        ...result,
        score: result.score * 0.3,
        matchType: 'keyword'
      });
    }
  }
  
  // Sort by combined score and return top results
  return Array.from(combinedResults.values())
    .sort((a, b) => b.score - a.score)
    .slice(0, limit);
}

async function keywordSearch(
  postgres: PostgresResource,
  query: string,
  limit: number,
  filters?: SearchFilters
): Promise<SearchResultItem[]> {
  
  const { Client } = await import('pg');
  const client = new Client({
    host: postgres.host,
    port: postgres.port,
    database: postgres.database,
    user: postgres.username,
    password: postgres.password
  });
  
  try {
    await client.connect();
    
    let whereClause = '1=1';
    const params: any[] = [query];
    let paramIndex = 2;
    
    // Add filters to WHERE clause
    if (filters?.fileTypes && filters.fileTypes.length > 0) {
      whereClause += ` AND file_type = ANY($${paramIndex})`;
      params.push(filters.fileTypes);
      paramIndex++;
    }
    
    if (filters?.folderPaths && filters.folderPaths.length > 0) {
      whereClause += ` AND (${filters.folderPaths.map(() => `folder_path LIKE $${paramIndex++} || '%'`).join(' OR ')})`;
      params.push(...filters.folderPaths);
    }
    
    if (filters?.dateRange) {
      whereClause += ` AND uploaded_at BETWEEN $${paramIndex} AND $${paramIndex + 1}`;
      params.push(filters.dateRange.start, filters.dateRange.end);
      paramIndex += 2;
    }
    
    if (filters?.tags && filters.tags.length > 0) {
      whereClause += ` AND tags && $${paramIndex}`;
      params.push(filters.tags);
      paramIndex++;
    }
    
    const query_sql = `
      SELECT 
        id, original_name as filename, folder_path, file_type,
        size_bytes as size, uploaded_at, description, tags,
        ts_rank(
          to_tsvector('english', 
            COALESCE(original_name, '') || ' ' ||
            COALESCE(description, '') || ' ' ||
            COALESCE(ocr_text, '') || ' ' ||
            array_to_string(COALESCE(tags, ARRAY[]::text[]), ' ')
          ),
          plainto_tsquery('english', $1)
        ) as score
      FROM files
      WHERE ${whereClause}
        AND (
          to_tsvector('english', 
            COALESCE(original_name, '') || ' ' ||
            COALESCE(description, '') || ' ' ||
            COALESCE(ocr_text, '') || ' ' ||
            array_to_string(COALESCE(tags, ARRAY[]::text[]), ' ')
          ) @@ plainto_tsquery('english', $1)
        )
      ORDER BY score DESC
      LIMIT ${limit}
    `;
    
    const result = await client.query(query_sql, params);
    
    return result.rows.map(row => ({
      id: row.id,
      filename: row.filename,
      folderPath: row.folder_path,
      fileType: row.file_type,
      size: parseInt(row.size),
      uploadedAt: row.uploaded_at,
      description: row.description,
      tags: row.tags || [],
      score: parseFloat(row.score),
      matchType: 'keyword' as const
    }));
    
  } finally {
    await client.end();
  }
}

async function generateQueryEmbedding(ollama: OllamaResource, query: string): Promise<number[]> {
  try {
    const response = await fetch(`${ollama.base_url}/api/embeddings`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        model: 'nomic-embed-text',
        prompt: query
      })
    });
    
    if (!response.ok) {
      throw new Error(`Embedding API error: ${response.statusText}`);
    }
    
    const result = await response.json();
    return result.embedding;
    
  } catch (error) {
    console.error('Error generating query embedding:', error);
    throw error;
  }
}

async function searchVectorDatabase(
  qdrant: QdrantResource,
  collection: string,
  queryVector: number[],
  limit: number,
  minScore: number,
  filters?: SearchFilters
): Promise<any[]> {
  
  const searchPayload: any = {
    vector: queryVector,
    limit,
    score_threshold: minScore,
    with_payload: true
  };
  
  // Add filters if provided
  if (filters) {
    const filter: any = { must: [] };
    
    if (filters.fileTypes && filters.fileTypes.length > 0) {
      filter.must.push({
        key: 'file_type',
        match: { any: filters.fileTypes }
      });
    }
    
    if (filters.folderPaths && filters.folderPaths.length > 0) {
      filter.must.push({
        key: 'folder_path',
        match: { any: filters.folderPaths }
      });
    }
    
    if (filters.tags && filters.tags.length > 0) {
      filter.must.push({
        key: 'tags',
        match: { any: filters.tags }
      });
    }
    
    if (filter.must.length > 0) {
      searchPayload.filter = filter;
    }
  }
  
  try {
    const response = await fetch(`${qdrant.url}/collections/${collection}/points/search`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(searchPayload)
    });
    
    if (!response.ok) {
      throw new Error(`Qdrant search error: ${response.statusText}`);
    }
    
    const result = await response.json();
    return result.result || [];
    
  } catch (error) {
    console.error('Error searching vector database:', error);
    throw error;
  }
}

async function enrichWithFileDetails(
  postgres: PostgresResource,
  vectorResults: any[],
  matchType: string
): Promise<SearchResultItem[]> {
  
  if (vectorResults.length === 0) return [];
  
  const fileIds = vectorResults.map(r => r.payload.file_id);
  const files = await getFilesByIds(postgres, fileIds);
  
  // Combine vector results with file details
  return vectorResults.map(vectorResult => {
    const file = files.find(f => f.id === vectorResult.payload.file_id);
    if (!file) return null;
    
    return {
      ...file,
      score: vectorResult.score,
      matchType: matchType as any
    };
  }).filter(r => r !== null) as SearchResultItem[];
}

async function getFilesByIds(postgres: PostgresResource, fileIds: string[]): Promise<SearchResultItem[]> {
  const { Client } = await import('pg');
  const client = new Client({
    host: postgres.host,
    port: postgres.port,
    database: postgres.database,
    user: postgres.username,
    password: postgres.password
  });
  
  try {
    await client.connect();
    
    const query = `
      SELECT 
        id, original_name as filename, folder_path, file_type,
        size_bytes as size, uploaded_at, description, tags
      FROM files 
      WHERE id = ANY($1)
    `;
    
    const result = await client.query(query, [fileIds]);
    
    return result.rows.map(row => ({
      id: row.id,
      filename: row.filename,
      folderPath: row.folder_path,
      fileType: row.file_type,
      size: parseInt(row.size),
      uploadedAt: row.uploaded_at,
      description: row.description,
      tags: row.tags || [],
      score: 1.0, // Will be overridden
      matchType: 'semantic' as const
    }));
    
  } finally {
    await client.end();
  }
}

function applyFilters(results: SearchResultItem[], filters?: SearchFilters): SearchResultItem[] {
  if (!filters) return results;
  
  return results.filter(result => {
    if (filters.sizeRange) {
      if (result.size < filters.sizeRange.min || result.size > filters.sizeRange.max) {
        return false;
      }
    }
    
    if (filters.hasDescription !== undefined) {
      if (filters.hasDescription && !result.description) return false;
      if (!filters.hasDescription && result.description) return false;
    }
    
    return true;
  });
}

function extractSnippet(vectorResults: any[], fileId: string, query: string): string {
  const fileChunks = vectorResults.filter(r => r.payload.file_id === fileId);
  if (fileChunks.length === 0) return '';
  
  // Get the best matching chunk
  const bestChunk = fileChunks[0];
  const content = bestChunk.payload.content || '';
  
  // Simple snippet extraction - take first 200 characters around query terms
  const queryWords = query.toLowerCase().split(/\s+/);
  const lowerContent = content.toLowerCase();
  
  let bestIndex = 0;
  let maxMatches = 0;
  
  for (let i = 0; i < content.length - 200; i += 50) {
    const segment = lowerContent.slice(i, i + 200);
    const matches = queryWords.reduce((count, word) => 
      count + (segment.includes(word) ? 1 : 0), 0);
    
    if (matches > maxMatches) {
      maxMatches = matches;
      bestIndex = i;
    }
  }
  
  return content.slice(bestIndex, bestIndex + 200) + '...';
}

async function rerankResults(
  ollama: OllamaResource,
  query: string,
  results: SearchResultItem[]
): Promise<SearchResultItem[]> {
  
  // Simple reranking based on content relevance
  // In a production system, you might use a cross-encoder model
  
  const scoredResults = results.map(result => {
    let bonusScore = 0;
    
    // Boost score if query terms appear in filename
    const queryWords = query.toLowerCase().split(/\s+/);
    const filename = result.filename.toLowerCase();
    
    for (const word of queryWords) {
      if (filename.includes(word)) {
        bonusScore += 0.1;
      }
    }
    
    // Boost score if query terms appear in tags
    for (const tag of result.tags) {
      for (const word of queryWords) {
        if (tag.toLowerCase().includes(word)) {
          bonusScore += 0.05;
        }
      }
    }
    
    return {
      ...result,
      score: result.score + bonusScore
    };
  });
  
  return scoredResults.sort((a, b) => b.score - a.score);
}

async function generateSearchSuggestions(
  postgres: PostgresResource,
  query: string,
  results: SearchResultItem[]
): Promise<string[]> {
  
  // Generate suggestions based on popular searches and similar queries
  const suggestions: string[] = [];
  
  if (results.length === 0) {
    // If no results, suggest broader terms
    const queryWords = query.split(/\s+/);
    if (queryWords.length > 1) {
      suggestions.push(...queryWords);
    }
    suggestions.push('recent files', 'documents', 'photos');
  } else {
    // If results found, suggest related tags
    const allTags = results.flatMap(r => r.tags);
    const tagCounts = allTags.reduce((counts, tag) => {
      counts[tag] = (counts[tag] || 0) + 1;
      return counts;
    }, {} as Record<string, number>);
    
    const popularTags = Object.entries(tagCounts)
      .sort(([, a], [, b]) => b - a)
      .slice(0, 3)
      .map(([tag]) => tag);
    
    suggestions.push(...popularTags);
  }
  
  return suggestions.slice(0, 5);
}

async function storeSearchQuery(
  postgres: PostgresResource,
  query: string,
  searchType: string,
  userId?: string
): Promise<void> {
  
  const { Client } = await import('pg');
  const client = new Client({
    host: postgres.host,
    port: postgres.port,
    database: postgres.database,
    user: postgres.username,
    password: postgres.password
  });
  
  try {
    await client.connect();
    
    const insertQuery = `
      INSERT INTO search_history (query, search_type, user_id, created_at)
      VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
    `;
    
    await client.query(insertQuery, [query, searchType, userId || 'anonymous']);
    
  } catch (error) {
    console.error('Error storing search query:', error);
    // Non-critical error, don't throw
  } finally {
    await client.end();
  }
}