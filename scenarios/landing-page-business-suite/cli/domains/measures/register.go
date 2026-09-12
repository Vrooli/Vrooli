// Package measures binds the typed commercial analytics surface from the CLI
// manifest to generated Connect clients.
package measures

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/cli-core/cliapp"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/measures"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/measures/measuresv1connect"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

const defaultWindow = "this_week"

type invoke func(context.Context, *measuresv1.TimeWindow) (proto.Message, int64, error)

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := lpbsconnect.NewMeasuresServiceClient(httpClient, baseURL)
	bindings := map[string]cliapp.PrimitiveHandler{
		"MeasuresService.CountSubscriptionsCreated": primitive("Count subscriptions created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountSubscriptionsCreated(ctx, connect.NewRequest(&lpbsv1.CountSubscriptionsCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountCreditTransactionsCreated": primitive("Count credit transactions created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountCreditTransactionsCreated(ctx, connect.NewRequest(&lpbsv1.CountCreditTransactionsCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountCheckoutSessionsCreated": primitive("Count checkout sessions created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountCheckoutSessionsCreated(ctx, connect.NewRequest(&lpbsv1.CountCheckoutSessionsCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountBundleProductsCreated": primitive("Count bundle products created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountBundleProductsCreated(ctx, connect.NewRequest(&lpbsv1.CountBundleProductsCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountBundlePricesCreated": primitive("Count bundle prices created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountBundlePricesCreated(ctx, connect.NewRequest(&lpbsv1.CountBundlePricesCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountSubscriptionSchedulesCreated": primitive("Count subscription schedules created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountSubscriptionSchedulesCreated(ctx, connect.NewRequest(&lpbsv1.CountSubscriptionSchedulesCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountIntroCouponUsageCreated": primitive("Count introductory coupon usage in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountIntroCouponUsageCreated(ctx, connect.NewRequest(&lpbsv1.CountIntroCouponUsageCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountPaymentAnomaliesCreated": primitive("Count payment anomalies created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountPaymentAnomaliesCreated(ctx, connect.NewRequest(&lpbsv1.CountPaymentAnomaliesCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountUsersCreated": primitive("Count customer accounts created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountUsersCreated(ctx, connect.NewRequest(&lpbsv1.CountUsersCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountUserSessionsCreated": primitive("Count user sessions created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountUserSessionsCreated(ctx, connect.NewRequest(&lpbsv1.CountUserSessionsCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountAuthTokensCreated": primitive("Count authentication tokens created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountAuthTokensCreated(ctx, connect.NewRequest(&lpbsv1.CountAuthTokensCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountAPIKeysCreated": primitive("Count API keys created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountAPIKeysCreated(ctx, connect.NewRequest(&lpbsv1.CountAPIKeysCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountCreditReservationsCreated": primitive("Count credit reservations created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountCreditReservationsCreated(ctx, connect.NewRequest(&lpbsv1.CountCreditReservationsCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountSubscriptionTierLimitsCreated": primitive("Count subscription tier limits created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountSubscriptionTierLimitsCreated(ctx, connect.NewRequest(&lpbsv1.CountSubscriptionTierLimitsCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountUsageRecordsCreated": primitive("Count usage records created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountUsageRecordsCreated(ctx, connect.NewRequest(&lpbsv1.CountUsageRecordsCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountAdminSessionsCreated": primitive("Count administrator sessions created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountAdminSessionsCreated(ctx, connect.NewRequest(&lpbsv1.CountAdminSessionsCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountAdminUsersCreated": primitive("Count administrator accounts created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountAdminUsersCreated(ctx, connect.NewRequest(&lpbsv1.CountAdminUsersCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountAssetsCreated": primitive("Count content assets created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountAssetsCreated(ctx, connect.NewRequest(&lpbsv1.CountAssetsCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountDownloadAppsCreated": primitive("Count downloadable applications created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountDownloadAppsCreated(ctx, connect.NewRequest(&lpbsv1.CountDownloadAppsCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountDownloadArtifactsCreated": primitive("Count downloadable artifacts created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountDownloadArtifactsCreated(ctx, connect.NewRequest(&lpbsv1.CountDownloadArtifactsCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountDownloadAssetsCreated": primitive("Count downloadable assets created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountDownloadAssetsCreated(ctx, connect.NewRequest(&lpbsv1.CountDownloadAssetsCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountDownloadStorageSettingsCreated": primitive("Count download storage settings created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountDownloadStorageSettingsCreated(ctx, connect.NewRequest(&lpbsv1.CountDownloadStorageSettingsCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountFeedbackRequestsCreated": primitive("Count feedback requests created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountFeedbackRequestsCreated(ctx, connect.NewRequest(&lpbsv1.CountFeedbackRequestsCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountMetricsEventsCreated": primitive("Count metrics events created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountMetricsEventsCreated(ctx, connect.NewRequest(&lpbsv1.CountMetricsEventsCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountRemoteProfilesCreated": primitive("Count remote profiles created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountRemoteProfilesCreated(ctx, connect.NewRequest(&lpbsv1.CountRemoteProfilesCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
		"MeasuresService.CountWaitlistEmailsCreated": primitive("Count waitlist signups created in a time window.", func(ctx context.Context, window *measuresv1.TimeWindow) (proto.Message, int64, error) {
			response, err := client.CountWaitlistEmailsCreated(ctx, connect.NewRequest(&lpbsv1.CountWaitlistEmailsCreatedRequest{Window: window}))
			if err != nil {
				return nil, 0, err
			}
			return response.Msg, response.Msg.GetCount(), nil
		}),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, "measures", bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("measures: load manifest: %w", err)
	}
	return group, nil
}

func primitive(description string, run invoke) cliapp.PrimitiveHandler {
	return cliapp.ProtoList(
		func(ctx cliapp.OperationContext) (proto.Message, error) {
			windowName := strings.TrimSpace(ctx.Flag("window"))
			if windowName == "" {
				windowName = defaultWindow
			}
			window, err := timeWindow(windowName)
			if err != nil {
				return nil, err
			}
			message, _, err := run(context.Background(), window)
			if err != nil {
				return nil, cliapp.WrapAPIError("measure", err, nil)
			}
			return message, nil
		},
		func(ctx cliapp.OperationContext, message proto.Message) cliapp.ListReport {
			return cliapp.ListReport{
				Summary:        []string{fmt.Sprintf("%s (%s).", description, ctx.Flag("window"))},
				ResultsHeading: "Measure result",
				Results:        []string{fmt.Sprint(message)},
			}
		},
	)
}

func timeWindow(value string) (*measuresv1.TimeWindow, error) {
	key := "TIME_WINDOW_TOKEN_" + strings.ToUpper(value)
	integer, ok := measuresv1.TimeWindowToken_value[key]
	if !ok || measuresv1.TimeWindowToken(integer) == measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_UNSPECIFIED {
		return nil, fmt.Errorf("unknown time window %q (use this_week, last_7d, last_30d, this_month, last_month, or this_quarter)", value)
	}
	return &measuresv1.TimeWindow{Window: &measuresv1.TimeWindow_Token{Token: measuresv1.TimeWindowToken(integer)}}, nil
}
