# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Token-economy adds a **decentralized value exchange protocol** that enables scenarios to mint, transfer, and redeem both fungible and non-fungible tokens. This creates a micro-economy layer where achievements, rewards, and value can flow between any Vrooli scenarios, turning the entire ecosystem into a gamified, incentive-aligned platform. Every action in any scenario can now have economic consequences and rewards.

### Intelligence Amplification
**How does this capability make future agents smarter?**
This capability enables agents to:
- **Learn optimal incentive structures** by analyzing token flow patterns and user behavior
- **Create self-balancing economies** that adjust reward rates based on system-wide metrics
- **Coordinate multi-scenario workflows** using tokens as signals for resource allocation
- **Build reputation systems** that persist across all scenarios
- **Implement complex behavioral nudges** through economic mechanisms
- **Enable market-based priority systems** where agents bid tokens for computational resources

### Recursive Value
**What new scenarios become possible after this exists?**
1. **marketplace-hub** - A decentralized marketplace where users trade scenario-generated assets
2. **achievement-leaderboard** - Global ranking system across all scenarios using NFT badges
3. **family-governance-dao** - Voting system where family members use tokens to make household decisions
4. **skill-certification-platform** - Issues verifiable credentials as NFTs for completed learning paths
5. **resource-auction-house** - Scenarios bid tokens for priority access to limited resources
6. **cross-household-exchange** - Enables token trading between different Vrooli installations
7. **defi-bridge** - Eventual connection to real blockchain networks for value extraction

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Token minting API with configurable supply limits per scenario
  - [ ] Transfer mechanism between wallets with atomic transaction guarantees
  - [ ] PostgreSQL-based immutable transaction ledger
  - [ ] Multi-tenancy via scenario-authenticator integration
  - [ ] Balance query API with Redis caching for performance
  - [ ] Scenario permission registry for minting/burning rights
  - [ ] Basic wallet UI showing all token balances
  - [ ] CLI for all token operations (mint, transfer, balance, history)
  - [ ] Parent admin controls for household economy management
  - [ ] Exchange rate mechanism between different token types
  
- **Should Have (P1)**
  - [ ] NFT (ERC-721 style) achievement system
  - [ ] Token burning mechanism for redemption
  - [ ] Scheduled token distributions (allowances, recurring rewards)
  - [ ] Transaction history with filtering and export
  - [ ] Token metadata storage (icons, descriptions, creation date)
  - [ ] Bulk transfer operations for efficiency
  - [ ] Token swap pools for automatic exchanges
  - [ ] Rich notification system for token events
  - [ ] Analytics dashboard for token flow visualization
  - [ ] Smart contract-like rules engine for conditional transfers
  
- **Nice to Have (P2)**
  - [ ] Cross-household token bridges
  - [ ] Token staking mechanisms
  - [ ] Governance voting using tokens
  - [ ] Time-locked token vesting
  - [ ] Token-gated content access
  - [ ] DEX-style automated market maker
  - [ ] Layer 2 scaling via state channels
  - [ ] zk-proof privacy features
  - [ ] Real blockchain bridge (Ethereum/Polygon)
  - [ ] Token-based compute resource allocation

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Transaction Throughput | > 1000 TPS | Load testing with concurrent transfers |
| API Response Time | < 100ms for balance queries | P95 latency monitoring |
| Ledger Write Speed | < 50ms per transaction | Database performance metrics |
| Cache Hit Rate | > 90% for balance queries | Redis metrics |
| UI Load Time | < 500ms | Lighthouse performance score |
| Transaction Finality | < 1 second | End-to-end transaction timing |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with scenario-authenticator
- [ ] Performance targets met under load
- [ ] Parent controls fully functional
- [ ] Multi-scenario integration validated
- [ ] Security audit passed (no double-spend, overflow issues)
- [ ] UI works on mobile and desktop
- [ ] Documentation complete (API, CLI, integration guide)

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: postgres
    purpose: Immutable transaction ledger and wallet state
    integration_pattern: Direct SQL via Go database/sql
    access_method: CLI command - resource-postgres
    
  - resource_name: redis
    purpose: High-speed balance caching and session management
    integration_pattern: Direct connection via go-redis
    access_method: CLI command - resource-redis
    
  - resource_name: qdrant
    purpose: Similarity search for token recommendations
    integration_pattern: REST API for vector operations
    access_method: Direct API via HTTP client
    
