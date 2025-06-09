import { MockedProvider } from "@apollo/client/testing";
import { ApiKeyPermission } from "@vrooli/shared";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { BrowserRouter } from "react-router-dom";
import { SettingsApiView } from "./SettingsApiView";

// Mock the required hooks and services
jest.mock("../../hooks/useLazyFetch", () => ({
    useLazyFetch: () => [
        jest.fn(),
        {
            loading: false,
            error: null,
            data: null,
        },
    ],
}));

jest.mock("../../utils/session", () => ({
    getUser: () => ({
        id: "user-123",
        name: "Test User",
        email: "test@example.com",
    }),
}));

jest.mock("react-i18next", () => ({
    useTranslation: () => ({
        t: (key: string) => key,
        i18n: { language: "en" },
    }),
}));

// Mock GraphQL queries
const GET_API_KEYS_QUERY = `
    query GetApiKeys {
        apiKeys {
            edges {
                node {
                    id
                    name
                    permissions
                    creditsUsed
                    creditsLimitHard
                    creditsLimitSoft
                    stopAtLimit
                    createdAt
                    updatedAt
                }
            }
        }
    }
`;

const GET_EXTERNAL_API_KEYS_QUERY = `
    query GetExternalApiKeys {
        apiKeysExternal {
            edges {
                node {
                    id
                    name
                    service
                    createdAt
                    updatedAt
                }
            }
        }
    }
`;

const CREATE_API_KEY_MUTATION = `
    mutation CreateApiKey($input: ApiKeyCreateInput!) {
        createApiKey(input: $input) {
            id
            name
            key
            permissions
            creditsLimitHard
            creditsLimitSoft
            stopAtLimit
            createdAt
        }
    }
`;

const mockApiKeys = [
    {
        id: "1",
        name: "Development Key",
        permissions: [ApiKeyPermission.ReadPublic, ApiKeyPermission.ReadPrivate],
        creditsUsed: 150,
        creditsLimitHard: 1000,
        creditsLimitSoft: 800,
        stopAtLimit: true,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    },
    {
        id: "2",
        name: "Production Key",
        permissions: [ApiKeyPermission.ReadPublic],
        creditsUsed: 50,
        creditsLimitHard: 5000,
        creditsLimitSoft: null,
        stopAtLimit: false,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    },
];

const mockExternalApiKeys = [
    {
        id: "ext-1",
        name: "OpenAI Key",
        service: "openai",
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    },
    {
        id: "ext-2",
        name: "GitHub Token",
        service: "github",
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    },
];

const createMockProvider = (overrides: any = {}) => {
    const defaultMocks = [
        {
            request: {
                query: GET_API_KEYS_QUERY,
            },
            result: {
                data: {
                    apiKeys: {
                        edges: mockApiKeys.map(key => ({ node: key })),
                    },
                },
            },
        },
        {
            request: {
                query: GET_EXTERNAL_API_KEYS_QUERY,
            },
            result: {
                data: {
                    apiKeysExternal: {
                        edges: mockExternalApiKeys.map(key => ({ node: key })),
                    },
                },
            },
        },
        {
            request: {
                query: CREATE_API_KEY_MUTATION,
                variables: {
                    input: {
                        name: "Test Key",
                        permissions: [ApiKeyPermission.ReadPublic],
                        creditsLimitHard: 1000,
                        creditsLimitSoft: null,
                        stopAtLimit: true,
                    },
                },
            },
            result: {
                data: {
                    createApiKey: {
                        id: "new-key-id",
                        name: "Test Key",
                        key: "sk-test-1234567890abcdef", // Raw key for one-time display
                        permissions: [ApiKeyPermission.ReadPublic],
                        creditsLimitHard: 1000,
                        creditsLimitSoft: null,
                        stopAtLimit: true,
                        createdAt: new Date().toISOString(),
                    },
                },
            },
        },
    ];

    return [...defaultMocks, ...overrides];
};

const renderSettingsApiView = (mocks: any[] = []) => {
    return render(
        <BrowserRouter>
            <MockedProvider mocks={createMockProvider(mocks)} addTypename={false}>
                <SettingsApiView />
            </MockedProvider>
        </BrowserRouter>
    );
};

