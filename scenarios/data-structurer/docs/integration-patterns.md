# Data Structurer Integration Patterns

This document outlines how other scenarios can integrate with data-structurer to leverage universal data transformation capabilities.

## 🔌 Integration Methods

### 1. REST API Integration

**Direct API Calls** - For scenarios with custom Go/Node.js/Python backends

```bash
# Create a schema for your use case
curl -X POST http://localhost:8080/api/v1/schemas \
  -H "Content-Type: application/json" \
  -d '{
    "name": "prospect-contacts",
    "description": "Extract contact information from prospect data",
    "schema_definition": {
      "type": "object",
      "properties": {
        "first_name": {"type": "string"},
        "last_name": {"type": "string"},
        "email": {"type": "string", "format": "email"},
        "company": {"type": "string"},
        "phone": {"type": "string"}
      },
      "required": ["first_name", "email"]
    }
  }'

# Process unstructured data
curl -X POST http://localhost:8080/api/v1/process \
  -H "Content-Type: application/json" \
  -d '{
    "schema_id": "<schema-uuid>",
    "input_type": "text",
    "input_data": "John Smith, CEO at TechCorp, john@techcorp.com, +1-555-123-4567"
  }'
```

### 2. CLI Integration

**Shell Scripts and Automation** - For scenarios using bash/shell scripting

```bash
#!/bin/bash

# Create schema from template
SCHEMA_ID=$(data-structurer create-from-template \
  "contact-person" "email-prospects" | jq -r '.id')

# Process a batch of files
for file in ./prospect-data/*.pdf; do
  data-structurer process "$SCHEMA_ID" "$file"
done

# Export structured data
data-structurer get-data "$SCHEMA_ID" --format csv > prospects.csv
```

### 3. n8n Workflow Integration

**Workflow Orchestration** - For scenarios using n8n automation

```json
{
  "nodes": [
    {
      "name": "HTTP Request - Create Schema",
      "type": "n8n-nodes-base.httpRequest",
      "parameters": {
        "url": "http://localhost:8080/api/v1/schemas",
        "method": "POST",
        "jsonBody": "{{ $json.schema_definition }}"
      }
    },
    {
      "name": "HTTP Request - Process Data",
      "type": "n8n-nodes-base.httpRequest",  
      "parameters": {
        "url": "http://localhost:8080/api/v1/process",
        "method": "POST",
        "jsonBody": {
          "schema_id": "{{ $('HTTP Request - Create Schema').first().json.id }}",
          "input_type": "text",
          "input_data": "{{ $json.raw_text }}"
        }
      }
    }
  ]
}
```

## 📋 Scenario-Specific Integration Examples

### Email Outreach Manager

**Use Case**: Process prospect lists from any format (LinkedIn exports, business cards, web scraping)

```bash
# 1. Create contact schema (or use template)
CONTACT_SCHEMA=$(data-structurer create-from-template \
  "contact-person" "outreach-prospects")

# 2. Process prospect data
for prospect_file in ./uploads/*; do
  RESULT=$(data-structurer process "$CONTACT_SCHEMA" "$prospect_file" --json)
  CONFIDENCE=$(echo "$RESULT" | jq -r '.confidence_score')
  
  # Only use high-confidence extractions
  if (( $(echo "$CONFIDENCE > 0.8" | bc -l) )); then
    echo "High confidence prospect: $prospect_file"
  fi
done

# 3. Export for outreach campaigns
data-structurer get-data "$CONTACT_SCHEMA" --format csv > outreach_list.csv
```

### Research Assistant

**Use Case**: Structure research documents and extract key information

```javascript
// API integration example for research-assistant
const createResearchSchema = async () => {
  const schema = {
    name: "research-documents",
    description: "Extract key information from research papers",
    schema_definition: {
      type: "object",
      properties: {
        title: {type: "string"},
        authors: {type: "array", items: {type: "string"}},
        abstract: {type: "string"},
        key_findings: {type: "array", items: {type: "string"}},
        methodology: {type: "string"},
        citations: {type: "array", items: {type: "string"}},
        publication_year: {type: "integer"}
      }
    }
  };
  
  const response = await fetch('http://localhost:8080/api/v1/schemas', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(schema)
  });
  
  return response.json();
};

const processResearchPaper = async (schemaId, pdfPath) => {
  const response = await fetch('http://localhost:8080/api/v1/process', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({
      schema_id: schemaId,
      input_type: "file",
      input_data: pdfPath
    })
  });
  
  return response.json();
};
```

### Competitor Change Monitor

**Use Case**: Structure competitor data from various sources (websites, press releases, SEC filings)

