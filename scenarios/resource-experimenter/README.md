# Resource Experimenter

## 🧪 Purpose & Strategic Value

Resource Experimenter is a **meta-scenario** that accelerates Vrooli's evolution by systematically exploring how new resources can enhance existing scenarios. This is a force-multiplier for the entire ecosystem:

- **Rapid prototyping**: Test resource combinations without manual coding
- **Discovery engine**: Find unexpected synergies between resources and scenarios  
- **Revenue multiplier**: Each successful experiment creates a new variant worth $10K-50K
- **Learning accelerator**: Generated experiments become training data for future AI agents

## 🎯 Cross-Scenario Impact

Resource Experimenter is critical infrastructure that other scenarios depend on:

- **prompt-manager**: Uses experiments to discover better prompt structures
- **agent-metareasoning-manager**: Learns from experiment results to improve agent coordination
- **product-manager**: Analyzes experiments to identify high-value feature combinations
- **scenario-generator**: Incorporates successful experiments as templates for new scenarios

## 🔧 Core Features

- **AI-powered copying**: Uses Claude Code to intelligently copy and modify scenarios
- **Resource integration**: Systematically adds new resources to existing scenarios
- **Experiment tracking**: PostgreSQL database tracks all experiments and results
- **Template system**: Successful experiments become reusable templates
- **Async processing**: Non-blocking experiment execution for parallel exploration

## 🎨 UI/UX Style

**Professional lab interface** - Clean, scientific aesthetic:
- Dashboard showing active experiments with progress bars
- Resource compatibility matrix visualization
- Side-by-side comparison of original vs modified scenarios
- Real-time logs streaming from Claude Code
- Success rate metrics and resource impact charts

## 🔌 Dependencies

### Required Resources
- **claude-code**: AI assistant for scenario modification (core dependency)
- **postgres**: Experiment tracking and results storage

### Recommended Resources (for enhanced experiments)
- **ollama**: For scenarios needing local LLM capabilities
- **n8n**: For workflow automation in modified scenarios
- **browserless**: For web-based validation of experiments

## 📂 Structure

```
resource-experimenter/
├── .vrooli/service.json      # Resource configuration
├── api/                      # Go API for experiment management
├── cli/                      # Command-line interface
├── initialization/           
│   └── postgres/            # Database schema and templates
├── prompts/                 # Claude Code prompts for modifications
├── ui/                      # React dashboard
└── README.md               # This file
```

## 🚀 Usage Examples

### CLI Commands
```bash
# List available scenarios for experimentation
resource-experimenter list-scenarios

# Start a new experiment
resource-experimenter experiment \
  --scenario "invoice-generator" \
  --resource "comfyui" \
  --prompt "Add visual invoice templates with AI-generated logos"

# Check experiment status
resource-experimenter status <experiment-id>

# Apply successful experiment to create new scenario
resource-experimenter apply <experiment-id> --name "visual-invoice-generator"
```

### API Endpoints
- `GET /api/experiments` - List all experiments
- `POST /api/experiments` - Create new experiment
- `GET /api/experiments/{id}` - Get experiment details
- `GET /api/templates` - List successful experiment templates
- `GET /api/scenarios` - List available scenarios for experimentation

## 🔄 Integration Points

### Incoming Integrations
- **scenario-generator** calls this to test resource combinations
- **agent-metareasoning-manager** uses this for capability expansion
- **product-manager** queries this for feature possibility analysis

### Outgoing Integrations
- Calls **claude-code** resource for AI-powered modifications
- Stores results in **postgres** for persistence
- Can trigger **n8n** workflows for validation (future enhancement)

## 📊 Success Metrics

- **Experiment success rate**: >70% of experiments should produce working scenarios
- **Resource coverage**: Each resource should be tested with 5+ scenarios
- **Template reuse**: Successful experiments used 3+ times as templates
- **Time to experiment**: <5 minutes from request to modified scenario

## 🔮 Future Enhancements

- N8n workflow for automated validation of modified scenarios
- Integration with browserless for visual testing
- Machine learning to predict resource compatibility
- Automatic generation of documentation for new scenario variants
- Webhook notifications when experiments complete