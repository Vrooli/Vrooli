package categories

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	categoriesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/categories/categories_v1connect"
	internal "signal-inbox/internal/categories"
	"signal-inbox/internal/module"
)

func Module(service *internal.Service) module.Module {
	path, handler := categoriesconnect.NewCategoriesServiceHandler(NewConnectHandler(service))
	return module.Module{Name: "categories", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}

func Schema() string { return internal.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "categories_create", Path: categoriesconnect.CategoriesServiceCreateCategoryProcedure, Method: "POST", Summary: "Create category", Description: "Creates an operator-defined category at runtime.", Category: "categories", Request: &module.Schema{Type: "CreateCategoryRequest"}, Response: &module.Schema{Type: "CreateCategoryResponse"}},
	{ID: "categories_list", Path: categoriesconnect.CategoriesServiceListCategoriesProcedure, Method: "POST", Summary: "List categories", Description: "Lists active categories, optionally including retired categories.", Category: "categories", Request: &module.Schema{Type: "ListCategoriesRequest"}, Response: &module.Schema{Type: "ListCategoriesResponse"}},
	{ID: "categories_rename", Path: categoriesconnect.CategoriesServiceRenameCategoryProcedure, Method: "POST", Summary: "Rename category", Description: "Renames a non-reserved category.", Category: "categories", Request: &module.Schema{Type: "RenameCategoryRequest"}, Response: &module.Schema{Type: "RenameCategoryResponse"}},
	{ID: "categories_retire", Path: categoriesconnect.CategoriesServiceRetireCategoryProcedure, Method: "POST", Summary: "Retire category", Description: "Retires a category and reassigns confirmed signals to the reserved fallback without deleting signals.", Category: "categories", Request: &module.Schema{Type: "RetireCategoryRequest"}, Response: &module.Schema{Type: "RetireCategoryResponse"}},
	{ID: "categories_get_classification", Path: categoriesconnect.CategoriesServiceGetClassificationProcedure, Method: "POST", Summary: "Get classification", Description: "Returns the latest advisory or confirmed classification for a signal.", Category: "categories", Request: &module.Schema{Type: "GetClassificationRequest"}, Response: &module.Schema{Type: "GetClassificationResponse"}},
	{ID: "categories_confirm_classification", Path: categoriesconnect.CategoriesServiceConfirmClassificationProcedure, Method: "POST", Summary: "Confirm classification", Description: "Appends an operator confirmation or override while retaining the proposal.", Category: "categories", Request: &module.Schema{Type: "ConfirmClassificationRequest"}, Response: &module.Schema{Type: "ConfirmClassificationResponse"}},
}