optional:
  - resource_name: minio
    purpose: Store token metadata, icons, and transaction receipts
    fallback: Store metadata in PostgreSQL JSONB
    access_method: S3-compatible API
    
  - resource_name: ollama
    purpose: AI-powered exchange rate recommendations
    fallback: Use fixed exchange rates
    access_method: CLI command - resource-ollama
```

### Resource Integration Standards
```yaml
integration_priorities:
  2_resource_cli:
    - command: resource-postgres query
      purpose: Database operations for ledger
    - command: resource-redis get/set
      purpose: Cache operations for balances
    - command: resource-ollama generate
      purpose: Exchange rate optimization
  
  3_direct_api:
    - justification: Need atomic transactions with PostgreSQL
      endpoint: Direct SQL connection required
    - justification: Redis pub/sub for real-time updates
      endpoint: Redis connection for event streaming

shared_workflow_criteria:
  - Token operations are too critical for workflow abstraction
  - Direct API calls ensure ACID compliance
  - This scenario IS the shared resource for other scenarios
```

### Data Models
```yaml
primary_entities:
  - name: Token
    storage: postgres
    schema: |
      {
        id: UUID
        household_id: UUID
        symbol: VARCHAR(10)
        name: VARCHAR(100)
        type: ENUM('fungible', 'non-fungible')
        total_supply: DECIMAL(36,18)
        decimals: INT
        creator_scenario: VARCHAR(100)
        metadata: JSONB
        created_at: TIMESTAMP
      }
    relationships: Has many Transactions, belongs to Household
    
  - name: Wallet
    storage: postgres
    schema: |
      {
        id: UUID
        household_id: UUID
        user_id: UUID
        address: VARCHAR(42) // 0x prefixed hex
        type: ENUM('user', 'scenario', 'treasury')
        metadata: JSONB
        created_at: TIMESTAMP
      }
    relationships: Has many Balances, belongs to User and Household
    
  - name: Balance
    storage: postgres/redis
    schema: |
      {
        wallet_id: UUID
        token_id: UUID
        amount: DECIMAL(36,18)
        locked_amount: DECIMAL(36,18)
        updated_at: TIMESTAMP
      }
    relationships: Belongs to Wallet and Token
    
  - name: Transaction
    storage: postgres
    schema: |
      {
        id: UUID
        hash: VARCHAR(66) // SHA256 hex
        from_wallet: UUID
        to_wallet: UUID
        token_id: UUID
        amount: DECIMAL(36,18)
        type: ENUM('mint', 'transfer', 'burn', 'swap')
        metadata: JSONB
        status: ENUM('pending', 'confirmed', 'failed')
        created_at: TIMESTAMP
        confirmed_at: TIMESTAMP
      }
    relationships: References Wallets and Token
    
  - name: Achievement
    storage: postgres/qdrant
    schema: |
      {
        id: UUID
        token_id: UUID // NFT token
        scenario: VARCHAR(100)
        type: VARCHAR(50)
        title: VARCHAR(200)
        description: TEXT
        criteria: JSONB
        icon_url: VARCHAR(500)
        rarity: ENUM('common', 'rare', 'epic', 'legendary')
        earned_at: TIMESTAMP
      }
    relationships: Is a special type of NFT Token
    
  - name: ExchangeRate
    storage: postgres/redis
    schema: |
      {
        from_token: UUID
        to_token: UUID
        rate: DECIMAL(36,18)
        liquidity: DECIMAL(36,18)
        last_updated: TIMESTAMP
      }
    relationships: References Token pairs
