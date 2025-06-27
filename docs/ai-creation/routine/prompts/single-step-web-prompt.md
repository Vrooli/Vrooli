# RoutineWeb (Web Search) Prompt

## Overview

You are an expert AI system designer tasked with creating **RoutineWeb** routines for Vrooli. These are single-step routines that perform web searches and information retrieval from the internet. They serve as atomic building blocks for research, fact-checking, and real-time information gathering within larger workflows.

## Purpose

RoutineWeb routines are used for:
- **Research tasks**: Gather information on specific topics
- **Fact-checking**: Verify claims and statements against web sources
- **Real-time data**: Retrieve current information and news
- **Market research**: Analyze trends and competitive intelligence
- **Content discovery**: Find relevant articles, papers, and resources

## Execution Context

### Tier 3 Execution
RoutineWeb operates at **Tier 3** of Vrooli's architecture:
- **Search engine integration**: Direct queries to major search engines
- **Result processing**: Extract and structure search results
- **Template-based queries**: Dynamic search query construction
- **Content extraction**: Parse and summarize web content

### Integration with Multi-Step Workflows
RoutineWeb routines are commonly used as subroutines within RoutineMultiStep workflows for:
- **Information gathering steps**: Collect data for analysis
- **Verification steps**: Validate information against web sources  
- **Research steps**: Find supporting evidence and context
- **Content enrichment steps**: Add external information to workflows

## JSON Structure Requirements

### Complete RoutineWeb Structure

```json
{
  "id": "[generate-19-digit-snowflake-id]",
  "publicId": "[generate-10-12-char-alphanumeric]",
  "resourceType": "Routine",
  "isPrivate": false,
  "permissions": "{}",
  "isInternal": false,
  "tags": [],
  "versions": [
    {
      "id": "[generate-19-digit-snowflake-id]",
      "publicId": "[generate-10-12-char-alphanumeric]", 
      "versionLabel": "1.0.0",
      "versionNotes": "Initial version",
      "isComplete": true,
      "isPrivate": false,
      "versionIndex": 0,
      "isAutomatable": true,
      "resourceSubType": "RoutineWeb",
      "config": {
        "__version": "1.0",
        "callDataWeb": {
          "__version": "1.0",
          "schema": {
            "queryTemplate": "{{input.searchTerm}} {{input.context}} site:{{input.targetSite}}",
            "searchEngine": "google",
            "maxResults": 10,
            "language": "en",
            "region": "us",
            "timeRange": "{{input.timeFilter}}",
            "contentType": "any",
            "outputMapping": {
              "searchResults": "results",
              "totalCount": "total",
              "searchQuery": "query",
              "searchMetadata": "metadata"
            },
            "extractContent": true,
            "summaryLength": 200
          }
        },
        "formInput": {
          "__version": "1.0",
          "schema": {
            "elements": [
              {
                "fieldName": "searchTerm",
                "id": "search_term_input",
                "label": "Search Term",
                "type": "TextInput",
                "isRequired": true,
                "placeholder": "Enter search query"
              }
            ]
          }
        },
        "formOutput": {
          "__version": "1.0", 
          "schema": {
            "elements": [
              {
                "fieldName": "searchResults",
                "id": "search_results_output",
                "label": "Search Results",
                "type": "JSON"
              },
              {
                "fieldName": "totalCount",
                "id": "total_count_output",
                "label": "Total Results Found",
                "type": "IntegerInput"
              }
            ]
          }
        },
        "executionStrategy": "deterministic"
      },
      "translations": [
        {
          "id": "[generate-19-digit-snowflake-id]",
          "language": "en",
          "name": "Descriptive Web Search Routine Name",
          "description": "What this web search routine finds and how it helps with research.",
          "instructions": "How to use this routine and what search terms to provide."
        }
      ]
    }
  ]
}
```

## Configuration Details

### callDataWeb Schema

#### Required Fields
- **`queryTemplate`**: Search query with template variables
- **`searchEngine`**: Search engine to use (google, bing, duckduckgo)
- **`maxResults`**: Maximum number of results to return

