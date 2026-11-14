package remote

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	secpECDSA "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	authorizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	fundingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding"
	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	sessionPrivateKeyEnv   = "VROOLI_TREASURY_SESSION_PRIVATE_KEY"
	sessionPublicKeyHeader = "X-Vrooli-Treasury-Session-Public-Key"
	sessionProofHeader     = "X-Vrooli-Treasury-Session-Proof"
)

// agentSettlementIntent mirrors Treasury's canonical signed JSON contract.
// Keep field order and JSON names aligned with walletops.AgentSettlementIntent.
type agentSettlementIntent struct {
	AuthorizationID string `json:"authorization_id"`
	MandateID       string `json:"mandate_id"`
	BudgetID        string `json:"budget_id"`
	WalletID        string `json:"wallet_id"`
	WalletAddress   string `json:"wallet_address"`
	Network         string `json:"network"`
	AgentSubject    string `json:"agent_subject"`
	AgentRunID      string `json:"agent_run_id"`
	SettlementID    string `json:"settlement_id"`
	InstrumentID    string `json:"instrument_id"`
	IdempotencyKey  string `json:"idempotency_key"`
	AmountMinor     int64  `json:"amount_minor"`
	Currency        string `json:"currency"`
	Counterparty    string `json:"counterparty"`
	Generation      int64  `json:"generation"`
	ExpiresAt       string `json:"expires_at"`
}

func treasurySessionRequestHeaders(core *cliapp.ScenarioApp) cliapp.RequestPreparer {
	return treasurySessionRequestHeadersWithKeySource(core, func() string { return os.Getenv(sessionPrivateKeyEnv) })
}

func treasurySessionRequestHeadersWithKeySource(core *cliapp.ScenarioApp, keySource func() string) cliapp.RequestPreparer {
	return func(_ cliapp.OperationContext, method string, request proto.Message) (map[string]string, error) {
		if !strings.HasPrefix(method, "AgentSpend.") {
			return nil, nil
		}
		operation, target := "", ""
		switch method {
		case "AgentSpend.ProposeCharge":
			operation = "treasury.spend"
			target = cliutil.TreasurySpendIntentAuthorizationTarget(cliutil.TreasurySpendIntent{
				AuthorizationID: protoStringField(request, "id"), MandateID: protoStringField(request, "mandate_id"), AmountMinor: protoInt64Field(request, "amount_minor"),
				Currency: protoStringField(request, "currency"), Counterparty: protoStringField(request, "counterparty"), AuthorizationIdempotencyKey: protoStringField(request, "idempotency_key"),
				SettlementID: protoStringField(request, "settlement_id"), InstrumentID: protoStringField(request, "instrument_id"), SettlementIdempotencyKey: protoStringField(request, "settlement_idempotency_key"),
			})
		case "AgentSpend.GetAuthorization":
			operation, target = "treasury.read", "treasury:authorization/"+protoStringField(request, "id")
		case "AgentSpend.GetBudgetWallet":
			operation, target = "treasury.read", "treasury:budget/"+protoStringField(request, "budget_id")
		case "AgentSpend.GetBudgetHeadroom":
			operation, target = "treasury.read", "treasury:budget/"+protoStringField(request, "budget_id")
		case "AgentSpend.ReportOutcome":
			if keySource == nil {
				return nil, fmt.Errorf("run-owned Treasury session-key source is unavailable")
			}
			privateKey := strings.TrimPrefix(strings.TrimSpace(keySource()), "0x")
			key, err := parseSessionPrivateKey(privateKey)
			if err != nil {
				return nil, err
			}
			proof, exactTarget, err := prepareSettlementProofAndTarget(core, request, key)
			if err != nil {
				return nil, err
			}
			headers, err := treasuryAgentOperationHeaders("treasury.spend", exactTarget)
			if err != nil {
				return nil, err
			}
			if headers == nil {
				headers = make(map[string]string)
			}
			headers[sessionProofHeader] = proof
			return headers, nil
		default:
			return nil, fmt.Errorf("unsupported Treasury agent operation %q", method)
		}
		headers, err := treasuryAgentOperationHeaders(operation, target)
		if err != nil {
			return nil, err
		}
		if method != "AgentSpend.GetBudgetWallet" {
			return headers, nil
		}
		if keySource == nil {
			return headers, nil
		}
		privateKey := strings.TrimPrefix(strings.TrimSpace(keySource()), "0x")
		if privateKey == "" {
			if method == "AgentSpend.GetBudgetWallet" {
				return headers, nil // Operator reads and manually provisioned callers remain unchanged.
			}
			return nil, fmt.Errorf("run-owned Treasury session key is unavailable")
		}
		key, err := parseSessionPrivateKey(privateKey)
		if err != nil {
			return nil, err
		}
		publicKey := "0x" + hex.EncodeToString(key.PubKey().SerializeUncompressed())
		if protoStringField(request, "mandate_id") == "" {
			return headers, nil
		}
		if headers == nil {
			headers = make(map[string]string)
		}
		headers[sessionPublicKeyHeader] = publicKey
		return headers, nil
	}
}