```

### API Contract
```yaml
endpoints:
  # Token Management
  - method: POST
    path: /api/v1/tokens/create
    purpose: Create new token type (scenario admin only)
    input_schema: |
      {
        symbol: string
        name: string
        type: "fungible" | "non-fungible"
        initial_supply?: number
        metadata?: object
      }
    output_schema: |
      {
        token_id: string
        address: string
      }
    sla:
      response_time: 200ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/tokens/mint
    purpose: Mint new tokens (authorized scenarios only)
    input_schema: |
      {
        token_id: string
        to_wallet: string
        amount: number
        metadata?: object
      }
    output_schema: |
      {
        transaction_hash: string
        new_supply: number
      }
    sla:
      response_time: 100ms
      availability: 99.9%
      
  - method: POST
    path: /api/v1/tokens/transfer
    purpose: Transfer tokens between wallets
    input_schema: |
      {
        from_wallet: string
        to_wallet: string
        token_id: string
        amount: number
        memo?: string
      }
    output_schema: |
      {
        transaction_hash: string
        new_balance: number
      }
    sla:
      response_time: 100ms
      availability: 99.99%
      
  - method: GET
    path: /api/v1/wallets/{wallet_id}/balance
    purpose: Get all token balances for wallet
    output_schema: |
      {
        balances: [{
          token_id: string
          symbol: string
          amount: number
          locked: number
          value_usd?: number
        }]
      }
    sla:
      response_time: 50ms
      availability: 99.99%
      
  - method: POST
    path: /api/v1/tokens/swap
    purpose: Exchange one token for another
    input_schema: |
      {
        from_token: string
        to_token: string
        amount: number
        slippage_tolerance?: number
      }
    output_schema: |
      {
        transaction_hash: string
        received_amount: number
        exchange_rate: number
      }
    sla:
      response_time: 200ms
      availability: 99.9%
      
  - method: GET
    path: /api/v1/achievements/{user_id}
    purpose: Get all achievements/NFTs for user
    output_schema: |
      {
        achievements: [{
          id: string
          title: string
          scenario: string
          rarity: string
          earned_at: string
        }]
      }
    sla:
      response_time: 100ms
      availability: 99.9%
      
  # Admin Endpoints
  - method: POST
    path: /api/v1/admin/economy/rules
    purpose: Set household economy rules (parent only)
    input_schema: |
      {
        daily_limits?: object
        exchange_rates?: object
        allowed_scenarios?: string[]
      }
    
  - method: GET
    path: /api/v1/admin/analytics
    purpose: Get token flow analytics
    output_schema: |
      {
        total_transactions: number
        token_velocity: object
        top_earners: array
        scenario_activity: object
      }
```

### Event Interface
```yaml
published_events:
  - name: token.minted
    payload: { token_id, amount, to_wallet, scenario }
    subscribers: [scenario-authenticator, achievement-tracker]
    
  - name: token.transferred
    payload: { transaction_hash, from, to, amount, token_id }
    subscribers: [notification-hub, analytics-engine]
    
  - name: achievement.earned
    payload: { user_id, achievement_id, scenario, rarity }
    subscribers: [notification-hub, leaderboard-system]
    
  - name: exchange.completed
    payload: { from_token, to_token, amount, rate }
    subscribers: [market-maker, analytics-engine]
    
consumed_events:
  - name: scenario.task_completed
    action: Mint reward tokens based on task difficulty
    
  - name: authenticator.user_created
    action: Create wallet for new user
    
  - name: household.settings_updated
    action: Update economy rules and limits
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: token-economy
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show token economy health and stats
    flags: [--json, --verbose, --household <id>]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and API version
    flags: [--json]

