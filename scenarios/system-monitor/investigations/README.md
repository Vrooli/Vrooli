# Investigation Scripts

This directory contains reusable investigation scripts that can be run by OpenCode agents and system administrators to diagnose performance issues.

## Structure

```
investigations/
├── README.md              # This file
├── manifest.json          # Script registry and metadata
├── templates/              # Script templates for common patterns
│   ├── cpu-investigation.sh
│   ├── memory-investigation.sh
│   └── process-analysis.sh
├── active/                 # Ready-to-use investigation scripts
│   ├── high-cpu-analysis.sh
│   ├── memory-leak-detector.sh
│   └── process-genealogy.sh
└── results/               # Investigation results (auto-cleaned)
    └── [timestamp-script-name]/
```

## Script Format

Each script should include metadata in comments:

```bash
#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: High CPU Analysis
# DESCRIPTION: Identifies processes consuming excessive CPU and traces their origins
# CATEGORY: performance
# TRIGGERS: cpu_usage > 80%, load_average > 4.0
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-11
# LAST_MODIFIED: 2025-09-11
# VERSION: 1.0

# Your investigation logic here
```

## Usage by OpenCode Agents

OpenCode agents have full read/write access to this directory and should:

1. **Check existing scripts** before creating new ones
2. **Improve existing scripts** rather than duplicating
3. **Follow naming conventions**: `[problem-type]-[specific-focus].sh`
4. **Include proper metadata** in script headers
5. **Output structured data** (JSON preferred) for UI consumption
6. **Clean up temporary files** and respect the results retention policy

## Guidelines

- Scripts should be **idempotent** and **safe to run multiple times**
- Use **timeouts** for commands that might hang
- **Avoid destructive operations** - investigation only
- **Log all actions** for audit trail
- **Handle errors gracefully** with meaningful messages