func treasuryAgentOperationHeaders(operation, target string) (map[string]string, error) {
	if strings.TrimSpace(os.Getenv(cliutil.EnvIdentityToken)) == "" {
		return nil, nil
	}
	// Resolve the currently approved, run-bound operation grant for each
	// Treasury agent call. This keeps baseline identity separate from the
	// human-approved operation proof and permits a new exact grant to take
	// effect without restarting the run.
	credentials, err := cliutil.ResolveAgentCredentialsForOperation(context.Background(), operation, target)
	if err != nil {
		return nil, fmt.Errorf("resolve current Treasury operation grant: %w", err)
	}
	if credentials.BaselineToken == "" || credentials.OperationToken == "" {
		return nil, fmt.Errorf("current Agent Manager Treasury operation grant is unavailable")
	}
	return map[string]string{
		cliutil.HeaderAgentIdentityToken:  credentials.BaselineToken,
		cliutil.HeaderAgentOperationToken: credentials.OperationToken,
	}, nil
}

func parseSessionPrivateKey(value string) (*secp256k1.PrivateKey, error) {
	raw, err := hex.DecodeString(value)
	if err != nil || len(raw) != 32 {
		return nil, fmt.Errorf("run-owned Treasury session key must be 32-byte hex")
	}
	var scalar secp256k1.ModNScalar
	if scalar.SetByteSlice(raw) || scalar.IsZero() {
		return nil, fmt.Errorf("run-owned Treasury session key is outside secp256k1 scalar range")
	}
	return secp256k1.PrivKeyFromBytes(raw), nil
}

func prepareSettlementProof(core *cliapp.ScenarioApp, request proto.Message, key *secp256k1.PrivateKey) (string, error) {
	proof, _, err := prepareSettlementProofAndTarget(core, request, key)
	return proof, err
}

func prepareSettlementProofAndTarget(core *cliapp.ScenarioApp, request proto.Message, key *secp256k1.PrivateKey) (string, string, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := authorizationconnect.NewAgentSpendClient(httpClient, baseURL)
	return prepareSettlementProofFromClientAndTarget(client, request, key)
}

type agentSpendProofReader interface {
	GetAuthorization(context.Context, *connect.Request[authorizationv1.GetAuthorizationRequest]) (*connect.Response[authorizationv1.GetAuthorizationResponse], error)
	GetBudgetWallet(context.Context, *connect.Request[fundingv1.GetBudgetWalletRequest]) (*connect.Response[fundingv1.BudgetWalletResponse], error)
}

func prepareSettlementProofFromClient(client agentSpendProofReader, request proto.Message, key *secp256k1.PrivateKey) (string, error) {
	proof, _, err := prepareSettlementProofFromClientAndTarget(client, request, key)
	return proof, err
}

func prepareSettlementProofFromClientAndTarget(client agentSpendProofReader, request proto.Message, key *secp256k1.PrivateKey) (string, string, error) {
	return prepareSettlementProofFromClientAtAndTarget(client, request, key, time.Now().UTC())
}