custom_commands:
  # Wallet Operations
  - name: wallet-create
    description: Create new wallet for user or scenario
    api_endpoint: /api/v1/wallets/create
    arguments:
      - name: type
        type: string
        required: true
        description: Wallet type (user|scenario|treasury)
    flags:
      - name: --user-id
        description: User ID for user wallet
      - name: --scenario
        description: Scenario name for scenario wallet
    output: Wallet address and initial setup
    
  - name: balance
    description: Check token balances
    api_endpoint: /api/v1/wallets/{wallet}/balance
    arguments:
      - name: wallet
        type: string
        required: false
        description: Wallet address (defaults to current user)
    flags:
      - name: --token
        description: Filter by specific token
      - name: --all-users
        description: Show all household balances
    output: Token balances table
    
  # Token Operations  
  - name: mint
    description: Mint new tokens (scenario only)
    api_endpoint: /api/v1/tokens/mint
    arguments:
      - name: token
        type: string
        required: true
        description: Token symbol or ID
      - name: amount
        type: int
        required: true
        description: Amount to mint
      - name: to
        type: string
        required: true
        description: Recipient wallet address
    flags:
      - name: --metadata
        description: JSON metadata for NFTs
    output: Transaction hash and confirmation
    
  - name: transfer
    description: Transfer tokens between wallets
    api_endpoint: /api/v1/tokens/transfer
    arguments:
      - name: token
        type: string
        required: true
        description: Token to transfer
      - name: amount
        type: int
        required: true
        description: Amount to transfer
      - name: to
        type: string
        required: true
        description: Recipient wallet
    flags:
      - name: --from
        description: Source wallet (default current user)
      - name: --memo
        description: Transaction memo
    output: Transaction hash and new balance
    
  - name: swap
    description: Exchange tokens at current rate
    api_endpoint: /api/v1/tokens/swap
    arguments:
      - name: from-token
        type: string
        required: true
        description: Token to swap from
      - name: to-token
        type: string
        required: true
        description: Token to receive
      - name: amount
        type: int
        required: true
        description: Amount to swap
    flags:
      - name: --slippage
        description: Max slippage tolerance (%)
    output: Swap details and received amount
    
  - name: history
    description: View transaction history
    api_endpoint: /api/v1/transactions
    flags:
      - name: --wallet
        description: Filter by wallet
      - name: --token
        description: Filter by token
      - name: --limit
        description: Number of transactions
      - name: --export
        description: Export as CSV
    output: Transaction list or CSV file
    
  # Achievement Commands
  - name: achievements
    description: View earned achievements/NFTs
    api_endpoint: /api/v1/achievements/{user}
    arguments:
      - name: user
        type: string
        required: false
        description: User ID (default current)
    flags:
      - name: --scenario
        description: Filter by scenario
      - name: --rarity
        description: Filter by rarity
    output: Achievement gallery
    
  # Admin Commands
  - name: admin-set-rate
    description: Set exchange rate (parent only)
    api_endpoint: /api/v1/admin/economy/rates
    arguments:
      - name: from-token
        type: string
        required: true
      - name: to-token
        type: string
        required: true
      - name: rate
        type: float
        required: true
    output: Updated rate confirmation
    
  - name: admin-stats
    description: View economy statistics
    api_endpoint: /api/v1/admin/analytics
    flags:
      - name: --period
        description: Time period (day|week|month)
      - name: --scenario
        description: Filter by scenario
    output: Analytics dashboard data
```

### CLI-API Parity Requirements
- Every API endpoint has corresponding CLI command
- CLI provides both human-readable and JSON output
- Wallet addresses can use aliases (e.g., @alice instead of 0x...)
- Transaction hashes are shortened in human output
- Batch operations supported via --file flag

### Implementation Standards
```yaml
implementation_requirements:
  - architecture: Thin wrapper over api/lib functions
  - language: Go for consistency
  - dependencies: Minimal - reuse API client
  - error_handling: Clear errors with recovery suggestions
  - configuration: 
      - ~/.vrooli/token-economy/config.yaml
      - ENV vars: TOKEN_ECONOMY_API_URL, TOKEN_ECONOMY_WALLET
      - Command flags override all
  
installation:
  - install_script: Creates symlink in ~/.vrooli/bin/
  - path_update: Adds to PATH if needed
  - permissions: 755 executable
  - documentation: Comprehensive --help
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **scenario-authenticator**: User identity and multi-tenancy foundation
- **postgres resource**: Database for immutable ledger
- **redis resource**: Caching layer for performance
- **notification-hub** (optional): For transaction notifications

### Downstream Enablement
**What future capabilities does this unlock?**
- **marketplace-hub**: Trading platform for scenario assets
- **governance-dao**: Voting with token weights
- **defi-protocols**: Lending, staking, yield farming
- **cross-chain-bridge**: Connect to real blockchain
- **compute-marketplace**: Bid tokens for GPU/CPU time
- **subscription-manager**: Token-based SaaS subscriptions

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: chore-tracking
    capability: Mint ChoreCoins for completed tasks
    interface: API - POST /api/v1/tokens/mint
    
  - scenario: retro-game-launcher
    capability: Check GameTokens before allowing play
    interface: API - GET /api/v1/wallets/{wallet}/balance
    
  - scenario: study-buddy
    capability: Award StudyBadges (NFTs) for milestones
    interface: API - POST /api/v1/achievements/award
    
  - scenario: ALL
    capability: Universal value exchange protocol
    interface: Full API/CLI/Event suite
    
