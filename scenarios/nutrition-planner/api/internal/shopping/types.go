package shopping

import (
	"context"
	"errors"
	"fmt"
	"time"

	"nutrition-planner/internal/cost"
	"nutrition-planner/internal/decimalx"
	"nutrition-planner/internal/inventory"
	planning "nutrition-planner/internal/planning"
	recipe "nutrition-planner/internal/recipe"
)

type Line struct {
	Key             string           `json:"key"`
	Label           string           `json:"label"`
	Need            string           `json:"need"`
	Stock           string           `json:"stock"`
	Missing         string           `json:"missing"`
	PackageCount    string           `json:"packageCount"`
	Price           string           `json:"price"`
	PortionCost     string           `json:"portionCost,omitempty"`
	CheckoutTotal   string           `json:"checkoutTotal,omitempty"`
	ActualSpend     string           `json:"actualSpend,omitempty"`
	ItemID          string           `json:"-"`
	Unit            string           `json:"-"`
	Amount          decimalx.Decimal `json:"-"`
	SourceRecipes   []string         `json:"sourceRecipeIds,omitempty"`
	Checked         bool             `json:"checked"`
	HaveThis        bool             `json:"haveThis"`
	ActualQuantity  string           `json:"actualQuantity,omitempty"`
	ActualUnit      string           `json:"actualUnit,omitempty"`
	ActualPrice     string           `json:"actualPrice,omitempty"`
	PurchaseOmitted bool             `json:"purchaseOmitted,omitempty"`
}

type Change struct {
	Key    string `json:"key"`
	Before *Line  `json:"before,omitempty"`
	After  *Line  `json:"after,omitempty"`
}

// Diff reports ingredient lines whose demand identity enters or leaves a plan.
// It carries each source line intact so unknown stock, package, and price data
// stay explicit in a review instead of being guessed from recipe names.
func Diff(before, after []Line) []Change {
	previous := make(map[string]Line, len(before))
	next := make(map[string]Line, len(after))
	for _, line := range before {
		previous[line.Key] = line
	}
	for _, line := range after {
		next[line.Key] = line
	}
	changes := make([]Change, 0)
	for key, line := range previous {
		if _, remains := next[key]; !remains {
			copy := line
			changes = append(changes, Change{Key: key, Before: &copy})
		}
	}
	for key, line := range next {
		old, existed := previous[key]
		if !existed {
			copy := line
			changes = append(changes, Change{Key: key, After: &copy})
		} else if !sameImpact(old, line) {
			beforeCopy, afterCopy := old, line
			changes = append(changes, Change{Key: key, Before: &beforeCopy, After: &afterCopy})
		}
	}
	for i := 0; i < len(changes); i++ {
		for j := i + 1; j < len(changes); j++ {
			if changes[j].Key < changes[i].Key {
				changes[i], changes[j] = changes[j], changes[i]
			}
		}
	}
	return changes
}

func sameImpact(a, b Line) bool {
	return a.Need == b.Need && a.Stock == b.Stock && a.Missing == b.Missing && a.PackageCount == b.PackageCount && a.Price == b.Price && a.PortionCost == b.PortionCost && a.CheckoutTotal == b.CheckoutTotal && a.ActualSpend == b.ActualSpend
}

type Repository interface {
	Checked(context.Context, string) (map[string]bool, error)
	SetChecked(context.Context, string, string, bool) error
	HaveThis(context.Context, string) (map[string]bool, error)
	SetHaveThis(context.Context, string, string, bool) error
	ConfirmPurchases(context.Context, string, string, []PurchaseLine) error
	LatestPurchaseReview(context.Context, string) (map[string]PurchaseLine, error)
}

// PurchaseLine is the user's reviewed actual for one derived grocery row.
// Omitted rows are retained with Omitted=true and never create stock events.
type PurchaseLine struct {
	Key, ItemID string
	Amount      decimalx.Decimal
	Unit, Price string
	Omitted     bool
}

// PersistedState is the user-authored portion of the shopping domain that is
// safe to include in a workspace backup. It deliberately excludes inventory.
type PersistedState struct {
	Checks   map[string]bool   `json:"checks"`
	HaveThis map[string]bool   `json:"haveThis"`
	Reviews  []PersistedReview `json:"reviews"`
}

type PersistedReview struct {
	ID          string          `json:"id"`
	PayloadHash string          `json:"payloadHash"`
	CreatedAt   string          `json:"createdAt"`
	Lines       []PersistedLine `json:"lines"`
}

type PersistedLine struct {
	Key     string `json:"key"`
	ItemID  string `json:"itemId"`
	Amount  string `json:"amount"`
	Unit    string `json:"unit"`
	Price   string `json:"price"`
	Omitted bool   `json:"omitted"`
}

type PortableRepository interface {
	Repository
	PersistedState(context.Context, string) (PersistedState, error)
}

