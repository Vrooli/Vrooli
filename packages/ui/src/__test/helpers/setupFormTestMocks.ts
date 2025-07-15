/**
 * @deprecated This approach has been superseded by createFormTestSuite.ts
 * 
 * Please use the new mock consolidation system instead:
 * 
 * import { createFormTestSuite } from "./createFormTestSuite.js";
 * 
 * const suite = createFormTestSuite({
 *     component: MyComponent,
 *     defaultProps: { ... },
 * });
 * 
 * // Then use suite.setup.generateMockCode() to get mock setup
 * 
 * This file provides a template for setting up mocks in form tests.
 * Since vi.mock() must be called at the top level, we provide code snippets
 * that can be easily copied into test files.
 */

/**
 * Generate the mock setup code for a form test.
 * Copy the returned string into your test file at the top level.
 * 
 * @example
 * // In your test file:
 * const mockSetup = generateFormMockSetup({ 
 *   testConfig: 'commentFormTestConfig',
 *   lazyFetchData: { threads: [...] }
 * });
 * // Then manually copy the generated code to the top of your test file
 */
export function generateFormMockSetup(options: {
    testConfig: string;
    lazyFetchData?: any;
    additionalMocks?: Record<string, string>;
}) {
    const { testConfig, lazyFetchData, additionalMocks = {} } = options;
    
    return `
// Import mock utilities
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

// Mock hooks using the central registry
vi.mock("../../../hooks/useLazyFetch.js", () => ({
    useLazyFetch: createUseLazyFetchMock(${JSON.stringify(lazyFetchData, null, 2)})
}));

vi.mock("../../../hooks/useStandardUpsertForm.ts", () => ({
    useStandardUpsertForm: createUseStandardUpsertFormMock()
}));

vi.mock("../../../hooks/useManagedObject.js", () => ({
    useManagedObject: createUseManagedObjectMock(${testConfig})
}));

// Mock UI components
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

${Object.entries(additionalMocks).map(([path, mockName]) => 
    `vi.mock("${path}", () => ({ ${mockName} }));`,
).join("\n")}
`;
}

/**
 * Get the standard test boilerplate for form tests
 */
export function getFormTestBoilerplate(componentName: string, defaultPropsCode: string) {
    return `
describe("${componentName}", () => {
    const mockSession = createMockSession();

    const defaultProps = ${defaultPropsCode};

    // Create the simple form tester once for all tests
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

            // TODO: Fill in form fields
            // await user.type(screen.getByTestId("input-field"), "value");
            
            await user.click(screen.getByTestId("submit-button"));
            expect(mockFunctions.onSubmit).toHaveBeenCalled();
        });
    });
});
`;
}