#### Optional Fields
- **`language`**: Search language (default: "en")
- **`region`**: Geographic region for results (default: "us")
- **`timeRange`**: Time filter for results
- **`contentType`**: Type of content to search
- **`outputMapping`**: How to map search results to output fields
- **`extractContent`**: Whether to extract full content
- **`summaryLength`**: Length of content summaries

### Query Template Configuration

#### Basic Search Query
```json
{
  "queryTemplate": "{{input.searchTerm}}"
}
```

#### Advanced Search Query
```json
{
  "queryTemplate": "{{input.topic}} {{input.modifier}} {{input.timeContext}}"
}
```

#### Site-Specific Search
```json
{
  "queryTemplate": "{{input.searchTerm}} site:{{input.targetSite}}"
}
```

#### Complex Search Query
```json
{
  "queryTemplate": "\"{{input.exactPhrase}}\" {{input.additionalTerms}} -{{input.excludeTerms}} filetype:{{input.fileType}}"
}
```

### Search Engine Options

#### Google Search
```json
{
  "searchEngine": "google",
  "language": "en",
  "region": "us"
}
```

#### Bing Search
```json
{
  "searchEngine": "bing",
  "language": "en",
  "region": "us"
}
```

#### DuckDuckGo Search
```json
{
  "searchEngine": "duckduckgo",
  "language": "en"
}
```

### Time Range Filters

#### Predefined Time Ranges
```json
{
  "timeRange": "{{input.timeFilter}}"
}
```

Options: "hour", "day", "week", "month", "year", "any"

#### Custom Time Range
```json
{
  "timeRange": "{{input.startDate}}..{{input.endDate}}"
}
```

### Content Type Filters

#### General Content
```json
{
  "contentType": "any"
}
```

#### Specific Content Types
```json
{
  "contentType": "{{input.contentFilter}}"
}
```

Options: "news", "images", "videos", "academic", "shopping", "books"

### Output Mapping Configuration

#### Standard Output Mapping
```json
{
  "outputMapping": {
    "searchResults": "results",
    "totalCount": "total",
    "searchQuery": "query",
    "searchMetadata": "metadata"
  }
}
```

#### Custom Output Mapping
```json
{
  "outputMapping": {
    "titles": "results[*].title",
    "urls": "results[*].url", 
    "summaries": "results[*].snippet",
    "domains": "results[*].domain",
    "relevanceScores": "results[*].score"
  }
}
```

## Common Use Cases & Templates

### 1. General Research
```json
{
  "callDataWeb": {
    "schema": {
      "queryTemplate": "{{input.researchTopic}} {{input.context}} {{input.year}}",
      "searchEngine": "google",
      "maxResults": 15,
      "language": "{{input.language}}",
      "timeRange": "{{input.timeFilter}}",
      "outputMapping": {
        "articles": "results",
        "totalFound": "total",
        "searchUsed": "query"
      },
      "extractContent": true,
      "summaryLength": 250
    }
  }
}
```

### 2. News and Current Events
```json
{
  "callDataWeb": {
    "schema": {
      "queryTemplate": "{{input.newsKeyword}} {{input.location}} news",
      "searchEngine": "google",
      "maxResults": 20,
      "contentType": "news",
      "timeRange": "{{input.timePeriod}}",
      "outputMapping": {
        "newsArticles": "results",
        "headlines": "results[*].title",
        "sources": "results[*].source"
      }
    }
  }
}
```

### 3. Academic Research
```json
{
  "callDataWeb": {
    "schema": {
      "queryTemplate": "{{input.academicTopic}} {{input.methodology}} site:scholar.google.com OR site:arxiv.org OR site:pubmed.ncbi.nlm.nih.gov",
      "searchEngine": "google",
      "maxResults": 25,
      "language": "en",
      "outputMapping": {
        "papers": "results",
        "abstracts": "results[*].snippet",
        "citations": "results[*].url"
      },
      "extractContent": true
    }
  }
}
```

