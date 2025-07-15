/**
 * Simplified Form Test Suite Creator
 * 
 * This provides a practical solution for mock consolidation that works around
 * Vitest's vi.mock() top-level requirement while dramatically reducing boilerplate.
 * 
 * Based on previous attempts in UIFormTestFactory and setupFormTestMocks, but
 * simplified for real-world usage.
 */

// AI_CHECK: TYPE_SAFETY=replaced-8-any-types-with-specific-types | LAST: 2025-06-28

import { type ComponentType } from "react";
import { vi } from "vitest";
import { 
    createUseLazyFetchMock,
    createUseManagedObjectMock,
    createUseStandardUpsertFormMock,
    mockBox,
    mockBottomActionsButtons,
    mockDialog,
    mockMarkdownDisplay,
    mockTopBar,
    mockTranslatedAdvancedInput,
    mockUseIsMobile,
    mockUseWindowSize,
    resetFormMocks, 
} from "../mocks/formMocks.js";
import { 
    createSimpleFormTester,
    createMockSession,
    renderFormComponent,
    type FormComponentTestConfig,
} from "./formComponentTestHelpers.js";

/**
 * Configuration for creating a form test suite
 */
export interface FormTestSuiteConfig<TProps = Record<string, unknown>> {
    /** The form component to test */
    component: ComponentType<TProps>;
    /** Default props for the component */
    defaultProps: TProps;
    /** Optional mock session (will create default if not provided) */
    mockSession?: Record<string, unknown>;
    /** Data for useLazyFetch mock */
    lazyFetchData?: Record<string, unknown>;
    /** Additional custom mocks */
    customMocks?: Record<string, unknown>;
    /** Form test config (if using fixtures) */
    formTestConfig?: Record<string, unknown>;
}

/**
 * Result of creating a form test suite
 */
export interface FormTestSuite<TProps = Record<string, unknown>> {
    /** Pre-configured test utilities */
    utils: {
        /** Render the component with all mocks set up */
        render: () => ReturnType<typeof renderFormComponent>;
        /** Simple form tester for one-line tests */
        tester: ReturnType<typeof createSimpleFormTester>;
        /** Mock session */
        mockSession: Record<string, unknown>;
        /** Default props */
        defaultProps: TProps;
    };
    
    /** Standard test functions ready to use */
    tests: {
        /** Test basic component rendering */
        rendering: () => Promise<void>;
        /** Test form submission with valid data */
        submission: (formData?: Record<string, unknown>) => Promise<void>;
        /** Test form cancellation */
        cancellation: () => Promise<void>;
        /** Test a single input element */
        singleInput: (field: string, value: unknown, inputType?: string) => Promise<void>;
        /** Test multiple input elements */
        multipleInputs: (fields: Array<[string, unknown, string?]>) => Promise<void>;
    };
    
    /** Mock setup and cleanup functions */
    setup: {
        /** Call this in beforeEach */
        beforeEach: () => void;
        /** Call this in afterEach */
        afterEach: () => void;
        /** Generate the vi.mock() code (for copy-paste) */
        generateMockCode: () => string;
    };
}

/**
 * Create a complete form test suite with mocks and utilities
 * 
 * This function creates all the testing utilities you need for a form component,
 * but still requires you to set up vi.mock() calls at the top level (due to Vitest limitation).
 * 
 * Use the generateMockCode() function to get the exact vi.mock() calls to copy-paste.
 */
