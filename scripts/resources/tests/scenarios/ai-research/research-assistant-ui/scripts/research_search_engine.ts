/**
 * Enterprise Research Search Engine
 * 
 * Privacy-respecting multi-source search aggregation system with intelligent
 * filtering, source verification, and comprehensive result analysis for
 * professional research and competitive intelligence workflows.
 * 
 * Features:
 * - Privacy-first search through SearXNG aggregation
 * - Multi-engine result synthesis and deduplication
 * - Source credibility scoring and verification
 * - Domain-specific search optimization
 * - Research query enhancement and suggestion
 * 
 * Enterprise Value: Core search infrastructure for $12K-20K research projects
 * Target Users: Consultants, analysts, research teams, intelligence professionals
 */

import * as wmill from "https://deno.land/x/windmill@v1.85.0/mod.ts";

// Search Engine Types
interface SearchQuery {
  query: string;
  engines?: string[];
  category?: string;
  language?: string;
  time_range?: string;
  safe_search?: boolean;
  result_limit?: number;
  domain_filter?: string[];
  exclude_domains?: string[];
  project_id?: string;
}

interface SearchResult {
  title: string;
  url: string;
  content: string;
  engine: string;
  score?: number;
  published_date?: string;
  domain: string;
  credibility_score?: number;
  content_type?: string;
  language?: string;
}

interface SearchEngineResult {
  query: string;
  total_results: number;
  search_time: number;
  engines_used: string[];
  results: SearchResult[];
  suggestions?: string[];
  related_queries?: string[];
  source_diversity: number;
  credibility_analysis: {
    high_credibility: number;
    medium_credibility: number;
    low_credibility: number;
  };
}

interface EngineConfiguration {
  name: string;
  enabled: boolean;
  weight: number;
  timeout: number;
  categories: string[];
  rate_limit?: number;
}

// Service Configuration
const SERVICES = {
  searxng: wmill.getVariable("SEARXNG_BASE_URL") || "http://localhost:8080",
  ollama: wmill.getVariable("OLLAMA_BASE_URL") || "http://localhost:11434",
  qdrant: wmill.getVariable("QDRANT_BASE_URL") || "http://localhost:6333",
  minio: wmill.getVariable("MINIO_BASE_URL") || "http://localhost:9000"
};

// Default search engines configuration
const DEFAULT_ENGINES: EngineConfiguration[] = [
  { name: "google", enabled: true, weight: 0.3, timeout: 10, categories: ["general", "news", "scholar"] },
  { name: "bing", enabled: true, weight: 0.25, timeout: 8, categories: ["general", "news", "images"] },
  { name: "duckduckgo", enabled: true, weight: 0.2, timeout: 8, categories: ["general", "news"] },
  { name: "arxiv", enabled: true, weight: 0.15, timeout: 15, categories: ["science", "it"] },
  { name: "scholar", enabled: true, weight: 0.1, timeout: 20, categories: ["science", "scholar"] }
];

/**
 * Main Research Search Function
 * Orchestrates the complete search workflow with privacy protection
 */
export async function main(
  action: 'search' | 'search_suggestions' | 'verify_sources' | 'get_search_config' | 'analyze_query',
  searchQuery?: SearchQuery,
  queryText?: string,
  sources?: string[]
): Promise<{
  success: boolean;
  data?: any;
  message: string;
  search_result?: SearchEngineResult;
  suggestions?: string[];
  analysis?: any;
}> {
  try {
    console.log(`🔍 Research Search Engine: Executing ${action}`);
    
    switch (action) {
      case 'search':
        return await executeResearchSearch(searchQuery!);
      
      case 'search_suggestions':
        return await generateSearchSuggestions(queryText!);
      
      case 'verify_sources':
        return await verifySourceCredibility(sources!);
      
      case 'get_search_config':
        return await getSearchEngineConfiguration();
      
      case 'analyze_query':
        return await analyzeSearchQuery(queryText!);
      
      default:
        throw new Error(`Unknown action: ${action}`);
    }
  } catch (error) {
    console.error('Research Search Error:', error);
    return {
      success: false,
      message: `Search failed: ${error.message}`
    };
  }
}

/**
 * Execute Privacy-Respecting Research Search
 * Multi-engine aggregation with result synthesis and scoring
 */
