# Token Economy Integration Guide

## For Scenario Developers

This guide shows how to integrate your scenario with the Token Economy to enable rewards, payments, and achievements.

## 🚀 Quick Integration

### 1. Register Your Scenario

First, register your scenario to get minting permissions:

```bash
# Contact token-economy admin to register
# You'll receive:
# - Scenario API key
# - Allowed token types
# - Minting limits
```

### 2. Basic Integration Pattern

```go
// In your scenario's API
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

const TOKEN_ECONOMY_API = "http://localhost:11080/api/v1"

// Reward user for completing task
func rewardUser(userWallet string, amount float64) error {
    payload := map[string]interface{}{
        "token_id": "YOUR_TOKEN",
        "to_wallet": userWallet,
        "amount": amount,
        "metadata": map[string]string{
            "scenario": "your-scenario",
            "reason": "task_completion",
        },
    }
    
    data, _ := json.Marshal(payload)
    resp, err := http.Post(
        TOKEN_ECONOMY_API + "/tokens/mint",
        "application/json",
        bytes.NewBuffer(data),
    )
    
    return err
}

// Check if user has enough tokens
func checkBalance(userWallet string, required float64) (bool, error) {
    resp, err := http.Get(TOKEN_ECONOMY_API + "/wallets/" + userWallet + "/balance")
    if err != nil {
        return false, err
    }
    
    var result struct {
        Balances []struct {
            TokenID string  `json:"token_id"`
            Amount  float64 `json:"amount"`
        } `json:"balances"`
    }
    
    json.NewDecoder(resp.Body).Decode(&result)
    
    for _, balance := range result.Balances {
        if balance.TokenID == "YOUR_TOKEN" {
            return balance.Amount >= required, nil
        }
    }
    
    return false, nil
}
```

### 3. JavaScript/Node.js Integration

```javascript
const TOKEN_ECONOMY_API = 'http://localhost:11080/api/v1';

// Reward user
async function rewardUser(userWallet, amount) {
    const response = await fetch(`${TOKEN_ECONOMY_API}/tokens/mint`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            token_id: 'YOUR_TOKEN',
            to_wallet: userWallet,
            amount: amount,
            metadata: {
                scenario: 'your-scenario',
                timestamp: Date.now()
            }
        })
    });
    
    return response.json();
}

// Check balance before allowing action
async function canAfford(userWallet, cost) {
    const response = await fetch(`${TOKEN_ECONOMY_API}/wallets/${userWallet}/balance`);
    const data = await response.json();
    
    const balance = data.balances.find(b => b.token_id === 'YOUR_TOKEN');
    return balance && balance.amount >= cost;
}

// Charge tokens for service
async function chargeUser(userWallet, amount) {
    // Transfer to your scenario's treasury wallet
    const response = await fetch(`${TOKEN_ECONOMY_API}/tokens/transfer`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            from_wallet: userWallet,
            to_wallet: 'YOUR_SCENARIO_WALLET',
            token_id: 'YOUR_TOKEN',
            amount: amount
        })
    });
    
    return response.json();
}
```

## 📋 Common Integration Patterns

### Pattern 1: Task Rewards
```javascript
// User completes a task
async function onTaskComplete(userId, taskId, difficulty) {
    const reward = calculateReward(difficulty);
    await rewardUser(getUserWallet(userId), reward);
    
    // Optional: Award achievement for milestone
    if (await isAchievementEarned(userId, taskId)) {
        await awardAchievement(userId, 'task-master');
    }
}
```

### Pattern 2: Pay-to-Play
```javascript
// User wants to access premium feature
async function accessPremiumFeature(userId) {
    const wallet = getUserWallet(userId);
    const cost = 10; // tokens
    
    if (!await canAfford(wallet, cost)) {
        throw new Error('Insufficient tokens');
    }
    
    await chargeUser(wallet, cost);
    // Grant access to feature
}
```

### Pattern 3: Subscription Model
```javascript
// Daily/weekly token deduction
async function processSubscription(userId) {
    const wallet = getUserWallet(userId);
    const dailyCost = 5;
    
    try {
        await chargeUser(wallet, dailyCost);
        return { active: true };
    } catch (error) {
        // Subscription expired
        return { active: false, reason: 'insufficient_funds' };
    }
}
```

### Pattern 4: Achievement System
```javascript
// Award NFT achievement
async function awardAchievement(userId, achievementType) {
    const response = await fetch(`${TOKEN_ECONOMY_API}/achievements/award`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            user_id: userId,
            achievement: achievementType,
            scenario: 'your-scenario',
            rarity: calculateRarity(achievementType)
        })
    });
    
    return response.json();
}
```

## 🎯 Best Practices

### 1. Token Design
- Create meaningful token names (CHORE, STUDY, GAME)
- Set appropriate decimal places (usually 2 for simplicity)
- Consider max supply to prevent inflation

### 2. Balance Management
- Cache balance checks (1-minute TTL recommended)
- Batch mint operations when possible
- Use metadata to track mint reasons

### 3. Error Handling
```javascript
async function safeTransfer(from, to, amount) {
    try {
        const result = await transfer(from, to, amount);
        return { success: true, data: result };
    } catch (error) {
        if (error.message.includes('Insufficient balance')) {
            return { success: false, error: 'not_enough_tokens' };
        }
        // Log other errors
        console.error('Transfer failed:', error);
        return { success: false, error: 'transfer_failed' };
    }
}
```

### 4. Multi-tenancy
```javascript
// Always respect household isolation
async function getHouseholdTokens(householdId) {
    // Only return tokens for this household
    const tokens = await fetchTokens();
    return tokens.filter(t => t.household_id === householdId);
}
```

## 🔄 Webhook Integration

Subscribe to token events:

```javascript
// Listen for token events
const WebSocket = require('ws');
const ws = new WebSocket('ws://localhost:11082/events');

ws.on('message', (data) => {
    const event = JSON.parse(data);
    
    switch(event.type) {
        case 'token.transferred':
            if (event.to_wallet === YOUR_WALLET) {
                onTokensReceived(event);
            }
            break;
            
        case 'achievement.earned':
            if (event.scenario === 'your-scenario') {
                updateAchievementDisplay(event);
            }
            break;
    }
});
```

## 📊 Analytics Integration

Track your scenario's token economy:

```javascript
async function getScenarioStats() {
    const stats = await fetch(`${TOKEN_ECONOMY_API}/admin/analytics?scenario=your-scenario`);
    return stats.json();
}

// Returns:
{
    tokens_minted: 10000,
    tokens_burned: 500,
    active_users: 150,
    transaction_volume: 25000,
    top_earners: [...],
    achievement_stats: {...}
}
```

## 🚨 Common Issues

### Issue 1: Permission Denied
```
Error: Scenario not authorized to mint tokens
```
**Solution**: Register your scenario with token-economy admin

### Issue 2: Rate Limiting
```
Error: Too many requests
```
**Solution**: Implement request batching and caching

### Issue 3: Transaction Failed
```
Error: Transaction could not be processed
```
**Solution**: Check wallet addresses are valid, amounts are positive

## 📚 Additional Resources

- [API Documentation](./api.md)
- [Token Economics Best Practices](./economics.md)
- [Security Guidelines](./security.md)

## 💬 Support

For integration support:
- Check existing integrations in other scenarios
- Review test cases in `tests/integration-test.sh`
- Contact the Token Economy maintainers

---

Remember: **Every token minted should provide real value to users!**