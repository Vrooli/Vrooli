/**
 * AI Code Validation Example
 * This demonstrates how AI-generated code can be validated using Judge0
 */

// Simulated AI-generated code (in practice, this would come from Ollama or another AI)
const aiGeneratedCode = `
function isPrime(n) {
    if (n <= 1) return false;
    if (n <= 3) return true;
    if (n % 2 === 0 || n % 3 === 0) return false;
    
    for (let i = 5; i * i <= n; i += 6) {
        if (n % i === 0 || n % (i + 2) === 0) {
            return false;
        }
    }
    return true;
}

// Test the function
const testCases = [2, 3, 4, 17, 20, 29, 100];
testCases.forEach(num => {
    console.log(\`\${num} is \${isPrime(num) ? 'prime' : 'not prime'}\`);
});
`;

// Test cases for validation
const testCases = [
    { input: "2", expected: "2 is prime" },
    { input: "4", expected: "4 is not prime" },
    { input: "17", expected: "17 is prime" },
    { input: "100", expected: "100 is not prime" }
];

// Validation function
async function validateAICode(code, language, tests) {
    console.log("🤖 Validating AI-generated code...");
    console.log("📝 Code to validate:");
    console.log(code);
    console.log("\n🧪 Running test cases...");
    
    let passed = 0;
    let failed = 0;
    
    for (const test of tests) {
        // In practice, this would call Judge0 API
        // For this example, we'll simulate the validation
        console.log(`\nTest: Checking if output contains "${test.expected}"`);
        
        // Simulate Judge0 submission
        const result = await submitToJudge0(code, language, test.input);
        
        if (result.status === "accepted" && result.output.includes(test.expected)) {
            console.log("✅ PASSED");
            passed++;
        } else {
            console.log("❌ FAILED");
            console.log(`   Expected: ${test.expected}`);
            console.log(`   Got: ${result.output}`);
            failed++;
        }
    }
    
    console.log(`\n📊 Results: ${passed} passed, ${failed} failed`);
    return { passed, failed, total: tests.length };
}

// Simulated Judge0 API call
async function submitToJudge0(code, language, stdin) {
    // In practice, this would be an actual API call:
    // const response = await fetch('http://localhost:2358/submissions', {
    //     method: 'POST',
    //     headers: {
    //         'Content-Type': 'application/json',
    //         'X-Auth-Token': API_KEY
    //     },
    //     body: JSON.stringify({
    //         source_code: code,
    //         language_id: getLanguageId(language),
    //         stdin: stdin,
    //         wait: true
    //     })
    // });
    
    // Simulated response
    return {
        status: "accepted",
        output: `${stdin} is ${isPrime(parseInt(stdin)) ? 'prime' : 'not prime'}`
    };
}

// Helper function to check if a number is prime (for simulation)
function isPrime(n) {
    if (n <= 1) return false;
    if (n <= 3) return true;
    if (n % 2 === 0 || n % 3 === 0) return false;
    
    for (let i = 5; i * i <= n; i += 6) {
        if (n % i === 0 || n % (i + 2) === 0) {
            return false;
        }
    }
    return true;
}

// Run the validation
validateAICode(aiGeneratedCode, "javascript", testCases)
    .then(results => {
        if (results.failed === 0) {
            console.log("\n🎉 AI-generated code is valid and ready for use!");
        } else {
            console.log("\n⚠️  AI-generated code needs revision");
        }
    });

// Integration example with Vrooli AI tiers
console.log(`
📋 Integration with Vrooli AI Tiers:

1. Tier 1 (Coordination): "Generate a prime number checker"
2. Tier 2 (Process): Routes to code generation AI
3. AI generates code (like above)
4. Tier 3 (Execution): Validates code with Judge0
5. Results returned to user with confidence score

This ensures all AI-generated code is tested before deployment!
`);