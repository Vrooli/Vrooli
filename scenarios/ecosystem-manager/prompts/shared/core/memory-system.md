# 🧠 Vrooli's Memory System

## Critical Understanding

Qdrant is Vrooli's **long-term memory** - it remembers every solution, pattern, failure, and capability. Always search first to learn from past work and avoid repeating mistakes.

## MANDATORY: Search Before Starting

```bash
# Try memory search first, but fallback to file search if issues
vrooli resource qdrant search "your task keywords"

# If Qdrant is down/slow/returns bad results, immediately use file search
rg -i "error pattern" /home/matthalloran8/Vrooli --type md
grep -r "integration approach" /home/matthalloran8/Vrooli/scenarios
find /home/matthalloran8/Vrooli -name "*keyword*" -type f
```

**Search for**: Similar implementations, known failures, best practices, integration examples, error patterns.

**Important**: If memory search fails, takes too long, or results look irrelevant, skip it and use standard file search instead. Don't waste time troubleshooting Qdrant - file search is reliable.

## Memory-Driven Development

Search first, then adapt proven approaches. Avoid documented failures. Every solution builds on past work.