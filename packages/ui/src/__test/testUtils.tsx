import React from "react";
import StyledEngineProvider from "@mui/material/StyledEngineProvider";
import { ThemeProvider } from "@mui/material/styles";
import { createTheme } from "@mui/material/styles";
import type { Theme } from "@mui/material";
import { render as rtlRender, type RenderOptions } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { BrowserRouter } from "react-router-dom";
import { SessionContext } from "../contexts/session.js";
import { DEFAULT_THEME, themes } from "../utils/display/theme.js";
import type { StartedPostgreSQLContainer, StartedGenericContainer } from "testcontainers";

function withFontSize(theme: Theme, fontSize: number): Theme {
    return createTheme({
        ...theme,
        typography: {
            fontSize,
        },
    });
}

function withIsLeftHanded(theme: Theme, isLeftHanded: boolean): Theme {
    return createTheme({
        ...theme,
        isLeftHanded,
    });
}

// Mock values or functions to simulate theme and session context
const defaultFontSize = 14;
const defaultIsLeftHanded = false;

// Create a default theme object based on your customizations
const defaultCustomTheme = withIsLeftHanded(withFontSize(themes[DEFAULT_THEME], defaultFontSize), defaultIsLeftHanded);

// Original render function for component tests
function render(ui, {
    theme = defaultCustomTheme,
    session = undefined, // Default session value
    ...renderOptions
} = {}) {
    function Wrapper({ children }) {
        return (
            <StyledEngineProvider injectFirst>
                <ThemeProvider theme={theme}>
                    <SessionContext.Provider value={session}>
                        {children}
                    </SessionContext.Provider>
                </ThemeProvider>
            </StyledEngineProvider>
        );
    }

    return rtlRender(ui, { wrapper: Wrapper, ...renderOptions });
}

// Enhanced render function for integration tests with all providers
interface CustomRenderOptions extends Omit<RenderOptions, 'wrapper'> {
    theme?: Theme;
    session?: any;
    initialEntries?: string[];
    withRouter?: boolean;
}

export const renderWithProviders = (
    ui: React.ReactElement,
    options: CustomRenderOptions = {}
) => {
    const { 
        theme = defaultCustomTheme, 
        session = undefined,
        initialEntries = ['/'],
        withRouter = true,
        ...renderOptions 
    } = options;
    
    const AllProviders = ({ children }: { children: React.ReactNode }) => {
        const content = (
            <StyledEngineProvider injectFirst>
                <ThemeProvider theme={theme}>
                    <SessionContext.Provider value={session}>
                        {children}
                    </SessionContext.Provider>
                </ThemeProvider>
            </StyledEngineProvider>
        );

        return withRouter ? (
            <BrowserRouter>{content}</BrowserRouter>
        ) : content;
    };

    const user = userEvent.setup();
    
    return {
        user,
        ...rtlRender(ui, { wrapper: AllProviders, ...renderOptions }),
    };
};

// Import integration utilities from setup
import { 
    getTestDatabaseClient, 
    createTestTransaction, 
    createTestAPIClient 
} from './setup.vitest.js';

// Testcontainers setup helper
export interface TestContainers {
    postgres: StartedPostgreSQLContainer;
    redis: StartedGenericContainer;
    cleanup: () => Promise<void>;
}

export const setupTestContainers = async (): Promise<TestContainers> => {
    // Retrieve containers from global setup
    const postgres = (global as any).__UI_TEST_POSTGRES_CONTAINER__;
    const redis = (global as any).__UI_TEST_REDIS_CONTAINER__;
    
    if (!postgres || !redis) {
        throw new Error('Test containers not available. Make sure global setup completed successfully.');
    }
    
    return {
        postgres,
        redis,
        cleanup: async () => {
            // Cleanup is handled by global teardown
            console.log('Test containers will be cleaned up by global teardown');
        },
    };
};

// Database testing utilities (only available in integration tests)
export const withTestTransaction = createTestTransaction;
export const getTestDatabase = getTestDatabaseClient;
export const createTestClient = createTestAPIClient;

// Form testing utilities
export const fillForm = async (
    user: ReturnType<typeof userEvent.setup>,
    formData: Record<string, any>,
    getByRole: any
) => {
    for (const [field, value] of Object.entries(formData)) {
        if (typeof value === 'string') {
            const input = getByRole('textbox', { name: new RegExp(field, 'i') });
            await user.clear(input);
            await user.type(input, value);
        } else if (typeof value === 'boolean') {
            const checkbox = getByRole('checkbox', { name: new RegExp(field, 'i') });
            if (value !== checkbox.checked) {
                await user.click(checkbox);
            }
        }
        // Add more field types as needed
    }
};

export const submitForm = async (
    user: ReturnType<typeof userEvent.setup>,
    getByRole: any,
    buttonText: string = 'submit'
) => {
    const submitButton = getByRole('button', { name: new RegExp(buttonText, 'i') });
    await user.click(submitButton);
};

// Wait utilities for async operations
export const waitForDatabaseRecord = async (
    table: string,
    condition: Record<string, any>,
    timeout: number = 5000
): Promise<any> => {
    if (!getTestDatabaseClient) {
        throw new Error('Database utilities not available. This function only works in integration tests.');
    }
    
    const prisma = getTestDatabaseClient();
    if (!prisma) {
        throw new Error('Database client not available');
    }
    
    const startTime = Date.now();
    
    while (Date.now() - startTime < timeout) {
        try {
            const record = await (prisma as any)[table].findFirst({
                where: condition,
            });
            
            if (record) {
                return record;
            }
        } catch (error) {
            // Continue polling
        }
        
        await new Promise(resolve => setTimeout(resolve, 100));
    }
    
    throw new Error(`Database record not found within ${timeout}ms: ${JSON.stringify(condition)}`);
};

// Cleanup utilities
export const cleanupTestData = async (tables: string[]) => {
    if (!getTestDatabaseClient) {
        return; // Not in integration test environment
    }
    
    const prisma = getTestDatabaseClient();
    if (!prisma) {
        return;
    }
    
    // Delete in reverse order to handle foreign key constraints
    for (const table of tables.reverse()) {
        try {
            await (prisma as any)[table].deleteMany({
                where: {
                    // Only delete test records (those with test- prefix in IDs)
                    id: {
                        startsWith: 'test-',
                    },
                },
            });
        } catch (error) {
            console.warn(`Failed to cleanup table ${table}:`, error);
        }
    }
};

// Integration test helpers
export const createIntegrationTestSuite = (suiteName: string) => {
    let containers: TestContainers;
    
    return {
        setupContainers: async () => {
            containers = await setupTestContainers();
            return containers;
        },
        
        withDatabase: withTestTransaction,
        
        createAPIClient: () => createTestClient?.(),
        
        cleanup: async () => {
            if (containers) {
                await containers.cleanup();
            }
        },
        
        // Helper for common test patterns
        testFormSubmission: async (
            component: React.ReactElement,
            formData: Record<string, any>,
            expectedDbRecord?: { table: string; condition: Record<string, any> }
        ) => {
            const { user, getByRole } = renderWithProviders(component);
            
            // Fill and submit form
            await fillForm(user, formData, getByRole);
            await submitForm(user, getByRole);
            
            // Verify database if expected record provided
            if (expectedDbRecord) {
                const { table, condition } = expectedDbRecord;
                const record = await waitForDatabaseRecord(table, condition);
                return record;
            }
        },
    };
};

// Re-export everything
export * from "@testing-library/react";
export { userEvent };
// Override the render method
export { render };
