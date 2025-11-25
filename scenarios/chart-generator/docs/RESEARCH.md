# Research & References

## Testing Architecture

### Phased Testing System
- Located in `/home/matthalloran8/Vrooli/scripts/scenarios/testing/`
- Phases: structure → dependencies → unit → integration → business → performance
- Each phase has dedicated script in `test/phases/`

### Requirements Validation
- Requirements must reference actual test files relative to scenario root
- Valid test locations:
  - `api/**/*_test.go` - Go unit tests
  - `ui/src/**/*.test.tsx` - React component tests
  - `cli/*.bats` - CLI integration tests
  - `test/playbooks/**/*.json` - UI automation workflows

### Completeness Scoring
- Penalties for:
  - Monolithic test files (≥4 requirements per file)
  - Lack of multi-layer validation (P0/P1 should have 2+ test layers)
  - 100% 1:1 operational target mapping (suggests auto-generation)
- Scoring components:
  - Quality metrics (50pts): requirements passing %, operational targets %, tests passing %
  - Coverage metrics (15pts): test coverage ratio, validation depth score
  - Quantity metrics (10pts): requirement count, target count, test count
  - UI metrics (25pts): template type, file count, API integration, routing, LOC

## Chart Generation Stack

### D3.js Choice
Selected for maximum flexibility and professional output quality over:
- Chart.js (simpler but less customizable)
- Plotly (built-in interactivity but less control)

### Export Pipeline
- PNG: Browserless (headless Chrome) with fallback to Go native generation
- SVG: Native D3.js output
- PDF: Vector graphics support for print quality

### Performance Targets
- <2000ms for complex charts (1000+ data points)
- Current: 15-18ms typical generation time

## Related Scenarios
- browser-automation-studio - Reference for proper test structure and playbook organization
- business-reports - Downstream consumer of chart generation
- research-assistant - Downstream consumer for publication-quality charts

## External References
- [D3.js Documentation](https://d3js.org/)
- [WCAG 2.1 Accessibility Guidelines](https://www.w3.org/WAI/WCAG21/quickref/)
- [Phased Testing Architecture](/home/matthalloran8/Vrooli/docs/testing/architecture/PHASED_TESTING.md)