consumes_from:
  - scenario: scenario-authenticator
    capability: User identity and household context
    fallback: Single-tenant mode
    
  - scenario: notification-hub
    capability: Send transaction alerts
    fallback: Log events only
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: professional
  inspiration: "Coinbase meets Nintendo achievement system"
  
  visual_style:
    color_scheme: custom
    primary_colors:
      - "#6366F1" # Indigo for primary actions
      - "#10B981" # Green for positive values
      - "#EF4444" # Red for negative values
      - "#F59E0B" # Amber for achievements
    typography: modern
    layout: dashboard
    animations: subtle
    
  personality:
    tone: friendly
    mood: energetic
    target_feeling: "Empowered and rewarded"

style_references:
  professional: 
    - "Clean, modern fintech aesthetic"
    - "Data-rich but not overwhelming"
  creative:
    - "Achievement badges with personality"
    - "Satisfying animation on token receipt"
  playful:
    - "Particle effects for rare achievements"
    - "Coin sound effects (optional)"
```

### Target Audience Alignment
- **Primary Users**: Families with children 8+
- **User Expectations**: Banking app polish with game-like fun
- **Accessibility**: WCAG AA compliance, large touch targets
- **Responsive Design**: Mobile-first, works on tablets

### Brand Consistency Rules
- Must feel trustworthy (handling value)
- Should feel rewarding and fun
- Professional enough for parents
- Engaging enough for kids
- Clear visual hierarchy for financial data

## 💰 Value Proposition

### Business Value
- **Primary Value**: Increases user retention 10x through gamification
- **Revenue Potential**: $30K - $50K per deployment
- **Cost Savings**: Replaces multiple reward/allowance apps
- **Market Differentiator**: First truly composable token economy

### Technical Value
- **Reusability Score**: 10/10 - Every scenario can use tokens
- **Complexity Reduction**: Makes incentives trivial to implement
- **Innovation Enablement**: Enables entirely new business models

## 🧬 Evolution Path

### Version 1.0 (Current)
- Basic token minting and transfer
- PostgreSQL ledger with Redis cache
- Multi-tenancy via authenticator
- Simple exchange rates
- Parent admin controls

### Version 2.0 (Planned)
- NFT achievements with rarity
- Automated market makers
- Staking and yield generation
- Cross-household bridges
- Advanced analytics

### Long-term Vision
- Full DeFi protocol suite
- Real blockchain integration
- AI-optimized economies
- Decentralized governance
- Global Vrooli token network

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with token config
    - PostgreSQL schema initialization
    - Redis cache warming
    - Admin UI deployment
    
  deployment_targets:
    - local: Docker Compose with persistent volumes
    - kubernetes: StatefulSet for ledger consistency
    - cloud: RDS + ElastiCache + Lambda
    
  revenue_model:
    - type: subscription
    - pricing_tiers:
        - family: $10/month
        - business: $50/month
        - enterprise: Custom
    - trial_period: 30 days
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: token-economy
    category: platform
    capabilities: 
      - Token minting and management
      - Multi-token wallets
      - Achievement NFTs
      - Exchange protocols
      - Transaction ledger
    interfaces:
      - api: http://localhost:11080/api/v1
      - cli: token-economy
      - events: token.*
      
  metadata:
    description: Universal token economy for scenario rewards
    keywords: [tokens, rewards, achievements, economy, wallet]
    dependencies: [scenario-authenticator]
    enhances: ALL
```

