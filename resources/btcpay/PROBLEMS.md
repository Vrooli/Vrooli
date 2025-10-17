# BTCPay Server Known Issues

## Current Limitations

### 1. Lightning Network Considerations
**Problem**: LND requires wallet initialization and blockchain sync
**Impact**: Lightning features not immediately available after setup
**Workaround**: Run `vrooli resource btcpay lightning setup` and wait for sync
**Status**: By design - blockchain sync required

### 2. Lightning Channel Requirements
**Problem**: Opening Lightning channels requires on-chain Bitcoin balance
**Impact**: Cannot create channels without funding the wallet first
**Solution**: Send Bitcoin to LND wallet, then open channels
**Status**: Standard Lightning Network requirement

### 3. API Key Requirements
**Problem**: Most BTCPay API operations require an API key
**Impact**: Store management and advanced features need manual key generation
**Workaround**: Generate API key through web interface at http://localhost:23000
**Status**: By design - security requirement

### 4. Bitcoin Core Requirement
**Problem**: Full BTCPay operation requires Bitcoin Core or equivalent blockchain node
**Impact**: Cannot process actual Bitcoin transactions without blockchain access
**Solution**: Either connect to external node or run Bitcoin Core container
**Status**: Not implemented - would require significant resources

### 5. Lightning Network Configuration
**Problem**: LND container requires bitcoin.active or litecoin.active configuration
**Impact**: Lightning container fails to start without proper chain configuration
**Workaround**: Need to provide proper lnd.conf with chain settings
**Status**: Partial implementation - container deploys but needs config file

### 6. ~~Crowdfunding CLI Argument Parsing~~ (Resolved 2025-09-26)
**Problem**: CLI framework passed flags as positional arguments to crowdfunding functions
**Impact**: Creating campaigns with --flag syntax didn't parse correctly
**Solution**: Updated btcpay::crowdfunding::create_campaign to handle both positional and flag-based arguments
**Status**: Resolved - Now supports both formats for backward compatibility

### 7. Missing Timeouts in Health Checks
**Problem**: Some health check calls missing timeout wrapper
**Impact**: Could hang indefinitely if service unresponsive
**Solution**: Added `timeout 5` to all curl commands
**Status**: Resolved (2025-09-26)

### 8. ~~POS Configuration Argument Parsing~~ (Resolved 2025-09-26) 
**Problem**: CLI framework could pass flags incorrectly to POS configure function
**Impact**: Store name and currency could get swapped when using flags
**Solution**: Updated btcpay::pos::configure to handle both positional and flag-based arguments
**Status**: Resolved - Now supports both formats for consistency

## Resolved Issues

### 1. PostgreSQL Connection String
**Problem**: Incorrect connection string format caused authentication failures
**Original**: `User ID=btcpay;Password=...`
**Fixed**: `Server=btcpay-postgres;Port=5432;Database=btcpayserver;User Id=btcpay;Password=...`
**Status**: Resolved

### 2. v2.0 Contract Compliance
**Problem**: Missing test structure per universal contract
**Solution**: Created test/run-tests.sh and test/phases/ directory structure
**Status**: Resolved

### 3. Port Registry Integration
**Problem**: Port was hardcoded instead of using central registry
**Solution**: Updated to source from scripts/resources/port_registry.sh
**Status**: Resolved

### 4. PostgreSQL Startup Race Condition
**Problem**: BTCPay started before PostgreSQL was ready
**Solution**: Added pg_isready wait loop with 30-second timeout
**Status**: Resolved

### 5. NBXplorer Integration
**Problem**: BTCPay Server requires NBXplorer for blockchain synchronization
**Solution**: Added NBXplorer container with PostgreSQL configuration
**Status**: Resolved - NBXplorer now starts automatically with BTCPay

## Configuration Notes

- BTCPay runs on port 23000 (mapped from internal 49392)
- PostgreSQL uses standard port 5432 internally
- Both containers use btcpay-network Docker network
- Data persisted in ${BTCPAY_DATA_DIR}

## Testing Limitations

- Smoke tests can verify container status but not full functionality
- Integration tests limited without NBXplorer
- Cannot test payment processing without blockchain connection