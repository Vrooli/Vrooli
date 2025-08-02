# Resume Screening Assistant

## 🎯 Overview

AI-powered recruitment automation system that parses resumes, evaluates candidates, and matches them to job requirements using advanced NLP and semantic search capabilities.

## 📋 Prerequisites

- Required resources: `unstructured-io`, `ollama`, `qdrant`
- Optional resources: `browserless` (for LinkedIn profile enrichment)

## 🚀 Quick Start

```bash
# Run the test scenario
./test.sh

# Run with custom timeout
TEST_TIMEOUT=900 ./test.sh
```

## 💼 Business Value

### Use Cases
- Automated resume screening at scale
- Candidate ranking and scoring
- Skills gap analysis
- Diversity and inclusion metrics

### Target Market
- HR departments processing high volumes
- Recruiting agencies
- Staffing firms
- Enterprise talent acquisition teams

### Revenue Potential
- Project range: $2,500 - $6,000
- Monthly SaaS: $500 - $2,000/month

## 🔧 Technical Details

### Architecture
1. **Document Processing**: Unstructured-IO extracts text and structure from resumes
2. **AI Analysis**: Ollama evaluates qualifications and scores candidates
3. **Vector Storage**: Qdrant enables semantic search across candidate pool
4. **Web Enrichment**: Optional Browserless for LinkedIn data

### Data Flow
1. Resume upload (PDF/DOCX)
2. Text extraction and parsing
3. Skill and experience identification
4. Candidate scoring against job requirements
5. Storage in searchable database
6. Matching and ranking results

## 🧪 Test Coverage

This scenario validates:
- ✅ Multiple resume format support
- ✅ Accurate data extraction
- ✅ AI-powered candidate scoring
- ✅ Semantic job matching
- ✅ Scalable processing

## 📊 Success Metrics

- Parsing accuracy: >95%
- Processing speed: <5 seconds per resume
- Matching relevance: >80% accuracy
- Volume capacity: 1000+ resumes/hour

## 🚧 Troubleshooting

| Issue | Solution |
|-------|----------|
| Parsing errors | Check document format compatibility |
| Low match scores | Refine job requirement prompts |
| Slow processing | Increase resource allocation |

## 🏷️ Tags

`hr-tech`, `recruitment`, `resume-parsing`, `candidate-matching`, `automation`