export function createFormTestSuite<TProps = Record<string, unknown>>(
    config: FormTestSuiteConfig<TProps>,
): FormTestSuite<TProps> {
    const {
        component,
        defaultProps,
        mockSession = createMockSession(),
        lazyFetchData,
        customMocks = {},
        formTestConfig,
    } = config;

    // Create test utilities
    const formTester = createSimpleFormTester(component, defaultProps, mockSession);
    
    const renderConfig: FormComponentTestConfig<TProps> = {
        defaultProps,
        mockSession,
    };

    // Standard test functions
    const tests = {
        async rendering() {
            const { container } = renderFormComponent(component, renderConfig);
            expect(container).toBeTruthy();
            expect(container.firstChild).toBeTruthy();
        },

        async submission(formData?: Record<string, unknown>) {
            const { fillForm, findElement, user } = renderFormComponent(component, renderConfig);
            
            if (formData) {
                await fillForm(formData);
            }
            
            // For CommentUpsert, we need to fill the text field to enable submission
            if (!formData) {
                const textField = findElement({ testId: "input-text" });
                await user.type(textField, "Test comment content");
            }
            
            const submitButton = findElement({ testId: "submit-button" });
            await user.click(submitButton);
            
            // Verify callback was called (assumes it's a mock)
            const propsWithCallbacks = defaultProps as Record<string, unknown>;
            const callback = propsWithCallbacks.onCompleted || propsWithCallbacks.onSubmit;
            if (callback && vi.isMockFunction(callback)) {
                expect(callback).toHaveBeenCalled();
            }
        },

        async cancellation() {
            const { findElement, user } = renderFormComponent(component, renderConfig);
            
            const cancelButton = findElement({ testId: "cancel-button" });
            await user.click(cancelButton);
            
            // Verify callback was called
            const propsWithCallbacks = defaultProps as Record<string, unknown>;
            const callback = propsWithCallbacks.onClose || propsWithCallbacks.onCancel;
            if (callback && vi.isMockFunction(callback)) {
                expect(callback).toHaveBeenCalled();
            }
        },

        async singleInput(field: string, value: unknown, inputType = "text") {
            return formTester.testElement(field, value, inputType as "text" | "checkbox" | "select");
        },

        async multipleInputs(fields: Array<[string, unknown, string?]>) {
            return formTester.testElements(fields as Array<[string, unknown, ("text" | "checkbox" | "select")?]>);
        },
    };

    // Mock setup functions
    const setup = {
        beforeEach: () => {
            resetFormMocks();
        },

        afterEach: () => {
            vi.restoreAllMocks();
        },

        generateMockCode: () => {
            const componentName = component.displayName || component.name || "UnknownComponent";
            return generateFormMockSetupCode({
                componentName,
                lazyFetchData,
                customMocks,
                formTestConfig: formTestConfig ? "formTestConfig" : undefined,
            });
        },
    };

    return {
        utils: {
            render: () => renderFormComponent(component, renderConfig),
            tester: formTester,
            mockSession,
            defaultProps,
        },
        tests,
        setup,
    };
}

/**
 * Generate the vi.mock() code needed for a form test
 * This addresses the previous attempts' code generation approach but simplifies it
 */
function generateFormMockSetupCode(options: {
    componentName: string;
    lazyFetchData?: Record<string, unknown>;
    customMocks?: Record<string, unknown>;
    formTestConfig?: string;
}): string {
    const { componentName, lazyFetchData, customMocks = {}, formTestConfig } = options;

    const lazyFetchDataString = lazyFetchData 
        ? JSON.stringify(lazyFetchData, null, 2)
        : `{
    threads: [{
        comment: {
            id: "test-comment-id",
            you: { canUpdate: true, canDelete: true }
        }
    }]
}`;

    const customMockEntries = Object.entries(customMocks)
        .map(([path, mock]) => `vi.mock("${path}", () => (${JSON.stringify(mock)}));`)
        .join("\n");

    return `// Generated mock setup for ${componentName}
// Copy this code to the top of your test file (before imports)

import { vi } from "vitest";

// Mock all dependencies using the central registry
vi.mock("../../../hooks/useLazyFetch.js", () => ({
    useLazyFetch: createUseLazyFetchMock(${lazyFetchDataString})
}));

vi.mock("../../../hooks/useStandardUpsertForm.ts", () => ({
    useStandardUpsertForm: createUseStandardUpsertFormMock()
}));

${formTestConfig ? `vi.mock("../../../hooks/useManagedObject.js", () => ({
    useManagedObject: createUseManagedObjectMock(${formTestConfig})
}));` : ""}

vi.mock("../../../hooks/useWindowSize.js", () => ({
    useWindowSize: mockUseWindowSize
}));

vi.mock("../../../hooks/useIsMobile.js", () => ({
    useIsMobile: mockUseIsMobile
}));

// Mock UI components using central registry
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

${customMockEntries}

// Import mock utilities AFTER mocks
import { 
    createUseLazyFetchMock,
    createUseManagedObjectMock,
    createUseStandardUpsertFormMock,
    mockBox,
    mockBottomActionsButtons,
    mockDialog,
    mockMarkdownDisplay,
    mockTopBar,
    mockTranslatedAdvancedInput,
    mockUseIsMobile,
    mockUseWindowSize,
    resetFormMocks 
} from "../../../__test/mocks/formMocks.js";

// Import test helpers
import { createFormTestSuite } from "../../../__test/helpers/createFormTestSuite.js";
import { ${componentName} } from "./${componentName}.js";

// Now you can use createFormTestSuite in your tests...
`;
}

