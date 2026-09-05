---
name: "documentation-search"
description: "Deep documentation discovery workflow for contextual search, reference following, and relevance ranking."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["search","documentation"]
  tags: ["skill"]
  icon: "search"
  status: "active"
  revision: 43
  createdAt: "2026-01-26T00:00:00Z"
  updatedAt: "2026-02-04T13:13:54Z"
  requires:
    scenarios: []
    commands: []
  origin:
    kind: "authored"
---
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