### 4. Fact Checking
```json
{
  "callDataWeb": {
    "schema": {
      "queryTemplate": "\"{{input.claimToCheck}}\" fact check verification",
      "searchEngine": "google",
      "maxResults": 10,
      "contentType": "news",
      "outputMapping": {
        "factCheckResults": "results",
        "verificationSources": "results[*].source",
        "credibilityScores": "results[*].credibility"
      }
    }
  }
}
```

### 5. Competitive Intelligence
```json
{
  "callDataWeb": {
    "schema": {
      "queryTemplate": "{{input.companyName}} {{input.analysisType}} {{input.industry}} -site:{{input.excludeSite}}",
      "searchEngine": "google", 
      "maxResults": 30,
      "timeRange": "{{input.timeRange}}",
      "outputMapping": {
        "competitorInfo": "results",
        "marketAnalysis": "results[*].snippet",
        "businessNews": "results[*].title"
      },
      "extractContent": true,
      "summaryLength": 300
    }
  }
}
```

### 6. Technical Documentation Search
```json
{
  "callDataWeb": {
    "schema": {
      "queryTemplate": "{{input.technology}} {{input.problemDescription}} documentation tutorial",
      "searchEngine": "google",
      "maxResults": 12,
      "outputMapping": {
        "documentation": "results",
        "tutorials": "results[*].url",
        "examples": "results[*].snippet"
      },
      "extractContent": true
    }
  }
}
```

## Form Configuration

### Input Form Elements

#### Basic Search Parameters
```json
{
  "fieldName": "searchTerm",
  "type": "TextInput",
  "isRequired": true,
  "label": "Search Term",
  "placeholder": "Enter your search query"
},
{
  "fieldName": "context",
  "type": "TextInput",
  "label": "Additional Context",
  "placeholder": "Add context or modifiers"
}
```

#### Search Refinement
```json
{
  "fieldName": "timeFilter",
  "type": "Selector",
  "label": "Time Range",
  "options": ["any", "day", "week", "month", "year"],
  "defaultValue": "any"
},
{
  "fieldName": "contentType", 
  "type": "Selector",
  "label": "Content Type",
  "options": ["any", "news", "academic", "images", "videos"],
  "defaultValue": "any"
}
```

#### Advanced Options
```json
{
  "fieldName": "language",
  "type": "Selector",
  "label": "Language",
  "options": ["en", "es", "fr", "de", "zh", "ja"],
  "defaultValue": "en"
},
{
  "fieldName": "maxResults",
  "type": "IntegerInput",
  "label": "Maximum Results",
  "defaultValue": 10,
  "min": 1,
  "max": 100
}
```

### Output Form Elements

#### Search Results
```json
{
  "fieldName": "searchResults",
  "type": "JSON",
  "label": "Search Results"
},
{
  "fieldName": "totalCount",
  "type": "IntegerInput",
  "label": "Total Results Found"
},
{
  "fieldName": "searchQuery",
  "type": "TextInput",
  "label": "Actual Query Used"
}
```

#### Processed Results
```json
{
  "fieldName": "summaries",
  "type": "JSON",
  "label": "Content Summaries"
},
{
  "fieldName": "relevantUrls",
  "type": "JSON",
  "label": "Most Relevant URLs"
},
{
  "fieldName": "keyFindings",
  "type": "TextInput",
  "label": "Key Findings"
}
```

## Template Variables

### Input Variables
```json
{
  "queryTemplate": "{{input.mainTopic}} {{input.subtopic}} {{input.year}}"
}
```

### System Variables
```json
{
  "queryTemplate": "{{input.searchTerm}} {{userLanguage}} {{now().year}}"
}
```

### Conditional Variables
```json
{
  "queryTemplate": "{{input.searchTerm}}{{#if input.location}} {{input.location}}{{/if}}{{#if input.excludeTerms}} -{{input.excludeTerms}}{{/if}}"
}
```

## Execution Strategies

### Deterministic Strategy (Recommended)
**Best for**: Consistent research queries, fact-checking, data gathering
```json
{
  "executionStrategy": "deterministic"
}
```

