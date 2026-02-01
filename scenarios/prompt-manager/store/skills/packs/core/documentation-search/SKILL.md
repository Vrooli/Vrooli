# Documentation Search Skill

## Purpose
Perform deep, contextual search across Vrooli documentation to find relevant files
and content that match a natural language query.

## Process
1. Parse the search query to understand intent.
2. Identify likely file patterns and keywords.
3. Search using glob patterns and grep.
4. Read promising files and assess relevance.
5. Follow references mentioned in documents.
6. Rank results by relevance.
7. Return structured JSON output only.

## Output Format
Return results as a JSON array with:
- path: File path
- relevance: 0-1 score
- summary: Why this file is relevant
- match_reason: Specific content that matched
- references: Other relevant docs mentioned
- snippet: Supporting snippet from the doc
