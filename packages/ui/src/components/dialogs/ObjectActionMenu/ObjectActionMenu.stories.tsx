import type { Meta, StoryObj } from "@storybook/react";
import Button from "@mui/material/Button";
import { generatePK } from "@vrooli/shared";
import { useRef, useState } from "react";
import { signedInNoPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { ObjectAction } from "../../../utils/actions/objectActions.js";
import { ObjectActionMenu } from "./ObjectActionMenu.js";
import { centeredDecorator } from "../../../__test/helpers/storybookDecorators.tsx";

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
        canDelete: true,
        canComment: true,
        canCopy: true,
        canReport: true,
        canShare: true,
        canBookmark: true,
        canReact: true,
        reaction: null,
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
        canComment: true,
        canCopy: true,
        canReport: true,
        canShare: true,
        canBookmark: true,
        canReact: true,
        reaction: { emotion: "Like", __typename: "Reaction" },
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
        canComment: false,
        canCopy: false,
        canReport: true,
        canShare: true,
        canBookmark: true,
        canReact: false,
        reaction: null,
    },
};

const mockNote = {
    __typename: "Note" as const,
    id: generatePK().toString(),
    name: "Important Notes",
    isPrivate: false,
    you: {
        isBookmarked: false,
        canUpdate: true,
        canDelete: true,
        canComment: true,
        canCopy: true,
        canReport: false,
        canShare: true,
        canBookmark: true,
        canReact: true,
        reaction: null,
    },
};

// Mock action data factory
const createMockActionData = (availableActions: ObjectAction[]) => ({
    availableActions,
    onActionStart: (action: ObjectAction) => console.log("Action started:", action),
    onActionComplete: (action: string, data: any) => console.log("Action completed:", action, data),
    // Mock dialog states
    isBookmarkDialogOpen: false,
    bookmarkFor: null,
    isBulkDeleteDialogOpen: false,
    selectedForDelete: [],
    isDeleteDialogOpen: false,
    deleteObject: null,
    isReportDialogOpen: false,
    reportObject: null,
    isShareDialogOpen: false,
    shareObject: null,
    // Mock action handlers
    closeBookmarkDialog: () => console.log("Bookmark dialog closed"),
    openBookmarkDialog: () => console.log("Bookmark dialog opened"),
    closeBulkDeleteDialog: () => console.log("Bulk delete dialog closed"),
    closeDeleteDialog: () => console.log("Delete dialog closed"),
    closeReportDialog: () => console.log("Report dialog closed"),
    closeShareDialog: () => console.log("Share dialog closed"),
});

const meta: Meta<typeof ObjectActionMenu> = {
    title: "Components/Dialogs/ObjectActionMenu",
    component: ObjectActionMenu,
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
        anchorEl: {
            control: false,
            description: "Element to anchor the menu to",
        },
        object: {
            control: { type: "object" },
            description: "The object to show actions for",
        },
        actionData: {
            control: { type: "object" },
            description: "Object action data and handlers",
        },
        exclude: {
            control: { type: "object" },
            description: "Actions to exclude from the menu",
        },
        onClose: {
            action: "menu-closed",
            description: "Callback when menu is closed",
        },
    },
    decorators: [centeredDecorator],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Showcase with controls
export const Showcase: Story = {
    render: (args) => {
        const buttonRef = useRef<HTMLButtonElement>(null);
        const [anchorEl, setAnchorEl] = useState<Element | null>(null);

        const handleClick = () => {
            setAnchorEl(buttonRef.current);
        };

        const handleClose = () => {
            setAnchorEl(null);
            args.onClose?.();
        };

        const mockActionData = createMockActionData([
            ObjectAction.Edit,
            ObjectAction.VoteUp,
            ObjectAction.Bookmark,
            ObjectAction.Comment,
            ObjectAction.Share,
            ObjectAction.Fork,
            ObjectAction.Report,
            ObjectAction.Delete,
        ]);

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Button
                    ref={buttonRef}
                    onClick={handleClick}
                    variant="contained"
                >
                    Open Action Menu
                </Button>
                <ObjectActionMenu
                    {...args}
                    anchorEl={anchorEl}
                    actionData={mockActionData}
                    onClose={handleClose}
                />
                <div style={{ textAlign: "center", fontSize: "14px", color: "var(--text-secondary)" }}>
                    <p>Menu State: {anchorEl ? "Open" : "Closed"}</p>
                    <p>Object Type: {args.object?.__typename}</p>
                </div>
            </div>
        );
    },
    args: {
        object: mockProject,
        exclude: [],
    },
};

