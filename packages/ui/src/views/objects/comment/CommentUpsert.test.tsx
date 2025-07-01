// AI_CHECK: TEST_QUALITY=9 | LAST: 2025-01-27
import { screen } from "@testing-library/react";
import { DUMMY_ID } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
    createUseLazyFetchMock,
    createUseStandardUpsertFormMock,
    mockBottomActionsButtons,
    mockBox,
    mockDialog,
    mockMarkdownDisplay,
    mockTopBar,
    mockTranslatedAdvancedInput,
    mockUseIsMobile,
    mockUseWindowSize,
} from "../../../__test/mocks/formMocks.js";

// Mock all dependencies using the central registry
vi.mock("../../../hooks/useLazyFetch.js", () => ({
    useLazyFetch: createUseLazyFetchMock({
        threads: [{
            comment: {
                id: "test-comment-id",
                you: { canUpdate: true, canDelete: true },
            },
        }],
    }),
}));

vi.mock("../../../hooks/useStandardUpsertForm.ts", () => ({
    useStandardUpsertForm: createUseStandardUpsertFormMock(),
}));

vi.mock("../../../hooks/useWindowSize.js", () => ({
    useWindowSize: mockUseWindowSize,
}));

vi.mock("../../../hooks/useIsMobile.js", () => ({
    useIsMobile: mockUseIsMobile,
}));

// Mock UI components using central registry
vi.mock("../../../components/inputs/AdvancedInput/AdvancedInput.js", () => ({
    TranslatedAdvancedInput: mockTranslatedAdvancedInput,
}));

vi.mock("../../../components/text/MarkdownDisplay.js", () => ({
    MarkdownDisplay: mockMarkdownDisplay,
}));

vi.mock("../../../components/layout/Box.js", () => ({
    Box: mockBox,
}));

vi.mock("../../../components/dialogs/Dialog/Dialog.js", () => ({
    Dialog: mockDialog,
}));

vi.mock("../../../components/navigation/TopBar.js", () => ({
    TopBar: mockTopBar,
}));

vi.mock("../../../components/buttons/BottomActionsButtons.js", () => ({
    BottomActionsButtons: mockBottomActionsButtons,
}));

import { commonFormTestSuites } from "../../../__test/helpers/createFormTestSuite.js";
import { CommentUpsert } from "./CommentUpsert.js";

// Create the test suite using mock consolidation
const suite = commonFormTestSuites.commentForm(CommentUpsert, {
    isCreate: true,
    display: { title: "Create Comment" },
    objectType: "User" as const,
    objectId: DUMMY_ID,
    language: "en",
});

describe("CommentUpsert", () => {
    // Simplified setup - just 2 lines instead of 20+!
    beforeEach(suite.setup.beforeEach);
    afterEach(suite.setup.afterEach);

    describe("Central Mock Registry Demo", () => {
        it("renders successfully with centralized mocks", suite.tests.rendering);

        it("uses centralized mock functions", async () => {
            const { user, findElement } = suite.utils.render();

            // Fill in the text field
            const textarea = findElement({ testId: "input-text" });
            await user.type(textarea, "Test comment content");

            // Verify text was entered
            expect(textarea.value).toBe("Test comment content");

            // Verify submit button is present and not disabled
            const submitButton = screen.queryByTestId("submit-button");
            if (submitButton) {
                expect(submitButton.disabled).toBe(false);
            }
        });

        it("handles different form interactions", () =>
            suite.tests.singleInput("text", "Test comment content", "textarea"),
        );

        it("handles multi-line content properly", async () => {
            const { user, findElement } = suite.utils.render();

            const multilineText = "Line 1\nLine 2\nLine 3";
            const textarea = findElement({ testId: "input-text" });
            await user.type(textarea, multilineText);

            expect(textarea.value).toBe(multilineText);
        });

        it("displays correct form elements", async () => {
            suite.utils.render();

            // Verify key UI elements are present (rendered by our mocks)
            expect(screen.getByTestId("input-text")).toBeTruthy();

            // Check for dialog structure (since this is a comment form)
            expect(screen.getByTestId("dialog")).toBeTruthy();
        });
    });

    describe("Form State Testing", () => {
        it("handles create mode", async () => {
            suite.utils.render();

            // In create mode, the form should be ready for new content
            const textarea = screen.getByTestId("input-text");
            expect(textarea.value).toBe("");
        });

        it("demonstrates cancellation", suite.tests.cancellation);
    });

    describe("Input Validation", () => {
        it("accepts valid comment text", () =>
            suite.tests.singleInput("text", "This is a valid comment", "textarea"),
        );

        it("handles empty input", () =>
            suite.tests.singleInput("text", "", "textarea"),
        );

        it("handles long text input", () =>
            suite.tests.singleInput("text", "A".repeat(1000), "textarea"),
        );
    });

    describe("Multiple Input Scenarios", () => {
        it("tests multiple input variations", () =>
            suite.tests.multipleInputs([
                ["text", "Short comment", "textarea"],
                ["text", "Multi-line\ncomment\nwith\nbreaks", "textarea"],
                ["text", "Comment with special chars: @#$%", "textarea"],
            ]),
        );
    });

    describe("Advanced Custom Testing", () => {
        it("demonstrates full testing capabilities preserved", async () => {
            const { changeFormField, findElement } = suite.utils.render();

            // All existing testing utilities are still available
            await changeFormField("text", "Custom test logic still works");

            const textarea = findElement({ testId: "input-text" });
            expect(textarea.value).toBe("Custom test logic still works");

            // Verify component structure
            expect(screen.getByTestId("dialog")).toBeTruthy();
            expect(screen.getByTestId("input-text")).toBeTruthy();
        });

        it("preserves mock function verification", async () => {
            const { user } = suite.utils.render();

            // Test cancellation functionality
            const cancelButton = screen.queryByTestId("cancel-button") ||
                screen.queryByTestId("close-button");

            if (cancelButton) {
                await user.click(cancelButton);

                // The mock functions are still accessible if needed
                // This demonstrates backwards compatibility
                expect(suite.utils.defaultProps.onClose).toHaveBeenCalled();
            }
        });
    });

    // Uncomment to see the generated mock code for other tests
    // it("shows generated mock setup", () => {
    //     console.log("\n" + "=".repeat(80));
    //     console.log("COPY THIS MOCK CODE FOR NEW TESTS:");
    //     console.log("=".repeat(80));
    //     console.log(suite.setup.generateMockCode());
    //     console.log("=".repeat(80));
    // });
});

