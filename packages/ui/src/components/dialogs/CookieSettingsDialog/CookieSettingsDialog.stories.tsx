import type { Meta, StoryObj } from "@storybook/react";
import { useCallback, useState } from "react";
import { CookieSettingsDialog } from "./CookieSettingsDialog.js";

const meta: Meta<typeof CookieSettingsDialog> = {
    title: "Components/Dialogs/CookieSettingsDialog",
    component: CookieSettingsDialog,
    parameters: {
        docs: {
            description: {
                component: "A modern, polished cookie settings dialog with custom Switch and Button components. Features color-coded categories, improved UX with multiple action options, and Tailwind CSS styling.",
            },
        },
    },
};

export default meta;
type Story = StoryObj<typeof CookieSettingsDialog>;

/**
 * Default cookie settings dialog with all features
 */
export const Default: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(true);
        const handleClose = useCallback(() => {
            setIsOpen(false);
            // Reopen after a short delay for demo purposes
            setTimeout(() => setIsOpen(true), 1000);
        }, []);

        return (
            <CookieSettingsDialog
                handleClose={handleClose}
                isOpen={isOpen}
            />
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Shows the polished cookie settings dialog with custom components, color-coded categories, and improved button layout.",
            },
        },
    },
};

/**
 * Closed state (for testing behavior)
 */
export const Closed: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);
        const handleClose = useCallback(() => {
            setIsOpen(false);
        }, []);

        return (
            <div className="tw-p-4">
                <button 
                    className="tw-px-4 tw-py-2 tw-bg-blue-500 tw-text-white tw-rounded hover:tw-bg-blue-600"
                    onClick={() => setIsOpen(true)}
                >
                    Open Cookie Settings
                </button>
                <CookieSettingsDialog
                    handleClose={handleClose}
                    isOpen={isOpen}
                />
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Demonstrates the dialog in closed state with a trigger button.",
            },
        },
    },
};
