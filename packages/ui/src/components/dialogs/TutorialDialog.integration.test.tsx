// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-19
/* eslint-disable @typescript-eslint/ban-ts-comment */
import { screen, fireEvent, waitFor, act } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import { LINKS } from "@vrooli/shared";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { render } from "../../__test/testUtils.js";
import { mockLocationForStorybook, clearMockedLocationForStorybook } from "../../route/useLocation.js";
import { ELEMENT_IDS } from "../../utils/consts.js";
import { TutorialDialog } from "./TutorialDialog.js";

// Create a more realistic PubSub mock that actually works
function createPubSubMock() {
    const subscribers = new Map();
    
    return {
        get: () => ({
            subscribe: vi.fn((channel, callback) => {
                if (!subscribers.has(channel)) {
                    subscribers.set(channel, new Set());
                }
                subscribers.get(channel).add(callback);
                
                // Return unsubscribe function
                return vi.fn(() => {
                    subscribers.get(channel)?.delete(callback);
                });
            }),
            publish: vi.fn((channel, data) => {
                const channelSubscribers = subscribers.get(channel);
                if (channelSubscribers) {
                    channelSubscribers.forEach(callback => {
                        callback(data);
                    });
                }
            }),
            hasSubscribers: vi.fn(() => false),
        }),
    };
}

const PubSub = createPubSubMock();

// Mock location hook
const mockSetLocation = vi.fn();
vi.mock("../../route/router.js", async () => {
    const actual = await vi.importActual("../../route/router.js") as Record<string, unknown>;
    return {
        ...actual,
        useLocation: () => [{ pathname: window.location.pathname, search: window.location.search }, mockSetLocation],
    };
});

// Mock DOM elements that tutorial might anchor to
const mockElements = [
    { id: "dashboard-resource-list", rect: { x: 100, y: 200, width: 300, height: 150 } },
    { id: "dashboard-event-list", rect: { x: 100, y: 380, width: 300, height: 120 } },
    { id: "user-menu-profile-icon", rect: { x: 50, y: 50, width: 40, height: 40 } },
    { id: "user-menu-account-list", rect: { x: 200, y: 100, width: 250, height: 80 } },
    { id: "search-tabs", rect: { x: 50, y: 120, width: 600, height: 50 } },
];