```python
import requests
import json

class CompetitorDataStructurer:
    def __init__(self, api_base="http://localhost:8080"):
        self.api_base = api_base
        self.company_schema_id = self._get_or_create_company_schema()
    
    def _get_or_create_company_schema(self):
        # Use existing company template
        templates_response = requests.get(f"{self.api_base}/api/v1/schema-templates")
        templates = templates_response.json()["templates"]
        
        company_template = next(
            (t for t in templates if t["name"] == "company-organization"), 
            None
        )
        
        if company_template:
            schema_data = {
                "name": "competitor-companies",
                "description": "Competitor company information extraction"
            }
            
            response = requests.post(
                f"{self.api_base}/api/v1/schemas/from-template/{company_template['id']}",
                json=schema_data
            )
            
            return response.json()["id"]
    
    def process_competitor_data(self, raw_data, source_type="text"):
        """Process competitor data from any source"""
        response = requests.post(
            f"{self.api_base}/api/v1/process",
            json={
                "schema_id": self.company_schema_id,
                "input_type": source_type,
                "input_data": raw_data
            }
        )
        
        return response.json()
    
    def get_all_competitor_data(self):
        """Retrieve all processed competitor data"""
        response = requests.get(
            f"{self.api_base}/api/v1/data/{self.company_schema_id}"
        )
        
        return response.json()["data"]

# Usage
structurer = CompetitorDataStructurer()

# Process competitor website content
website_content = scrape_competitor_website("https://competitor.com/about")
result = structurer.process_competitor_data(website_content)

# Process SEC filing
sec_filing = download_sec_filing("COMPETITOR-TICKER")  
result = structurer.process_competitor_data(sec_filing, "file")
```

### Resume Screening Assistant

**Use Case**: Process resumes in any format and extract standardized information

```bash
#!/bin/bash
# resume-processing-integration.sh

RESUME_SCHEMA_ID="resume-extraction"

# Create or get resume schema
if ! data-structurer get-schema "$RESUME_SCHEMA_ID" >/dev/null 2>&1; then
  echo "Creating resume extraction schema..."
  
  cat > /tmp/resume-schema.json << EOF
{
  "type": "object",
  "properties": {
    "candidate_name": {"type": "string"},
    "email": {"type": "string", "format": "email"},
    "phone": {"type": "string"},
    "experience_years": {"type": "integer"},
    "skills": {"type": "array", "items": {"type": "string"}},
    "education": {"type": "array", "items": {
      "type": "object",
      "properties": {
        "degree": {"type": "string"},
        "institution": {"type": "string"},
        "year": {"type": "integer"}
      }
    }},
    "work_history": {"type": "array", "items": {
      "type": "object", 
      "properties": {
        "company": {"type": "string"},
        "position": {"type": "string"},
        "duration": {"type": "string"},
        "achievements": {"type": "array", "items": {"type": "string"}}
      }
    }},
    "summary": {"type": "string"}
  }
}
EOF
  
  SCHEMA_RESULT=$(data-structurer create-schema \
    "$RESUME_SCHEMA_ID" /tmp/resume-schema.json \
    "Extract structured information from resumes")
  RESUME_SCHEMA_ID=$(echo "$SCHEMA_RESULT" | jq -r '.id')
fi

# Process resume directory
for resume in ./resumes/*; do
  echo "Processing: $resume"
  
  RESULT=$(data-structurer process "$RESUME_SCHEMA_ID" "$resume" --json)
  CONFIDENCE=$(echo "$RESULT" | jq -r '.confidence_score // 0')
  STATUS=$(echo "$RESULT" | jq -r '.status')
  
  if [[ "$STATUS" == "completed" ]] && (( $(echo "$CONFIDENCE > 0.7" | bc -l) )); then
    echo "✅ Successfully processed $resume (confidence: ${CONFIDENCE})"
  else
    echo "⚠️  Low confidence or failed: $resume"
  fi
done

# Export structured resume data
echo "Exporting structured resume data..."
data-structurer get-data "$RESUME_SCHEMA_ID" --format csv > structured_resumes.csv

echo "Resume processing complete. Results in: structured_resumes.csv"
```

## 🔄 Workflow Patterns

### Pattern 1: Schema Once, Use Everywhere

```bash
# 1. Create schema once per data type
SCHEMA_ID=$(data-structurer create-schema "financial-transactions" ./transaction-schema.json)

# 2. Use across multiple data sources
data-structurer process "$SCHEMA_ID" ./bank-statements/*.pdf
data-structurer process "$SCHEMA_ID" ./receipts/*.jpg  
data-structurer process "$SCHEMA_ID" ./invoices/*.docx

# 3. Single consistent output format
data-structurer get-data "$SCHEMA_ID" --format csv > all_transactions.csv
```

### Pattern 2: Template-Driven Development

