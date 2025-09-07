# Scenario Generator Backlog System

## Overview

This folder-based backlog system allows you to manage scenario generation requests through simple YAML files. You can add, edit, and prioritize scenarios manually or through the UI.

## Directory Structure

```
backlog/
├── pending/        # Scenarios waiting to be generated
├── in-progress/    # Currently being processed
├── completed/      # Successfully generated scenarios
├── failed/         # Failed generation attempts
├── templates/      # Template files for creating new items
└── README.md       # This file
```

## How to Use

### Adding a New Scenario to the Backlog

1. **Manual Method:**
   - Copy `templates/scenario-template.yaml` to `pending/`
   - Rename it with format: `XXX-descriptive-name.yaml` (XXX = priority number)
   - Edit the file with your scenario details
   - Save the file - it will appear in the UI automatically

2. **UI Method:**
   - Go to the "Backlog" tab in the web interface
   - Click "Add to Backlog"
   - Fill in the form
   - Submit to create the YAML file

### Priority System

Files are processed based on their filename prefix:
- `001-` through `099-` = HIGH priority
- `100-` through `199-` = MEDIUM priority  
- `200-` through `999-` = LOW priority

To change priority, simply rename the file with a different number prefix.

### File Format

Each backlog item is a YAML file with the following structure:

```yaml
id: unique-identifier
name: "Scenario Name"
description: "Brief description"
prompt: |
  Detailed prompt for AI generation
complexity: simple|intermediate|advanced
category: business-category
priority: high|medium|low
estimated_revenue: 25000
tags: [tag1, tag2, tag3]
metadata:
  requested_by: "Person/Team"
  requested_date: "YYYY-MM-DD"
  deadline: "YYYY-MM-DD"
resources_required: [postgres, redis, claude-code, etc]
validation_criteria:
  - "Criterion 1"
  - "Criterion 2"
notes: |
  Additional context
```

## Workflow States

### pending/
New scenarios waiting to be processed. Add your YAML files here.

### in-progress/
Scenarios currently being generated. Files are automatically moved here when generation starts.

### completed/
Successfully generated scenarios. Files include generation results and metadata.

### failed/
Failed generation attempts. Files include error messages and can be moved back to pending/ after fixes.

## Best Practices

1. **Use Clear Naming:** File names should be descriptive: `001-invoice-saas.yaml` not `001-thing.yaml`

2. **Detailed Prompts:** The more specific your prompt, the better the generated scenario

3. **Resource Planning:** List all required resources to ensure proper setup

4. **Validation Criteria:** Define clear success criteria for testing

5. **Version Control:** Commit backlog changes to Git for history tracking

## Bulk Operations

### Import Multiple Scenarios
```bash
# Copy multiple YAML files to pending/
cp ~/my-scenarios/*.yaml ./pending/
```

### Reorder Priorities
```bash
# Renumber all files with 10-increment spacing
i=10; for f in pending/*.yaml; do 
  mv "$f" "pending/$(printf "%03d" $i)-${f##*/}"; 
  i=$((i+10)); 
done
```

### Archive Old Items
```bash
# Move completed items older than 30 days to archive
find completed/ -name "*.yaml" -mtime +30 -exec mv {} archive/ \;
```

## Integration with CI/CD

The backlog can be automated through scripts:

```bash
# Add a new scenario from CI/CD
cat > pending/$(date +%s)-automated.yaml << EOF
id: automated-$(date +%s)
name: "Automated Scenario"
# ... rest of YAML
EOF
```

## Monitoring

Check backlog status:
```bash
# Count items in each state
echo "Pending: $(ls pending/*.yaml 2>/dev/null | wc -l)"
echo "In Progress: $(ls in-progress/*.yaml 2>/dev/null | wc -l)"
echo "Completed: $(ls completed/*.yaml 2>/dev/null | wc -l)"
echo "Failed: $(ls failed/*.yaml 2>/dev/null | wc -l)"
```

## Troubleshooting

### Scenario Not Appearing in UI
- Check YAML syntax: `yamllint pending/your-file.yaml`
- Ensure file has `.yaml` extension
- Verify file permissions are readable

### Generation Stuck
- Check `in-progress/` folder for stuck items
- Move back to `pending/` to retry
- Check API logs for errors

### Priority Not Working
- Ensure filename starts with 3-digit number: `001-`, not `1-` or `01-`
- Lower numbers = higher priority

## Tips

- **Quick Edit:** Use VS Code with YAML extension for syntax highlighting
- **Validation:** Install `yamllint` to validate files before adding
- **Templates:** Create custom templates in `templates/` for different scenario types
- **Backup:** Regularly backup the backlog folder to prevent data loss

---

For more information, see the main [Scenario Generator README](../README.md)