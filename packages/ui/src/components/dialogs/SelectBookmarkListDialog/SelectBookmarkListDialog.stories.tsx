import type { Meta, StoryObj } from "@storybook/react";
import { generatePK, type BookmarkList } from "@vrooli/shared";
import { useState } from "react";
import { signedInNoPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { Button } from "../../buttons/Button.js";
import { SelectBookmarkListDialog } from "./SelectBookmarkListDialog.js";
import { centeredDecorator } from "../../../__test/helpers/storybookDecorators.tsx";

const meta: Meta<typeof SelectBookmarkListDialog> = {
    title: "Components/Dialogs/SelectBookmarkListDialog",
    component: SelectBookmarkListDialog,
    parameters: {
        layout: "fullscreen",
        backgrounds: { disable: true },
        docs: {
            story: {
                inline: false,
                iframeHeight: 600,
            },
        },
        session: signedInNoPremiumWithCreditsSession,
    },
    tags: ["autodocs"],
    argTypes: {
        isOpen: {
            control: { type: "boolean" },
            description: "Whether the dialog is open",
        },
        objectId: {
            control: { type: "text" },
            description: "ID of the object to bookmark",
        },
        objectType: {
            control: { type: "select" },
            options: ["Api", "Bot", "Comment", "Meeting", "Note", "Project", "Routine", "Standard", "Team", "User"],
            description: "Type of object to bookmark",
        },
        isCreate: {
            control: { type: "boolean" },
            description: "Whether this is for creating a new bookmark",
        },
        onClose: {
            action: "closed",
            description: "Callback when dialog is closed",
        },
    },
    decorators: [
        (Story, context) => {
            // Mock bookmark lists for the story
            const mockBookmarkLists: BookmarkList[] = [
                {
                    __typename: "BookmarkList",
                    id: generatePK().toString(),
                    label: "Favorites",
                    bookmarksCount: 12,
                    isPrivate: false,
                    you: {
                        canDelete: true,
                        canUpdate: true,
                    },
                },
                {
                    __typename: "BookmarkList",
                    id: generatePK().toString(),
                    label: "AI Projects",
                    bookmarksCount: 8,
                    isPrivate: false,
                    you: {
                        canDelete: true,
                        canUpdate: true,
                    },
                },
                {
                    __typename: "BookmarkList",
                    id: generatePK().toString(),
                    label: "Learning Resources",
                    bookmarksCount: 25,
                    isPrivate: true,
                    you: {
                        canDelete: true,
                        canUpdate: true,
                    },
                },
                {
                    __typename: "BookmarkList",
                    id: generatePK().toString(),
                    label: "Work Projects",
                    bookmarksCount: 5,
                    isPrivate: false,
                    you: {
                        canDelete: true,
                        canUpdate: true,
                    },
                },
                {
                    __typename: "BookmarkList",
                    id: generatePK().toString(),
                    label: "Personal",
                    bookmarksCount: 3,
                    isPrivate: true,
                    you: {
                        canDelete: true,
                        canUpdate: true,
                    },
                },
            ];

            // Mock the bookmark lists store
            const originalUseBookmarkListsStore = require("../../../stores/bookmarkListsStore.js").useBookmarkListsStore;
            
            return centeredDecorator(Story);
        },
    ],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Mock object IDs
const mockProjectId = generatePK().toString();
const mockRoutineId = generatePK().toString();
const mockUserId = generatePK().toString();

// Showcase with controls
export const Showcase: Story = {
    render: (args) => {
        const [isOpen, setIsOpen] = useState(false);

        return (
            <>
                <Button variant="primary" onClick={() => setIsOpen(true)}>
                    Select Bookmark Lists
                </Button>
                <SelectBookmarkListDialog
                    {...args}
                    isOpen={isOpen || args.isOpen}
                    onClose={(inList) => {
                        setIsOpen(false);
                        args.onClose?.(inList);
                    }}
                />
            </>
        );
    },
    args: {
        isOpen: false,
        objectId: mockProjectId,
        objectType: "Project",
        isCreate: true,
    },
};

// Adding new bookmark to lists
export const AddToBookmarkLists: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(true);

        return (
            <>
                <Button variant="primary" onClick={() => setIsOpen(true)}>
                    Add to Bookmark Lists
                </Button>
                <SelectBookmarkListDialog
                    isOpen={isOpen}
                    objectId={mockProjectId}
                    objectType="Project"
                    isCreate={true}
                    onClose={(inList) => {
                        setIsOpen(false);
                        console.log("Object is in bookmark list:", inList);
                    }}
                />
            </>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Adding a new object to bookmark lists. Shows checkboxes for all available lists with create list option.",
            },
        },
    },
};

// Editing existing bookmarks
export const EditBookmarks: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(true);

        return (
            <>
                <Button variant="secondary" onClick={() => setIsOpen(true)}>
                    Edit Bookmarks
                </Button>
                <SelectBookmarkListDialog
                    isOpen={isOpen}
                    objectId={mockRoutineId}
                    objectType="Routine"
                    isCreate={false}
                    onClose={(inList) => {
                        setIsOpen(false);
                        console.log("Object is in bookmark list:", inList);
                    }}
                />
            </>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Editing existing bookmarks for an object. Pre-selects lists that already contain the object.",
            },
        },
    },
};

// Different object types
export const BookmarkUser: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(true);

        return (
            <>
                <Button variant="outline" onClick={() => setIsOpen(true)}>
                    Bookmark User
                </Button>
                <SelectBookmarkListDialog
                    isOpen={isOpen}
                    objectId={mockUserId}
                    objectType="User"
                    isCreate={true}
                    onClose={(inList) => {
                        setIsOpen(false);
                        console.log("User is in bookmark list:", inList);
                    }}
                />
            </>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Bookmarking a user profile to lists.",
            },
        },
    },
};

export const BookmarkTeam: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(true);

        return (
            <>
                <Button variant="ghost" onClick={() => setIsOpen(true)}>
                    Bookmark Team
                </Button>
                <SelectBookmarkListDialog
                    isOpen={isOpen}
                    objectId={generatePK().toString()}
                    objectType="Team"
                    isCreate={true}
                    onClose={(inList) => {
                        setIsOpen(false);
                        console.log("Team is in bookmark list:", inList);
                    }}
                />
            </>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Bookmarking a team to lists.",
            },
        },
    },
};

export const BookmarkStandard: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(true);

        return (
            <>
                <Button variant="primary" onClick={() => setIsOpen(true)}>
                    Bookmark Standard
                </Button>
                <SelectBookmarkListDialog
                    isOpen={isOpen}
                    objectId={generatePK().toString()}
                    objectType="Standard"
                    isCreate={true}
                    onClose={(inList) => {
                        setIsOpen(false);
                        console.log("Standard is in bookmark list:", inList);
                    }}
                />
            </>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Bookmarking a standard to lists.",
            },
        },
    },
};

// No object selected
export const NoObjectSelected: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(true);

        return (
            <>
                <Button variant="danger" onClick={() => setIsOpen(true)}>
                    No Object Selected
                </Button>
                <SelectBookmarkListDialog
                    isOpen={isOpen}
                    objectId={null}
                    objectType="Project"
                    isCreate={true}
                    onClose={(inList) => {
                        setIsOpen(false);
                        console.log("Object is in bookmark list:", inList);
                    }}
                />
            </>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Dialog behavior when no object ID is provided.",
            },
        },
    },
};
