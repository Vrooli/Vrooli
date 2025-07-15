/**
 * Mock Consolidation Utilities
 * 
 * Helper functions to migrate existing tests to the new mock consolidation system
 * and analyze current mock usage patterns.
 */

import { readFile, writeFile } from "fs/promises";
import { join } from "path";

/**
 * Common mock patterns found in form tests
 */
export const COMMON_MOCK_PATTERNS = {
    hooks: [
        "useLazyFetch",
        "useStandardUpsertForm", 
        "useManagedObject",
        "useWindowSize",
        "useIsMobile",
        "useIsLargeScreen",
        "useTheme",
    ],
    components: [
        "TranslatedAdvancedInput",
        "MarkdownDisplay",
        "Box",
        "Dialog",
        "TopBar",
        "BottomActionsButtons",
        "BaseForm",
    ],
    contexts: [
        "SessionContext",
        "ThemeProvider",
    ],
} as const;

/**
 * Extract mock patterns from a test file
 */
export function analyzeMockPatterns(testFileContent: string): {
    foundMocks: string[];
    missingMocks: string[];
    suggestions: string[];
    complexity: "low" | "medium" | "high";
} {
    const lines = testFileContent.split("\n");
    const mockLines = lines.filter(line => line.trim().startsWith("vi.mock("));
    
    const foundMocks = mockLines.map(line => {
        const match = line.match(/vi\.mock\("([^"]+)"/);
        return match ? match[1] : "";
    }).filter(Boolean);

    const allCommonMocks = [
        ...COMMON_MOCK_PATTERNS.hooks,
        ...COMMON_MOCK_PATTERNS.components,
        ...COMMON_MOCK_PATTERNS.contexts,
    ];

    const missingMocks = allCommonMocks.filter(mockName => 
        !foundMocks.some(found => found.includes(mockName)),
    );

    const suggestions = [];
    const mockCount = mockLines.length;

    if (mockCount > 10) {
        suggestions.push("Consider using createFormTestSuite() to reduce mock boilerplate");
    }
    
    if (mockCount > 20) {
        suggestions.push("High mock complexity - excellent candidate for consolidation");
    }

    if (foundMocks.some(mock => mock.includes("formMocks"))) {
        suggestions.push("Already using central mock registry - good!");
    }

    const complexity = mockCount < 5 ? "low" : mockCount < 15 ? "medium" : "high";

    return {
        foundMocks,
        missingMocks,
        suggestions,
        complexity,
    };
}

/**
 * Generate a migration template for an existing test file
 */
