package ledger

import (
	"context"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	"google.golang.org/protobuf/encoding/protojson"
)

func (h *handlers) cryptoPreview(c cliapp.OperationContext) (*ingestpb.PreviewCryptoObservationResponse, error) {
	in := &ingestpb.PreviewCryptoObservationRequest{}
	if err := protojson.Unmarshal([]byte(c.Flag("observation")), in); err != nil {
		return nil, err
	}
	out, err := h.clientI.PreviewCryptoObservation(context.Background(), connect.NewRequest(in))
	if err != nil {
		return nil, err
	}
	return out.Msg, nil
}

func cryptoPreviewReport(_ cliapp.OperationContext, out *ingestpb.PreviewCryptoObservationResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Crypto observation preview; journal posting withheld and budget uncredited."}, ResultsHeading: "Holds", Results: out.Holds}
}
