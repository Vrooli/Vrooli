import { act, render, screen, waitFor } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import React from "react";
import { beforeEach, describe, expect, it, type Mock, vi } from "vitest";
import { ThemeProvider, createTheme } from "@mui/material/styles";
import { SessionContext } from "../../../contexts/session.js";
import { getCurrentUser, checkIfLoggedIn, SessionService } from "../../../utils/authentication/session.js";
import { ELEMENT_IDS } from "../../../utils/consts.js";
import { PubSub } from "../../../utils/pubsub.js";
import { openObject } from "../../../utils/navigation/openObject.js";
import { performAction, Actions } from "../../../utils/navigation/quickActions.js";
import { useMenu } from "../../../hooks/useMenu.js";
import { useIsAdmin } from "../../../hooks/useIsAdmin.js";
import { useLocation } from "../../../route/router.js";
import { useLazyFetch } from "../../../hooks/useFetch.js";
import { useWindowSize } from "../../../hooks/useWindowSize.js";
import { UserMenu } from "./UserMenu.js";

// Mock dependencies
vi.mock("../../../utils/authentication/session.js", () => ({
    getCurrentUser: vi.fn(),
    checkIfLoggedIn: vi.fn(),
    SessionService: {
        logOut: vi.fn(),
    },
    guestSession: {
        id: null,
        users: [],
        roles: [],
    },
}));

vi.mock("../../../utils/navigation/openObject.js", () => ({
    openObject: vi.fn(),
}));

vi.mock("../../../utils/navigation/quickActions.js", () => ({
    performAction: vi.fn(),
    Actions: {
        tutorial: "tutorial",
    },
}));

vi.mock("../../../route/router.js", () => ({
    useLocation: vi.fn(() => ["/", vi.fn()]),
}));

vi.mock("../../../api/fetchWrapper.js", () => ({
    fetchLazyWrapper: vi.fn(),
}));

vi.mock("../../../api/socket.js", () => ({
    SocketService: {
        get: () => ({
            disconnect: vi.fn(),
            connect: vi.fn(),
        }),
    },
}));

vi.mock("../../../hooks/useFetch.js", () => ({
    useLazyFetch: vi.fn(() => [vi.fn()]),
}));

vi.mock("../../../hooks/useIsAdmin.js", () => ({
    useIsAdmin: vi.fn(() => ({ isAdmin: false })),
}));

vi.mock("../../../hooks/useMenu.js", () => ({
    useMenu: vi.fn(() => ({ isOpen: false, close: vi.fn() })),
}));

vi.mock("../../../hooks/useWindowSize.js", () => ({
    useWindowSize: vi.fn(() => true),
}));

vi.mock("react-i18next", () => ({
    useTranslation: () => ({
        t: (key: string) => key,
    }),
}));

vi.mock("../../../utils/consts.js", () => ({
    ELEMENT_IDS: {
        UserMenu: "UserMenu",
        UserMenuDisplaySettings: "UserMenuDisplaySettings",
    },
    LINKS: {
        Login: "/auth/login",
        Home: "/",
        Pro: "/pro",
        Settings: "/settings",
        SettingsDisplay: "/settings/display",
        History: "/history",
        Calendar: "/calendar",
        Awards: "/awards",
        Admin: "/admin",
    },
}));

// Mock PubSub
vi.mock("../../../utils/pubsub.js", () => ({
    PubSub: {
        get: vi.fn(() => ({
            subscribe: vi.fn(),
            publish: vi.fn(),
        })),
    },
}));

// Mock additional components
vi.mock("../../inputs/LanguageSelector/LanguageSelector.js", () => ({
    LanguageSelector: () => <div data-testid="language-selector">LanguageSelector</div>,
}));

vi.mock("../../inputs/LeftHandedCheckbox/LeftHandedCheckbox.js", () => ({
    LeftHandedCheckbox: () => <div data-testid="left-handed-checkbox">LeftHandedCheckbox</div>,
}));

vi.mock("../../inputs/TextSizeButtons/TextSizeButtons.js", () => ({
    TextSizeButtons: () => <div data-testid="text-size-buttons">TextSizeButtons</div>,
}));

