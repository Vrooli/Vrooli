import type { Meta, StoryObj } from "@storybook/react";
import { generatePK } from "@vrooli/shared";
import { useState } from "react";
import { Button } from "../../buttons/Button.js";
import { ShareObjectDialog } from "./ShareObjectDialog.js";
import { centeredDecorator } from "../../../__test/helpers/storybookDecorators.tsx";

const meta: Meta<typeof ShareObjectDialog> = {
    title: "Components/Dialogs/ShareObjectDialog",
    component: ShareObjectDialog,
    parameters: {
        layout: "fullscreen",
        backgrounds: { disable: true },
        docs: {
            story: {
                inline: false,
                iframeHeight: 600,
            },
        },
    },
    tags: ["autodocs"],
    argTypes: {
        open: {
            control: { type: "boolean" },
            description: "Whether the dialog is open",
        },
        object: {
            control: { type: "object" },
            description: "The object to share",
        },
        onClose: {
            action: "closed",
            description: "Callback when dialog is closed",
        },
    },
    decorators: [centeredDecorator],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Mock objects for different types
const mockProject = {
    __typename: "Project" as const,
    id: generatePK().toString(),
    handle: "awesome-project",
    name: "Awesome AI Project",
    isPrivate: false,
    you: {
        isBookmarked: false,
        canUpdate: true,
        canDelete: false,
    },
};

const mockRoutine = {
    __typename: "Routine" as const,
    id: generatePK().toString(),
    isInternal: false,
    isPrivate: false,
    name: "Data Processing Routine",
    you: {
        isBookmarked: true,
        canUpdate: false,
        canDelete: false,
    },
};

const mockUser = {
    __typename: "User" as const,
    id: generatePK().toString(),
    handle: "john_doe",
    name: "John Doe",
    you: {
        isBookmarked: false,
        canUpdate: false,
        canDelete: false,
    },
};

const mockTeam = {
    __typename: "Team" as const,
    id: generatePK().toString(),
    handle: "awesome-team",
    name: "Awesome Development Team",
    isPrivate: false,
    you: {
        isBookmarked: true,
        canUpdate: false,
        canDelete: false,
    },
};

const mockStandard = {
    __typename: "Standard" as const,
    id: generatePK().toString(),
    name: "API Response Standard",
    isPrivate: false,
    you: {
        isBookmarked: false,
        canUpdate: true,
        canDelete: false,
    },
};

const mockComment = {
    __typename: "Comment" as const,
    id: generatePK().toString(),
    content: "This is a great project! I really like the approach you've taken here.",
    you: {
        isBookmarked: false,
        canUpdate: false,
        canDelete: false,
    },
};

// Showcase with controls
export const Showcase: Story = {
    render: (args) => {
        const [isOpen, setIsOpen] = useState(false);

        return (
            <>
                <Button variant="primary" onClick={() => setIsOpen(true)}>
                    Open Share Dialog
                </Button>
                <ShareObjectDialog
                    {...args}
                    open={isOpen || args.open}
                    onClose={() => {
                        setIsOpen(false);
                        args.onClose?.();
                    }}
                />
            </>
        );
    },
    args: {
        open: false,
        object: mockProject,
    },
};

// Project sharing
export const ShareProject: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(true);

        return (
            <>
                <Button variant="primary" onClick={() => setIsOpen(true)}>
                    Share Project
                </Button>
                <ShareObjectDialog
                    open={isOpen}
                    object={mockProject}
                    onClose={() => setIsOpen(false)}
                />
            </>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Sharing a project with QR code, link, and object export options.",
            },
        },
    },
};

// Routine sharing
export const ShareRoutine: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(true);

        return (
            <>
                <Button variant="secondary" onClick={() => setIsOpen(true)}>
                    Share Routine
                </Button>
                <ShareObjectDialog
                    open={isOpen}
                    object={mockRoutine}
                    onClose={() => setIsOpen(false)}
                />
            </>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Sharing a routine with all available sharing options.",
            },
        },
    },
};

// User profile sharing
export const ShareUser: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(true);

        return (
            <>
                <Button variant="outline" onClick={() => setIsOpen(true)}>
                    Share User Profile
                </Button>
                <ShareObjectDialog
                    open={isOpen}
                    object={mockUser}
                    onClose={() => setIsOpen(false)}
                />
            </>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Sharing a user profile.",
            },
        },
    },
};

// Team sharing
export const ShareTeam: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(true);

        return (
            <>
                <Button variant="ghost" onClick={() => setIsOpen(true)}>
                    Share Team
                </Button>
                <ShareObjectDialog
                    open={isOpen}
                    object={mockTeam}
                    onClose={() => setIsOpen(false)}
                />
            </>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Sharing a team page.",
            },
        },
    },
};

// Standard sharing
export const ShareStandard: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(true);

        return (
            <>
                <Button variant="primary" onClick={() => setIsOpen(true)}>
                    Share Standard
                </Button>
                <ShareObjectDialog
                    open={isOpen}
                    object={mockStandard}
                    onClose={() => setIsOpen(false)}
                />
            </>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Sharing a standard definition.",
            },
        },
    },
};

// Comment sharing
export const ShareComment: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(true);

        return (
            <>
                <Button variant="danger" onClick={() => setIsOpen(true)}>
                    Share Comment
                </Button>
                <ShareObjectDialog
                    open={isOpen}
                    object={mockComment}
                    onClose={() => setIsOpen(false)}
                />
            </>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Sharing a comment with limited sharing options.",
            },
        },
    },
};

// No object (fallback)
export const NoObject: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(true);

        return (
            <>
                <Button variant="outline" onClick={() => setIsOpen(true)}>
                    Share Current Page
                </Button>
                <ShareObjectDialog
                    open={isOpen}
                    object={null}
                    onClose={() => setIsOpen(false)}
                />
            </>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Share dialog with no specific object - defaults to current page URL.",
            },
        },
    },
};