// Project actions (owner)
export const ProjectOwnerActions: Story = {
    render: () => {
        const buttonRef = useRef<HTMLButtonElement>(null);
        const [anchorEl, setAnchorEl] = useState<Element | null>(null);

        const handleClick = () => setAnchorEl(buttonRef.current);
        const handleClose = () => setAnchorEl(null);

        const mockActionData = createMockActionData([
            ObjectAction.Edit,
            ObjectAction.VoteUp,
            ObjectAction.Bookmark,
            ObjectAction.Comment,
            ObjectAction.Share,
            ObjectAction.Fork,
            ObjectAction.Delete,
        ]);

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Button
                    ref={buttonRef}
                    onClick={handleClick}
                    variant="outlined"
                >
                    Project Actions (Owner)
                </Button>
                <ObjectActionMenu
                    anchorEl={anchorEl}
                    object={mockProject}
                    actionData={mockActionData}
                    onClose={handleClose}
                />
                <div style={{ textAlign: "center", fontSize: "14px", color: "var(--text-secondary)" }}>
                    <p>Full permissions on own project</p>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Action menu for a project owned by the current user - shows all available actions.",
            },
        },
    },
};

// Routine actions (bookmarked, voted)
export const RoutineBookmarkedActions: Story = {
    render: () => {
        const buttonRef = useRef<HTMLButtonElement>(null);
        const [anchorEl, setAnchorEl] = useState<Element | null>(null);

        const handleClick = () => setAnchorEl(buttonRef.current);
        const handleClose = () => setAnchorEl(null);

        const mockActionData = createMockActionData([
            ObjectAction.VoteDown, // Already voted up, so show vote down
            ObjectAction.BookmarkUndo, // Already bookmarked, so show unbookmark
            ObjectAction.Comment,
            ObjectAction.Share,
            ObjectAction.Fork,
            ObjectAction.Report,
        ]);

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Button
                    ref={buttonRef}
                    onClick={handleClick}
                    variant="outlined"
                >
                    Routine Actions (Bookmarked)
                </Button>
                <ObjectActionMenu
                    anchorEl={anchorEl}
                    object={mockRoutine}
                    actionData={mockActionData}
                    onClose={handleClose}
                />
                <div style={{ textAlign: "center", fontSize: "14px", color: "var(--text-secondary)" }}>
                    <p>Already bookmarked and voted - shows undo actions</p>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Action menu for a routine that's already bookmarked and voted on.",
            },
        },
    },
};

// User actions (limited)
export const UserLimitedActions: Story = {
    render: () => {
        const buttonRef = useRef<HTMLButtonElement>(null);
        const [anchorEl, setAnchorEl] = useState<Element | null>(null);

        const handleClick = () => setAnchorEl(buttonRef.current);
        const handleClose = () => setAnchorEl(null);

        const mockActionData = createMockActionData([
            ObjectAction.Bookmark,
            ObjectAction.Share,
            ObjectAction.Report,
        ]);

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Button
                    ref={buttonRef}
                    onClick={handleClick}
                    variant="outlined"
                >
                    User Actions (Limited)
                </Button>
                <ObjectActionMenu
                    anchorEl={anchorEl}
                    object={mockUser}
                    actionData={mockActionData}
                    onClose={handleClose}
                />
                <div style={{ textAlign: "center", fontSize: "14px", color: "var(--text-secondary)" }}>
                    <p>Limited actions for user profile</p>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Action menu for a user profile with limited available actions.",
            },
        },
    },
};

// Note actions with exclusions
export const NoteWithExclusions: Story = {
    render: () => {
        const buttonRef = useRef<HTMLButtonElement>(null);
        const [anchorEl, setAnchorEl] = useState<Element | null>(null);

        const handleClick = () => setAnchorEl(buttonRef.current);
        const handleClose = () => setAnchorEl(null);

        // Available actions but excluding some
        const mockActionData = createMockActionData([
            ObjectAction.Edit,
            ObjectAction.VoteUp,
            ObjectAction.Bookmark,
            ObjectAction.Comment,
            ObjectAction.Share,
            ObjectAction.Fork,
            ObjectAction.Delete,
        ]);

        const excludeActions = [ObjectAction.Edit, ObjectAction.Delete];

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Button
                    ref={buttonRef}
                    onClick={handleClick}
                    variant="outlined"
                >
                    Note Actions (Excluded Edit/Delete)
                </Button>
                <ObjectActionMenu
                    anchorEl={anchorEl}
                    object={mockNote}
                    actionData={mockActionData}
                    exclude={excludeActions}
                    onClose={handleClose}
                />
                <div style={{ textAlign: "center", fontSize: "14px", color: "var(--text-secondary)" }}>
                    <p>Edit and Delete actions excluded</p>
                    <p>Useful when these actions are handled elsewhere</p>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Action menu with specific actions excluded - useful when some actions are handled by other UI elements.",
            },
        },
    },
};