type Evidence struct {
	Inventory inventory.Repository
	Costs     cost.Repository
	Now       func() time.Time
}

func Derive(draft planning.Draft, recipes []recipe.Recipe, checked map[string]bool) []Line {
	byID := make(map[string]recipe.Recipe, len(recipes))
	for _, item := range recipes {
		byID[recipeRevisionKey(item.ID, item.Revision)] = item
	}
	lines := make(map[string]*Line)
	for _, occurrence := range draft.Occurrences {
		// Open, skipped, and social slots without a selected recipe are agenda
		// state, not ingredient demand. Keep them out of the grocery checklist.
		if occurrence.RecipeID == "" {
			continue
		}
		item, ok := byID[recipeRevisionKey(occurrence.RecipeID, occurrence.RecipeRevision)]
		if !ok {
			key := "recipe:" + occurrence.RecipeID
			if lines[key] == nil {
				lines[key] = &Line{Key: key, Label: "Ingredients for " + occurrence.RecipeName, Need: "unknown", Stock: "unknown", Missing: "unknown", PackageCount: "unknown", Price: "unknown", PortionCost: "unknown", CheckoutTotal: "unknown", ActualSpend: "unknown", Amount: decimalx.Unknown}
			}
			lines[key].SourceRecipes = appendUnique(lines[key].SourceRecipes, occurrence.RecipeID)
			continue
		}
		inputs := make(map[string]bool)
		ingredients := make(map[string]recipe.Ingredient)
		for _, ingredient := range item.Ingredients {
			id := ingredient.ID
			if id == "" {
				id = ingredient.Name
			}
			if id != "" {
				ingredients[id] = ingredient
			}
			if ingredient.Name != "" {
				ingredients[ingredient.Name] = ingredient
			}
		}
		for _, method := range item.Methods {
			for _, step := range method.Steps {
				for _, input := range step.Inputs {
					if input != "" {
						inputs[input] = true
					}
				}
			}
		}
		if len(inputs) == 0 {
			inputs["recipe:"+item.ID] = true
		}
		for input := range inputs {
			key := "ingredient:" + input
			if lines[key] == nil {
				lines[key] = &Line{Key: key, Label: input, Need: "unknown", Stock: "unknown", Missing: "unknown", PackageCount: "unknown", Price: "unknown", PortionCost: "unknown", CheckoutTotal: "unknown", ActualSpend: "unknown", ItemID: input, Amount: decimalx.Unknown}
			}
			if ingredient, found := ingredients[input]; found {
				amount, amountErr := decimalx.Parse(ingredient.Amount)
				yield, yieldErr := decimalx.Parse(item.CanonicalYield)
				quantity, quantityErr := decimalx.Parse(occurrence.Quantity)
				if amountErr == nil && yieldErr == nil && quantityErr == nil && !yield.IsZero() && ingredient.Unit != "" {
					scaled, _ := decimalx.Mul(amount, quantity)
					scaled, _ = decimalx.Div(scaled, yield)
					line := lines[key]
					if line.Unit == "" {
						line.Unit = ingredient.Unit
						line.Amount = scaled
					} else if line.Unit == ingredient.Unit && !line.Amount.IsUnknown() {
						line.Amount, _ = decimalx.Add(line.Amount, scaled)
					} else {
						line.Amount = decimalx.Unknown
					}
					if !line.Amount.IsUnknown() {
						line.Need = line.Amount.String() + " " + line.Unit
					}
				} else {
					lines[key].Amount = decimalx.Unknown
				}
			}
			lines[key].SourceRecipes = appendUnique(lines[key].SourceRecipes, item.ID)
		}
	}
	out := make([]Line, 0, len(lines))
	for _, line := range lines {
		line.Checked = checked[line.Key]
		if line.PortionCost == "" {
			line.PortionCost = "unknown"
		}
		if line.CheckoutTotal == "" {
			line.CheckoutTotal = "unknown"
		}
		if line.ActualSpend == "" {
			line.ActualSpend = "unknown"
		}
		out = append(out, *line)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Key < out[i].Key {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func LoadSelectedRecipes(ctx context.Context, recipes recipe.Service, workspaceID string, draft planning.Draft) ([]recipe.Recipe, error) {
	selected := make(map[string]recipe.Recipe)
	for _, occurrence := range draft.Occurrences {
		if occurrence.RecipeID == "" || occurrence.RecipeRevision <= 0 {
			continue
		}
		key := recipeRevisionKey(occurrence.RecipeID, occurrence.RecipeRevision)
		if _, exists := selected[key]; exists {
			continue
		}
		item, err := recipes.GetRevision(ctx, occurrence.RecipeID, workspaceID, occurrence.RecipeRevision)
		if err != nil {
			var missing recipe.ErrNotFound
			if errors.As(err, &missing) {
				continue
			}
			return nil, err
		}
		selected[key] = item
	}
	out := make([]recipe.Recipe, 0, len(selected))
	for _, item := range selected {
		out = append(out, item)
	}
	return out, nil
}

// Enrich adds only current, unconditional inventory and price evidence. Any
// mismatch or stale evidence remains unknown instead of being converted to zero.
func Enrich(ctx context.Context, workspaceID string, lines []Line, inventoryRepo inventory.Repository, costRepo cost.Repository, now time.Time) ([]Line, error) {
	events, err := inventoryRepo.List(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	stockByItem := map[string]cost.Stock{}
	for _, line := range lines {
		if line.ItemID == "" || line.Unit == "" {
			continue
		}
		amount, unit, known := decimalx.KnownInt(0), "", true
		for _, event := range events {
			if event.ItemID != line.ItemID || event.Kind == inventory.BatchPortion || event.Kind == inventory.PortionUndo || event.Kind == inventory.YieldCorrection {
				continue
			}
			if unit == "" {
				unit = event.Unit
			}
			if unit != event.Unit {
				known = false
				break
			}
			value := event.Amount
			if event.Kind == inventory.Waste || event.Kind == inventory.Preparation {
				value, _ = decimalx.Mul(value, decimalx.KnownInt(-1))
			}
			amount, _ = decimalx.Add(amount, value)
		}
		if known && unit == line.Unit {
			stockByItem[line.ItemID] = cost.Stock{Amount: amount, Unit: unit}
		} else {
			stockByItem[line.ItemID] = cost.Stock{Amount: decimalx.Unknown, Unit: line.Unit}
		}
	}
	for i := range lines {
		line := &lines[i]
		if line.Amount.IsUnknown() || line.Unit == "" || line.ItemID == "" {
			continue
		}
		stock, exists := stockByItem[line.ItemID]
		if !exists {
			stock = cost.Stock{Amount: decimalx.Unknown, Unit: line.Unit}
		}
		observations, err := costRepo.List(ctx, workspaceID, line.ItemID)
		if err != nil {
			return nil, err
		}
		var offers []cost.Package
		currency := ""
		exponent := -1
		mixedCurrency := false
		for _, offer := range observations {
			if !offer.Available || offer.MembershipRequired || offer.CouponRequired || offer.MinimumBuy != "" || offer.PackageUnit != line.Unit || now.Before(offer.ObservedAt) || now.Sub(offer.ObservedAt) > 30*24*time.Hour || (!offer.ValidThrough.IsZero() && now.After(offer.ValidThrough)) {
				continue
			}
			if currency == "" {
				currency = offer.Price.Currency
				exponent = offer.Price.Exponent
			} else if currency != offer.Price.Currency || exponent != offer.Price.Exponent {
				mixedCurrency = true
			}
			offers = append(offers, cost.Package{ID: offer.ID, ItemID: offer.ItemID, Amount: offer.PackageAmount, Unit: offer.PackageUnit, Price: offer.Price, Observed: offer.ObservedAt.Format(time.RFC3339)})
		}
		if mixedCurrency {
			offers = nil
		}
		result, err := cost.Calculate(cost.Requirement{ItemID: line.ItemID, Amount: line.Amount, Unit: line.Unit}, stock, offers)
		if err != nil {
			return nil, err
		}
		if !result.Available.IsUnknown() {
			line.Stock = result.Available.String() + " " + line.Unit
		}
		if !result.Missing.IsUnknown() {
			line.Missing = result.Missing.String() + " " + line.Unit
		}
		if !result.PackagesToBuy.IsUnknown() {
			line.PackageCount = result.PackagesToBuy.String()
		}
		if len(offers) > 0 {
			selected := offers[0]
			for _, offer := range offers[1:] {
				if offer.Price.Minor < selected.Price.Minor {
					selected = offer
				}
			}
			line.Price = selected.Price.String() + " / " + selected.Amount.String() + " " + selected.Unit
		}
		line.PortionCost = formatMinor(result.AllocatedMinorCost, result.Currency, offers)
		line.CheckoutTotal = formatMinor(result.CheckoutMinorCost, result.Currency, offers)
	}
	return lines, nil
}

func formatMinor(value decimalx.Decimal, currency string, offers []cost.Package) string {
	if value.IsUnknown() || currency == "" {
		return "unknown"
	}
	exponent := 0
	for _, offer := range offers {
		if offer.Price.Currency == currency {
			exponent = offer.Price.Exponent
			break
		}
	}
	denominator := int64(1)
	for i := 0; i < exponent; i++ {
		denominator *= 10
	}
	major, err := decimalx.Div(value, decimalx.KnownInt(denominator))
	if err != nil {
		return "unknown"
	}
	return major.String() + " " + currency
}

func recipeRevisionKey(id string, revision int64) string {
	return id + "@" + fmt.Sprint(revision)
}

func appendUnique(values []string, value string) []string {
	for _, item := range values {
		if item == value {
			return values
		}
	}
	return append(values, value)
}
