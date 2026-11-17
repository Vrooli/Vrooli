package cases

import (
	"log"
	"strings"

	forensics "github.com/vrooli/chain-forensics"

	"chain-forensics/internal/cases"
	"chain-forensics/internal/module"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	casesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/chain-forensics/v1/cases/cases_v1connect"
)

// RPCConfig contains public chain reader endpoints resolved at the application
// composition root. Keeping ambient environment reads out of this domain
// module makes endpoint selection explicit and replaceable in tests.
type RPCConfig struct {
	EthereumURL      string
	BaseURL          string
	ArbitrumURL      string
	SolanaURL        string
	SolanaHistoryURL string
}

func Module(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger, rpc RPCConfig) module.Module {
	repo := cases.NewSQLiteRepository(db, clk)
	if logger == nil {
		logger = log.Default()
	}
	ethEndpoint := strings.TrimSpace(rpc.EthereumURL)
	if ethEndpoint == "" {
		ethEndpoint = forensics.DefaultEthereumRPC
	}
	baseEndpoint := strings.TrimSpace(rpc.BaseURL)
	if baseEndpoint == "" {
		baseEndpoint = forensics.DefaultBaseRPC
	}
	arbitrumEndpoint := strings.TrimSpace(rpc.ArbitrumURL)
	if arbitrumEndpoint == "" {
		arbitrumEndpoint = forensics.DefaultArbitrumRPC
	}
	solanaEndpoint := strings.TrimSpace(rpc.SolanaURL)
	if solanaEndpoint == "" {
		solanaEndpoint = forensics.DefaultSolanaRPC
	}
	solanaHistoryEndpoint := strings.TrimSpace(rpc.SolanaHistoryURL)
	if solanaHistoryEndpoint == "" {
		solanaHistoryEndpoint = forensics.DefaultSolanaHistoryRPC
	}
	eth, err := forensics.NewEVMRPCClient(ethEndpoint, nil, nil)
	if err != nil {
		logger.Fatalf("Ethereum public RPC: %v", err)
	}
	base, err := forensics.NewEVMRPCClient(baseEndpoint, nil, nil)
	if err != nil {
		logger.Fatalf("Base public RPC: %v", err)
	}
	arbitrum, err := forensics.NewEVMRPCClient(arbitrumEndpoint, nil, nil)
	if err != nil {
		logger.Fatalf("Arbitrum public RPC: %v", err)
	}
	solana, err := forensics.NewSolanaRPCClient(solanaEndpoint, nil, nil)
	if err != nil {
		logger.Fatalf("Solana public RPC: %v", err)
	}
	solanaHistory, err := forensics.NewSolanaRPCClient(solanaHistoryEndpoint, nil, nil)
	if err != nil {
		logger.Fatalf("Solana history public RPC: %v", err)
	}
	bitcoin, err := forensics.NewMempoolClient("", nil, nil)
	if err != nil {
		logger.Fatalf("Bitcoin public reader: %v", err)
	}
	readers := map[forensics.Chain]EVMReader{forensics.ChainEthereum: eth, forensics.ChainBase: base, forensics.ChainArbitrum: arbitrum}
	wallets := map[forensics.Chain]EVMWalletReader{forensics.ChainEthereum: eth, forensics.ChainBase: base}
	path, handler := casesconnect.NewCasesServiceHandler(NewHandlerWithReaders(repo, logger, readers, bitcoin, bitcoin).WithWalletReaders(wallets).WithSolanaReader(solanaHistory).WithSolanaSignatureReader(solanaHistory).WithSolanaTransactionReader(solanaHistory).WithSolanaTokenAccountReader(solana))
	return module.Module{Name: "cases", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}
func Schema() string { return cases.Schema() }