// Empty actions
export const EmptyActions: Story = {
    render: () => {
        const buttonRef = useRef<HTMLButtonElement>(null);
        const [anchorEl, setAnchorEl] = useState<Element | null>(null);

        const handleClick = () => setAnchorEl(buttonRef.current);
        const handleClose = () => setAnchorEl(null);

        const mockActionData = createMockActionData([]);

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Button
                    ref={buttonRef}
                    onClick={handleClick}
                    variant="outlined"
                >
                    No Actions Available
                </Button>
                <ObjectActionMenu
                    anchorEl={anchorEl}
                    object={mockUser}
                    actionData={mockActionData}
                    onClose={handleClose}
                />
                <div style={{ textAlign: "center", fontSize: "14px", color: "var(--text-secondary)" }}>
                    <p>Menu with no available actions</p>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Action menu when no actions are available for the object.",
            },
        },
    },
};

// Different object types comparison
export const ObjectTypeComparison: Story = {
    render: () => {
        const [anchorEls, setAnchorEls] = useState<{ [key: string]: Element | null }>({});

        const handleClick = (objectType: string, element: Element | null) => {
            setAnchorEls(prev => ({ ...prev, [objectType]: element }));
        };

        const handleClose = (objectType: string) => {
            setAnchorEls(prev => ({ ...prev, [objectType]: null }));
        };

        const objects = [
            { 
                name: "Project", 
                object: mockProject, 
                actions: [ObjectAction.Edit, ObjectAction.VoteUp, ObjectAction.Bookmark, ObjectAction.Share, ObjectAction.Fork, ObjectAction.Delete], 
            },
            { 
                name: "Routine", 
                object: mockRoutine, 
                actions: [ObjectAction.VoteDown, ObjectAction.BookmarkUndo, ObjectAction.Comment, ObjectAction.Share, ObjectAction.Fork], 
            },
            { 
                name: "User", 
                object: mockUser, 
                actions: [ObjectAction.Bookmark, ObjectAction.Share, ObjectAction.Report], 
            },
            { 
                name: "Note", 
                object: mockNote, 
                actions: [ObjectAction.Edit, ObjectAction.VoteUp, ObjectAction.Bookmark, ObjectAction.Comment, ObjectAction.Share, ObjectAction.Delete], 
            },
        ];

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <div style={{ display: "flex", gap: "10px", flexWrap: "wrap" }}>
                    {objects.map(({ name, object, actions }) => {
                        const buttonRef = useRef<HTMLButtonElement>(null);
                        return (
                            <div key={name}>
                                <Button
                                    ref={buttonRef}
                                    onClick={() => handleClick(name, buttonRef.current)}
                                    variant="outlined"
                                    size="small"
                                >
                                    {name}
                                </Button>
                                <ObjectActionMenu
                                    anchorEl={anchorEls[name] || null}
                                    object={object}
                                    actionData={createMockActionData(actions)}
                                    onClose={() => handleClose(name)}
                                />
                            </div>
                        );
                    })}
                </div>
                <div style={{ textAlign: "center", fontSize: "14px", color: "var(--text-secondary)" }}>
                    <p>Compare action menus for different object types</p>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Comparison of action menus for different object types to see how actions vary.",
            },
        },
    },
};

// Always open for design testing
export const AlwaysOpen: Story = {
    render: () => {
        const buttonRef = useRef<HTMLButtonElement>(null);

        const mockActionData = createMockActionData([
            ObjectAction.Edit,
            ObjectAction.VoteUp,
            ObjectAction.Bookmark,
            ObjectAction.Comment,
            ObjectAction.Share,
            ObjectAction.Fork,
            ObjectAction.Report,
            ObjectAction.Delete,
        ]);

        const handleClose = () => console.log("Menu closed");

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <Button
                    ref={buttonRef}
                    variant="contained"
                    disabled
                >
                    Menu Anchor
                </Button>
                <ObjectActionMenu
                    anchorEl={buttonRef.current}
                    object={mockProject}
                    actionData={mockActionData}
                    onClose={handleClose}
                />
                <div style={{ textAlign: "center", fontSize: "14px", color: "var(--text-secondary)" }}>
                    <p>Menu that stays open for design testing</p>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Action menu that stays open for design and interaction testing.",
            },
        },
    },
};