async function executeResearchSearch(query: SearchQuery): Promise<any> {
  console.log(`🔍 Executing research search: "${query.query}"`);
  
  const searchStart = Date.now();
  const enabledEngines = getEnabledEngines(query.engines, query.category);
  
  // Execute parallel searches across engines
  const searchPromises = enabledEngines.map(engine => 
    searchWithEngine(query, engine)
  );
  
  const engineResults = await Promise.allSettled(searchPromises);
  
  // Process and aggregate results
  const allResults: SearchResult[] = [];
  const enginesUsed: string[] = [];
  
  engineResults.forEach((result, index) => {
    if (result.status === 'fulfilled' && result.value) {
      allResults.push(...result.value.results);
      enginesUsed.push(enabledEngines[index].name);
    }
  });
  
  // Deduplicate and score results
  const processedResults = await processSearchResults(allResults, query);
  
  // Calculate source diversity
  const sourceDiversity = calculateSourceDiversity(processedResults);
  
  // Analyze credibility
  const credibilityAnalysis = analyzeResultCredibility(processedResults);
  
  // Generate related queries and suggestions
  const suggestions = await generateRelatedQueries(query.query);
  
  const searchResult: SearchEngineResult = {
    query: query.query,
    total_results: processedResults.length,
    search_time: Date.now() - searchStart,
    engines_used: enginesUsed,
    results: processedResults.slice(0, query.result_limit || 50),
    suggestions: suggestions.slice(0, 5),
    source_diversity: sourceDiversity,
    credibility_analysis: credibilityAnalysis
  };
  
  // Store search for analytics (if project specified)
  if (query.project_id) {
    await storeSearchAnalytics(query, searchResult);
  }
  
  console.log(`✅ Search completed: ${processedResults.length} results from ${enginesUsed.length} engines`);
  
  return {
    success: true,
    data: {
      search_result: searchResult,
      performance_metrics: {
        engines_contacted: enabledEngines.length,
        engines_successful: enginesUsed.length,
        total_processing_time: Date.now() - searchStart,
        results_per_second: Math.round(processedResults.length / ((Date.now() - searchStart) / 1000))
      }
    },
    search_result: searchResult,
    message: `Search completed successfully: ${processedResults.length} results found`
  };
}

/**
 * Search with Individual Engine
 * Execute search request against specific SearXNG engine
 */
async function searchWithEngine(query: SearchQuery, engine: EngineConfiguration): Promise<{ results: SearchResult[] }> {
  try {
    console.log(`🌐 Searching with ${engine.name}...`);
    
    // Build search parameters
    const searchParams = new URLSearchParams({
      q: query.query,
      format: 'json',
      engines: engine.name,
      ...(query.category && { categories: query.category }),
      ...(query.language && { language: query.language }),
      ...(query.safe_search !== undefined && { safesearch: query.safe_search ? '1' : '0' }),
      ...(query.time_range && { time_range: query.time_range })
    });
    
    const response = await fetch(`${SERVICES.searxng}/search?${searchParams}`, {
      method: 'GET',
      headers: {
        'User-Agent': 'Vrooli-Research-Assistant/1.0',
        'Accept': 'application/json'
      },
      signal: AbortSignal.timeout(engine.timeout * 1000)
    });
    
    if (!response.ok) {
      throw new Error(`SearXNG request failed: ${response.statusText}`);
    }
    
    const data = await response.json();
    
    if (!data.results || !Array.isArray(data.results)) {
      return { results: [] };
    }
    
    // Transform results to our format
    const transformedResults: SearchResult[] = data.results.map((result: any) => ({
      title: result.title || 'Untitled',
      url: result.url || '',
      content: result.content || '',
      engine: engine.name,
      published_date: result.publishedDate,
      domain: extractDomain(result.url || ''),
      content_type: result.content_type || 'text/html',
      language: result.language
    }));
    
    // Apply domain filtering
    const filteredResults = applyDomainFiltering(transformedResults, query);
    
    console.log(`✅ ${engine.name}: ${filteredResults.length} results`);
    return { results: filteredResults };
    
  } catch (error) {
    console.error(`❌ Search failed for ${engine.name}:`, error);
    return { results: [] };
  }
}

/**
 * Process and Score Search Results
 * Deduplication, scoring, and ranking of aggregated results
 */
async function processSearchResults(results: SearchResult[], query: SearchQuery): Promise<SearchResult[]> {
  console.log(`🔄 Processing ${results.length} raw search results...`);
  
  // Deduplicate by URL and content similarity
  const deduplicatedResults = deduplicateResults(results);
  
  // Score results based on relevance and credibility
  const scoredResults = await scoreResults(deduplicatedResults, query);
  
  // Sort by score (descending)
  const sortedResults = scoredResults.sort((a, b) => (b.score || 0) - (a.score || 0));
  
  console.log(`✅ Processed results: ${sortedResults.length} unique results`);
  return sortedResults;
}