func prepareSettlementProofFromClientAt(client agentSpendProofReader, request proto.Message, key *secp256k1.PrivateKey, now time.Time) (string, error) {
	proof, _, err := prepareSettlementProofFromClientAtAndTarget(client, request, key, now)
	return proof, err
}

func prepareSettlementProofFromClientAtAndTarget(client agentSpendProofReader, request proto.Message, key *secp256k1.PrivateKey, now time.Time) (string, string, error) {
	if client == nil || key == nil {
		return "", "", fmt.Errorf("Treasury proof client and session key are required")
	}
	authorizationID := protoStringField(request, "authorization_id")
	settlementID := protoStringField(request, "settlement_id")
	instrumentID := protoStringField(request, "instrument_id")
	idempotencyKey := protoStringField(request, "idempotency_key")
	if authorizationID == "" || settlementID == "" || instrumentID == "" || idempotencyKey == "" {
		return "", "", fmt.Errorf("settlement request requires authorization, settlement, instrument, and idempotency IDs")
	}
	authorizationTarget := "treasury:authorization/" + authorizationID
	credentials, err := treasuryAgentOperationHeaders("treasury.read", authorizationTarget)
	if err != nil {
		return "", "", err
	}
	authorizationRequest := connect.NewRequest(&authorizationv1.GetAuthorizationRequest{Id: authorizationID})
	for name, value := range credentials {
		authorizationRequest.Header().Set(name, value)
	}
	authorizationResponse, err := client.GetAuthorization(context.Background(), authorizationRequest)
	if err != nil {
		return "", "", fmt.Errorf("read Treasury authorization before signing: %w", err)
	}
	if authorizationResponse == nil || authorizationResponse.Msg == nil || authorizationResponse.Msg.GetAuthorization() == nil {
		return "", "", fmt.Errorf("Treasury returned no authorization to sign")
	}
	auth := authorizationResponse.Msg.GetAuthorization()
	if auth.GetVerdict() != authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_APPROVED && auth.GetVerdict() != authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_SETTLED {
		return "", "", fmt.Errorf("Treasury authorization is not approved; no session proof was created")
	}
	if auth.GetExpiresAt() == nil || !now.Before(auth.GetExpiresAt().AsTime()) {
		return "", "", fmt.Errorf("Treasury authorization is expired or has no expiry; no session proof was created")
	}
	if auth.GetMandateId() == "" || auth.GetBudgetId() == "" || auth.GetRequestingAgent() == "" {
		return "", "", fmt.Errorf("Treasury authorization is not bound to a mandate, budget, and agent")
	}
	if auth.GetSettlementId() == "" || auth.GetInstrumentId() == "" || auth.GetSettlementIdempotencyKey() == "" || auth.GetSettlementId() != settlementID || auth.GetInstrumentId() != instrumentID || auth.GetSettlementIdempotencyKey() != idempotencyKey {
		return "", "", fmt.Errorf("settlement request does not match the immutable terms approved with its authorization")
	}
	credentials, err = treasuryAgentOperationHeaders("treasury.read", "treasury:budget/"+auth.GetBudgetId())
	if err != nil {
		return "", "", err
	}
	walletRequest := connect.NewRequest(&fundingv1.GetBudgetWalletRequest{BudgetId: auth.GetBudgetId(), MandateId: auth.GetMandateId()})
	for name, value := range credentials {
		walletRequest.Header().Set(name, value)
	}
	walletRequest.Header().Set(sessionPublicKeyHeader, "0x"+hex.EncodeToString(key.PubKey().SerializeUncompressed()))
	walletResponse, err := client.GetBudgetWallet(context.Background(), walletRequest)
	if err != nil {
		return "", "", fmt.Errorf("read mandate-bound wallet before signing: %w", err)
	}
	if walletResponse == nil || walletResponse.Msg == nil || walletResponse.Msg.GetWallet() == nil {
		return "", "", fmt.Errorf("Treasury returned no mandate-bound budget wallet")
	}
	wallet := walletResponse.Msg.GetWallet()
	if wallet.GetAgentSubject() != auth.GetRequestingAgent() || wallet.GetAgentRunId() == "" || wallet.GetMandateId() != auth.GetMandateId() || wallet.GetBudgetId() != auth.GetBudgetId() || wallet.GetSessionGeneration() <= 0 || wallet.GetSessionExpiresAt() == nil {
		return "", "", fmt.Errorf("Treasury wallet session does not match the approved authorization")
	}
	if !now.Before(wallet.GetSessionExpiresAt().AsTime()) {
		return "", "", fmt.Errorf("Treasury agent session is expired; no session proof was created")
	}
	intent := agentSettlementIntent{
		AuthorizationID: auth.GetId(), MandateID: auth.GetMandateId(), BudgetID: auth.GetBudgetId(),
		WalletID: wallet.GetWalletId(), WalletAddress: wallet.GetAddress(), Network: wallet.GetNetwork(),
		AgentSubject: auth.GetRequestingAgent(), AgentRunID: wallet.GetAgentRunId(),
		SettlementID: settlementID, InstrumentID: instrumentID, IdempotencyKey: idempotencyKey,
		AmountMinor: auth.GetAmountMinor(), Currency: auth.GetCurrency(), Counterparty: auth.GetCounterparty(),
		Generation: wallet.GetSessionGeneration(), ExpiresAt: wallet.GetSessionExpiresAt().AsTime().UTC().Format(time.RFC3339Nano),
	}
	for _, value := range []string{intent.AuthorizationID, intent.MandateID, intent.BudgetID, intent.WalletID, intent.WalletAddress, intent.Network, intent.AgentSubject, intent.AgentRunID, intent.SettlementID, intent.InstrumentID, intent.IdempotencyKey, intent.Currency, intent.Counterparty, intent.ExpiresAt} {
		if strings.TrimSpace(value) == "" {
			return "", "", fmt.Errorf("Treasury settlement intent is incomplete")
		}
	}
	if intent.AmountMinor <= 0 {
		return "", "", fmt.Errorf("Treasury settlement authorization has no positive amount")
	}
	proof, err := signSessionIntent(key, intent)
	exactTarget := cliutil.TreasurySettlementTarget(auth.GetId(), auth.GetMandateId(), auth.GetAmountMinor(), auth.GetCurrency(), auth.GetCounterparty(), auth.GetSettlementId(), auth.GetInstrumentId(), auth.GetSettlementIdempotencyKey())
	return proof, exactTarget, err
}