export function generateMigrationTemplate(options: {
    componentName: string;
    existingMocks: string[];
    defaultProps: Record<string, any>;
    testType: "dialog" | "page" | "crud" | "custom";
}): string {
    const { componentName, existingMocks, defaultProps, testType } = options;
    
    const propsString = JSON.stringify(defaultProps, null, 2)
        .split("\n")
        .map(line => "    " + line)
        .join("\n");

    let suiteCreation = "";
    switch (testType) {
        case "dialog":
            suiteCreation = `const suite = commonFormTestSuites.dialogForm(${componentName}, baseProps);`;
            break;
        case "crud":
            suiteCreation = `const suites = commonFormTestSuites.crudForm(${componentName}, baseProps);
// Use suites.create for create mode tests
// Use suites.update for update mode tests`;
            break;
        case "custom":
        default:
            suiteCreation = `const suite = createFormTestSuite({
    component: ${componentName},
    defaultProps,
    // Add custom configuration as needed
});`;
    }

    return `// MIGRATION TEMPLATE for ${componentName}
// 
// 1. Replace the existing vi.mock() calls with the generated code from suite.setup.generateMockCode()
// 2. Replace manual test setup with the pre-configured suite
// 3. Convert individual tests to use suite.tests.* methods

import { describe, it, beforeEach, afterEach, vi } from "vitest";
import { createFormTestSuite, commonFormTestSuites } from "../../__test/helpers/createFormTestSuite.js";
import { ${componentName} } from "./${componentName}.js";

// TODO: Copy the generated mock code here (from suite.setup.generateMockCode())

const baseProps = ${propsString.trimStart()};

${suiteCreation}

describe("${componentName}", () => {
    // Simplified setup - just 2 lines!
    beforeEach(suite.setup.beforeEach);
    afterEach(suite.setup.afterEach);

    describe("Basic Functionality", () => {
        it("renders successfully", suite.tests.rendering);
        
        it("handles form submission", () => 
            suite.tests.submission({ /* form data */ })
        );
        
        it("handles cancellation", suite.tests.cancellation);
    });

    describe("Input Testing", () => {
        it("tests individual inputs", () => 
            suite.tests.singleInput("fieldName", "testValue", "text")
        );

        it("tests multiple inputs", () => 
            suite.tests.multipleInputs([
                ["field1", "value1", "text"],
                ["field2", "value2", "textarea"],
                // Add more as needed
            ])
        );
    });

    describe("Custom Tests", () => {
        it("demonstrates advanced testing", async () => {
            const { user, findElement } = suite.utils.render();
            
            // Your existing custom test logic here
            // This preserves all your existing test capabilities
        });
    });

    // Uncomment to generate mock code
    // it("generates mock setup code", () => {
    //     console.log(suite.setup.generateMockCode());
    // });
});

/*
MIGRATION CHECKLIST:
☐ 1. Copy generated mock code to top of file
☐ 2. Update imports to use new helpers
☐ 3. Replace manual setup/teardown with suite methods
☐ 4. Convert simple tests to use suite.tests.* methods
☐ 5. Preserve custom test logic using suite.utils.render()
☐ 6. Test that all functionality still works
☐ 7. Remove old boilerplate code
☐ 8. Update any custom mock configurations

BENEFITS AFTER MIGRATION:
✅ Reduced boilerplate (typically 40-60% less code)
✅ Consistent mock setup across all form tests
✅ Easy to add new tests with one-line methods
✅ Backwards compatible with existing test patterns
✅ Centralized mock management
*/`;
}

/**
 * Find all form test files that could benefit from consolidation
 */
export async function findCandidateTestFiles(baseDir: string): Promise<Array<{
    filePath: string;
    componentName: string;
    mockCount: number;
    complexity: "low" | "medium" | "high";
    recommendations: string[];
}>> {
    // This would typically use fs to scan directories
    // For now, return a mock implementation showing the concept
    
    return [
        {
            filePath: "src/views/objects/comment/CommentUpsert.test.tsx",
            componentName: "CommentUpsert",
            mockCount: 25,
            complexity: "high",
            recommendations: [
                "Excellent candidate for createFormTestSuite()",
                "Could reduce 215 lines to ~80 lines",
                "Use commonFormTestSuites.commentForm() pattern",
            ],
        },
        {
            filePath: "src/views/objects/project/ProjectCrud.test.tsx",
            componentName: "ProjectCrud",
            mockCount: 18,
            complexity: "medium",
            recommendations: [
                "Good candidate for commonFormTestSuites.crudForm()",
                "Separate create/update test suites would help organization",
            ],
        },
        // Add more as discovered...
    ];
}

/**
 * Validate that a test suite configuration will work
 */
export function validateTestSuiteConfig(config: {
    component: string;
    defaultProps: Record<string, any>;
    mockDependencies: string[];
}): {
    isValid: boolean;
    warnings: string[];
    errors: string[];
} {
    const warnings: string[] = [];
    const errors: string[] = [];

    // Check for required props
    const requiredProps = ["onClose", "onCompleted"];
    const missingProps = requiredProps.filter(prop => !(prop in config.defaultProps));
    
    if (missingProps.length > 0) {
        warnings.push(`Missing common props: ${missingProps.join(", ")}`);
    }

    // Check for common mock dependencies
    const commonHooks = ["useStandardUpsertForm", "useLazyFetch"];
    const missingHooks = commonHooks.filter(hook => 
        !config.mockDependencies.some(dep => dep.includes(hook)),
    );

    if (missingHooks.length > 0) {
        warnings.push(`Consider mocking common hooks: ${missingHooks.join(", ")}`);
    }

    // Validate prop types
    if (config.defaultProps.isCreate !== undefined && typeof config.defaultProps.isCreate !== "boolean") {
        errors.push("isCreate prop should be boolean");
    }

    return {
        isValid: errors.length === 0,
        warnings,
        errors,
    };
}

