# SearXNG Integration Examples

This document provides comprehensive, production-ready examples for integrating SearXNG with various tools and frameworks in the Vrooli ecosystem.

## Table of Contents
- [n8n Workflows](#n8n-workflows)
- [Ollama Integration](#ollama-integration)
- [LangChain Integration](#langchain-integration)
- [Python Scripts](#python-scripts)
- [Node.js/TypeScript](#nodejstypescript)
- [Shell Scripts](#shell-scripts)

## n8n Workflows

### Basic Search Workflow
Import this workflow directly into n8n:

```json
{
  "name": "SearXNG Web Search and Analysis",
  "nodes": [
    {
      "parameters": {
        "method": "GET",
        "url": "http://localhost:8280/search",
        "queryParametersUi": {
          "parameter": [
            {
              "name": "q",
              "value": "={{ $json[\"query\"] }}"
            },
            {
              "name": "format",
              "value": "json"
            },
            {
              "name": "categories",
              "value": "general"
            }
          ]
        },
        "options": {
          "response": {
            "response": {
              "responseFormat": "json"
            }
          }
        }
      },
      "name": "Search Web",
      "type": "n8n-nodes-base.httpRequest",
      "typeVersion": 4.1,
      "position": [460, 260]
    },
    {
      "parameters": {
        "jsCode": "// Extract top 5 results\nconst results = $input.first().json.results;\nconst topResults = results.slice(0, 5).map(r => ({\n  title: r.title,\n  url: r.url,\n  content: r.content || 'No description',\n  engine: r.engine\n}));\n\nreturn topResults;"
      },
      "name": "Process Results",
      "type": "n8n-nodes-base.code",
      "typeVersion": 2,
      "position": [660, 260]
    }
  ],
  "connections": {
    "Search Web": {
      "main": [
        [
          {
            "node": "Process Results",
            "type": "main",
            "index": 0
          }
        ]
      ]
    }
  }
}
```

### Advanced Research Workflow with AI Analysis

```json
{
  "name": "AI-Powered Research Assistant",
  "nodes": [
    {
      "parameters": {
        "operation": "text",
        "text": "{{ $json[\"topic\"] }}"
      },
      "name": "Research Topic",
      "type": "n8n-nodes-base.set",
      "typeVersion": 2,
      "position": [260, 260]
    },
    {
      "parameters": {
        "method": "GET",
        "url": "http://localhost:8280/search",
        "queryParametersUi": {
          "parameter": [
            {
              "name": "q",
              "value": "={{ $json[\"topic\"] }}"
            },
            {
              "name": "format",
              "value": "json"
            },
            {
              "name": "time_range",
              "value": "month"
            },
            {
              "name": "language",
              "value": "en"
            }
          ]
        }
      },
      "name": "Search Recent Articles",
      "type": "n8n-nodes-base.httpRequest",
      "typeVersion": 4.1,
      "position": [460, 260]
    },
    {
      "parameters": {
        "method": "POST",
        "url": "http://localhost:11434/api/generate",
        "jsonParameters": true,
        "options": {},
        "bodyParametersJson": "{\n  \"model\": \"llama3.1:8b\",\n  \"prompt\": \"Analyze and summarize these search results about {{ $('Research Topic').first().json.topic }}:\\n\\n{{ $json.results.slice(0,5).map(r => r.title + ': ' + r.content).join('\\n\\n') }}\\n\\nProvide a comprehensive summary with key insights.\",\n  \"stream\": false\n}"
      },
      "name": "AI Analysis",
      "type": "n8n-nodes-base.httpRequest",
      "typeVersion": 4.1,
      "position": [660, 260]
    },
    {
      "parameters": {
        "jsCode": "// Combine search results with AI analysis\nconst searchResults = $('Search Recent Articles').first().json;\nconst aiAnalysis = $('AI Analysis').first().json;\n\nreturn {\n  topic: $('Research Topic').first().json.topic,\n  timestamp: new Date().toISOString(),\n  totalResults: searchResults.number_of_results,\n  topSources: searchResults.results.slice(0, 5).map(r => ({\n    title: r.title,\n    url: r.url,\n    engine: r.engine\n  })),\n  aiSummary: aiAnalysis.response,\n  searchMetadata: {\n    query: searchResults.query,\n    responseTime: searchResults.response_time\n  }\n};"
      },
      "name": "Compile Report",
      "type": "n8n-nodes-base.code",
      "typeVersion": 2,
      "position": [860, 260]
    }
  ]
}
```

### Competitive Intelligence Workflow

```json
{
  "name": "Competitor Monitoring",
  "nodes": [
    {
      "parameters": {
        "rule": {
          "interval": [
            {
              "field": "hours",
              "hoursInterval": 6
            }
          ]
        }
      },
      "name": "Every 6 Hours",
      "type": "n8n-nodes-base.scheduleTrigger",
      "typeVersion": 1.1,
      "position": [260, 260]
    },
    {
      "parameters": {
        "jsCode": "// Define competitors to monitor\nreturn [\n  { competitor: 'Company A', keywords: 'Company A news announcements' },\n  { competitor: 'Company B', keywords: 'Company B product launches' },\n  { competitor: 'Company C', keywords: 'Company C partnerships funding' }\n];"
      },
      "name": "Competitor List",
      "type": "n8n-nodes-base.code",
      "typeVersion": 2,
      "position": [460, 260]
    },
    {
      "parameters": {
        "method": "GET",
        "url": "http://localhost:8280/search",
        "queryParametersUi": {
          "parameter": [
            {
              "name": "q",
              "value": "={{ $json[\"keywords\"] }}"
            },
            {
              "name": "format",
              "value": "json"
            },
            {
              "name": "time_range",
              "value": "day"
            },
            {
              "name": "categories",
              "value": "news"
            }
          ]
        }
      },
      "name": "Search Competitor News",
      "type": "n8n-nodes-base.httpRequest",
      "typeVersion": 4.1,
      "position": [660, 260]
    },
    {
      "parameters": {
        "conditions": {
          "number": [
            {
              "value1": "={{ $json[\"results\"].length }}",
              "operation": "larger",
              "value2": 0
            }
          ]
        }
      },
      "name": "New Results?",
      "type": "n8n-nodes-base.if",
      "typeVersion": 1,
      "position": [860, 260]
    }
  ]
}
```

## Ollama Integration

### Python Script for Research Assistant

```python
#!/usr/bin/env python3
"""
SearXNG + Ollama Research Assistant
Searches the web and uses AI to analyze and summarize findings
"""

import requests
import json
from typing import List, Dict
import sys

SEARXNG_URL = "http://localhost:8280"
OLLAMA_URL = "http://localhost:11434"

def search_web(query: str, **kwargs) -> Dict:
    """Search using SearXNG"""
    params = {
        'q': query,
        'format': 'json',
        **kwargs
    }
    
    response = requests.get(f"{SEARXNG_URL}/search", params=params)
    response.raise_for_status()
    return response.json()

def analyze_with_ollama(content: str, model: str = "llama3.1:8b") -> str:
    """Analyze content using Ollama"""
    payload = {
        'model': model,
        'prompt': content,
        'stream': False
    }
    
    response = requests.post(f"{OLLAMA_URL}/api/generate", json=payload)
    response.raise_for_status()
    return response.json()['response']

def research_topic(topic: str, depth: int = 5) -> Dict:
    """Complete research workflow"""
    print(f"🔍 Researching: {topic}")
    
    # Search for information
    search_results = search_web(
        topic,
        categories='general,science',
        time_range='year',
        language='en'
    )
    
    # Extract top results
    top_results = search_results['results'][:depth]
    
    # Prepare content for analysis
    content_for_analysis = f"Topic: {topic}\n\nTop search results:\n\n"
    for i, result in enumerate(top_results, 1):
        content_for_analysis += f"{i}. {result['title']}\n"
        content_for_analysis += f"   Source: {result['url']}\n"
        content_for_analysis += f"   Summary: {result.get('content', 'No summary')}\n\n"
    
    content_for_analysis += "\nPlease provide:\n1. A comprehensive summary of the findings\n2. Key insights and patterns\n3. Potential areas for further research"
    
    # Get AI analysis
    print("🤖 Analyzing with AI...")
    analysis = analyze_with_ollama(content_for_analysis)
    
    return {
        'topic': topic,
        'search_stats': {
            'total_results': search_results.get('number_of_results', 0),
            'response_time': search_results.get('response_time', 0),
            'engines_used': list(set(r['engine'] for r in top_results))
        },
        'top_sources': [
            {
                'title': r['title'],
                'url': r['url'],
                'engine': r['engine']
            } for r in top_results
        ],
        'ai_analysis': analysis
    }

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: research_assistant.py <topic>")
        sys.exit(1)
    
    topic = " ".join(sys.argv[1:])
    result = research_topic(topic)
    
    print("\n" + "="*50)
    print("📊 RESEARCH REPORT")
    print("="*50)
    print(f"\nTopic: {result['topic']}")
    print(f"Total Results Found: {result['search_stats']['total_results']:,}")
    print(f"Search Engines Used: {', '.join(result['search_stats']['engines_used'])}")
    print(f"\n🔗 Top Sources:")
    for source in result['top_sources']:
        print(f"  • {source['title']}")
        print(f"    {source['url']}")
    print(f"\n🤖 AI Analysis:\n{result['ai_analysis']}")
```

### Bash Script for Quick Research

```bash
#!/bin/bash
# quick_research.sh - Search and analyze with one command

set -euo pipefail

# Check arguments
if [ $# -eq 0 ]; then
    echo "Usage: $0 <search query>"
    exit 1
fi

QUERY="$*"
SEARXNG_URL="http://localhost:8280"
OLLAMA_URL="http://localhost:11434"

echo "🔍 Searching for: $QUERY"

# Search
RESULTS=$(curl -s "${SEARXNG_URL}/search?q=${QUERY// /%20}&format=json&categories=general")

# Extract top 3 results
TOP_RESULTS=$(echo "$RESULTS" | jq -r '.results[:3] | map("\(.title)\n\(.content // "No description")\nSource: \(.url)\n") | join("\n---\n")')

# Prepare prompt
PROMPT="Based on these search results for '$QUERY':\n\n$TOP_RESULTS\n\nProvide a brief summary and key insights."

# Analyze with Ollama
echo "🤖 Analyzing results..."
ANALYSIS=$(curl -s -X POST "${OLLAMA_URL}/api/generate" \
    -d "{\"model\": \"llama3.1:8b\", \"prompt\": \"$PROMPT\", \"stream\": false}" \
    | jq -r '.response')

echo -e "\n📊 Analysis:\n$ANALYSIS"
```

## LangChain Integration

### Python LangChain Example

```python
#!/usr/bin/env python3
"""
LangChain integration with SearXNG for RAG applications
"""

from langchain.tools import Tool
from langchain.agents import initialize_agent, AgentType
from langchain.llms import Ollama
import requests
import json

class SearXNGSearchTool:
    """Custom LangChain tool for SearXNG"""
    
    def __init__(self, base_url="http://localhost:8280"):
        self.base_url = base_url
    
    def search(self, query: str) -> str:
        """Execute search and return formatted results"""
        try:
            response = requests.get(
                f"{self.base_url}/search",
                params={'q': query, 'format': 'json', 'categories': 'general'}
            )
            response.raise_for_status()
            
            results = response.json()['results'][:3]
            
            formatted = []
            for r in results:
                formatted.append(f"Title: {r['title']}\nURL: {r['url']}\nSummary: {r.get('content', 'N/A')}")
            
            return "\n\n".join(formatted)
        except Exception as e:
            return f"Search error: {str(e)}"

# Initialize the search tool
searxng = SearXNGSearchTool()

# Create LangChain tool
search_tool = Tool(
    name="Web Search",
    func=searxng.search,
    description="Search the web for current information. Input should be a search query."
)

# Initialize LLM (using local Ollama)
llm = Ollama(
    base_url="http://localhost:11434",
    model="llama3.1:8b"
)

# Create agent
agent = initialize_agent(
    tools=[search_tool],
    llm=llm,
    agent=AgentType.ZERO_SHOT_REACT_DESCRIPTION,
    verbose=True
)

# Example usage
if __name__ == "__main__":
    question = "What are the latest developments in quantum computing?"
    
    result = agent.run(question)
    print(f"\n📊 Final Answer:\n{result}")
```

### TypeScript LangChain Example

```typescript
// searxng-langchain.ts
import { Tool } from "@langchain/core/tools";
import { ChatOllama } from "@langchain/community/chat_models/ollama";
import { AgentExecutor, createReactAgent } from "langchain/agents";
import axios from "axios";

class SearXNGTool extends Tool {
  name = "web-search";
  description = "Search the web for current information. Input should be a search query.";
  
  private baseUrl = "http://localhost:8280";
  
  async _call(query: string): Promise<string> {
    try {
      const response = await axios.get(`${this.baseUrl}/search`, {
        params: {
          q: query,
          format: 'json',
          categories: 'general'
        }
      });
      
      const results = response.data.results.slice(0, 3);
      
      return results.map((r: any) => 
        `Title: ${r.title}\nURL: ${r.url}\nSummary: ${r.content || 'N/A'}`
      ).join('\n\n');
    } catch (error) {
      return `Search error: ${error.message}`;
    }
  }
}

async function runResearchAgent(question: string) {
  // Initialize tools
  const searchTool = new SearXNGTool();
  
  // Initialize LLM
  const llm = new ChatOllama({
    baseUrl: "http://localhost:11434",
    model: "llama3.1:8b",
  });
  
  // Create agent
  const agent = await createReactAgent({
    llm,
    tools: [searchTool],
  });
  
  const executor = new AgentExecutor({
    agent,
    tools: [searchTool],
  });
  
  // Run query
  const result = await executor.run({ input: question });
  
  console.log(`📊 Result: ${result}`);
  return result;
}

// Example usage
runResearchAgent("What are the latest AI breakthroughs?")
  .then(console.log)
  .catch(console.error);
```

## Python Scripts

### Advanced Search API Client

```python
#!/usr/bin/env python3
"""
Complete Python client for SearXNG with all features
"""

import requests
from typing import Optional, List, Dict, Any
from datetime import datetime
import json

class SearXNGClient:
    """Full-featured SearXNG API client"""
    
    def __init__(self, base_url: str = "http://localhost:8280"):
        self.base_url = base_url
        self.session = requests.Session()
    
    def search(
        self,
        query: str,
        categories: Optional[str] = None,
        engines: Optional[List[str]] = None,
        language: Optional[str] = None,
        time_range: Optional[str] = None,
        safe_search: Optional[int] = None,
        page: int = 1,
        format: str = 'json'
    ) -> Dict[str, Any]:
        """
        Perform a search with all available options
        
        Args:
            query: Search query
            categories: Comma-separated categories (general,images,news,videos,etc)
            engines: List of specific engines to use
            language: Language code (en, de, fr, etc)
            time_range: Time filter (day, week, month, year)
            safe_search: Safe search level (0=off, 1=moderate, 2=strict)
            page: Page number for pagination
            format: Response format (json, html, csv, rss)
        
        Returns:
            Search results dictionary
        """
        params = {'q': query, 'format': format, 'pageno': page}
        
        if categories:
            params['categories'] = categories
        if engines:
            params['engines'] = ','.join(engines)
        if language:
            params['language'] = language
        if time_range:
            params['time_range'] = time_range
        if safe_search is not None:
            params['safesearch'] = safe_search
        
        response = self.session.get(f"{self.base_url}/search", params=params)
        response.raise_for_status()
        
        return response.json() if format == 'json' else response.text
    
    def get_stats(self) -> Dict[str, Any]:
        """Get service statistics"""
        response = self.session.get(f"{self.base_url}/stats")
        response.raise_for_status()
        return response.json()
    
    def get_config(self) -> Dict[str, Any]:
        """Get service configuration"""
        response = self.session.get(f"{self.base_url}/config")
        response.raise_for_status()
        return response.json()
    
    def health_check(self) -> bool:
        """Check if service is healthy"""
        try:
            response = self.session.get(f"{self.base_url}/healthz", timeout=5)
            return response.status_code == 200
        except:
            return False
    
    def batch_search(self, queries: List[str], **kwargs) -> List[Dict[str, Any]]:
        """Perform multiple searches"""
        results = []
        for query in queries:
            try:
                result = self.search(query, **kwargs)
                results.append({
                    'query': query,
                    'success': True,
                    'data': result
                })
            except Exception as e:
                results.append({
                    'query': query,
                    'success': False,
                    'error': str(e)
                })
        return results
    
    def search_with_filters(self, query: str) -> Dict[str, Any]:
        """
        Convenience method for common filtered searches
        """
        # Recent news
        news = self.search(query, categories='news', time_range='week')
        
        # Academic papers
        academic = self.search(query, categories='science', engines=['google_scholar'])
        
        # Images
        images = self.search(query, categories='images', safe_search=1)
        
        # Videos
        videos = self.search(query, categories='videos')
        
        return {
            'news': news.get('results', [])[:3],
            'academic': academic.get('results', [])[:3],
            'images': images.get('results', [])[:3],
            'videos': videos.get('results', [])[:3]
        }

# Example usage
if __name__ == "__main__":
    client = SearXNGClient()
    
    # Basic search
    results = client.search("artificial intelligence breakthroughs")
    print(f"Found {results['number_of_results']} results")
    
    # Advanced search
    filtered = client.search(
        "machine learning",
        categories="science,news",
        time_range="month",
        language="en"
    )
    
    # Multi-category search
    multi = client.search_with_filters("quantum computing")
    print(f"News articles: {len(multi['news'])}")
    print(f"Academic papers: {len(multi['academic'])}")
```

## Node.js/TypeScript

### Complete TypeScript Client

```typescript
// searxng-client.ts
import axios, { AxiosInstance } from 'axios';

interface SearchOptions {
  categories?: string;
  engines?: string[];
  language?: string;
  timeRange?: 'day' | 'week' | 'month' | 'year';
  safeSearch?: 0 | 1 | 2;
  page?: number;
  format?: 'json' | 'html' | 'csv' | 'rss';
}

interface SearchResult {
  title: string;
  url: string;
  content?: string;
  engine: string;
  score?: number;
  category?: string;
  publishedDate?: string;
}

interface SearchResponse {
  query: string;
  number_of_results: number;
  results: SearchResult[];
  response_time: number;
  suggestions?: string[];
}

class SearXNGClient {
  private client: AxiosInstance;
  
  constructor(baseURL: string = 'http://localhost:8280') {
    this.client = axios.create({
      baseURL,
      timeout: 30000,
    });
  }
  
  async search(query: string, options: SearchOptions = {}): Promise<SearchResponse> {
    const params: any = {
      q: query,
      format: options.format || 'json',
      pageno: options.page || 1,
    };
    
    if (options.categories) params.categories = options.categories;
    if (options.engines) params.engines = options.engines.join(',');
    if (options.language) params.language = options.language;
    if (options.timeRange) params.time_range = options.timeRange;
    if (options.safeSearch !== undefined) params.safesearch = options.safeSearch;
    
    const response = await this.client.get('/search', { params });
    return response.data;
  }
  
  async getStats(): Promise<any> {
    const response = await this.client.get('/stats');
    return response.data;
  }
  
  async getConfig(): Promise<any> {
    const response = await this.client.get('/config');
    return response.data;
  }
  
  async healthCheck(): Promise<boolean> {
    try {
      await this.client.get('/healthz');
      return true;
    } catch {
      return false;
    }
  }
  
  async batchSearch(
    queries: string[],
    options: SearchOptions = {}
  ): Promise<Array<{ query: string; results?: SearchResponse; error?: string }>> {
    const results = await Promise.allSettled(
      queries.map(q => this.search(q, options))
    );
    
    return results.map((result, index) => ({
      query: queries[index],
      ...(result.status === 'fulfilled'
        ? { results: result.value }
        : { error: result.reason.message }),
    }));
  }
  
  // Specialized search methods
  async searchNews(query: string, recent: boolean = true): Promise<SearchResponse> {
    return this.search(query, {
      categories: 'news',
      timeRange: recent ? 'week' : undefined,
    });
  }
  
  async searchImages(query: string, safe: boolean = true): Promise<SearchResponse> {
    return this.search(query, {
      categories: 'images',
      safeSearch: safe ? 1 : 0,
    });
  }
  
  async searchScience(query: string): Promise<SearchResponse> {
    return this.search(query, {
      categories: 'science',
      engines: ['google_scholar', 'pubmed'],
    });
  }
}

// Express.js API wrapper example
import express from 'express';

const app = express();
const searxng = new SearXNGClient();

app.get('/api/search', async (req, res) => {
  try {
    const { q, ...options } = req.query;
    const results = await searxng.search(q as string, options as SearchOptions);
    res.json(results);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

app.get('/api/research/:topic', async (req, res) => {
  try {
    const { topic } = req.params;
    
    // Parallel searches across different categories
    const [news, academic, general] = await Promise.all([
      searxng.searchNews(topic),
      searxng.searchScience(topic),
      searxng.search(topic),
    ]);
    
    res.json({
      topic,
      timestamp: new Date().toISOString(),
      news: news.results.slice(0, 3),
      academic: academic.results.slice(0, 3),
      general: general.results.slice(0, 5),
      totalResults: news.number_of_results + academic.number_of_results + general.number_of_results,
    });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

export { SearXNGClient, SearchOptions, SearchResponse, SearchResult };
```

## Shell Scripts

### Complete CLI Wrapper

```bash
#!/bin/bash
# searxng-cli.sh - Complete command-line interface for SearXNG

set -euo pipefail

SEARXNG_URL="${SEARXNG_URL:-http://localhost:8280}"

# Help function
show_help() {
    cat << EOF
SearXNG CLI - Privacy-respecting metasearch

Usage: $0 [OPTIONS] <command> [arguments]

Commands:
    search <query>           Basic search
    news <query>            Search news
    images <query>          Search images
    science <query>         Search academic papers
    advanced <query>        Advanced search with options
    batch <file>            Batch search from file
    monitor <query>         Monitor for new results
    export <query> <file>   Export results to file

Options:
    -c, --category <cat>    Set category (general,news,images,videos,etc)
    -l, --language <lang>   Set language (en,de,fr,etc)
    -t, --time <range>      Set time range (day,week,month,year)
    -e, --engines <list>    Comma-separated engine list
    -p, --page <num>        Page number
    -f, --format <fmt>      Output format (json,csv,text)
    -s, --safe              Enable safe search
    -v, --verbose           Verbose output

Examples:
    $0 search "quantum computing"
    $0 news "AI breakthroughs" --time day
    $0 advanced "machine learning" -c science -l en -t month
    $0 batch queries.txt
    $0 monitor "breaking news" --time day

EOF
}

# Parse command line arguments
COMMAND=""
QUERY=""
CATEGORY="general"
LANGUAGE=""
TIME_RANGE=""
ENGINES=""
PAGE="1"
FORMAT="json"
SAFE_SEARCH="0"
VERBOSE=0

# Search function
do_search() {
    local query="$1"
    local params="q=$(urlencode "$query")&format=$FORMAT&pageno=$PAGE&categories=$CATEGORY"
    
    [[ -n "$LANGUAGE" ]] && params="$params&language=$LANGUAGE"
    [[ -n "$TIME_RANGE" ]] && params="$params&time_range=$TIME_RANGE"
    [[ -n "$ENGINES" ]] && params="$params&engines=$ENGINES"
    [[ "$SAFE_SEARCH" != "0" ]] && params="$params&safesearch=$SAFE_SEARCH"
    
    [[ $VERBOSE -eq 1 ]] && echo "🔍 Searching: $query" >&2
    
    curl -s "${SEARXNG_URL}/search?${params}"
}

# URL encode function
urlencode() {
    python3 -c "import urllib.parse; print(urllib.parse.quote('$1'))"
}

# Format JSON output
format_results() {
    jq -r '.results[] | "[\(.engine)] \(.title)\n  \(.url)\n  \(.content // "No description")\n"'
}

# News search
search_news() {
    CATEGORY="news"
    TIME_RANGE="${TIME_RANGE:-week}"
    do_search "$1" | format_results
}

# Image search
search_images() {
    CATEGORY="images"
    SAFE_SEARCH="1"
    do_search "$1" | jq -r '.results[] | "[\(.engine)] \(.title)\n  Image: \(.url)\n  Thumbnail: \(.thumbnail // "N/A")\n"'
}

# Science/academic search
search_science() {
    CATEGORY="science"
    ENGINES="google_scholar,pubmed"
    do_search "$1" | format_results
}

# Advanced search with all options
search_advanced() {
    do_search "$1" | format_results
}

# Batch search from file
batch_search() {
    local file="$1"
    [[ ! -f "$file" ]] && { echo "File not found: $file"; exit 1; }
    
    while IFS= read -r query; do
        [[ -z "$query" || "$query" =~ ^# ]] && continue
        echo "=== Query: $query ==="
        do_search "$query" | format_results
        echo
    done < "$file"
}

# Monitor for new results
monitor_search() {
    local query="$1"
    local last_results=""
    
    echo "📡 Monitoring for: $query (Press Ctrl+C to stop)"
    
    while true; do
        current_results=$(do_search "$query" | jq -r '.results[:3] | map(.url) | join(",")')
        
        if [[ "$current_results" != "$last_results" && -n "$last_results" ]]; then
            echo "[$(date '+%Y-%m-%d %H:%M:%S')] New results found!"
            do_search "$query" | format_results | head -20
        fi
        
        last_results="$current_results"
        sleep 300  # Check every 5 minutes
    done
}

# Export results
export_results() {
    local query="$1"
    local output="$2"
    
    echo "📁 Exporting results to: $output"
    
    case "${output##*.}" in
        json)
            do_search "$query" > "$output"
            ;;
        csv)
            do_search "$query" | jq -r '.results[] | [.title, .url, .engine, .content] | @csv' > "$output"
            ;;
        txt|*)
            do_search "$query" | format_results > "$output"
            ;;
    esac
    
    echo "✅ Exported $(wc -l < "$output") lines"
}

# Main script logic
case "${1:-}" in
    search)
        shift
        search_advanced "$@"
        ;;
    news)
        shift
        search_news "$@"
        ;;
    images)
        shift
        search_images "$@"
        ;;
    science)
        shift
        search_science "$@"
        ;;
    advanced)
        shift
        # Parse additional options here
        search_advanced "$@"
        ;;
    batch)
        shift
        batch_search "$@"
        ;;
    monitor)
        shift
        monitor_search "$@"
        ;;
    export)
        shift
        export_results "$@"
        ;;
    -h|--help|help)
        show_help
        ;;
    *)
        echo "Unknown command: ${1:-}"
        show_help
        exit 1
        ;;
esac
```

## Integration Best Practices

### 1. Error Handling
Always implement proper error handling and retries:

```python
import time
from typing import Optional

def search_with_retry(query: str, max_retries: int = 3) -> Optional[Dict]:
    for attempt in range(max_retries):
        try:
            return search_web(query)
        except requests.RequestException as e:
            if attempt < max_retries - 1:
                time.sleep(2 ** attempt)  # Exponential backoff
                continue
            raise
    return None
```

### 2. Rate Limiting
Respect rate limits to avoid overloading the service:

```python
from functools import wraps
import time

def rate_limit(calls: int = 10, period: int = 60):
    def decorator(func):
        times = []
        
        @wraps(func)
        def wrapper(*args, **kwargs):
            now = time.time()
            times[:] = [t for t in times if now - t < period]
            
            if len(times) >= calls:
                sleep_time = period - (now - times[0])
                time.sleep(sleep_time)
            
            times.append(now)
            return func(*args, **kwargs)
        
        return wrapper
    return decorator

@rate_limit(calls=10, period=60)
def search_limited(query: str):
    return search_web(query)
```

### 3. Caching
Implement caching for frequently searched queries:

```python
from functools import lru_cache
import hashlib

@lru_cache(maxsize=100)
def cached_search(query: str, categories: str = 'general'):
    cache_key = hashlib.md5(f"{query}:{categories}".encode()).hexdigest()
    return search_web(query, categories=categories)
```

### 4. Monitoring
Add monitoring to track usage and performance:

```python
import logging
from datetime import datetime

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def monitored_search(query: str) -> Dict:
    start_time = datetime.now()
    
    try:
        result = search_web(query)
        duration = (datetime.now() - start_time).total_seconds()
        
        logger.info(f"Search completed: query='{query}', "
                   f"results={len(result.get('results', []))}, "
                   f"duration={duration:.2f}s")
        
        return result
    except Exception as e:
        logger.error(f"Search failed: query='{query}', error={e}")
        raise
```

## Testing Your Integration

### Unit Test Example

```python
import unittest
from unittest.mock import patch, Mock

class TestSearXNGIntegration(unittest.TestCase):
    
    @patch('requests.get')
    def test_search_success(self, mock_get):
        mock_response = Mock()
        mock_response.json.return_value = {
            'results': [
                {'title': 'Test', 'url': 'http://example.com'}
            ]
        }
        mock_response.raise_for_status = Mock()
        mock_get.return_value = mock_response
        
        result = search_web("test query")
        
        self.assertIn('results', result)
        self.assertEqual(len(result['results']), 1)
        mock_get.assert_called_once()
    
    def test_integration_live(self):
        """Test against live SearXNG instance"""
        # Only run if service is available
        if not SearXNGClient().health_check():
            self.skipTest("SearXNG service not available")
        
        client = SearXNGClient()
        result = client.search("test")
        
        self.assertIn('results', result)
        self.assertIsInstance(result['results'], list)
```

## Troubleshooting Common Issues

### Connection Refused
```bash
# Check if SearXNG is running
vrooli resource searxng status

# Restart if needed
vrooli resource searxng manage restart
```

### Rate Limiting
```python
# Implement exponential backoff
def handle_rate_limit(response):
    if response.status_code == 429:
        retry_after = int(response.headers.get('Retry-After', 60))
        time.sleep(retry_after)
        return True
    return False
```

### JSON Parsing Errors
```python
# Safe JSON parsing
def safe_parse(response_text):
    try:
        return json.loads(response_text)
    except json.JSONDecodeError:
        logger.error(f"Invalid JSON: {response_text[:100]}")
        return {'error': 'Invalid response format'}
```

## Additional Resources

- [SearXNG Official Documentation](https://docs.searxng.org/)
- [n8n Workflow Documentation](https://docs.n8n.io/)
- [Ollama API Reference](https://ollama.ai/docs/api)
- [LangChain Documentation](https://docs.langchain.com/)

## Support

For issues or questions about these integrations:
1. Check the troubleshooting section above
2. Review logs: `vrooli resource searxng logs`
3. Verify health: `vrooli resource searxng status`
4. Test manually: `curl http://localhost:8280/search?q=test&format=json`