func signSessionIntent(key *secp256k1.PrivateKey, intent agentSettlementIntent) (string, error) {
	encoded, err := json.Marshal(intent)
	if err != nil {
		return "", fmt.Errorf("encode Treasury settlement intent: %w", err)
	}
	message := "Vrooli Treasury mandate session settlement v1\n" + string(encoded)
	hash := sha3.NewLegacyKeccak256()
	_, _ = hash.Write([]byte("\x19Ethereum Signed Message:\n" + strconv.Itoa(len([]byte(message))) + message))
	digest := hash.Sum(nil)
	compact := secpECDSA.SignCompact(key, digest, false)
	if len(compact) != 65 || compact[0] < 27 || compact[0] > 30 {
		return "", fmt.Errorf("secp256k1 signer returned an invalid recovery ID")
	}
	signature := make([]byte, 65)
	copy(signature[:64], compact[1:])
	signature[64] = compact[0] - 27
	return "0x" + hex.EncodeToString(signature), nil
}

func protoStringField(message proto.Message, name string) string {
	if message == nil {
		return ""
	}
	reflection := message.ProtoReflect()
	field := reflection.Descriptor().Fields().ByName(protoreflect.Name(name))
	if field == nil || field.Kind() != protoreflect.StringKind {
		return ""
	}
	return reflection.Get(field).String()
}

func protoInt64Field(message proto.Message, name string) int64 {
	if message == nil {
		return 0
	}
	reflection := message.ProtoReflect()
	field := reflection.Descriptor().Fields().ByName(protoreflect.Name(name))
	if field == nil || field.Kind() != protoreflect.Int64Kind {
		return 0
	}
	return reflection.Get(field).Int()
}
