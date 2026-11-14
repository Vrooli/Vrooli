package receiptfunding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/database"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	"google.golang.org/protobuf/encoding/protojson"
)

const LPBSServiceCredentialIdentity = "vrooli/landing-page-business-suite"

type LPBSCredentialReader interface {
	Resolve(context.Context, string, string) (string, error)
}

type LPBSCredentialFunc func(context.Context, string, string) (string, error)

func (f LPBSCredentialFunc) Resolve(ctx context.Context, identity, field string) (string, error) {
	return f(ctx, identity, field)
}

type LPBSCreditLotClient struct {
	url         string
	credentials LPBSCredentialReader
	client      *http.Client
}

func NewLPBSCreditLotClient(url string, credentials LPBSCredentialReader, client *http.Client) (*LPBSCreditLotClient, error) {
	if strings.TrimSpace(url) == "" || credentials == nil {
		return nil, errors.New("LPBS URL and credential authority are required")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &LPBSCreditLotClient{url: strings.TrimRight(url, "/"), credentials: credentials, client: client}, nil
}

func (c *LPBSCreditLotClient) GrantAcceptedReceipt(ctx context.Context, receipt *ingestpb.AcceptedCryptoReceipt) (CreditLotLink, error) {
	ctx = lpbsCreditLotRequestContext(ctx)
	if receipt == nil || receipt.GetId() == "" {
		return CreditLotLink{}, errors.New("accepted Money Ledger receipt is required")
	}
	credential, err := c.credentials.Resolve(ctx, LPBSServiceCredentialIdentity, "service-secret")
	if err != nil || strings.TrimSpace(credential) == "" {
		return CreditLotLink{}, errors.New("LPBS service credential is unavailable")
	}
	receiptJSON, err := protojson.Marshal(receipt)
	if err != nil {
		return CreditLotLink{}, err
	}
	body, err := json.Marshal(struct {
		Receipt json.RawMessage `json:"receipt"`
	}{Receipt: receiptJSON})
	if err != nil {
		return CreditLotLink{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/api/v1/internal/crypto-funding/accepted-receipts", bytes.NewReader(body))
	if err != nil {
		return CreditLotLink{}, err
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(credential))
	request.Header.Set("Content-Type", "application/json")
	if database.IsTestMode(ctx) {
		request.Header.Set("X-Vrooli-Test-Mode", "1")
	}
	response, err := c.client.Do(request)
	if err != nil {
		return CreditLotLink{}, err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 64*1024))
	if err != nil {
		return CreditLotLink{}, err
	}
	if response.StatusCode != http.StatusOK {
		return CreditLotLink{}, fmt.Errorf("LPBS credit lot owner returned HTTP %d", response.StatusCode)
	}
	var result struct {
		LotID     string `json:"credit_lot_id"`
		ReceiptID string `json:"receipt_id"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil || result.LotID == "" || result.ReceiptID != receipt.GetId() {
		return CreditLotLink{}, errors.New("LPBS returned an invalid credit-lot association")
	}
	return CreditLotLink{ID: result.LotID, Owner: "landing-page-business-suite", ReceiptID: result.ReceiptID}, nil
}
