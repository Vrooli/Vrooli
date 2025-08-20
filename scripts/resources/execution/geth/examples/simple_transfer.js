// Simple Ether Transfer Example
// This script demonstrates how to send Ether between accounts

// Get accounts
var accounts = eth.accounts;
console.log("Available accounts:", accounts);

if (accounts.length < 2) {
    console.log("Need at least 2 accounts for transfer demo");
    console.log("Creating new account...");
    personal.newAccount("password123");
    accounts = eth.accounts;
}

var sender = accounts[0];
var receiver = accounts[1] || accounts[0]; // Use same account if only one exists

// Check initial balances
console.log("Sender balance:", web3.fromWei(eth.getBalance(sender), "ether"), "ETH");
console.log("Receiver balance:", web3.fromWei(eth.getBalance(receiver), "ether"), "ETH");

// Unlock sender account (dev mode auto-unlocks)
if (eth.mining || eth.syncing === false) {
    try {
        personal.unlockAccount(sender, "", 0);
    } catch(e) {
        console.log("Account already unlocked or in dev mode");
    }
}

// Send 1 Ether
var amount = web3.toWei(1, "ether");
var txHash = eth.sendTransaction({
    from: sender,
    to: receiver,
    value: amount,
    gas: 21000
});

console.log("Transaction sent! Hash:", txHash);

// Wait for transaction to be mined
var receipt = null;
var attempts = 0;
while (!receipt && attempts < 30) {
    receipt = eth.getTransactionReceipt(txHash);
    if (!receipt) {
        admin.sleep(1);
        attempts++;
    }
}

if (receipt) {
    console.log("Transaction mined!");
    console.log("Block number:", receipt.blockNumber);
    console.log("Gas used:", receipt.gasUsed);
    
    // Check final balances
    console.log("\nFinal balances:");
    console.log("Sender balance:", web3.fromWei(eth.getBalance(sender), "ether"), "ETH");
    console.log("Receiver balance:", web3.fromWei(eth.getBalance(receiver), "ether"), "ETH");
} else {
    console.log("Transaction not mined after 30 seconds");
}