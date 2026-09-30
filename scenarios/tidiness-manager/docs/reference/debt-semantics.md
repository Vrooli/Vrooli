# Tidiness debt semantics

Tidiness Manager exposes two related but deliberately different measures.

`DebtScore` is the shared maturity count of WARNING and INFO findings that are
not marked `uncheckable`. `DUPLICATED_BOILERPLATE` is uncheckable: it remains
visible but does not increase DebtScore.

`duplication_line_debt` measures actionable duplication only. For an
`opportunity`, it is the normalized group's largest physical span multiplied by
the number of extra locations. A `high-leverage` cross-package opportunity uses
a multiplier of two. `structural` and `incidental` groups have zero line debt.

Consequently, a scenario can show visible structural repetition while reporting
zero duplication debt. Conversely, a small number of long cross-boundary
opportunities can carry more debt than many short groups. Budgets and ratchets
should use `duplication_line_debt`, not raw detector-group count.

The producer ranks root-cause opportunities by total line debt, with
high-leverage opportunities first on ties. Raw clone locations remain evidence;
the ranked opportunities are the recommended remediation queue.

[CODE: api/file_role.go]
[CODE: api/duplication_opportunities.go]