### Reasoning Strategy  
**Best for**: Complex research requiring query refinement
```json
{
  "executionStrategy": "reasoning"
}
```

### Conversational Strategy
**Best for**: Interactive research with user feedback
```json
{
  "executionStrategy": "conversational"
}
```

## Validation Rules

### Critical Requirements
1. **Resource Type**: Must be `"Routine"` 
2. **Resource Sub Type**: Must be `"RoutineWeb"`
3. **Config Version**: Must have `"__version": "1.0"` at root level
4. **Call Data**: Must have `callDataWeb` with proper schema
5. **Query Template**: Must be valid search query with template variables
6. **Search Engine**: Must specify supported search engine
7. **Max Results**: Must be reasonable number (1-100)
8. **Forms**: Must have both `formInput` and `formOutput`
9. **IDs**: 19-digit snowflake IDs for all `id` fields
10. **Public IDs**: 10-12 character alphanumeric for `publicId` fields
11. **Tags**: Must be empty array `[]`

### Web-Specific Requirements
1. **Query Template**: Valid search query syntax
2. **Template Variables**: Proper `{{variable}}` syntax
3. **Search Engine**: Supported engine (google, bing, duckduckgo)
4. **Result Limits**: Reasonable maxResults value
5. **Output Mapping**: Valid JSON path expressions

### Quality Requirements  
1. **Query Design**: Clear, effective search queries
2. **Result Processing**: Appropriate output mapping
3. **Content Extraction**: Reasonable summary lengths
4. **Error Handling**: Handle search failures gracefully

## Generation Guidelines

### 1. Understand the Research Need
- **Information Type**: What kind of information is needed?
- **Sources**: Which websites or content types are most relevant?
- **Timeliness**: Is current or historical information needed?
- **Scope**: How broad or specific should the search be?

### 2. Design the Search Query
- **Keywords**: Choose effective search terms
- **Operators**: Use search operators for precision
- **Filters**: Apply appropriate content and time filters
- **Exclusions**: Remove irrelevant results

### 3. Configure Search Parameters
- **Search Engine**: Choose best engine for the use case
- **Result Count**: Balance completeness with performance
- **Content Processing**: Decide on summary and extraction needs

### 4. Plan Output Structure
- **Result Format**: How should results be structured?
- **Data Extraction**: What information to extract from results?
- **Metadata**: What search metadata is useful?

### 5. Consider User Experience
- **Input Simplicity**: Make searches easy to configure
- **Result Clarity**: Present results in usable format
- **Error Handling**: Provide helpful feedback on search issues

## Quality Checklist

Before generating a RoutineWeb routine, verify:

### Structure:
- [ ] `resourceSubType` is `"RoutineWeb"`
- [ ] `callDataWeb` schema is complete and valid
- [ ] Both `formInput` and `formOutput` are defined
- [ ] All IDs follow correct format
- [ ] `executionStrategy` matches the search type

### Search Configuration:
- [ ] Query template uses proper template variable syntax
- [ ] Search engine is supported and appropriate
- [ ] Maximum results is reasonable (1-100)
- [ ] Time and content filters are properly configured
- [ ] Output mapping extracts useful information

### Query Design:
- [ ] Search query is effective for the intended research
- [ ] Template variables are used appropriately
- [ ] Search operators improve precision
- [ ] Excluded terms filter irrelevant results

### Forms:
- [ ] Input form captures necessary search parameters
- [ ] Output form includes results and metadata
- [ ] Field names match template variables
- [ ] Required fields are properly marked

### User Experience:
- [ ] Search purpose is clear from name and description
- [ ] Instructions help users formulate effective queries
- [ ] Results are structured for easy consumption
- [ ] Error cases are handled appropriately

### Metadata:
- [ ] Name clearly describes the search purpose
- [ ] Description explains what information will be found
- [ ] Instructions guide users on effective search terms
- [ ] Version information is complete

Generate RoutineWeb routines that provide reliable, efficient web search capabilities while being easily composable into larger research and analysis workflows.