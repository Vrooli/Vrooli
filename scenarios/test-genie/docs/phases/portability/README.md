# Portability

The portability phase evaluates declared resource platform support against the
operating system observed by the validator. An unsupported or incomplete
declaration is a failed target, not a passing measurement.

The phase is provider-backed by scenario-dependency-analyzer and is selected
for scenario targets. It is intentionally observational and is not eligible
for result caching.
