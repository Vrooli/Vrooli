# Audit Scope

Session constraints for audits and investigations. These boundaries ensure thorough analysis without unintended changes.

## Session Boundaries

### ALLOWED
- Reading and analyzing code
- Documenting findings
- Creating diagrams or visualizations
- Writing reports and recommendations
- Identifying patterns and anti-patterns
- Measuring metrics
- Running read-only tools (linters, analyzers)

### NOT ALLOWED
- Modifying any source code
- Changing configuration files
- Fixing issues found during audit
- Refactoring
- Adding tests
- Making commits

## Deliverables

An audit should produce:

1. **Findings**: Clear list of issues discovered
2. **Severity**: Priority/impact classification
3. **Recommendations**: Actionable suggestions for each finding
4. **Evidence**: Code references, metrics, or examples

## Audit Process

1. **Scope Definition**: Clarify what is being audited
2. **Analysis**: Systematic review of the target area
3. **Documentation**: Record all findings with evidence
4. **Prioritization**: Rank findings by impact/urgency
5. **Recommendations**: Propose solutions without implementing

## Verification Checklist

Before completing any audit task:
- [ ] No new changes made outside the target scope (changes may exist in other parts of the project due to parallel agents - leave these be)
- [ ] All findings documented
- [ ] Evidence provided for each finding
- [ ] Recommendations are actionable
- [ ] Severity/priority assigned to findings
