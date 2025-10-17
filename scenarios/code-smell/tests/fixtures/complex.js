// Test file with complex code smell patterns

// Hard-coded port
const API_URL = "http://localhost:3000/api";

// Hard-coded home path
const CONFIG_PATH = "/home/matthalloran8/Vrooli/config";

// Empty catch block
try {
    doSomething();
} catch (e) {
}

// Deep nesting
function deeplyNested() {
    if (condition1) {
        if (condition2) {
            if (condition3) {
                if (condition4) {
                    if (condition5) {
                        // Too deep!
                        return "nested";
                    }
                }
            }
        }
    }
}

// Magic number
const timeout = 86400000; // What is this number?

// Unnecessary return await
async function fetchData() {
    return await fetch('/api/data');
}

// Console.log in production
function processData(data) {
    console.log("Processing:", data);
    return data;
}

// TODO: Fix this later
// FIXME: This is broken
// HACK: Temporary workaround

module.exports = {
    deeplyNested,
    fetchData,
    processData
};