```bash
# 1. Start with proven templates
data-structurer list-templates --category contacts
data-structurer list-templates --category business
data-structurer list-templates --category documents

# 2. Customize for your use case  
SCHEMA_ID=$(data-structurer create-from-template "contact-person" "crm-leads")

# 3. Immediate processing capability
data-structurer process "$SCHEMA_ID" ./lead-sources/*
```

### Pattern 3: Confidence-Based Processing

```bash
# Process with confidence filtering
process_with_confidence() {
  local schema_id="$1"
  local input_file="$2"
  local min_confidence="${3:-0.8}"
  
  RESULT=$(data-structurer process "$schema_id" "$input_file" --json)
  CONFIDENCE=$(echo "$RESULT" | jq -r '.confidence_score // 0')
  
  if (( $(echo "$CONFIDENCE >= $min_confidence" | bc -l) )); then
    echo "✅ High confidence result"
    return 0
  else
    echo "⚠️  Low confidence ($CONFIDENCE), manual review needed"
    return 1
  fi
}
```

## 🎛️ Environment Configuration

### Environment Variables

```bash
# Set in your scenario's environment
export DATA_STRUCTURER_API_URL="http://localhost:8080"
export DATA_STRUCTURER_DEFAULT_CONFIDENCE="0.8"
export DATA_STRUCTURER_BATCH_SIZE="10"
```

### Configuration Files

```yaml
# ~/.vrooli/data-structurer/config.yaml
api:
  base_url: "http://localhost:8080"
  timeout: 30
  retry_attempts: 3

processing:
  default_confidence_threshold: 0.8
  batch_size: 10
  async_processing: true

schemas:
  auto_create_from_templates: true
  default_templates:
    - "contact-person"
    - "document-metadata" 
    - "company-organization"
```

## 🚀 Performance Considerations

### Batch Processing

```bash
# Efficient batch processing
batch_process() {
  local schema_id="$1"
  local input_dir="$2"
  local batch_size="${3:-5}"
  
  find "$input_dir" -type f | while IFS= read -r -d '' file; do
    data-structurer process "$schema_id" "$file" &
    
    # Limit concurrent processes
    (($(jobs -r | wc -l) >= batch_size)) && wait
  done
  
  wait # Wait for all background jobs
}
```

### Caching and Reuse

```bash
# Cache schema IDs to avoid repeated lookups
get_or_create_schema() {
  local schema_name="$1"
  local schema_file="$2"
  local cache_file="$HOME/.vrooli/data-structurer/schema_cache.json"
  
  # Check cache first
  if [[ -f "$cache_file" ]]; then
    CACHED_ID=$(jq -r ".\"$schema_name\" // empty" "$cache_file")
    if [[ -n "$CACHED_ID" ]]; then
      echo "$CACHED_ID"
      return
    fi
  fi
  
  # Create new schema
  RESULT=$(data-structurer create-schema "$schema_name" "$schema_file" --json)
  SCHEMA_ID=$(echo "$RESULT" | jq -r '.id')
  
  # Update cache
  mkdir -p "$(dirname "$cache_file")"
  echo "{}" | jq ".\"$schema_name\" = \"$SCHEMA_ID\"" > "$cache_file"
  
  echo "$SCHEMA_ID"
}
```

## 🔧 Troubleshooting Common Issues

### Schema Validation Errors

```bash
# Validate schema before creation
validate_schema() {
  local schema_file="$1"
  
  # Check JSON syntax
  if ! jq . "$schema_file" >/dev/null 2>&1; then
    echo "❌ Invalid JSON syntax"
    return 1
  fi
  
  # Check required properties
  if ! jq -e '.type and .properties' "$schema_file" >/dev/null; then
    echo "❌ Schema must have 'type' and 'properties'"
    return 1
  fi
  
  echo "✅ Schema validation passed"
  return 0
}
```

### Processing Failures

```bash
# Robust processing with retry
process_with_retry() {
  local schema_id="$1"
  local input_data="$2"
  local max_retries="${3:-3}"
  local retry_count=0
  
  while [[ $retry_count -lt $max_retries ]]; do
    RESULT=$(data-structurer process "$schema_id" "$input_data" --json)
    STATUS=$(echo "$RESULT" | jq -r '.status')
    
    if [[ "$STATUS" == "completed" ]]; then
      echo "$RESULT"
      return 0
    fi
    
    echo "⚠️  Attempt $((retry_count + 1)) failed, retrying..."
    ((retry_count++))
    sleep 2
  done
  
  echo "❌ Processing failed after $max_retries attempts"
  return 1
}
```

---

**Remember**: Data-structurer is designed to be the universal data transformation layer for Vrooli. Every scenario that processes unstructured data should integrate with it rather than building custom parsers. This creates compound intelligence where each scenario's data handling becomes available to all others.