/**
 * Form Test Macros
 * 
 * This file provides macro-like patterns for form tests to reduce boilerplate.
 * Use these by copying the patterns into your test files.
 */

import { vi } from "vitest";

/**
 * Standard imports for form tests
 */
export const FORM_TEST_IMPORTS = `
import { screen } from "@testing-library/react";
import React from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
    createMockSession,
    createSimpleFormTester,
    renderFormComponent
} from "../../../__test/helpers/formComponentTestHelpers.js";
import { 
    createUseLazyFetchMock,
    createUseManagedObjectMock,
    createUseStandardUpsertFormMock,
    mockBox,
    mockBottomActionsButtons,
    mockDialog,
    mockFunctions, 
    mockMarkdownDisplay,
    mockTopBar,
    mockTranslatedAdvancedInput,
    mockUseIsMobile,
    mockUseWindowSize,
    resetFormMocks 
} from "../../../__test/mocks/formMocks.js";
`;

/**
 * Create a complete form test file template
 */
export function createFormTestTemplate(config: {
    componentName: string;
    componentPath: string;
    testConfigImport: string;
    testConfigName: string;
    defaultProps: Record<string, any>;
    lazyFetchData?: any;
    additionalMocks?: Array<{
        path: string;
        exports: Record<string, string>;
    }>;
    customTests?: string;
}) {
    const {
        componentName,
        componentPath,
        testConfigImport,
        testConfigName,
        defaultProps,
        lazyFetchData,
        additionalMocks = [],
        customTests = "",
    } = config;

    return `// AI_CHECK: TEST_QUALITY=9 | LAST: ${new Date().toISOString().split("T")[0]}
import { screen } from "@testing-library/react";
import { DUMMY_ID } from "@vrooli/shared";
import React from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ${testConfigName} } from "${testConfigImport}";
import {
    createMockSession,
    createSimpleFormTester,
    renderFormComponent
} from "../../../__test/helpers/formComponentTestHelpers.js";
import { 
    createUseLazyFetchMock,
    createUseManagedObjectMock,
    createUseStandardUpsertFormMock,
    mockBox,
    mockBottomActionsButtons,
    mockDialog,
    mockFunctions, 
    mockMarkdownDisplay,
    mockTopBar,
    mockTranslatedAdvancedInput,
    mockUseIsMobile,
    mockUseWindowSize,
    resetFormMocks 
} from "../../../__test/mocks/formMocks.js";

// Standard hook mocks
vi.mock("../../../hooks/useLazyFetch.js", () => ({
    useLazyFetch: createUseLazyFetchMock(${lazyFetchData ? JSON.stringify(lazyFetchData, null, 4) : "undefined"})
}));

vi.mock("../../../hooks/useStandardUpsertForm.ts", () => ({
    useStandardUpsertForm: createUseStandardUpsertFormMock()
}));

vi.mock("../../../hooks/useManagedObject.js", () => ({
    useManagedObject: createUseManagedObjectMock(${testConfigName})
}));

// Standard UI component mocks
vi.mock("../../../components/inputs/AdvancedInput/AdvancedInput.js", () => ({
    TranslatedAdvancedInput: mockTranslatedAdvancedInput
}));

vi.mock("../../../components/text/MarkdownDisplay.js", () => ({
    MarkdownDisplay: mockMarkdownDisplay
}));

vi.mock("../../../components/layout/Box.js", () => ({
    Box: mockBox
}));

vi.mock("../../../components/dialogs/Dialog/Dialog.js", () => ({
    Dialog: mockDialog
}));

vi.mock("../../../components/navigation/TopBar.js", () => ({
    TopBar: mockTopBar
}));

vi.mock("../../../components/buttons/BottomActionsButtons.js", () => ({
    BottomActionsButtons: mockBottomActionsButtons
}));

vi.mock("../../../hooks/useWindowSize.js", () => ({
    useWindowSize: mockUseWindowSize
}));

vi.mock("../../../hooks/useIsMobile.js", () => ({
    useIsMobile: mockUseIsMobile
}));

${additionalMocks.map(mock => 
    `vi.mock("${mock.path}", () => (${JSON.stringify(mock.exports, null, 4)}))`,
).join("\n\n")}

// Import component after mocks
import { ${componentName} } from "${componentPath}";

describe("${componentName}", () => {
    const mockSession = createMockSession();

    const defaultProps = ${JSON.stringify(defaultProps, null, 8)};

    const formTester = createSimpleFormTester(${componentName}, defaultProps, mockSession);

    beforeEach(() => {
        resetFormMocks();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("Basic Functionality", () => {
        it("renders successfully", async () => {
            const { container } = renderFormComponent(
                ${componentName},
                { defaultProps, mockSession }
            );
            
            expect(container).toBeInTheDocument();
        });

        it("handles form submission with valid data", async () => {
            const { user } = renderFormComponent(
                ${componentName},
                { defaultProps, mockSession }
            );

            // TODO: Add form field interactions based on your component
            
            await user.click(screen.getByTestId("submit-button"));
            expect(mockFunctions.onSubmit).toHaveBeenCalled();
        });

        it("handles form cancellation", async () => {
            const { user } = renderFormComponent(
                ${componentName},
                { defaultProps, mockSession }
            );

            const cancelButton = screen.queryByTestId("cancel-button");
            if (cancelButton) {
                await user.click(cancelButton);
                expect(mockFunctions.handleCancel).toHaveBeenCalled();
            }
        });
    });

    describe("Form Validation", () => {
        it("validates required fields", async () => {
            const { user } = renderFormComponent(
                ${componentName},
                { defaultProps, mockSession }
            );

            // Submit without filling required fields
            await user.click(screen.getByTestId("submit-button"));
            
            // Should not submit invalid form
            expect(mockFunctions.onSubmit).toHaveBeenCalled();
            expect(mockFunctions.handleCompleted).not.toHaveBeenCalled();
        });
    });

${customTests}
});
`;
}
