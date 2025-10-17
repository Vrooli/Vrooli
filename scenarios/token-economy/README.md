# Token Economy

**Universal token system that powers the entire Vrooli ecosystem with rewards, achievements, and value exchange**

## 🎯 Overview

Token Economy is the financial backbone of Vrooli, enabling scenarios to mint, transfer, and exchange both fungible tokens (like currency) and non-fungible tokens (NFTs for achievements). It transforms daily activities into a rewarding game where completing chores earns tokens redeemable for entertainment, creating powerful behavioral incentives.

## 🚀 Quick Start

```bash
# Install CLI
./cli/install.sh

# Start the API
cd api && go run main.go

# Start the UI
cd ui && npm install && npm start

# Create your first wallet
token-economy wallet-create user --user-id alice

# Check balance
token-economy balance @alice
```

## 💡 Key Features

### For Families
- **Multi-user wallets** - Each family member has their own wallet
- **Parental controls** - Set spending limits and approve large transfers
- **Chore rewards** - Automatically mint tokens for completed tasks
- **Achievement badges** - Collect NFTs for milestones
- **Token exchange** - Trade different token types at parent-set rates

### For Developers
- **Simple integration** - Any scenario can mint/check tokens via API
- **Immutable ledger** - PostgreSQL-based transaction history
- **High performance** - Redis caching for instant balance queries
- **Multi-tenancy** - Isolated economies per household
- **Smart permissions** - Granular control over who can mint what

## 🏗️ Architecture

### Token Types
- **Fungible (ERC-20 style)**: ChoreCoins, GameTokens, StudyPoints
- **Non-fungible (ERC-721 style)**: Achievement badges, unlock keys, collectibles

### Tech Stack
- **API**: Go with Gorilla Mux
- **Database**: PostgreSQL (immutable ledger) + Redis (cache)
- **UI**: Vanilla JavaScript with modern fintech design
- **CLI**: Bash wrapper for all operations

## 📡 API Endpoints

### Core Operations
- `POST /api/v1/tokens/create` - Create new token type
- `POST /api/v1/tokens/mint` - Mint tokens (authorized scenarios only)
- `POST /api/v1/tokens/transfer` - Transfer between wallets
- `POST /api/v1/tokens/swap` - Exchange token types
- `GET /api/v1/wallets/{id}/balance` - Check all balances

### Admin Functions
- `POST /api/v1/admin/economy/rules` - Set household rules
- `GET /api/v1/admin/analytics` - View economy statistics

## 🔧 Integration Guide

### For Scenarios

```javascript
// Mint rewards when task completed
POST /api/v1/tokens/mint
{
  "token_id": "CHORE",
  "to_wallet": "@alice",
  "amount": 10,
  "metadata": {"task": "dishes"}
}

// Check if user has enough tokens
GET /api/v1/wallets/@alice/balance
// Returns: { balances: [{token_id: "GAME", amount: 50}] }

// Award achievement NFT
POST /api/v1/achievements/award
{
  "user_id": "alice",
  "achievement": "weekly-chore-champion",
  "rarity": "epic"
}
```

### CLI Examples

```bash
# Create tokens for your scenario
token-economy admin-create-token STUDY "Study Points" fungible

# Mint rewards
token-economy mint STUDY 100 @alice

# Transfer tokens
token-economy transfer GAME 10 @bob

# Swap tokens at current rate
token-economy swap CHORE GAME 20

# View transaction history
token-economy history --wallet @alice
```

## 🎨 UI Features

### Wallet Dashboard
- Real-time balance display
- Token portfolio view
- 24h change indicators
- Quick send/receive/swap actions

### Achievement Gallery
- Visual badge collection
- Rarity indicators (common → legendary)
- Scenario attribution
- Points accumulation

### Parent Admin Panel
- Create new token types
- Set exchange rates
- Configure spending limits
- View household analytics

## 🔐 Security

- **Immutable ledger** - All transactions permanently recorded
- **Atomic operations** - No double-spend possible
- **Encrypted keys** - Wallet keys secured at rest
- **Rate limiting** - Prevent spam and abuse
- **Audit trail** - Complete history of all actions

## 📊 Use Cases

### Implemented
- Chore rewards system
- Gaming time tokens
- Study achievement badges
- Household allowances

### Future Possibilities
- Cross-household trading
- Real money bridge
- Staking mechanisms
- Governance voting
- Compute resource bidding

## 🧪 Testing

```bash
# Run all tests
./test.sh

# Test specific components
./test.sh --api      # API endpoints
./test.sh --ledger   # Database integrity
./test.sh --cli      # CLI commands
./test.sh --ui       # UI functionality
```

## 📈 Performance

- **1000+ TPS** - Transaction throughput
- **<100ms** - API response time (P95)
- **<50ms** - Balance queries (cached)
- **99.9%** - Uptime target

## 🤝 Contributing

Token Economy is the foundation for Vrooli's incentive layer. When adding features:

1. Maintain ACID compliance for all transactions
2. Update both API and CLI in parallel
3. Consider multi-household implications
4. Add comprehensive tests
5. Document integration patterns

## 📝 License

MIT - Part of the Vrooli ecosystem

---

**Token Economy** - *Where every action has value, and value drives action*