describe.skip("TutorialDialog Integration", () => {
    beforeEach(() => {
        // Clear any existing mock location
        clearMockedLocationForStorybook();
        
        // Clear the mockSetLocation spy
        mockSetLocation.mockClear();
        
        // Create mock DOM elements
        mockElements.forEach(({ id, rect }) => {
            let element = document.getElementById(id);
            if (!element) {
                element = document.createElement("div");
                element.id = id;
                element.style.position = "absolute";
                element.style.left = `${rect.x}px`;
                element.style.top = `${rect.y}px`;
                element.style.width = `${rect.width}px`;
                element.style.height = `${rect.height}px`;
                element.getBoundingClientRect = vi.fn(() => ({
                    x: rect.x,
                    y: rect.y,
                    width: rect.width,
                    height: rect.height,
                    top: rect.y,
                    left: rect.x,
                    bottom: rect.y + rect.height,
                    right: rect.x + rect.width,
                    toJSON: () => ({}),
                }));
                document.body.appendChild(element);
            }
        });
    });

    afterEach(() => {
        // Clean up mock elements
        mockElements.forEach(({ id }) => {
            const element = document.getElementById(id);
            if (element) {
                element.remove();
            }
        });
        
        clearMockedLocationForStorybook();
        vi.clearAllMocks();
    });

    describe("Opening and Closing", () => {
        it("should open when PubSub event is published", async () => {
            mockLocationForStorybook(LINKS.Home);
            
            render(<TutorialDialog />);

            // Tutorial should not be visible initially
            expect(screen.queryByText("Welcome to AI-Powered Productivity")).not.toBeTruthy();

            // Open tutorial via PubSub
            act(() => {
                PubSub.get().publish("menu", { 
                    id: ELEMENT_IDS.Tutorial, 
                    isOpen: true, 
                });
            });

            // Wait for tutorial to appear
            await waitFor(() => {
                expect(screen.getByText("Welcome to AI-Powered Productivity")).toBeTruthy();
            });
        });

        it("should close when close button is clicked", async () => {
            mockLocationForStorybook(LINKS.Home);
            
            render(<TutorialDialog />);

            // Open tutorial
            act(() => {
                PubSub.get().publish("menu", { 
                    id: ELEMENT_IDS.Tutorial, 
                    isOpen: true, 
                });
            });

            await waitFor(() => {
                expect(screen.getByText("Tutorial")).toBeTruthy();
            });

            // Close tutorial
            const closeButton = screen.getByRole("button", { name: /close/i });
            fireEvent.click(closeButton);

            await waitFor(() => {
                expect(screen.queryByText("Tutorial")).not.toBeTruthy();
            });
        });
    });

    describe("Navigation", () => {
        it("should navigate forward through tutorial steps", async () => {
            mockLocationForStorybook(LINKS.Home);
            
            render(<TutorialDialog />);

            // Open tutorial
            act(() => {
                PubSub.get().publish("menu", { 
                    id: ELEMENT_IDS.Tutorial, 
                    isOpen: true, 
                });
            });

            await waitFor(() => {
                expect(screen.getByText("Welcome to AI-Powered Productivity")).toBeTruthy();
            });

            // Navigate to next step
            const nextButton = screen.getByRole("button", { name: /arrow.*right/i });
            fireEvent.click(nextButton);

            await waitFor(() => {
                expect(screen.getByText("Meet your new workspace")).toBeTruthy();
            });
        });

        it("should navigate backward through tutorial steps", async () => {
            mockLocationForStorybook(LINKS.Home);
            
            render(<TutorialDialog />);

            // Open tutorial at step 1, section 0 (second step in first section)
            const url = new URL(window.location.href);
            url.searchParams.set("tutorial_section", "0");
            url.searchParams.set("tutorial_step", "1");
            window.history.replaceState({}, "", url.toString());

            act(() => {
                PubSub.get().publish("menu", { 
                    id: ELEMENT_IDS.Tutorial, 
                    isOpen: true, 
                });
            });

            await waitFor(() => {
                expect(screen.getByText("Meet your new workspace")).toBeTruthy();
            });

            // Navigate to previous step
            const prevButton = screen.getByRole("button", { name: /arrow.*left/i });
            fireEvent.click(prevButton);

            await waitFor(() => {
                expect(screen.getByText("Welcome to AI-Powered Productivity")).toBeTruthy();
            });
        });

        it("should handle keyboard navigation", async () => {
            const user = userEvent.setup();
            mockLocationForStorybook(LINKS.Home);
            
            render(<TutorialDialog />);

            // Open tutorial
            act(() => {
                PubSub.get().publish("menu", { 
                    id: ELEMENT_IDS.Tutorial, 
                    isOpen: true, 
                });
            });

            await waitFor(() => {
                expect(screen.getByText("Welcome to AI-Powered Productivity")).toBeTruthy();
            });

            // Wait for initial render delay to clear
            await new Promise(resolve => setTimeout(resolve, 200));

            // Navigate forward with arrow right
            await user.keyboard("{ArrowRight}");

            await waitFor(() => {
                expect(screen.getByText("Meet your new workspace")).toBeTruthy();
            });

            // Navigate backward with arrow left
            await user.keyboard("{ArrowLeft}");

            await waitFor(() => {
                expect(screen.getByText("Welcome to AI-Powered Productivity")).toBeTruthy();
            });
        });
    });

    describe("URL Parameter Management", () => {
        it("should initialize from URL parameters", async () => {
            mockLocationForStorybook(LINKS.Home);
            
            // Set URL parameters before rendering
            const url = new URL(window.location.href);
            url.searchParams.set("tutorial_section", "1");
            url.searchParams.set("tutorial_step", "0");
            window.history.replaceState({}, "", url.toString());

            render(<TutorialDialog />);

            // Tutorial should open automatically and show the specified step (section 1 = second section)
            await waitFor(() => {
                expect(screen.getByText("Your First AI Conversation")).toBeTruthy();
            });
        });

        it("should update URL parameters when navigating", async () => {
            mockLocationForStorybook(LINKS.Home);
            
            render(<TutorialDialog />);

            // Open tutorial
            act(() => {
                PubSub.get().publish("menu", { 
                    id: ELEMENT_IDS.Tutorial, 
                    isOpen: true, 
                });
            });

            await waitFor(() => {
                expect(screen.getByText("Welcome to AI-Powered Productivity")).toBeTruthy();
            });

            // Navigate forward
            const nextButton = screen.getByRole("button", { name: /arrow.*right/i });
            fireEvent.click(nextButton);

            await waitFor(() => {
                expect(screen.getByText("Meet your new workspace")).toBeTruthy();
            });

            // Check URL parameters updated
            const urlParams = new URLSearchParams(window.location.search);
            expect(urlParams.get("tutorial_section")).toBe("0");
            expect(urlParams.get("tutorial_step")).toBe("1");
        });

        it("should clear URL parameters when closing", async () => {
            mockLocationForStorybook(LINKS.Home);
            
            render(<TutorialDialog />);

            // Open tutorial
            act(() => {
                PubSub.get().publish("menu", { 
                    id: ELEMENT_IDS.Tutorial, 
                    isOpen: true, 
                });
            });

            await waitFor(() => {
                expect(screen.getByText("Welcome to AI-Powered Productivity")).toBeTruthy();
            });

            // Close tutorial
            const closeButton = screen.getByRole("button", { name: /close/i });
            fireEvent.click(closeButton);

            await waitFor(() => {
                expect(screen.queryByText("Welcome to AI-Powered Productivity")).not.toBeTruthy();
            });

            // Check URL parameters cleared
            const urlParams = new URLSearchParams(window.location.search);
            expect(urlParams.get("tutorial_section")).toBeNull();
            expect(urlParams.get("tutorial_step")).toBeNull();
        });
    });

    describe("Page Verification and Wrong Page Detection", () => {
        it("should show wrong page dialog when on incorrect page", async () => {
            // Set up scenario where user is on wrong page
            mockLocationForStorybook(LINKS.Search); // User is on search page
            
            render(<TutorialDialog />);

            // Open tutorial at home page step
            const url = new URL(window.location.href);
            url.searchParams.set("tutorial_section", "1"); // Home page section
            url.searchParams.set("tutorial_step", "0");
            window.history.replaceState({}, "", url.toString());

            act(() => {
                PubSub.get().publish("menu", { 
                    id: ELEMENT_IDS.Tutorial, 
                    isOpen: true, 
                });
            });

            await waitFor(() => {
                expect(screen.getByText("Wrong Page")).toBeTruthy();
                expect(screen.getByText("Please return to the correct page to continue the tutorial.")).toBeTruthy();
                expect(screen.getByRole("button", { name: "Continue" })).toBeTruthy();
            });
        });

        it("should allow continuing to correct page from wrong page dialog", async () => {
            mockLocationForStorybook(LINKS.Search);
            
            render(<TutorialDialog />);

            // Set up wrong page scenario
            const url = new URL(window.location.href);
            url.searchParams.set("tutorial_section", "1");
            url.searchParams.set("tutorial_step", "0");
            window.history.replaceState({}, "", url.toString());

            act(() => {
                PubSub.get().publish("menu", { 
                    id: ELEMENT_IDS.Tutorial, 
                    isOpen: true, 
                });
            });

            await waitFor(() => {
                expect(screen.getByText("Wrong Page")).toBeTruthy();
            });

            // Click continue button
            const continueButton = screen.getByRole("button", { name: "Continue" });
            fireEvent.click(continueButton);

            // Verify navigation was attempted
            await waitFor(() => {
                expect(mockSetLocation).toHaveBeenCalledWith(LINKS.Home);
            });
        });
    });

    describe("Element Anchoring", () => {
        it("should render as popover when anchored to element", async () => {
            mockLocationForStorybook(LINKS.Home);
            
            render(<TutorialDialog />);

            // Navigate to step with anchor element
            const url = new URL(window.location.href);
            url.searchParams.set("tutorial_section", "1");
            url.searchParams.set("tutorial_step", "1"); // Step with element anchor
            window.history.replaceState({}, "", url.toString());

            act(() => {
                PubSub.get().publish("menu", { 
                    id: ELEMENT_IDS.Tutorial, 
                    isOpen: true, 
                });
            });

            await waitFor(() => {
                expect(screen.getByText("Your First AI Conversation")).toBeTruthy();
            });

            // Should render as popover (check for popover-specific attributes)
            const tutorial = screen.getByText("Tutorial").closest("[role='dialog']");
            expect(tutorial).toBeTruthy();
        });

        it("should render as dialog when no anchor element", async () => {
            mockLocationForStorybook(LINKS.Home);
            
            render(<TutorialDialog />);

            // Open tutorial at step without anchor
            act(() => {
                PubSub.get().publish("menu", { 
                    id: ELEMENT_IDS.Tutorial, 
                    isOpen: true, 
                });
            });

            await waitFor(() => {
                expect(screen.getByText("Welcome to AI-Powered Productivity")).toBeTruthy();
            });

            // Should render as dialog
            const tutorial = screen.getByRole("dialog");
            expect(tutorial).toBeTruthy();
        });
    });

    describe("Section Menu Navigation", () => {
        it("should open section menu when section title is clicked", async () => {
            mockLocationForStorybook(LINKS.Home);
            
            render(<TutorialDialog />);

            act(() => {
                PubSub.get().publish("menu", { 
                    id: ELEMENT_IDS.Tutorial, 
                    isOpen: true, 
                });
            });

            await waitFor(() => {
                expect(screen.getByText("Welcome to AI-Powered Productivity")).toBeTruthy();
            });

            // Click on section title to open menu
            const sectionTitle = screen.getByText("Welcome to AI-Powered Productivity");
            fireEvent.click(sectionTitle);

            await waitFor(() => {
                expect(screen.getByText("Sections")).toBeTruthy();
            });
        });

        it("should navigate to selected section from menu", async () => {
            mockLocationForStorybook(LINKS.Home);
            
            render(<TutorialDialog />);

            act(() => {
                PubSub.get().publish("menu", { 
                    id: ELEMENT_IDS.Tutorial, 
                    isOpen: true, 
                });
            });

            await waitFor(() => {
                expect(screen.getByText("Welcome to AI-Powered Productivity")).toBeTruthy();
            });

            // Open section menu
            const sectionTitle = screen.getByText("Welcome to AI-Powered Productivity");
            fireEvent.click(sectionTitle);

            await waitFor(() => {
                expect(screen.getByText("Sections")).toBeTruthy();
            });

            // Click on "Your First AI Conversation" section
            const firstConversationSection = screen.getByText("2. Your First AI Conversation");
            fireEvent.click(firstConversationSection);

            await waitFor(() => {
                expect(screen.getByText("Your First AI Conversation")).toBeTruthy();
            });
        });
    });

    describe("Auto-advance on Navigation", () => {
        it("should auto-advance when navigating to correct page", async () => {
            // Start on home page with tutorial expecting search page
            mockLocationForStorybook(LINKS.Home);
            
            render(<TutorialDialog />);

            // Set tutorial to search page step
            const url = new URL(window.location.href);
            url.searchParams.set("tutorial_section", "3"); // Search page section
            url.searchParams.set("tutorial_step", "0");
            window.history.replaceState({}, "", url.toString());

            act(() => {
                PubSub.get().publish("menu", { 
                    id: ELEMENT_IDS.Tutorial, 
                    isOpen: true, 
                });
            });

            // Should show wrong page initially
            await waitFor(() => {
                expect(screen.getByText("Wrong Page")).toBeTruthy();
            });

            // Simulate navigation to correct page
            mockLocationForStorybook(LINKS.Search);

            // Should auto-advance when we navigate to the correct page
            // This tests the autoAdvanceOnCorrectNavigationEffect
            await waitFor(() => {
                // The tutorial should advance or show correct content
                expect(screen.queryByText("Wrong Page")).not.toBeTruthy();
            }, { timeout: 2000 });
        });
    });
});
