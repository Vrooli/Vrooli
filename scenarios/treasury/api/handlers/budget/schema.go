// Package budget is the API composition boundary for spend budgets.
package budget

import domain "treasury/internal/budget"

func Schema() string { return domain.Schema() }
