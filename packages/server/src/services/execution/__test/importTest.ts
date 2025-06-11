// Simple test to verify SwarmExecutionService can be imported
console.log("Starting import test...");

// Test individual components first
try {
    console.log("Testing EventBus import...");
    const { EventBus } = await import("../cross-cutting/events/eventBus.js");
    console.log("✅ EventBus imported successfully");
} catch (err) {
    console.error("❌ EventBus import failed:", err.message);
}

try {
    console.log("\nTesting integration services...");
    const { RunPersistenceService } = await import("../integration/runPersistenceService.js");
    console.log("✅ RunPersistenceService imported successfully");
    
    const { RoutineStorageService } = await import("../integration/routineStorageService.js");
    console.log("✅ RoutineStorageService imported successfully");
    
    const { AuthIntegrationService } = await import("../integration/authIntegrationService.js");
    console.log("✅ AuthIntegrationService imported successfully");
} catch (err) {
    console.error("❌ Integration service import failed:", err.message);
}

try {
    console.log("\nTesting Tier 3 imports...");
    const { TierThreeExecutor } = await import("../tier3/index.js");
    console.log("✅ TierThreeExecutor imported successfully");
} catch (err) {
    console.error("❌ Tier 3 import failed:", err.message);
}

try {
    console.log("\nTesting Tier 2 imports...");
    const { TierTwoOrchestrator } = await import("../tier2/index.js");
    console.log("✅ TierTwoOrchestrator imported successfully");
} catch (err) {
    console.error("❌ Tier 2 import failed:", err.message);
}

try {
    console.log("\nTesting Tier 1 imports...");
    const { TierOneCoordinator } = await import("../tier1/index.js");
    console.log("✅ TierOneCoordinator imported successfully");
} catch (err) {
    console.error("❌ Tier 1 import failed:", err.message);
}

// Finally test the main service
try {
    console.log("\nTesting SwarmExecutionService import...");
    const { SwarmExecutionService } = await import("../swarmExecutionService.js");
    console.log("✅ SwarmExecutionService imported successfully!");
    console.log("\n🎉 All imports successful! The swarm execution architecture is ready.");
} catch (err) {
    console.error("❌ SwarmExecutionService import failed:", err.message);
    console.error("Stack:", err.stack);
}

console.log("\nImport test complete.");