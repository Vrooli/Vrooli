package coupons

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Register exposes the complete administrator coupon surface through the
// generated CouponAdminService contract. Keeping every operation on the same
// transport as the UI prevents REST and Connect behavior from drifting.
func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{Title: "Admin Commerce - Coupons", Commands: []cliapp.Command{
		listCommand(deps), createCommand(deps), usageCommand(deps), getCommand(deps), updateCommand(deps), deleteCommand(deps),
		mappingsCommand(deps), setPlanCouponCommand(deps), removePlanCouponCommand(deps), previewCommand(deps),
	}}
}

func client(deps support.Dependencies) (lpbsconnect.CouponAdminServiceClient, error) {
	httpClient, baseURL, err := deps.AdminConnectHTTPClient()
	if err != nil {
		return nil, err
	}
	return lpbsconnect.NewCouponAdminServiceClient(httpClient, baseURL), nil
}

func listCommand(deps support.Dependencies) cliapp.Command {
	op := cliapp.ProtoList(func(cliapp.OperationContext) (*lpbsv1.ListCouponsResponse, error) {
		service, err := client(deps)
		if err != nil {
			return nil, err
		}
		response, err := service.ListCoupons(context.Background(), connect.NewRequest(&lpbsv1.ListCouponsRequest{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("list coupons", err, nil)
		}
		return response.Msg, nil
	}, func(cliapp.OperationContext, *lpbsv1.ListCouponsResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Stripe coupons and introductory mappings."}, ResultsHeading: "Coupons"}
	})
	return (cliapp.Command{Name: "admin-coupons-list", NeedsAPI: true, Description: "List coupons through the generated Connect contract", Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoList}}).WithPrimitive(op)
}

func createCommand(deps support.Dependencies) cliapp.Command {
	op := cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*lpbsv1.CreateCouponResponse, error) {
		request := &lpbsv1.CreateCouponRequest{}
		if err := decodeBody(ctx, request, "coupon creation"); err != nil {
			return nil, err
		}
		service, err := client(deps)
		if err != nil {
			return nil, err
		}
		response, err := service.CreateCoupon(context.Background(), connect.NewRequest(request))
		if err != nil {
			return nil, cliapp.WrapAPIError("create coupon", err, nil)
		}
		return response.Msg, nil
	}, func(cliapp.OperationContext, *lpbsv1.CreateCouponResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Coupon created."}}
	})
	return (cliapp.Command{Name: "admin-coupons-create", NeedsAPI: true, Description: "Create coupon through the generated Connect contract (--body JSON)", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "body", Description: "Coupon JSON payload or @file.json", Required: true}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoMutation}}).WithPrimitive(op)
}

func usageCommand(deps support.Dependencies) cliapp.Command {
	op := cliapp.ProtoList(func(cliapp.OperationContext) (*lpbsv1.ListCouponUsageResponse, error) {
		service, err := client(deps)
		if err != nil {
			return nil, err
		}
		response, err := service.ListCouponUsage(context.Background(), connect.NewRequest(&lpbsv1.ListCouponUsageRequest{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("list coupon usage", err, nil)
		}
		return response.Msg, nil
	}, func(cliapp.OperationContext, *lpbsv1.ListCouponUsageResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Local introductory coupon redemptions."}, ResultsHeading: "Usage"}
	})
	return (cliapp.Command{Name: "admin-coupons-usage", NeedsAPI: true, Description: "List coupon usage through the generated Connect contract", Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoList}}).WithPrimitive(op)
}

func getCommand(deps support.Dependencies) cliapp.Command {
	op := cliapp.ProtoList(func(ctx cliapp.OperationContext) (*lpbsv1.GetCouponResponse, error) {
		id, err := couponID(ctx)
		if err != nil {
			return nil, err
		}
		service, err := client(deps)
		if err != nil {
			return nil, err
		}
		response, err := service.GetCoupon(context.Background(), connect.NewRequest(&lpbsv1.GetCouponRequest{CouponId: id}))
		if err != nil {
			return nil, cliapp.WrapAPIError("get coupon", err, nil)
		}
		return response.Msg, nil
	}, func(cliapp.OperationContext, *lpbsv1.GetCouponResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Stripe coupon."}, ResultsHeading: "Coupon"}
	})
	return (cliapp.Command{Name: "admin-coupons-get", NeedsAPI: true, Description: "Get coupon through the generated Connect contract (COUPON_ID)", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "coupon_id", Required: true}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoList}}).WithPrimitive(op)
}

