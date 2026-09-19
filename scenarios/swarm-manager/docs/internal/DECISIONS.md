# Decisions

## Priority polarity

`priority` has one scenario-wide polarity: lower values are more urgent and
sort first. The range is 0 through 10, where 0 is highest priority and 10 is
lowest. This matches backlog ordering and the UI dependency-sort contract.

Goal priorities formerly used the inverse comparison. The persisted values were
transformed through the Goal API (`p → 10-p`) when the comparison changed, so
the relative ordering of existing brownfield goals was preserved. Goals without
a stored priority were assigned the neutral value 5 through the same API.

Seeded goals must supply both a non-empty description and an explicit priority;
the seed path never turns omission into a ranked default.
