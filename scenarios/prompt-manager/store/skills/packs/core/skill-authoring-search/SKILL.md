## Meta focus: Search Skill Authoring

Guide for creating **search** skills (the authored skill declares `modes[0] = "search"`). Search skills focus on discovery, mapping, and evidence gathering, not implementation.

Required reading:
- `docs/agent-system/SKILL_AUTHORING.md`

---

### **1. Category Scope**

**In scope:**
- Finding relevant documentation, code, or artifacts
- Tracing how something is implemented
- Mapping relationships between files, concepts, or systems
- Ranking and summarizing evidence

**Out of scope:**
- Writing or changing code
- Architectural decisions or refactors
- Feature design beyond what the evidence supports

---

### **2. Recommended Structure**

Search skills should be short and operational. Include:

1. **Focus statement** - what the search is trying to discover
2. **Search surfaces** - where to look first (repo, docs, logs, external)
3. **Query strategy** - keywords, synonyms, and constraints
4. **Evidence handling** - how to verify relevance and avoid speculation
5. **Output contract** - exact format (JSON, table, list)
6. **Stop conditions** - when to stop searching and report

---

### **3. Convergence Patterns**

Use simple decision trees or tables to make search order consistent.

Example decision tree:
```
Is the question about behavior in this repo?
  -> Start with rg/rg --files and local docs
Is it about external specs or APIs?
  -> Identify authoritative sources before secondary ones
No clear target?
  -> Expand keywords, then search references in docs
```

Example ranking table:

| Signal | Rank impact | Why it matters |
|---|---|---|
| Direct match in primary docs | High | Most authoritative evidence |
| Code reference + doc mention | High | Confirms implementation + intent |
| Only secondary references | Medium | Useful, but verify |
| No source, only inference | Low | Mark clearly as inference |

---

### **4. Evidence Handling Rules**

- Prefer primary sources (code, official docs, config) over secondary mentions
- If the skill requires quotes, limit to short excerpts and note the source
- If evidence is missing, say so and list what was searched

---

### **5. Output Contract**

Define the output format explicitly. Example JSON contract:

```
[
  {
    "path": "docs/reference/api-endpoints.md",
    "relevance": 0.9,
    "summary": "Defines the auth endpoints referenced in the query",
    "match_reason": "Contains /auth/login and /auth/logout",
    "references": ["docs/concepts/ARCHITECTURE.md"],
    "snippet": "POST /auth/login ..."
  }
]
```

If output is not JSON, specify the exact headings and required fields.

---

### **6. Anti-Gaming Guidance**

- Do not return huge lists without relevance ranking
- Do not invent sources or overstate confidence
- Do not skip verification when evidence is available

---

### **7. Registration Notes**

Registration follows `docs/agent-system/SKILL_AUTHORING.md` §"Registration and metadata"; the authored skill declares `modes[0] = "search"` and a description that reflects the search intent and output contract.

---

### **8. Output Expectations**

You may update:
- Search skills to improve clarity, accuracy, or output contracts
- `skill.json` entries for Search skills

You must:
- Keep search skills evidence-driven and non-speculative
- Define a clear output contract and stop conditions
- Avoid prescribing implementation or code changes