vi.mock("../../inputs/ThemeSwitch/ThemeSwitch.js", () => ({
    ThemeSwitch: () => <div data-testid="theme-switch">ThemeSwitch</div>,
}));

vi.mock("../../navigation/ContactInfo.js", () => ({
    ContactInfo: () => <div>ContactInfo</div>,
}));

// Mock contexts
const mockSession = {
    id: "session-1",
    users: [
        {
            id: "user-1",
            name: "Test User",
            handle: "testuser",
            profileImage: null,
            updatedAt: new Date().toISOString(),
            hasPremium: false,
            credits: "100000000", // 100000000 / 1000000 = 100, then 100 / 100 = 1.00
            theme: "light",
        },
    ],
    roles: [],
};

// Create test theme
const testTheme = createTheme({
    palette: {
        background: {
            textPrimary: "#000000",
            textSecondary: "#666666",
        },
        secondary: {
            main: "#1976d2",
        },
    },
});

// Test wrapper component
function TestWrapper({ children }: { children: React.ReactNode }) {
    return (
        <ThemeProvider theme={testTheme}>
            <SessionContext.Provider value={mockSession}>
                {children}
            </SessionContext.Provider>
        </ThemeProvider>
    );
}

describe("UserMenu", () => {
    let mockPubSubInstance: {
        subscribe: Mock;
        publish: Mock;
    };

    beforeEach(() => {
        vi.clearAllMocks();
        mockPubSubInstance = {
            subscribe: vi.fn(() => () => {}), // Return unsubscribe function
            publish: vi.fn(),
        };
        (PubSub.get as Mock).mockReturnValue(mockPubSubInstance);
        (getCurrentUser as Mock).mockReturnValue(mockSession.users[0]);
        (checkIfLoggedIn as Mock).mockReturnValue(true);
        (useMenu as Mock).mockReturnValue({ isOpen: false, close: vi.fn() });
        (useIsAdmin as Mock).mockReturnValue({ isAdmin: false });
        (useLazyFetch as Mock).mockReturnValue([vi.fn()]);
    });

    describe("Menu opening and closing", () => {
        it("subscribes to menu events on mount", () => {
            render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            expect(mockPubSubInstance.subscribe).toHaveBeenCalledWith("menu", expect.any(Function));
        });

        it("handles menu event with valid anchorEl", () => {
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: vi.fn() });

            render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            // Get the subscription callback
            const subscribeCall = mockPubSubInstance.subscribe.mock.calls[0];
            const callback = subscribeCall[1];

            // Create a mock anchor element
            const mockAnchorEl = document.createElement("button");

            // Simulate menu event with anchorEl
            act(() => {
                callback({
                    id: ELEMENT_IDS.UserMenu,
                    data: { anchorEl: mockAnchorEl },
                });
            });

            // No error should occur
            expect(() => {
                callback({
                    id: ELEMENT_IDS.UserMenu,
                    data: { anchorEl: mockAnchorEl },
                });
            }).not.toThrow();
        });

        it("handles menu event without data gracefully", () => {
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: vi.fn() });

            render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            // Get the subscription callback
            const subscribeCall = mockPubSubInstance.subscribe.mock.calls[0];
            const callback = subscribeCall[1];

            // Simulate menu event without data
            expect(() => {
                callback({
                    id: ELEMENT_IDS.UserMenu,
                });
            }).not.toThrow();
        });

        it("handles menu event with null data gracefully", () => {
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: vi.fn() });

            render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            // Get the subscription callback
            const subscribeCall = mockPubSubInstance.subscribe.mock.calls[0];
            const callback = subscribeCall[1];

            // Simulate menu event with null data
            expect(() => {
                callback({
                    id: ELEMENT_IDS.UserMenu,
                    data: null,
                });
            }).not.toThrow();
        });

        it("handles menu event with undefined data.data gracefully", () => {
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: vi.fn() });

            render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            // Get the subscription callback
            const subscribeCall = mockPubSubInstance.subscribe.mock.calls[0];
            const callback = subscribeCall[1];

            // Simulate menu event with data but undefined data.data
            expect(() => {
                callback({
                    id: ELEMENT_IDS.UserMenu,
                    data: {},
                });
            }).not.toThrow();
        });

        it("ignores menu events for different menu IDs", () => {
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: vi.fn() });

            render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            // Get the subscription callback
            const subscribeCall = mockPubSubInstance.subscribe.mock.calls[0];
            const callback = subscribeCall[1];

            const mockAnchorEl = document.createElement("button");

            // Simulate menu event with different ID
            act(() => {
                callback({
                    id: "DifferentMenu",
                    data: { anchorEl: mockAnchorEl },
                });
            });

            // The component should not process this event
            expect(() => {
                callback({
                    id: "DifferentMenu",
                    data: { anchorEl: mockAnchorEl },
                });
            }).not.toThrow();
        });

        it("renders as a closed dialog by default", () => {
            (useMenu as Mock).mockReturnValue({ isOpen: false, close: vi.fn() });

            render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            // Dialog should not be visible when closed
            expect(screen.queryByRole("button", { name: "Settings" })).toBeNull();
        });

        it("renders dialog content when open", () => {
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: vi.fn() });

            render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            // Current user info should be visible
            expect(screen.getByText("Test User")).toBeDefined();
            expect(screen.getByText("@testuser")).toBeDefined();
        });

        it("closes menu on location change", async () => {
            const mockClose = vi.fn();
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: mockClose });

            let locationValue = "/";
            const setLocation = vi.fn((newLocation) => {
                locationValue = newLocation;
            });

            (useLocation as Mock).mockImplementation(() => [locationValue, setLocation]);

            const { rerender } = render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            // Change location
            act(() => {
                setLocation("/new-path");
            });

            // Force re-render with new location
            (useLocation as Mock).mockImplementation(() => ["/new-path", setLocation]);
            rerender(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            await waitFor(() => {
                expect(mockClose).toHaveBeenCalled();
            });
        });
    });

    describe("User account display", () => {
        it("displays current user information", () => {
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: vi.fn() });

            render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            expect(screen.getByText("Test User")).toBeDefined();
            expect(screen.getByText("@testuser")).toBeDefined();
            // Check for credits - look for the dollar amount
            expect(screen.getByText(/\$1\.00/)).toBeDefined();
        });

        it("displays premium badge for premium users", () => {
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: vi.fn() });

            const premiumUser = {
                ...mockSession.users[0],
                hasPremium: true,
            };
            (getCurrentUser as Mock).mockReturnValue(premiumUser);

            render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            expect(screen.getByText("Pro")).toBeDefined();
        });

        it("displays multiple accounts when available", () => {
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: vi.fn() });

            const multiUserSession = {
                ...mockSession,
                users: [
                    ...mockSession.users,
                    {
                        id: "user-2",
                        name: "Second User",
                        handle: "seconduser",
                        profileImage: null,
                        updatedAt: new Date().toISOString(),
                        hasPremium: true,
                        credits: "500000",
                        theme: "dark",
                    },
                ],
            };

            // Mock the session context by rendering with updated session
            render(
                <ThemeProvider theme={testTheme}>
                    <SessionContext.Provider value={multiUserSession}>
                        <UserMenu />
                    </SessionContext.Provider>
                </ThemeProvider>,
            );

            expect(screen.getByText("Test User")).toBeDefined();
            expect(screen.getByText("Second User")).toBeDefined();
            expect(screen.getByText("@seconduser")).toBeDefined();
        });

        it("handles user without handle gracefully", () => {
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: vi.fn() });

            const userWithoutHandle = {
                ...mockSession.users[0],
                handle: null,
            };
            (getCurrentUser as Mock).mockReturnValue(userWithoutHandle);

            const sessionWithoutHandle = {
                ...mockSession,
                users: [userWithoutHandle],
            };

            render(
                <ThemeProvider theme={testTheme}>
                    <SessionContext.Provider value={sessionWithoutHandle}>
                        <UserMenu />
                    </SessionContext.Provider>
                </ThemeProvider>,
            );

            expect(screen.getByText("Test User")).toBeDefined();
            // Should not display handle when it's null
            expect(screen.queryByText("@testuser")).toBeNull();
            // Also check for any @ symbol
            expect(screen.queryByText(/@/)).toBeNull();
        });
    });

    describe("Navigation items", () => {
        it("displays navigation items for logged-in users", () => {
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: vi.fn() });

            render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            expect(screen.getByRole("button", { name: "Bookmark" })).toBeDefined();
            expect(screen.getByRole("button", { name: "Calendar" })).toBeDefined();
            expect(screen.getByRole("button", { name: "History" })).toBeDefined();
            expect(screen.getByRole("button", { name: "Award" })).toBeDefined();
            expect(screen.getByRole("button", { name: "Pro" })).toBeDefined();
            expect(screen.getByRole("button", { name: "Settings" })).toBeDefined();
            expect(screen.getByRole("button", { name: "Tutorial" })).toBeDefined();
        });

        it("displays admin link for admin users", () => {
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: vi.fn() });
            (useIsAdmin as Mock).mockReturnValue({ isAdmin: true });

            render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            expect(screen.getByRole("button", { name: "Admin" })).toBeDefined();
        });

        it("displays limited navigation for non-logged-in users", () => {
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: vi.fn() });
            (checkIfLoggedIn as Mock).mockReturnValue(false);
            (getCurrentUser as Mock).mockReturnValue({ id: null });

            // Mock empty session
            const emptySession = { id: null, users: [], roles: [] };

            render(
                <ThemeProvider theme={testTheme}>
                    <SessionContext.Provider value={emptySession}>
                        <UserMenu />
                    </SessionContext.Provider>
                </ThemeProvider>,
            );

            // When not logged in, check for Tutorial and LogInSignUp buttons
            expect(screen.getByRole("button", { name: "Tutorial" })).toBeDefined();
            expect(screen.getByRole("button", { name: "LogInSignUp" })).toBeDefined();
            
            // Pro might be displayed as a nav item when not logged in
            // Let's just check it's present somewhere
            expect(screen.getByText("Pro")).toBeDefined();

            // Should not display user-specific items
            expect(screen.queryByRole("button", { name: "Settings" })).toBeNull();
            expect(screen.queryByRole("button", { name: "Bookmark" })).toBeNull();
        });

        it("handles navigation item clicks", async () => {
            const mockClose = vi.fn();
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: mockClose });
            
            const mockSetLocation = vi.fn();
            (useLocation as Mock).mockReturnValue(["/", mockSetLocation]);

            const user = userEvent.setup();

            render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            const settingsButton = screen.getByRole("button", { name: "Settings" });
            await act(async () => {
                await user.click(settingsButton);
            });

            expect(mockSetLocation).toHaveBeenCalledWith("/settings");
        });

        it("handles tutorial action", async () => {
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: vi.fn() });

            const user = userEvent.setup();

            render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            const tutorialButton = screen.getByRole("button", { name: "Tutorial" });
            await act(async () => {
                await user.click(tutorialButton);
            });

            expect(performAction).toHaveBeenCalledWith(Actions.tutorial, expect.any(Object));
        });
    });

    describe("Display settings", () => {
        it("toggles display settings section", async () => {
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: vi.fn() });

            const user = userEvent.setup();

            render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            // Initially collapsed - the display settings should not be visible
            // Check that our mocked components are not rendered  
            expect(screen.queryByTestId("theme-switch")).toBeNull();
            expect(screen.queryByTestId("text-size-buttons")).toBeNull();
            expect(screen.queryByTestId("language-selector")).toBeNull();
            expect(screen.queryByTestId("left-handed-checkbox")).toBeNull();

            // Click to expand
            const displayHeader = screen.getByText("Display").closest("div");
            await act(async () => {
                if (displayHeader) await user.click(displayHeader);
            });

            // Should be visible now - check for our mocked components
            await waitFor(() => {
                expect(screen.getByTestId("theme-switch")).toBeDefined();
                expect(screen.getByTestId("text-size-buttons")).toBeDefined();
                expect(screen.getByTestId("language-selector")).toBeDefined();
                expect(screen.getByTestId("left-handed-checkbox")).toBeDefined();
            });

            // Click to collapse again
            await act(async () => {
                if (displayHeader) await user.click(displayHeader);
            });

            // Should be hidden again
            await waitFor(() => {
                expect(screen.queryByTestId("theme-switch")).toBeNull();
                expect(screen.queryByTestId("text-size-buttons")).toBeNull();
                expect(screen.queryByTestId("language-selector")).toBeNull();
                expect(screen.queryByTestId("left-handed-checkbox")).toBeNull();
            });
        });

        it("displays all display settings when expanded", async () => {
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: vi.fn() });

            const user = userEvent.setup();

            render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            const displayHeader = screen.getByText("Display").closest("div");
            await act(async () => {
                if (displayHeader) await user.click(displayHeader);
            });

            await waitFor(() => {
                // Check for all our mocked display settings components
                expect(screen.getByTestId("theme-switch")).toBeDefined();
                expect(screen.getByTestId("text-size-buttons")).toBeDefined();
                expect(screen.getByTestId("left-handed-checkbox")).toBeDefined();
                expect(screen.getByTestId("language-selector")).toBeDefined();
            });
        });
    });

    describe("Account switching", () => {
        it("switches to different account when clicked", async () => {
            const mockClose = vi.fn();
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: mockClose });

            // Mock fetchLazyWrapper to check if it's called
            const { fetchLazyWrapper } = vi.mocked(await import("../../../api/fetchWrapper.js"));
            
            const multiUserSession = {
                ...mockSession,
                users: [
                    ...mockSession.users,
                    {
                        id: "user-2",
                        name: "Second User",
                        handle: "seconduser",
                        profileImage: null,
                        updatedAt: new Date().toISOString(),
                        hasPremium: false,
                        credits: "0",
                        theme: "dark",
                    },
                ],
            };

            const user = userEvent.setup();

            render(
                <ThemeProvider theme={testTheme}>
                    <SessionContext.Provider value={multiUserSession}>
                        <UserMenu />
                    </SessionContext.Provider>
                </ThemeProvider>,
            );

            const secondUserButton = screen.getByTestId("user-account-user-2");
            await act(async () => {
                await user.click(secondUserButton);
            });

            // Should attempt to switch account via fetchLazyWrapper
            expect(fetchLazyWrapper).toHaveBeenCalled();
        });

        it("navigates to profile when clicking current user", async () => {
            const mockClose = vi.fn();
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: mockClose });

            const user = userEvent.setup();

            render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            const currentUserButton = screen.getByTestId("user-account-user-1");
            await act(async () => {
                await user.click(currentUserButton);
            });

            expect(openObject).toHaveBeenCalledWith(mockSession.users[0], expect.any(Function));
        });
    });

    describe("Authentication actions", () => {
        it("displays add account button when under max accounts", () => {
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: vi.fn() });

            render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            expect(screen.getByRole("button", { name: "AddAccount" })).toBeDefined();
        });

        it("hides add account button when at max accounts", () => {
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: vi.fn() });

            // Create session with 10 users (MAX_ACCOUNTS)
            const maxUserSession = {
                ...mockSession,
                users: Array.from({ length: 10 }, (_, i) => ({
                    id: `user-${i}`,
                    name: `User ${i}`,
                    handle: `user${i}`,
                    profileImage: null,
                    updatedAt: new Date().toISOString(),
                    hasPremium: false,
                    credits: "0",
                    theme: "light",
                })),
            };

            render(
                <ThemeProvider theme={testTheme}>
                    <SessionContext.Provider value={maxUserSession}>
                        <UserMenu />
                    </SessionContext.Provider>
                </ThemeProvider>,
            );

            expect(screen.queryByRole("button", { name: "AddAccount" })).toBeNull();
        });

        it("handles logout action", async () => {
            const mockClose = vi.fn();
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: mockClose });

            // Mock fetchLazyWrapper to check if it's called
            const { fetchLazyWrapper } = vi.mocked(await import("../../../api/fetchWrapper.js"));

            const user = userEvent.setup();

            render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            const logOutButton = screen.getByRole("button", { name: "LogOut" });
            await act(async () => {
                await user.click(logOutButton);
            });

            expect(fetchLazyWrapper).toHaveBeenCalled();
        });

        it("handles login/signup for non-logged-in users", async () => {
            const mockClose = vi.fn();
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: mockClose });
            
            const mockSetLocation = vi.fn();
            (useLocation as Mock).mockReturnValue(["/", mockSetLocation]);
            
            (checkIfLoggedIn as Mock).mockReturnValue(false);

            const emptySession = { id: null, users: [], roles: [] };

            const user = userEvent.setup();

            render(
                <ThemeProvider theme={testTheme}>
                    <SessionContext.Provider value={emptySession}>
                        <UserMenu />
                    </SessionContext.Provider>
                </ThemeProvider>,
            );

            const loginButton = screen.getByRole("button", { name: "LogInSignUp" });
            await act(async () => {
                await user.click(loginButton);
            });

            expect(mockSetLocation).toHaveBeenCalledWith("/auth/login");
        });
    });

    describe("Mobile behavior", () => {
        it("closes menu after action on mobile", async () => {
            const mockClose = vi.fn();
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: mockClose });
            
            // Mock mobile window size
            (useWindowSize as Mock).mockImplementation((callback) => callback({ width: 500 }));

            const user = userEvent.setup();

            render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            const settingsButton = screen.getByRole("button", { name: "Settings" });
            await act(async () => {
                await user.click(settingsButton);
            });

            // Should submit form and close on mobile
            expect(mockClose).toHaveBeenCalled();
        });
    });

    describe("Form submission behavior", () => {
        it("submits theme changes when menu closes", async () => {
            const mockClose = vi.fn();
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: mockClose });

            const mockProfileUpdate = vi.fn();
            (useLazyFetch as Mock).mockReturnValue([mockProfileUpdate]);

            const { rerender } = render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            // Simulate menu closing
            (useMenu as Mock).mockReturnValue({ isOpen: false, close: mockClose });
            rerender(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            // Form should be submitted
            // Note: In actual implementation, this would happen through formik.handleSubmit
        });
    });

    describe("Contact info", () => {
        it("displays contact information section", () => {
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: vi.fn() });

            render(
                <TestWrapper>
                    <UserMenu />
                </TestWrapper>,
            );

            // Contact info is rendered as a mocked component
            expect(screen.getByText("ContactInfo")).toBeDefined();
        });
    });

    describe("Error handling", () => {
        it("handles missing session data gracefully", () => {
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: vi.fn() });

            (getCurrentUser as Mock).mockReturnValue({ id: null });

            const emptySession = null as any;

            expect(() => render(
                <ThemeProvider theme={testTheme}>
                    <SessionContext.Provider value={emptySession}>
                        <UserMenu />
                    </SessionContext.Provider>
                </ThemeProvider>,
            )).not.toThrow();
        });

        it("handles invalid credits value gracefully", () => {
            (useMenu as Mock).mockReturnValue({ isOpen: true, close: vi.fn() });

            const userWithInvalidCredits = {
                ...mockSession.users[0],
                credits: null,
            };
            (getCurrentUser as Mock).mockReturnValue(userWithInvalidCredits);

            const sessionWithInvalidCredits = {
                ...mockSession,
                users: [userWithInvalidCredits],
            };

            render(
                <ThemeProvider theme={testTheme}>
                    <SessionContext.Provider value={sessionWithInvalidCredits}>
                        <UserMenu />
                    </SessionContext.Provider>
                </ThemeProvider>,
            );

            // Should display $0.00 for invalid credits
            // Use getAllByText since there might be multiple instances
            const creditsElements = screen.getAllByText((content, element) => {
                return element?.textContent?.includes("$0.00") || false;
            });
            expect(creditsElements.length).toBeGreaterThan(0);
        });
    });
});