/**
 * Helper to extract component usage patterns from test files
 */
export function extractComponentPatterns(testFileContent: string): {
    formInputs: string[];
    buttonActions: string[];
    validationTests: string[];
    formBehaviors: string[];
} {
    const lines = testFileContent.split("\n");
    
    // Extract test IDs and form interactions
    const testIdPattern = /getByTestId\("([^"]+)"\)/g;
    const userTypePattern = /user\.type\([^,]+,\s*"([^"]+)"/g;
    const clickPattern = /user\.click\([^)]+\)/g;
    const expectPattern = /expect\([^)]+\)\.([^(]+)/g;

    const formInputs: string[] = [];
    const buttonActions: string[] = [];
    const validationTests: string[] = [];
    const formBehaviors: string[] = [];

    lines.forEach(line => {
        // Find test IDs
        let match;
        while ((match = testIdPattern.exec(line)) !== null) {
            const testId = match[1];
            if (testId.startsWith("input-")) {
                formInputs.push(testId.replace("input-", ""));
            } else if (testId.includes("button")) {
                buttonActions.push(testId);
            }
        }

        // Find validation patterns
        if (line.includes("toHaveTextContent") || line.includes("toBeInTheDocument")) {
            if (line.includes("error") || line.includes("required")) {
                validationTests.push(line.trim());
            }
        }

        // Find form behaviors
        if (line.includes("onCompleted") || line.includes("onClose") || line.includes("onSubmit")) {
            formBehaviors.push(line.trim());
        }
    });

    return {
        formInputs: [...new Set(formInputs)],
        buttonActions: [...new Set(buttonActions)],
        validationTests: [...new Set(validationTests)],
        formBehaviors: [...new Set(formBehaviors)],
    };
}

/**
 * Generate a compatibility report for migrating to the new system
 */
export function generateCompatibilityReport(testFileContent: string): {
    compatibility: "high" | "medium" | "low";
    migrationEffort: "minimal" | "moderate" | "significant";
    benefits: string[];
    challenges: string[];
    recommendation: string;
} {
    const analysis = analyzeMockPatterns(testFileContent);
    const patterns = extractComponentPatterns(testFileContent);
    
    const mockCount = analysis.foundMocks.length;
    const testComplexity = testFileContent.split("\n").length;
    
    let compatibility: "high" | "medium" | "low" = "high";
    let migrationEffort: "minimal" | "moderate" | "significant" = "minimal";
    
    if (mockCount > 20 || testComplexity > 300) {
        compatibility = "medium";
        migrationEffort = "moderate";
    }
    
    if (testFileContent.includes("testcontainers") || testFileContent.includes("real database")) {
        compatibility = "low";
        migrationEffort = "significant";
    }

    const benefits = [
        `Reduce boilerplate by ~${Math.round((mockCount / (testComplexity / 10)) * 60)}%`,
        "Standardize mock setup across tests",
        "Improve test maintainability",
    ];

    if (mockCount > 15) {
        benefits.push("Significant reduction in mock setup complexity");
    }

    const challenges = [];
    if (patterns.validationTests.length > 10) {
        challenges.push("Complex validation tests may need custom migration");
    }
    
    if (testFileContent.includes("integration")) {
        challenges.push("Integration tests may require additional configuration");
    }

    let recommendation = "";
    if (compatibility === "high" && migrationEffort === "minimal") {
        recommendation = "Highly recommended - easy migration with significant benefits";
    } else if (compatibility === "medium") {
        recommendation = "Recommended - moderate effort but good benefits";
    } else {
        recommendation = "Consider keeping existing approach for complex integration tests";
    }

    return {
        compatibility,
        migrationEffort,
        benefits,
        challenges,
        recommendation,
    };
}
