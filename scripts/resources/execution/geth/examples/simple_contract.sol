// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Simple Storage Contract
// Demonstrates basic smart contract functionality
contract SimpleStorage {
    uint256 private storedValue;
    address public owner;
    
    event ValueChanged(uint256 newValue, address changedBy);
    
    constructor() {
        owner = msg.sender;
        storedValue = 0;
    }
    
    // Store a value
    function store(uint256 value) public {
        storedValue = value;
        emit ValueChanged(value, msg.sender);
    }
    
    // Retrieve the stored value
    function retrieve() public view returns (uint256) {
        return storedValue;
    }
    
    // Increment the stored value
    function increment() public {
        storedValue++;
        emit ValueChanged(storedValue, msg.sender);
    }
    
    // Only owner can reset
    function reset() public {
        require(msg.sender == owner, "Only owner can reset");
        storedValue = 0;
        emit ValueChanged(0, msg.sender);
    }
}