### Version Management
```yaml
versioning:
  current: 1.0.0
  minimum_compatible: 1.0.0
  
  breaking_changes:
    - version: 2.0.0
      description: Token ID format change
      migration: Script provided
      
  deprecations:
    - feature: Direct SQL token creation
      removal_version: 2.0.0
      alternative: Use API/CLI only
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Double-spend attack | Low | Critical | ACID transactions, locks |
| Integer overflow | Low | High | Use decimal type, bounds checking |
| Cache inconsistency | Medium | Medium | TTL + invalidation hooks |
| Database corruption | Low | Critical | WAL + backups + replicas |

### Operational Risks
- **Economic imbalance**: AI monitoring for inflation/deflation
- **Scenario misbehavior**: Rate limits and permission system
- **Parent lockout**: Recovery mechanism via support key
- **Token loss**: Wallet recovery via mnemonics

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: token-economy

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/token-economy
    - cli/install.sh
    - initialization/storage/postgres/schema.sql
    - initialization/storage/redis/config.json
    - scenario-test.yaml
    - ui/index.html
    - ui/package.json
    
  required_dirs:
    - api
    - cli
    - initialization
    - initialization/storage
    - ui
    - tests

resources:
  required: [postgres, redis]
  optional: [qdrant, minio, ollama]
  health_timeout: 60

tests:
  - name: "PostgreSQL ledger initialized"
    type: sql
    service: postgres
    query: "SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('tokens', 'wallets', 'transactions')"
    expect:
      rows: 
        - count: 3
        
  - name: "API creates token"
    type: http
    service: api
    endpoint: /api/v1/tokens/create
    method: POST
    body:
      symbol: TEST
      name: Test Token
      type: fungible
    expect:
      status: 201
      body:
        token_id: /^[0-9a-f-]{36}$/
        
  - name: "CLI mints tokens"
    type: exec
    command: ./cli/token-economy mint TEST 100 @alice
    expect:
      exit_code: 0
      output_contains: ["transaction_hash", "confirmed"]
      
  - name: "Transfer between wallets"
    type: http
    service: api
    endpoint: /api/v1/tokens/transfer
    method: POST
    body:
      from_wallet: @alice
      to_wallet: @bob
      token_id: TEST
      amount: 50
    expect:
      status: 200
      body:
        new_balance: 50
        
  - name: "Redis cache populated"
    type: exec
    command: resource-redis get "balance:@alice:TEST"
    expect:
      exit_code: 0
      output_contains: ["50"]
```

### Test Execution Gates
```bash
# All tests must pass via:
./test.sh --scenario token-economy --validation complete

# Individual test categories:
./test.sh --structure    # Verify file/directory structure
./test.sh --resources    # Check resource health
./test.sh --integration  # Run integration tests
./test.sh --performance  # Validate performance targets
```

### Performance Validation
- [ ] 1000 TPS sustained for 10 minutes
- [ ] Sub-100ms API response times
- [ ] No memory leaks over 24 hours
- [ ] Ledger consistency after 100K transactions

### Integration Validation
- [ ] Works with scenario-authenticator
- [ ] Other scenarios can mint/check tokens
- [ ] Parent controls functional
- [ ] Exchange rates working

### Capability Verification
- [ ] Full token lifecycle working
- [ ] Multi-tenancy isolated properly
- [ ] Achievement NFTs functional
- [ ] UI responsive and intuitive

## 📝 Implementation Notes

### Design Decisions
**PostgreSQL for ledger**: Chosen for ACID compliance and proven reliability
- Alternative considered: Blockchain
- Decision driver: Simplicity and performance
- Trade-offs: Less decentralization for more speed

**Redis for caching**: Chosen for speed and pub/sub
- Alternative considered: In-memory Go cache
- Decision driver: Shared across services
- Trade-offs: Additional dependency for performance

**Decimal type for amounts**: Prevent floating point errors
- Alternative considered: Integer with decimals
- Decision driver: Simplicity and accuracy
- Trade-offs: Slightly larger storage

### Known Limitations
- **No cross-household transfers**: v1 is household-isolated
  - Workaround: Manual admin bridge
  - Future fix: v2 will add bridges
  
- **No automatic market making**: Fixed exchange rates only
  - Workaround: Admin sets rates
  - Future fix: AMM in v2

### Security Considerations
- **Data Protection**: All wallet keys encrypted at rest
- **Access Control**: Scenario permissions via JWT
- **Audit Trail**: Every transaction logged immutably
- **Rate Limiting**: Prevent spam and DOS attacks

## 🔗 References

### Documentation
- README.md - User guide and quickstart
- docs/api.md - Complete API reference
- docs/integration.md - Scenario integration guide
- docs/economics.md - Token economy design patterns

### Related PRDs
- scenarios/scenario-authenticator/PRD.md
- scenarios/chore-tracking/PRD.md
- scenarios/retro-game-launcher/PRD.md

### External Resources
- [ERC-20 Token Standard](https://eips.ethereum.org/EIPS/eip-20)
- [ERC-721 NFT Standard](https://eips.ethereum.org/EIPS/eip-721)
- [Token Engineering Commons](https://tokenengineeringcommunity.github.io/)

---

**Last Updated**: 2025-09-07
**Status**: Draft
**Owner**: AI Agent
**Review Cycle**: Weekly validation against implementation