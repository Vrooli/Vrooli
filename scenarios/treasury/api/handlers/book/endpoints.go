package book

import (
	"treasury/internal/module"

	bookconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/book/book_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "treasury_book_create", Path: bookconnect.BookAdminServiceCreateBookProcedure, Method: "POST", Summary: "Create an operator-owned Treasury book", Category: "book", Request: &module.Schema{Type: "CreateBookRequest"}, Response: &module.Schema{Type: "CreateBookResponse"}, Errors: bookErrors},
	{ID: "treasury_book_get", Path: bookconnect.BookAdminServiceGetBookProcedure, Method: "POST", Summary: "Read a Treasury book", Category: "book", Request: &module.Schema{Type: "GetBookRequest"}, Response: &module.Schema{Type: "GetBookResponse"}, Errors: bookErrors},
}

var bookErrors = []module.ErrorDesc{
	{Status: 401, Code: "unauthenticated", Description: "Operator identity is required"},
	{Status: 403, Code: "permission_denied", Description: "Agent-realm or invalid operator credential"},
	{Status: 412, Code: "failed_precondition", Description: "Operator realm or book service is unavailable"},
}