func updateCommand(deps support.Dependencies) cliapp.Command {
	op := cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*lpbsv1.UpdateCouponResponse, error) {
		id, err := couponID(ctx)
		if err != nil {
			return nil, err
		}
		request := &lpbsv1.UpdateCouponRequest{CouponId: id}
		if err := decodeBody(ctx, request, "coupon update"); err != nil {
			return nil, err
		}
		if request.GetCouponId() != "" && request.GetCouponId() != id {
			return nil, fmt.Errorf("coupon_id in --body disagrees with positional coupon_id")
		}
		request.CouponId = id
		service, err := client(deps)
		if err != nil {
			return nil, err
		}
		response, err := service.UpdateCoupon(context.Background(), connect.NewRequest(request))
		if err != nil {
			return nil, cliapp.WrapAPIError("update coupon", err, nil)
		}
		return response.Msg, nil
	}, func(cliapp.OperationContext, *lpbsv1.UpdateCouponResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Coupon updated."}}
	})
	return (cliapp.Command{Name: "admin-coupons-update", NeedsAPI: true, Description: "Update coupon through the generated Connect contract (COUPON_ID --body JSON)", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "coupon_id", Required: true}}, Flags: []cliapp.Flag{{Name: "body", Description: "Coupon update JSON payload or @file.json", Required: true}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoMutation}}).WithPrimitive(op)
}

func deleteCommand(deps support.Dependencies) cliapp.Command {
	op := cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*lpbsv1.DeleteCouponResponse, error) {
		id, err := couponID(ctx)
		if err != nil {
			return nil, err
		}
		service, err := client(deps)
		if err != nil {
			return nil, err
		}
		response, err := service.DeleteCoupon(context.Background(), connect.NewRequest(&lpbsv1.DeleteCouponRequest{CouponId: id}))
		if err != nil {
			return nil, cliapp.WrapAPIError("delete coupon", err, nil)
		}
		return response.Msg, nil
	}, func(cliapp.OperationContext, *lpbsv1.DeleteCouponResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Coupon deleted."}}
	})
	return (cliapp.Command{Name: "admin-coupons-delete", NeedsAPI: true, Description: "Delete coupon through the generated Connect contract (COUPON_ID)", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "coupon_id", Required: true}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoMutation}}).WithPrimitive(op)
}

func mappingsCommand(deps support.Dependencies) cliapp.Command {
	op := cliapp.ProtoList(func(cliapp.OperationContext) (*lpbsv1.GetCouponMappingsResponse, error) {
		service, err := client(deps)
		if err != nil {
			return nil, err
		}
		response, err := service.GetCouponMappings(context.Background(), connect.NewRequest(&lpbsv1.GetCouponMappingsRequest{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("get coupon mappings", err, nil)
		}
		return response.Msg, nil
	}, func(cliapp.OperationContext, *lpbsv1.GetCouponMappingsResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Stripe price to coupon mappings."}, ResultsHeading: "Mappings"}
	})
	return (cliapp.Command{Name: "admin-coupon-mappings", NeedsAPI: true, Description: "List coupon mappings through the generated Connect contract", Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoList}}).WithPrimitive(op)
}

func setPlanCouponCommand(deps support.Dependencies) cliapp.Command {
	op := cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*lpbsv1.SetCouponForPlanResponse, error) {
		priceID, couponID, err := planCouponIDs(ctx)
		if err != nil {
			return nil, err
		}
		service, err := client(deps)
		if err != nil {
			return nil, err
		}
		response, err := service.SetCouponForPlan(context.Background(), connect.NewRequest(&lpbsv1.SetCouponForPlanRequest{PriceId: priceID, CouponId: couponID}))
		if err != nil {
			return nil, cliapp.WrapAPIError("set coupon for plan", err, nil)
		}
		return response.Msg, nil
	}, func(cliapp.OperationContext, *lpbsv1.SetCouponForPlanResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Coupon assigned to plan."}}
	})
	return (cliapp.Command{Name: "admin-plan-coupon-set", NeedsAPI: true, Description: "Set plan coupon through the generated Connect contract (PRICE_ID COUPON_ID)", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "price_id", Required: true}, {Name: "coupon_id", Required: true}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoMutation}}).WithPrimitive(op)
}

