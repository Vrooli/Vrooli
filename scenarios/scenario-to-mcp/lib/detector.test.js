#!/usr/bin/env node
/**
 * Basic test for MCP detector
 */

const MCPDetector = require('./detector');

async function runTests() {
    console.log('Testing MCP Detector...');

    const detector = new MCPDetector();

    try {
        // Test 1: List scenarios
        console.log('\n✓ Test 1: Listing scenarios');
        const scenarios = await detector.listScenarios();
        if (scenarios.length === 0) {
            console.error('  ✗ No scenarios found');
            process.exit(1);
        }
        console.log(`  Found ${scenarios.length} scenarios`);

        // Test 2: Scan all scenarios
        console.log('\n✓ Test 2: Scanning all scenarios');
        const results = await detector.scanAllScenarios();
        if (results.length === 0) {
            console.error('  ✗ Scan returned no results');
            process.exit(1);
        }
        console.log(`  Scanned ${results.length} scenarios`);

        // Test 3: Check specific scenario
        console.log('\n✓ Test 3: Checking specific scenario');
        const testScenario = scenarios[0];
        const status = await detector.checkScenarioMCPStatus(testScenario);
        if (!status || !status.name) {
            console.error('  ✗ Failed to get scenario status');
            process.exit(1);
        }
        console.log(`  Checked scenario: ${status.name}`);

        console.log('\n✅ All tests passed');
        process.exit(0);
    } catch (error) {
        console.error('\n✗ Test failed:', error.message);
        process.exit(1);
    }
}

runTests();
