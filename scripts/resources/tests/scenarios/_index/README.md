# Vrooli Resource Test Scenarios

## 📁 Structure Overview

This directory contains integration test scenarios for Vrooli's resource ecosystem. Each scenario is self-contained in its own directory with standardized files.

### Directory Structure
```
scenarios/
├── _index/                     # This directory - indexes and categorization
├── SCENARIO_TEMPLATE/          # Template for creating new scenarios
└── [scenario-name]/            # Individual test scenarios
    ├── metadata.yaml          # Machine-readable scenario configuration
    ├── README.md              # Human-readable documentation
    ├── test.sh                # Test implementation
    └── ui/                    # Optional UI components
        ├── deploy.sh          # UI deployment script
        ├── config.json        # UI configuration
        └── scripts/           # Backend scripts (TypeScript)
```

## 🏷️ Categories and Tags

Scenarios are organized using a flexible category and tag system defined in `categories.yaml`. This allows scenarios to:
- Belong to multiple categories as they evolve
- Be discovered by various criteria (resources, complexity, business value)
- Add new capabilities without restructuring directories

### Main Categories
- **ai-assistance**: AI-powered assistance and automation
- **ai-content-creation**: Content generation using AI
- **ai-automation**: Business process automation
- **ai-data-analysis**: Data processing and analysis
- **ai-recruitment**: HR and recruitment automation
- **ai-research**: Research and information gathering
- **real-time-monitoring**: Live monitoring and analytics
- **security**: Security and compliance scenarios

### Useful Tags
- `windmill-ui`: Scenarios with Windmill UI components
- `requires-ollama`: Scenarios using Ollama for LLM
- `enterprise-ready`: Production-ready scenarios
- `high-revenue-potential`: Scenarios with >$10k project potential
- `requires-display`: Scenarios needing graphical display
- `compliance-focused`: Security and compliance scenarios

## 🔍 Finding Scenarios

### By Category
```bash
# List all content creation scenarios
grep -A20 "ai-content-creation:" categories.yaml

# Find scenarios in multiple categories
grep -E "(ai-content-creation|ai-assistance):" -A10 categories.yaml
```

### By Resource
```bash
# Find all scenarios using Ollama
grep -B1 "ollama" */metadata.yaml

# Find scenarios using multiple resources
find . -name "metadata.yaml" -exec grep -l "whisper" {} \; | xargs grep -l "ollama"
```

### By Business Value
```bash
# Find high-revenue scenarios (>$10k)
find . -name "metadata.yaml" -exec grep -B2 -A2 "min: [0-9]\{5,\}" {} +

# Find enterprise-ready scenarios
grep -A5 "enterprise-ready:" categories.yaml
```

## 📝 Creating New Scenarios

1. **Copy the template**:
   ```bash
   cp -r SCENARIO_TEMPLATE/ my-new-scenario/
   ```

2. **Update metadata.yaml** with scenario details

3. **Implement test.sh** following the template structure

4. **Write README.md** with business context

5. **Add to categories.yaml** under appropriate categories and tags

6. **Test the scenario**:
   ```bash
   cd my-new-scenario && ./test.sh
   ```

## 🔌 Resource Integration Patterns

### Adding Resource Artifacts

For scenarios that need resource-specific artifacts (workflows, schemas, scripts), use this optional structure:

```
my-scenario/
├── metadata.yaml
├── test.sh
├── README.md
├── ui/                    # Optional: UI components
└── resources/             # Optional: Resource artifacts
    ├── n8n/              # n8n workflow JSON exports
    │   └── workflow.json
    ├── windmill/         # Windmill scripts and dependencies
    │   ├── main.py
    │   └── requirements.txt
    ├── postgres/         # Database schemas and data
    │   ├── schema.sql
    │   └── seed.sql
    ├── api/              # API testing collections
    │   ├── postman-collection.json
    │   └── environment.json
    └── config/           # Configuration files
        └── .env.template
```

### Declaring Artifacts in Metadata

Use the enhanced metadata.yaml format to declare artifacts:

```yaml
# Declare resource artifacts for automated setup
artifacts:
  n8n:
    workflows: ["workflow.json"]
  postgres:
    schema: "schema.sql"
    seed_data: "seed.sql"
  windmill:
    scripts: ["main.py"]
    workspace: "my-workspace"
```

### Using Resource Helpers in Tests

The test template includes helper functions for common resource operations:

```bash
# In your test.sh setup function:
setup_business_scenario() {
    # ... standard setup ...
    
    # Import all declared artifacts automatically
    import_all_artifacts
    
    # Or import specific resources manually
    import_n8n_workflow "workflow.json"
    init_postgres_database
    deploy_windmill_scripts
}
```

### Best Practices

1. **Keep artifacts organized**: Use subdirectories under `resources/` for each resource type
2. **Document integration steps**: Include setup instructions in your scenario README
3. **Use metadata declarations**: Declare artifacts in metadata.yaml for automated discovery
4. **Version control artifacts**: Include all necessary files for reproducibility
5. **Test independently**: Ensure scenarios work with fresh resource instances

## 🧪 Running Scenarios

### Individual Scenario
```bash
cd [scenario-name]
./test.sh
```

### By Category
```bash
# Run all scenarios in a category
for scenario in $(grep -A10 "ai-content-creation:" categories.yaml | grep "^\s*-" | sed 's/^\s*- //'); do
    echo "Running $scenario..."
    cd "$scenario" && ./test.sh && cd ..
done
```

### With Custom Configuration
```bash
# Extended timeout
TEST_TIMEOUT=1800 ./test.sh

# Skip cleanup for debugging
TEST_CLEANUP=false ./test.sh

# Custom resource URLs
OLLAMA_BASE_URL=http://remote:11434 ./test.sh
```

## 📊 Scenario Metadata Schema

Each scenario has a `metadata.yaml` file with:
- **scenario**: Basic info (id, name, description, version)
- **categories**: List of categories it belongs to
- **complexity**: basic, intermediate, or advanced
- **resources**: Required and optional resources (supports simple list or enhanced format with versions)
- **business**: Value proposition, revenue potential, target markets
- **testing**: Duration, timeout, special requirements
- **success_criteria**: Measurable outcomes
- **tags**: Additional classification tags
- **artifacts** (optional): Declare resource-specific files for automated setup
- **performance** (optional): Expected performance metrics (latency, throughput, resource usage)

## 🔧 Maintenance

### Adding a New Category
1. Edit `categories.yaml`
2. Add category with description and initial scenarios
3. Update this README

### Updating Resource Dependencies
1. Edit scenario's `metadata.yaml`
2. Update required/optional resources
3. Run scenario to verify

### Deprecating a Scenario
1. Add `deprecated: true` to metadata.yaml
2. Add deprecation notice to README
3. Update categories.yaml to remove from active lists