func removePlanCouponCommand(deps support.Dependencies) cliapp.Command {
	op := cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*lpbsv1.RemoveCouponFromPlanResponse, error) {
		priceID := strings.TrimSpace(ctx.Positional("price_id"))
		if priceID == "" {
			return nil, fmt.Errorf("price_id is required")
		}
		service, err := client(deps)
		if err != nil {
			return nil, err
		}
		response, err := service.RemoveCouponFromPlan(context.Background(), connect.NewRequest(&lpbsv1.RemoveCouponFromPlanRequest{PriceId: priceID}))
		if err != nil {
			return nil, cliapp.WrapAPIError("remove coupon from plan", err, nil)
		}
		return response.Msg, nil
	}, func(cliapp.OperationContext, *lpbsv1.RemoveCouponFromPlanResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Coupon removed from plan."}}
	})
	return (cliapp.Command{Name: "admin-plan-coupon-remove", NeedsAPI: true, Description: "Remove plan coupon through the generated Connect contract (PRICE_ID)", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "price_id", Required: true}}}, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoMutation}}).WithPrimitive(op)
}

func previewCommand(deps support.Dependencies) cliapp.Command {
	op := cliapp.ProtoList(func(cliapp.OperationContext) (*lpbsv1.GetCouponImportPreviewResponse, error) {
		service, err := client(deps)
		if err != nil {
			return nil, err
		}
		response, err := service.GetCouponImportPreview(context.Background(), connect.NewRequest(&lpbsv1.GetCouponImportPreviewRequest{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("get Stripe coupon preview", err, nil)
		}
		return response.Msg, nil
	}, func(cliapp.OperationContext, *lpbsv1.GetCouponImportPreviewResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Stripe coupons available for local import."}, ResultsHeading: "Coupons"}
	})
	return (cliapp.Command{Name: "admin-stripe-coupons-preview", NeedsAPI: true, Description: "Preview Stripe coupons through the generated Connect contract", Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoList}}).WithPrimitive(op)
}

func decodeBody(ctx cliapp.OperationContext, request proto.Message, operation string) error {
	payload, err := support.ParseBody(ctx.Flag("body"))
	if err != nil {
		return err
	}
	if err := protojson.Unmarshal(payload, request); err != nil {
		return fmt.Errorf("decode %s: %w", operation, err)
	}
	return nil
}

func couponID(ctx cliapp.OperationContext) (string, error) {
	id := strings.TrimSpace(ctx.Positional("coupon_id"))
	if id == "" {
		return "", fmt.Errorf("coupon_id is required")
	}
	return id, nil
}

func planCouponIDs(ctx cliapp.OperationContext) (string, string, error) {
	priceID, couponID := strings.TrimSpace(ctx.Positional("price_id")), strings.TrimSpace(ctx.Positional("coupon_id"))
	if priceID == "" || couponID == "" {
		return "", "", fmt.Errorf("price_id and coupon_id are required")
	}
	return priceID, couponID, nil
}
