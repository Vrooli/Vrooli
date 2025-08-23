// Example JavaScript file for testing Claude Code in sandbox
// This file demonstrates various code patterns for testing

class Calculator {
    constructor() {
        this.result = 0;
    }
    
    // Basic arithmetic operations
    add(a, b) {
        return a + b;
    }
    
    subtract(a, b) {
        return a - b;
    }
    
    // Potential bug: no error handling for division by zero
    divide(a, b) {
        return a / b;
    }
    
    // Complex calculation with potential issues
    calculateCompound(principal, rate, time) {
        // TODO: Add input validation
        const amount = principal * Math.pow((1 + rate/100), time);
        return amount;
    }
}

// Function with security considerations
function processUserInput(input) {
    // Potential security issue: eval usage
    try {
        return eval(input);
    } catch (error) {
        return "Invalid input";
    }
}

// Async function for testing
async function fetchData(url) {
    // Missing error handling
    const response = await fetch(url);
    const data = await response.json();
    return data;
}

// Export for testing
module.exports = {
    Calculator,
    processUserInput,
    fetchData
};