/**
 * Pre-configured test suite factory for common form patterns
 */
export const commonFormTestSuites = {
    /**
     * Standard dialog form (most common pattern)
     */
    dialogForm: <TProps extends Record<string, unknown>>(
        component: ComponentType<TProps>,
        baseProps: Omit<TProps, "isOpen" | "onClose" | "onCompleted">,
    ) => createFormTestSuite({
        component,
        defaultProps: {
            ...baseProps,
            isOpen: true,
            onClose: vi.fn(),
            onCompleted: vi.fn(),
        } as TProps,
    }),

    /**
     * Comment form (with specific mock data)
     */
    commentForm: <TProps extends Record<string, unknown>>(
        component: ComponentType<TProps>,
        baseProps: Omit<TProps, "isOpen" | "onClose" | "onCompleted">,
    ) => createFormTestSuite({
        component,
        defaultProps: {
            ...baseProps,
            isOpen: true,
            onClose: vi.fn(),
            onCompleted: vi.fn(),
        } as TProps,
        lazyFetchData: {
            threads: [{
                comment: {
                    id: "test-comment-id",
                    you: { canUpdate: true, canDelete: true },
                },
            }],
        },
    }),

    /**
     * CRUD form with create/update modes
     */
    crudForm: <TProps extends Record<string, unknown>>(
        component: ComponentType<TProps>,
        baseProps: Omit<TProps, "isCreate" | "isOpen" | "onClose" | "onCompleted" | "onDeleted">,
    ) => ({
        create: createFormTestSuite({
            component,
            defaultProps: {
                ...baseProps,
                isCreate: true,
                isOpen: true,
                onClose: vi.fn(),
                onCompleted: vi.fn(),
                onDeleted: vi.fn(),
            } as TProps,
        }),
        update: createFormTestSuite({
            component,
            defaultProps: {
                ...baseProps,
                isCreate: false,
                isOpen: true,
                onClose: vi.fn(),
                onCompleted: vi.fn(),
                onDeleted: vi.fn(),
            } as TProps,
        }),
    }),
};

/**
 * Quick test helper that wraps common form test patterns
 * 
 * Usage:
 * ```typescript
 * const suite = quickFormTest(MyComponent, defaultProps);
 * 
 * describe("MyComponent", () => {
 *   beforeEach(suite.setup.beforeEach);
 *   afterEach(suite.setup.afterEach);
 *   
 *   it("renders", suite.tests.rendering);
 *   it("submits", () => suite.tests.submission({ name: "test" }));
 *   it("cancels", suite.tests.cancellation);
 * });
 * ```
 */
export function quickFormTest<TProps = Record<string, unknown>>(
    component: ComponentType<TProps>,
    defaultProps: TProps,
    mockSession?: Record<string, unknown>,
) {
    return createFormTestSuite({
        component,
        defaultProps,
        mockSession,
    });
}
