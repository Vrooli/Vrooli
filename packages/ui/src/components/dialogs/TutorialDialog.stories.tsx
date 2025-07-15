import type { Meta, StoryObj } from "@storybook/react";
import React from "react";
import { baseSession } from "../../__test/storybookConsts.js";
import { TutorialDialog } from "./TutorialDialog.js";
import { Box } from "@mui/material";
import * as RouterModule from "../../route/router.js";

const meta: Meta<typeof TutorialDialog> = {
    title: "Components/Dialogs/TutorialDialog",
    component: TutorialDialog,
    args: {
        bypassPageValidation: true,
    },
    parameters: {
        docs: {
            description: {
                component: "Interactive tutorial dialog that guides users through the Vrooli interface. Supports both popover (anchored to elements) and dialog modes with navigation, keyboard shortcuts, and URL state management.\n\n**Note:** In Storybook, page validation is bypassed to allow viewing tutorial content regardless of URL structure.\n\n**Usage:** Use the 'Open Tutorial' button to start the tutorial. Navigate using arrow keys, the step dots, or the 'Jump to Section' menu.",
            },
        },
        session: baseSession,
    },
    decorators: [
        (Story) => {
            
            // Mock DOM elements that tutorial might anchor to
            const mockElements = [
                { id: "chat-input-area", rect: { x: 100, y: 600, width: 600, height: 80 }, label: "Chat Input Area" },
                { id: "chat-bubble-tree", rect: { x: 100, y: 250, width: 600, height: 330 }, label: "Chat Messages" },
                { id: "routine-executor", rect: { x: 750, y: 250, width: 350, height: 330 }, label: "Routine Executor" },
                { id: "user-menu-profile-icon", rect: { x: 1060, y: 20, width: 40, height: 40 }, label: "Profile" },
                { id: "user-menu-account-list", rect: { x: 850, y: 80, width: 250, height: 150 }, label: "Account List" },
            ];

            // Create mock elements in DOM
            React.useEffect(() => {
                mockElements.forEach(({ id, rect, label }) => {
                    let element = document.getElementById(id);
                    if (!element) {
                        element = document.createElement("div");
                        element.id = id;
                        element.style.position = "absolute";
                        element.style.left = `${rect.x}px`;
                        element.style.top = `${rect.y}px`;
                        element.style.width = `${rect.width}px`;
                        element.style.height = `${rect.height}px`;
                        element.style.backgroundColor = "rgba(0, 123, 255, 0.1)";
                        element.style.border = "2px dashed #007bff";
                        element.style.borderRadius = "8px";
                        element.style.zIndex = "1";
                        element.style.display = "flex";
                        element.style.alignItems = "center";
                        element.style.justifyContent = "center";
                        element.style.flexDirection = "column";
                        element.innerHTML = `
                            <div style="padding: 8px; text-align: center;">
                                <div style="font-size: 14px; font-weight: bold; color: #007bff; margin-bottom: 4px;">${label}</div>
                                <small style="font-size: 11px; color: #666;">${id}</small>
                            </div>
                        `;
                        element.classList.add("tutorial-mock-element");
                        document.body.appendChild(element);
                    }
                });

                // Cleanup function to remove mock elements when story unmounts
                return () => {
                    const mockElementsToRemove = document.querySelectorAll(".tutorial-mock-element");
                    mockElementsToRemove.forEach(element => {
                        element.remove();
                    });
                };
            }, []);

            return (
                <div style={{ minHeight: "100vh", position: "relative", padding: "60px 20px 20px" }}>
                    
                    <div style={{ maxWidth: "800px", margin: "0 auto" }}>
                        <h2 style={{ marginBottom: "16px" }}>Tutorial Dialog Test Environment</h2>
                        <p style={{ marginBottom: "24px", color: "#666" }}>
                            <strong>Note:</strong> Navigation controls (arrows, dots, close button) may not work in Storybook 
                            due to router limitations. This component is designed to work with the actual app's router.
                            The blue dashed boxes represent UI elements that the tutorial will highlight during different steps.
                        </p>
                        
                        <Box
                            sx={{
                                backgroundColor: "info.light",
                                color: "info.contrastText",
                                p: 2,
                                borderRadius: 1,
                                mb: 3,
                                border: 1,
                                borderColor: "info.main",
                            }}
                        >
                            <h3 style={{ marginTop: 0, marginBottom: "12px" }}>Navigation (in actual app):</h3>
                            <ul style={{ marginBottom: 0, paddingLeft: "20px" }}>
                                <li>Use <strong>Arrow Keys</strong> to move between steps</li>
                                <li>Click the <strong>step dots</strong> at the bottom to jump to any step</li>
                                <li>Click the <strong>section title</strong> to open the section menu</li>
                                <li>Press <strong>Escape</strong> or click the <strong>X</strong> to close the tutorial</li>
                                <li>The tutorial will anchor to UI elements and track progress via URL</li>
                            </ul>
                        </Box>
                    </div>
                    
                    <Story />
                </div>
            );
        },
    ],
};

export default meta;
type Story = StoryObj<typeof TutorialDialog>;

// Create a component that forces the tutorial to be visible
const ForcedOpenTutorialDialog = () => {
    React.useEffect(() => {
        // Update the browser URL to include tutorial parameters
        const url = new URL(window.location.href);
        url.searchParams.set("TutorialView", JSON.stringify({ section: 0, step: 0 }));
        window.history.replaceState({}, "", url.toString());
        
        // Force a re-render by changing the location slightly
        window.dispatchEvent(new PopStateEvent("popstate"));
        
        return () => {
            // Clean up on unmount
            const cleanUrl = new URL(window.location.href);
            cleanUrl.searchParams.delete("TutorialView");
            window.history.replaceState({}, "", cleanUrl.toString());
        };
    }, []);
    
    return <TutorialDialog bypassPageValidation={true} />;
};

/**
 * Interactive tutorial that you can navigate through manually.
 * The tutorial starts from the beginning and allows full navigation.
 */
export const InteractiveTutorial: Story = {
    render: () => <ForcedOpenTutorialDialog />,
    parameters: {
        docs: {
            description: {
                story: "A preview of the TutorialDialog component. The story attempts to force the dialog open by setting URL parameters. Navigation controls may have limited functionality in Storybook due to router constraints.",
            },
        },
    },
};