describe("SettingsApiView", () => {
    beforeEach(() => {
        jest.clearAllMocks();
    });

    describe("Initial Rendering", () => {
        it("renders the API key management interface", async () => {
            renderSettingsApiView();

            // Check for main sections
            expect(screen.getByText("ApiKeys")).toBeInTheDocument();
            expect(screen.getByText("ExternalAPIKeys")).toBeInTheDocument();

            // Wait for data to load
            await waitFor(() => {
                expect(screen.getByText("Development Key")).toBeInTheDocument();
                expect(screen.getByText("Production Key")).toBeInTheDocument();
            });
        });

        it("displays loading state initially", () => {
            renderSettingsApiView();
            
            // Should show loading indicators
            expect(screen.getByTestId("loading-api-keys") || screen.getByText("Loading...")).toBeInTheDocument();
        });

        it("shows empty state when no API keys exist", async () => {
            const emptyMocks = [
                {
                    request: { query: GET_API_KEYS_QUERY },
                    result: { data: { apiKeys: { edges: [] } } },
                },
                {
                    request: { query: GET_EXTERNAL_API_KEYS_QUERY },
                    result: { data: { apiKeysExternal: { edges: [] } } },
                },
            ];

            renderSettingsApiView(emptyMocks);

            await waitFor(() => {
                expect(screen.getByText("NoAPIKeysYet")).toBeInTheDocument();
            });
        });
    });

    describe("Internal API Key Management", () => {
        it("displays API key information correctly", async () => {
            renderSettingsApiView();

            await waitFor(() => {
                // Check key names
                expect(screen.getByText("Development Key")).toBeInTheDocument();
                expect(screen.getByText("Production Key")).toBeInTheDocument();

                // Check usage information
                expect(screen.getByText("150 / 1000")).toBeInTheDocument(); // credits used/limit
                expect(screen.getByText("50 / 5000")).toBeInTheDocument();
            });
        });

        it("opens create API key dialog when clicking create button", async () => {
            const user = userEvent.setup();
            renderSettingsApiView();

            await waitFor(() => {
                expect(screen.getByText("Development Key")).toBeInTheDocument();
            });

            const createButton = screen.getByRole("button", { name: /create.*key/i });
            await user.click(createButton);

            // Dialog should open
            expect(screen.getByRole("dialog")).toBeInTheDocument();
            expect(screen.getByText("CreateAPIKey")).toBeInTheDocument();
        });

        it("shows permission levels with security indicators", async () => {
            renderSettingsApiView();

            await waitFor(() => {
                // Should show permission descriptions
                expect(screen.getByText(/ReadPublic/)).toBeInTheDocument();
                expect(screen.getByText(/ReadPrivate/)).toBeInTheDocument();
            });
        });

        it("handles API key creation flow", async () => {
            const user = userEvent.setup();
            renderSettingsApiView();

            await waitFor(() => {
                expect(screen.getByText("Development Key")).toBeInTheDocument();
            });

            // Open create dialog
            const createButton = screen.getByRole("button", { name: /create.*key/i });
            await user.click(createButton);

            // Fill in form
            const nameInput = screen.getByLabelText(/name/i);
            await user.type(nameInput, "Test Key");

            // Select permissions
            const readPublicCheckbox = screen.getByRole("checkbox", { name: /ReadPublic/i });
            await user.click(readPublicCheckbox);

            // Set credit limit
            const creditLimitInput = screen.getByLabelText(/hard.*limit/i);
            await user.clear(creditLimitInput);
            await user.type(creditLimitInput, "1000");

            // Submit form
            const submitButton = screen.getByRole("button", { name: /create/i });
            await user.click(submitButton);

            // Should show one-time key display dialog
            await waitFor(() => {
                expect(screen.getByText("sk-test-1234567890abcdef")).toBeInTheDocument();
                expect(screen.getByText("CopyAPIKey")).toBeInTheDocument();
            });
        });

        it("validates required fields in create form", async () => {
            const user = userEvent.setup();
            renderSettingsApiView();

            await waitFor(() => {
                expect(screen.getByText("Development Key")).toBeInTheDocument();
            });

            // Open create dialog
            const createButton = screen.getByRole("button", { name: /create.*key/i });
            await user.click(createButton);

            // Try to submit without filling required fields
            const submitButton = screen.getByRole("button", { name: /create/i });
            await user.click(submitButton);

            // Should show validation errors
            await waitFor(() => {
                expect(screen.getByText(/required/i)).toBeInTheDocument();
            });
        });

        it("shows security level warnings for permissions", async () => {
            const user = userEvent.setup();
            renderSettingsApiView();

            await waitFor(() => {
                expect(screen.getByText("Development Key")).toBeInTheDocument();
            });

            // Open create dialog
            const createButton = screen.getByRole("button", { name: /create.*key/i });
            await user.click(createButton);

            // Select high-security permission
            const writeAuthCheckbox = screen.getByRole("checkbox", { name: /WriteAuth/i });
            await user.click(writeAuthCheckbox);

            // Should show security warning
            expect(screen.getByText(/High.*Security/i)).toBeInTheDocument();
        });
    });

    describe("External API Key Management", () => {
        it("displays external API keys correctly", async () => {
            renderSettingsApiView();

            await waitFor(() => {
                expect(screen.getByText("OpenAI Key")).toBeInTheDocument();
                expect(screen.getByText("GitHub Token")).toBeInTheDocument();
                expect(screen.getByText("openai")).toBeInTheDocument();
                expect(screen.getByText("github")).toBeInTheDocument();
            });
        });

        it("opens external key creation dialog", async () => {
            const user = userEvent.setup();
            renderSettingsApiView();

            await waitFor(() => {
                expect(screen.getByText("OpenAI Key")).toBeInTheDocument();
            });

            const addExternalButton = screen.getByRole("button", { name: /add.*external/i });
            await user.click(addExternalButton);

            // Dialog should open
            expect(screen.getByRole("dialog")).toBeInTheDocument();
            expect(screen.getByText("AddExternalAPIKey")).toBeInTheDocument();
        });

        it("handles external key creation with service selection", async () => {
            const user = userEvent.setup();
            renderSettingsApiView();

            await waitFor(() => {
                expect(screen.getByText("OpenAI Key")).toBeInTheDocument();
            });

            // Open external key dialog
            const addExternalButton = screen.getByRole("button", { name: /add.*external/i });
            await user.click(addExternalButton);

            // Fill in form
            const nameInput = screen.getByLabelText(/name/i);
            await user.type(nameInput, "My OpenAI Key");

            // Select service
            const serviceSelect = screen.getByLabelText(/service/i);
            await user.click(serviceSelect);
            const openaiOption = screen.getByRole("option", { name: /openai/i });
            await user.click(openaiOption);

            // Enter API key
            const keyInput = screen.getByLabelText(/api.*key/i);
            await user.type(keyInput, "sk-openai-test-key");

            // Submit
            const submitButton = screen.getByRole("button", { name: /add/i });
            await user.click(submitButton);

            // Should close dialog and refresh list
            await waitFor(() => {
                expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
            });
        });

        it("validates external key input", async () => {
            const user = userEvent.setup();
            renderSettingsApiView();

            await waitFor(() => {
                expect(screen.getByText("OpenAI Key")).toBeInTheDocument();
            });

            // Open external key dialog
            const addExternalButton = screen.getByRole("button", { name: /add.*external/i });
            await user.click(addExternalButton);

            // Try to submit without required fields
            const submitButton = screen.getByRole("button", { name: /add/i });
            await user.click(submitButton);

            // Should show validation errors
            await waitFor(() => {
                expect(screen.getByText(/required/i)).toBeInTheDocument();
            });
        });
    });

    describe("Credit Usage and Limits", () => {
        it("displays credit usage correctly", async () => {
            renderSettingsApiView();

            await waitFor(() => {
                // Should show usage bars/indicators
                expect(screen.getByText("150 / 1000")).toBeInTheDocument();
                expect(screen.getByText("50 / 5000")).toBeInTheDocument();
            });
        });

        it("shows warning when approaching limits", async () => {
            const highUsageMocks = [
                {
                    request: { query: GET_API_KEYS_QUERY },
                    result: {
                        data: {
                            apiKeys: {
                                edges: [
                                    {
                                        node: {
                                            ...mockApiKeys[0],
                                            creditsUsed: 950, // Very close to 1000 limit
                                        },
                                    },
                                ],
                            },
                        },
                    },
                },
            ];

            renderSettingsApiView(highUsageMocks);

            await waitFor(() => {
                expect(screen.getByText(/warning/i) || screen.getByText(/approaching/i)).toBeInTheDocument();
            });
        });

        it("shows error when limit exceeded", async () => {
            const exceededLimitMocks = [
                {
                    request: { query: GET_API_KEYS_QUERY },
                    result: {
                        data: {
                            apiKeys: {
                                edges: [
                                    {
                                        node: {
                                            ...mockApiKeys[0],
                                            creditsUsed: 1100, // Exceeds 1000 limit
                                        },
                                    },
                                ],
                            },
                        },
                    },
                },
            ];

            renderSettingsApiView(exceededLimitMocks);

            await waitFor(() => {
                expect(screen.getByText(/exceeded/i) || screen.getByText(/limit/i)).toBeInTheDocument();
            });
        });
    });

    describe("Key Actions", () => {
        it("allows editing API key settings", async () => {
            const user = userEvent.setup();
            renderSettingsApiView();

            await waitFor(() => {
                expect(screen.getByText("Development Key")).toBeInTheDocument();
            });

            // Click edit button for first key
            const editButtons = screen.getAllByRole("button", { name: /edit/i });
            await user.click(editButtons[0]);

            // Should open edit dialog
            expect(screen.getByRole("dialog")).toBeInTheDocument();
            expect(screen.getByText("EditAPIKey")).toBeInTheDocument();
        });

        it("allows revoking API keys", async () => {
            const user = userEvent.setup();
            renderSettingsApiView();

            await waitFor(() => {
                expect(screen.getByText("Development Key")).toBeInTheDocument();
            });

            // Click revoke button
            const revokeButtons = screen.getAllByRole("button", { name: /revoke/i });
            await user.click(revokeButtons[0]);

            // Should show confirmation dialog
            expect(screen.getByText(/confirm/i)).toBeInTheDocument();
            expect(screen.getByText(/revoke/i)).toBeInTheDocument();
        });

        it("allows deleting external API keys", async () => {
            const user = userEvent.setup();
            renderSettingsApiView();

            await waitFor(() => {
                expect(screen.getByText("OpenAI Key")).toBeInTheDocument();
            });

            // Click delete button for external key
            const deleteButtons = screen.getAllByRole("button", { name: /delete/i });
            await user.click(deleteButtons[0]);

            // Should show confirmation dialog
            expect(screen.getByText(/confirm/i)).toBeInTheDocument();
            expect(screen.getByText(/delete/i)).toBeInTheDocument();
        });
    });

    describe("Error Handling", () => {
        it("displays error message when API key creation fails", async () => {
            const errorMocks = [
                {
                    request: {
                        query: CREATE_API_KEY_MUTATION,
                        variables: expect.any(Object),
                    },
                    error: new Error("Failed to create API key"),
                },
            ];

            const user = userEvent.setup();
            renderSettingsApiView(errorMocks);

            await waitFor(() => {
                expect(screen.getByText("Development Key")).toBeInTheDocument();
            });

            // Try to create a key
            const createButton = screen.getByRole("button", { name: /create.*key/i });
            await user.click(createButton);

            // Fill and submit form
            const nameInput = screen.getByLabelText(/name/i);
            await user.type(nameInput, "Test Key");

            const submitButton = screen.getByRole("button", { name: /create/i });
            await user.click(submitButton);

            // Should show error message
            await waitFor(() => {
                expect(screen.getByText(/failed/i) || screen.getByText(/error/i)).toBeInTheDocument();
            });
        });

        it("handles network errors gracefully", async () => {
            const networkErrorMocks = [
                {
                    request: { query: GET_API_KEYS_QUERY },
                    error: new Error("Network error"),
                },
            ];

            renderSettingsApiView(networkErrorMocks);

            await waitFor(() => {
                expect(screen.getByText(/error/i) || screen.getByText(/failed/i)).toBeInTheDocument();
            });
        });
    });

    describe("Responsive Design", () => {
        it("adapts layout for mobile screens", () => {
            // Mock mobile viewport
            Object.defineProperty(window, "innerWidth", {
                writable: true,
                configurable: true,
                value: 375,
            });

            renderSettingsApiView();

            // Should use mobile-friendly layout
            // This would depend on the specific responsive implementation
        });
    });

    describe("Accessibility", () => {
        it("provides proper ARIA labels and roles", async () => {
            renderSettingsApiView();

            await waitFor(() => {
                expect(screen.getByText("Development Key")).toBeInTheDocument();
            });

            // Check for proper ARIA attributes
            const createButton = screen.getByRole("button", { name: /create.*key/i });
            expect(createButton).toHaveAttribute("aria-label");

            // Check for proper headings
            const headings = screen.getAllByRole("heading");
            expect(headings.length).toBeGreaterThan(0);
        });

        it("supports keyboard navigation", async () => {
            renderSettingsApiView();

            await waitFor(() => {
                expect(screen.getByText("Development Key")).toBeInTheDocument();
            });

            // Tab through focusable elements
            const focusableElements = screen.getAllByRole("button");
            expect(focusableElements.length).toBeGreaterThan(0);

            focusableElements.forEach(element => {
                expect(element).toHaveAttribute("tabIndex");
            });
        });
    });
});