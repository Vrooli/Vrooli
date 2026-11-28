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
	"github.com/vrooli/api-core/lpbsidentity"
	"github.com/vrooli/api-core/servicetoken"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	"google.golang.org/protobuf/encoding/protojson"
)

const LPBSServiceCredentialIdentity = "vrooli/landing-page-business-suite"

const LPBSLegacyServiceCredentialField = "service-secret"

type LPBSCredentialReader interface {
	Resolve(context.Context, string, string) (string, error)
}

type LPBSCredentialStatusReader interface {
	Status(context.Context, string, string) (credentialclient.CredentialStatus, error)
}

type LPBSCredentialFunc func(context.Context, string, string) (string, error)

func (f LPBSCredentialFunc) Resolve(ctx context.Context, identity, field string) (string, error) {
	return f(ctx, identity, field)
}

type LPBSServiceTokenSource interface {
	Authorization(context.Context) (string, error)
}

type LPBSCreditLotClient struct {
	url    string
	tokens LPBSServiceTokenSource
	legacy LPBSCredentialReader
	client *http.Client
}

func NewLPBSCreditLotClient(url string, tokens LPBSServiceTokenSource, legacy LPBSCredentialReader, client *http.Client) (*LPBSCreditLotClient, error) {
	if strings.TrimSpace(url) == "" || (tokens == nil && legacy == nil) {
		return nil, errors.New("LPBS URL and a scoped service identity or legacy credential source are required")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &LPBSCreditLotClient{url: strings.TrimRight(url, "/"), tokens: tokens, legacy: legacy, client: client}, nil
}

func (c *LPBSCreditLotClient) GrantAcceptedReceipt(ctx context.Context, receipt *ingestpb.AcceptedCryptoReceipt) (CreditLotLink, error) {
	ctx = lpbsCreditLotRequestContext(ctx)
	if receipt == nil || receipt.GetId() == "" {
		return CreditLotLink{}, errors.New("accepted Money Ledger receipt is required")
	}
	authorization, tokenErr := "", error(nil)
	if c.tokens != nil {
		authorization, tokenErr = c.tokens.Authorization(ctx)
		if tokenErr == nil {
			if strings.TrimSpace(authorization) == "" {
				return CreditLotLink{}, errors.New("LPBS scoped service identity returned empty authorization")
			}
		} else if !errors.Is(tokenErr, servicetoken.ErrNotIssued) {
			return CreditLotLink{}, errors.New("LPBS scoped service identity failed; refusing legacy downgrade")
		}
	}
	if tokenErr != nil || c.tokens == nil {
		statusReader, ok := c.legacy.(LPBSCredentialStatusReader)
		if !ok || c.legacy == nil {
			return CreditLotLink{}, errors.New("LPBS scoped service identity is unavailable")
		}
		principal := lpbsidentity.TreasuryAcceptedReceiptCredit.CredentialIdentity("treasury")
		status, statusErr := statusReader.Status(ctx, principal, "credential")
		if statusErr != nil || status.ProviderState != "available" || status.Configured {
			return CreditLotLink{}, errors.New("LPBS scoped service identity is unavailable")
		}
		credential, err := c.legacy.Resolve(ctx, LPBSServiceCredentialIdentity, LPBSLegacyServiceCredentialField)
		if err != nil || strings.TrimSpace(credential) == "" {
			return CreditLotLink{}, errors.New("LPBS service identity is unavailable")
		}
		authorization = "Bearer " + strings.TrimSpace(credential)
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
	request.Header.Set("Authorization", authorization)
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
		return CreditLotLink{}, fmt.Errorf("LPBS credit lot owner returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
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