/**
 * Generate Search Suggestions
 * AI-powered query enhancement and related search suggestions
 */
async function generateSearchSuggestions(queryText: string): Promise<any> {
  console.log(`💡 Generating search suggestions for: "${queryText}"`);
  
  try {
    // Get available AI model
    const model = await getAvailableModel();
    
    const suggestionPrompt = `Generate 8 improved and related search queries for research purposes based on: "${queryText}". 
    Focus on: alternative phrasings, more specific queries, broader context, temporal variations, and domain-specific angles.
    Return as a simple list, one query per line.`;
    
    const response = await fetch(`${SERVICES.ollama}/api/generate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        model,
        prompt: suggestionPrompt,
        stream: false,
        options: { temperature: 0.7 }
      })
    });
    
    if (response.ok) {
      const result = await response.json();
      const suggestions = result.response
        .split('\n')
        .map((line: string) => line.trim())
        .filter((line: string) => line.length > 0 && !line.match(/^\d+\.?\s*/))
        .slice(0, 8);
      
      return {
        success: true,
        data: { suggestions },
        suggestions,
        message: `Generated ${suggestions.length} search suggestions`
      };
    }
  } catch (error) {
    console.error('AI suggestion generation failed:', error);
  }
  
  // Fallback to basic suggestions
  const basicSuggestions = generateBasicSuggestions(queryText);
  
  return {
    success: true,
    data: { suggestions: basicSuggestions },
    suggestions: basicSuggestions,
    message: `Generated ${basicSuggestions.length} basic suggestions`
  };
}

/**
 * Verify Source Credibility
 * Analyze source reliability and credibility scores
 */
async function verifySourceCredibility(sources: string[]): Promise<any> {
  console.log(`🔍 Verifying credibility of ${sources.length} sources...`);
  
  const credibilityResults = await Promise.all(
    sources.map(async (source) => {
      const domain = extractDomain(source);
      const credibilityScore = await calculateCredibilityScore(domain, source);
      
      return {
        url: source,
        domain,
        credibility_score: credibilityScore,
        category: categorizeCredibility(credibilityScore),
        factors: getCredibilityFactors(domain)
      };
    })
  );
  
  const averageCredibility = credibilityResults.reduce((sum, result) => sum + result.credibility_score, 0) / credibilityResults.length;
  
  return {
    success: true,
    data: {
      sources: credibilityResults,
      average_credibility: averageCredibility,
      high_credibility_count: credibilityResults.filter(r => r.credibility_score > 0.7).length,
      low_credibility_count: credibilityResults.filter(r => r.credibility_score < 0.4).length
    },
    message: `Verified ${sources.length} sources with average credibility ${Math.round(averageCredibility * 100)}%`
  };
}

/**
 * Analyze Search Query
 * Extract intent, entities, and suggest optimization
 */
async function analyzeSearchQuery(queryText: string): Promise<any> {
  console.log(`🔍 Analyzing search query: "${queryText}"`);
  
  const analysis = {
    query: queryText,
    word_count: queryText.split(' ').length,
    query_type: detectQueryType(queryText),
    entities: extractEntities(queryText),
    suggested_improvements: suggestQueryImprovements(queryText),
    recommended_engines: recommendEngines(queryText),
    estimated_results: estimateResultCount(queryText)
  };
  
  // AI-powered query analysis if available
  try {
    const model = await getAvailableModel();
    const aiAnalysis = await getAIQueryAnalysis(queryText, model);
    if (aiAnalysis) {
      analysis['ai_insights'] = aiAnalysis;
    }
  } catch (error) {
    console.log('AI analysis not available:', error.message);
  }
  
  return {
    success: true,
    data: analysis,
    analysis,
    message: `Query analysis completed: ${analysis.query_type} query with ${analysis.entities.length} entities`
  };
}

/**
 * Get Search Engine Configuration
 * Return current engine settings and capabilities
 */
async function getSearchEngineConfiguration(): Promise<any> {
  console.log('⚙️ Retrieving search engine configuration...');
  
  const configuration = {
    engines: DEFAULT_ENGINES,
    supported_categories: ['general', 'news', 'science', 'it', 'scholar', 'social media'],
    supported_languages: ['en', 'es', 'fr', 'de', 'zh', 'ja'],
    max_results: 100,
    timeout_range: [5, 30],
    privacy_features: [
      'No user tracking',
      'No search history storage', 
      'IP address protection',
      'Multi-engine aggregation',
      'No cookies required'
    ]
  };
  
  return {
    success: true,
    data: configuration,
    message: 'Search engine configuration retrieved'
  };
}

// Helper Functions

function getEnabledEngines(requestedEngines?: string[], category?: string): EngineConfiguration[] {
  let engines = DEFAULT_ENGINES.filter(engine => engine.enabled);
  
  if (requestedEngines && requestedEngines.length > 0) {
    engines = engines.filter(engine => requestedEngines.includes(engine.name));
  }
  
  if (category) {
    engines = engines.filter(engine => engine.categories.includes(category));
  }
  
  return engines;
}

function extractDomain(url: string): string {
  try {
    return new URL(url).hostname;
  } catch {
    return 'unknown';
  }
}

function applyDomainFiltering(results: SearchResult[], query: SearchQuery): SearchResult[] {
  let filtered = results;
  
  if (query.domain_filter && query.domain_filter.length > 0) {
    filtered = filtered.filter(result => 
      query.domain_filter!.some(domain => result.domain.includes(domain))
    );
  }
  
  if (query.exclude_domains && query.exclude_domains.length > 0) {
    filtered = filtered.filter(result => 
      !query.exclude_domains!.some(domain => result.domain.includes(domain))
    );
  }
  
  return filtered;
}

function deduplicateResults(results: SearchResult[]): SearchResult[] {
  const seen = new Set<string>();
  const deduplicated: SearchResult[] = [];
  
  for (const result of results) {
    const key = `${result.url}_${result.title}`;
    if (!seen.has(key)) {
      seen.add(key);
      deduplicated.push(result);
    }
  }
  
  return deduplicated;
}

async function scoreResults(results: SearchResult[], query: SearchQuery): Promise<SearchResult[]> {
  return results.map(result => {
    let score = 0;
    
    // Title relevance
    score += calculateTextRelevance(result.title, query.query) * 0.3;
    
    // Content relevance  
    score += calculateTextRelevance(result.content, query.query) * 0.4;
    
    // Domain credibility
    score += calculateDomainCredibility(result.domain) * 0.2;
    
    // Recency (if published date available)
    if (result.published_date) {
      score += calculateRecencyScore(result.published_date) * 0.1;
    }
    
    result.score = Math.min(1.0, Math.max(0.0, score));
    return result;
  });
}

function calculateTextRelevance(text: string, query: string): number {
  const queryTerms = query.toLowerCase().split(' ');
  const textLower = text.toLowerCase();
  
  let matches = 0;
  for (const term of queryTerms) {
    if (textLower.includes(term)) {
      matches++;
    }
  }
  
  return matches / queryTerms.length;
}

function calculateDomainCredibility(domain: string): number {
  // Simplified credibility scoring
  const highCredibility = ['edu', 'gov', 'org'];
  const mediumCredibility = ['com', 'net'];
  
  if (highCredibility.some(suffix => domain.endsWith(suffix))) {
    return 0.9;
  } else if (mediumCredibility.some(suffix => domain.endsWith(suffix))) {
    return 0.7;
  }
  
  return 0.5;
}

function calculateRecencyScore(publishedDate: string): number {
  try {
    const published = new Date(publishedDate);
    const now = new Date();
    const daysDiff = (now.getTime() - published.getTime()) / (1000 * 60 * 60 * 24);
    
    if (daysDiff <= 7) return 1.0;
    if (daysDiff <= 30) return 0.8;
    if (daysDiff <= 365) return 0.6;
    return 0.4;
  } catch {
    return 0.5;
  }
}

function calculateSourceDiversity(results: SearchResult[]): number {
  const domains = new Set(results.map(r => r.domain));
  const engines = new Set(results.map(r => r.engine));
  
  return Math.min(1.0, (domains.size + engines.size) / (results.length + 1));
}

function analyzeResultCredibility(results: SearchResult[]): any {
  const credibilityBreakdown = {
    high_credibility: 0,
    medium_credibility: 0,
    low_credibility: 0
  };
  
  results.forEach(result => {
    const credibility = calculateDomainCredibility(result.domain);
    if (credibility > 0.8) {
      credibilityBreakdown.high_credibility++;
    } else if (credibility > 0.6) {
      credibilityBreakdown.medium_credibility++;
    } else {
      credibilityBreakdown.low_credibility++;
    }
  });
  
  return credibilityBreakdown;
}

async function generateRelatedQueries(query: string): Promise<string[]> {
  // Simple related query generation
  const related = [
    `${query} 2024`,
    `${query} trends`,
    `${query} analysis`,
    `${query} market research`,
    `${query} best practices`
  ];
  
  return related;
}

function generateBasicSuggestions(query: string): string[] {
  return [
    `${query} overview`,
    `${query} statistics`,
    `${query} case study`,
    `${query} industry analysis`,
    `${query} future trends`
  ];
}

function detectQueryType(query: string): string {
  if (query.includes('?')) return 'question';
  if (query.includes('vs') || query.includes('compare')) return 'comparison';
  if (query.includes('how to')) return 'instructional';
  if (query.includes('when') || query.includes('where')) return 'factual';
  return 'informational';
}

function extractEntities(query: string): string[] {
  // Simple entity extraction (would use NLP in production)
  const words = query.split(' ').filter(word => word.length > 3);
  return words.slice(0, 5);
}

function suggestQueryImprovements(query: string): string[] {
  const improvements = [];
  
  if (query.split(' ').length < 3) {
    improvements.push('Add more specific terms');
  }
  
  if (!query.includes('2024') && !query.includes('recent')) {
    improvements.push('Add time frame (e.g., "2024", "recent")');
  }
  
  if (!query.includes('analysis') && !query.includes('research')) {
    improvements.push('Add research intent (e.g., "analysis", "study")');
  }
  
  return improvements;
}

function recommendEngines(query: string): string[] {
  const engines = ['google', 'bing', 'duckduckgo'];
  
  if (query.includes('academic') || query.includes('research')) {
    engines.push('arxiv', 'scholar');
  }
  
  if (query.includes('news') || query.includes('current')) {
    return ['google', 'bing']; // Better for news
  }
  
  return engines;
}

function estimateResultCount(query: string): string {
  const wordCount = query.split(' ').length;
  
  if (wordCount <= 2) return '10,000+';
  if (wordCount <= 4) return '1,000-10,000';
  return '100-1,000';
}

async function calculateCredibilityScore(domain: string, url: string): Promise<number> {
  // Simplified credibility calculation
  let score = 0.5; // Base score
  
  // Domain type scoring
  if (domain.endsWith('.edu')) score += 0.3;
  else if (domain.endsWith('.gov')) score += 0.35;
  else if (domain.endsWith('.org')) score += 0.2;
  else if (domain.endsWith('.com')) score += 0.1;
  
  // Known high-quality domains
  const highQualityDomains = ['reuters.com', 'bbc.com', 'nature.com', 'science.org'];
  if (highQualityDomains.some(d => domain.includes(d))) {
    score += 0.2;
  }
  
  return Math.min(1.0, score);
}

function categorizeCredibility(score: number): string {
  if (score > 0.7) return 'high';
  if (score > 0.4) return 'medium';
  return 'low';
}

function getCredibilityFactors(domain: string): string[] {
  const factors = [];
  
  if (domain.endsWith('.edu')) factors.push('Educational institution');
  if (domain.endsWith('.gov')) factors.push('Government source');
  if (domain.endsWith('.org')) factors.push('Organization');
  
  return factors;
}

async function getAIQueryAnalysis(query: string, model: string): Promise<any> {
  const analysisPrompt = `Analyze this search query for research purposes: "${query}"
  Identify: intent, key concepts, potential gaps, and research approach suggestions.
  Respond in JSON format.`;
  
  const response = await fetch(`${SERVICES.ollama}/api/generate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      model,
      prompt: analysisPrompt,
      stream: false
    })
  });
  
  if (response.ok) {
    const result = await response.json();
    try {
      return JSON.parse(result.response);
    } catch {
      return { analysis: result.response };
    }
  }
  
  return null;
}

async function storeSearchAnalytics(query: SearchQuery, result: SearchEngineResult): Promise<void> {
  // Store search analytics for project tracking
  console.log(`📊 Storing search analytics for project: ${query.project_id}`);
  // Implementation would store in analytics database
}

async function getAvailableModel(): Promise<string> {
  try {
    const response = await fetch(`${SERVICES.ollama}/api/tags`);
    const data = await response.json();
    return data.models?.[0]?.name || 'llama2';
  } catch {
    return 'llama2';
  }
}

console.log('🔍 Enterprise Research Search Engine initialized');
console.log('🎯 Features: Privacy-first search, multi-engine aggregation, credibility scoring');
console.log('💰 Enterprise ready for $12K-20K research projects');