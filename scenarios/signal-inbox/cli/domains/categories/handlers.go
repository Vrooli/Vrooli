package categories

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	categoriesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/categories"
	categoriesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/categories/categories_v1connect"
)

type handlers struct {
	client categoriesconnect.CategoriesServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	client, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: categoriesconnect.NewCategoriesServiceClient(client, base)}
}

func (h *handlers) createCall(ctx cliapp.OperationContext) (*categoriesv1.CreateCategoryResponse, error) {
	resp, err := h.client.CreateCategory(context.Background(), connect.NewRequest(&categoriesv1.CreateCategoryRequest{Name: ctx.Flag("name"), Description: ctx.Flag("description")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create category", err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) createReport(_ cliapp.OperationContext, response *categoriesv1.CreateCategoryResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Created " + formatCategory(response.Category)}}
}
func (h *handlers) listCall(ctx cliapp.OperationContext) (*categoriesv1.ListCategoriesResponse, error) {
	resp, err := h.client.ListCategories(context.Background(), connect.NewRequest(&categoriesv1.ListCategoriesRequest{IncludeRetired: ctx.Flag("include-retired") == "true"}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list categories", err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) listReport(_ cliapp.OperationContext, response *categoriesv1.ListCategoriesResponse) cliapp.ListReport {
	rows := make([]string, 0, len(response.Categories))
	for _, category := range response.Categories {
		rows = append(rows, formatCategory(category))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d category(s).", len(rows))}, ResultsHeading: "Categories", Results: rows}
}
func (h *handlers) renameCall(ctx cliapp.OperationContext) (*categoriesv1.RenameCategoryResponse, error) {
	resp, err := h.client.RenameCategory(context.Background(), connect.NewRequest(&categoriesv1.RenameCategoryRequest{Id: ctx.Positional("id"), Name: ctx.Flag("name"), Description: ctx.Flag("description")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("rename category", err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) renameReport(_ cliapp.OperationContext, response *categoriesv1.RenameCategoryResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Renamed " + formatCategory(response.Category)}}
}
func (h *handlers) retireCall(ctx cliapp.OperationContext) (*categoriesv1.RetireCategoryResponse, error) {
	resp, err := h.client.RetireCategory(context.Background(), connect.NewRequest(&categoriesv1.RetireCategoryRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("retire category", err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) retireReport(_ cliapp.OperationContext, response *categoriesv1.RetireCategoryResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Retired " + formatCategory(response.Category)}}
}
func (h *handlers) getClassificationCall(ctx cliapp.OperationContext) (*categoriesv1.GetClassificationResponse, error) {
	resp, err := h.client.GetClassification(context.Background(), connect.NewRequest(&categoriesv1.GetClassificationRequest{SignalId: ctx.Positional("signal-id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get classification", err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) getClassificationReport(_ cliapp.OperationContext, response *categoriesv1.GetClassificationResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Fetched classification."}, ResultsHeading: "Classification", Results: []string{formatClassification(response.Classification)}}
}
func (h *handlers) confirmCall(ctx cliapp.OperationContext) (*categoriesv1.ConfirmClassificationResponse, error) {
	resp, err := h.client.ConfirmClassification(context.Background(), connect.NewRequest(&categoriesv1.ConfirmClassificationRequest{SignalId: ctx.Positional("signal-id"), CategoryId: ctx.Flag("category-id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("confirm classification", err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) confirmReport(_ cliapp.OperationContext, response *categoriesv1.ConfirmClassificationResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Recorded " + formatClassification(response.Classification)}}
}
func formatCategory(category *categoriesv1.Category) string {
	if category == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s — %s (reserved=%t retired=%t)", category.Id, category.Name, category.Reserved, category.RetiredAt != nil)
}
func formatClassification(classification *categoriesv1.Classification) string {
	if classification == nil {
		return "(nil)"
	}
	return fmt.Sprintf("signal=%s proposed=%s confidence=%.2f confirmed=%s state=%s", classification.SignalId, classification.ProposedCategoryId, classification.ProposedConfidence, classification.ConfirmedCategoryId, classification.State)
}
