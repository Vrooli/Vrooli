# Geth Resource

Ethereum client for blockchain development and smart contract deployment.

## Overview

Geth (Go Ethereum) is the official Go implementation of the Ethereum protocol. This resource provides:

- **Full Ethereum node** capabilities (dev, testnet, or mainnet)
- **Smart contract deployment** and interaction
- **JSON-RPC and WebSocket** APIs for dApp development
- **Development mode** with instant mining and prefunded accounts
- **Transaction management** and account operations

## Benefits for Vrooli

- **Blockchain Integration**: Enable decentralized scenarios and Web3 applications
- **Smart Contract Automation**: Deploy and manage contracts programmatically
- **DeFi Capabilities**: Build financial automation and DeFi scenarios
- **Token Management**: Create and manage ERC-20/721 tokens
- **Decentralized Storage**: Integrate with IPFS and other Web3 services

## Quick Start

```bash
# Install Geth in development mode
vrooli resource geth install dev

# Start the node
vrooli resource geth start

# Check status
vrooli resource geth status

# Deploy a smart contract
vrooli resource geth inject deploy contracts/MyContract.sol

# Check account balance
vrooli resource geth balance 0x123...

# Open Geth console
vrooli resource geth console
```

## Networks

- **dev**: Local development with instant mining
- **mainnet**: Ethereum mainnet (requires significant resources)
- **goerli**: Goerli testnet
- **sepolia**: Sepolia testnet

## API Endpoints

- **JSON-RPC**: `http://localhost:8545`
- **WebSocket**: `ws://localhost:8546`
- **P2P**: Port `30303`

## Directory Structure

```
~/.vrooli/geth/
├── data/          # Blockchain data
├── contracts/     # Smart contracts
├── scripts/       # Geth scripts
└── logs/          # Node logs
```

## Examples

See the `examples/` directory for:
- Simple smart contract deployment
- Token creation (ERC-20)
- Transaction examples
- Web3.js integration

## Testing

Run tests with:
```bash
vrooli resource geth test
```

## Environment Variables

- `GETH_NETWORK`: Network to use (default: dev)
- `GETH_PORT`: JSON-RPC port (default: 8545)
- `GETH_WS_PORT`: WebSocket port (default: 8546)
- `GETH_CHAIN_ID`: Chain ID for dev network (default: 1337)

## Security Notes

- Development mode is insecure by design (unlocked accounts)
- Use proper key management for production
- Never expose RPC endpoints publicly without authentication
- Be cautious with smart contract permissions

## Scenarios Enabled

- **DeFi Automation**: Yield farming, liquidity provision
- **NFT Management**: Minting, trading, collection management
- **DAO Operations**: Governance, voting, treasury management
- **Payment Processing**: Crypto payment gateway
- **Supply Chain**: Track products on blockchain
- **Identity Management**: Self-sovereign identity solutions