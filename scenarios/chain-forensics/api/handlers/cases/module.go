package cases

import (
	"log"
	"os"
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

func Module(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger) module.Module {
	repo := cases.NewSQLiteRepository(db, clk)
	if logger == nil {
		logger = log.Default()
	}
	ethEndpoint := strings.TrimSpace(os.Getenv("CHAIN_FORENSICS_ETHEREUM_RPC_URL"))
	if ethEndpoint == "" {
		ethEndpoint = forensics.DefaultEthereumRPC
	}
	baseEndpoint := strings.TrimSpace(os.Getenv("CHAIN_FORENSICS_BASE_RPC_URL"))
	if baseEndpoint == "" {
		baseEndpoint = forensics.DefaultBaseRPC
	}
	arbitrumEndpoint := strings.TrimSpace(os.Getenv("CHAIN_FORENSICS_ARBITRUM_RPC_URL"))
	if arbitrumEndpoint == "" {
		arbitrumEndpoint = forensics.DefaultArbitrumRPC
	}
	solanaEndpoint := strings.TrimSpace(os.Getenv("CHAIN_FORENSICS_SOLANA_RPC_URL"))
	if solanaEndpoint == "" {
		solanaEndpoint = forensics.DefaultSolanaRPC
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
	bitcoin, err := forensics.NewMempoolClient("", nil, nil)
	if err != nil {
		logger.Fatalf("Bitcoin public reader: %v", err)
	}
	readers := map[forensics.Chain]EVMReader{forensics.ChainEthereum: eth, forensics.ChainBase: base, forensics.ChainArbitrum: arbitrum}
	wallets := map[forensics.Chain]EVMWalletReader{forensics.ChainEthereum: eth, forensics.ChainBase: base}
	path, handler := casesconnect.NewCasesServiceHandler(NewHandlerWithReaders(repo, logger, readers, bitcoin, bitcoin).WithWalletReaders(wallets).WithSolanaReader(solana))
	return module.Module{Name: "cases", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}
func Schema() string { return cases.Schema() }
