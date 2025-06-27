import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import { action } from "@storybook/addon-actions";
import { ObjectActionDialogs } from "./ObjectActionDialogs.js";
import { ObjectAction } from "../../../utils/actions/objectActions.js";
import { Button } from "../../buttons/Button.js";
import { centeredDecorator } from "../../../__test/helpers/storybookDecorators.tsx";

const meta: Meta<typeof ObjectActionDialogs> = {
    title: "Components/Dialogs/ObjectActionDialogs",
    component: ObjectActionDialogs,
    parameters: {
        layout: "fullscreen",
    },
    decorators: [centeredDecorator],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Mock object for testing
const mockProject = {
    __typename: "Project" as const,
    id: "proj_123",
    handle: "my-awesome-project",
    name: "My Awesome Project",
    description: "A sample project for testing dialog functionality",
    isBookmarked: false,
    isBookmarkedByYou: false,
    bookmarkFor: "Project" as const,
    you: {
        canDelete: true,
        canUpdate: true,
        canRead: true,
        canBookmark: true,
        canShare: true,
        canReport: true,
    },
    owner: {
        __typename: "User" as const,
        id: "user_123", 
        name: "John Doe",
        handle: "johndoe",
    },
};

const mockObjectActionData = {
    availableActions: [
        ObjectAction.Bookmark,
        ObjectAction.Delete,
        ObjectAction.Report,
        ObjectAction.Share,
        ObjectAction.Stats,
        ObjectAction.Donate,
    ],
    isBookmarkDialogOpen: false,
    isDeleteDialogOpen: false,
    isDonateDialogOpen: false,
    isReportDialogOpen: false,
    isShareDialogOpen: false,
    isStatsDialogOpen: false,
    onActionStart: action("onActionStart"),
    onActionComplete: action("onActionComplete"),
    closeBookmarkDialog: action("closeBookmarkDialog"),
    closeDeleteDialog: action("closeDeleteDialog"),
    closeDonateDialog: action("closeDonateDialog"),
    closeReportDialog: action("closeReportDialog"),
    closeShareDialog: action("closeShareDialog"),
    closeStatsDialog: action("closeStatsDialog"),
};

export const Showcase: Story = {
    args: {
        object: mockProject,
        ...mockObjectActionData,
    },
    argTypes: {
        availableActions: {
            control: { type: "check" },
            options: Object.values(ObjectAction),
            description: "Available actions for the object",
        },
        isBookmarkDialogOpen: {
            control: { type: "boolean" },
            description: "Whether the bookmark dialog is open",
        },
        isDeleteDialogOpen: {
            control: { type: "boolean" },
            description: "Whether the delete dialog is open",
        },
        isDonateDialogOpen: {
            control: { type: "boolean" },
            description: "Whether the donate dialog is open",
        },
        isReportDialogOpen: {
            control: { type: "boolean" },
            description: "Whether the report dialog is open",
        },
        isShareDialogOpen: {
            control: { type: "boolean" },
            description: "Whether the share dialog is open",
        },
        isStatsDialogOpen: {
            control: { type: "boolean" },
            description: "Whether the stats dialog is open",
        },
    },
    render: (args) => {
        return (
            <div style={{ padding: "2rem" }}>
                <div style={{ marginBottom: "2rem", textAlign: "center" }}>
                    <h3>Object Action Dialogs</h3>
                    <p>This component manages multiple action dialogs for objects like Projects, Routines, etc.</p>
                    <p>Use the controls below to open different dialogs and test their functionality.</p>
                </div>
                
                <ObjectActionDialogs {...args} />
                
                <div style={{ 
                    marginTop: "2rem", 
                    padding: "1rem", 
                    border: "1px solid #ddd", 
                    borderRadius: "8px",
                    backgroundColor: "white",
                }}>
                    <h4>Object Details:</h4>
                    <p><strong>Type:</strong> {args.object?.__typename}</p>
                    <p><strong>Name:</strong> {args.object?.name}</p>
                    <p><strong>Handle:</strong> {args.object?.handle}</p>
                </div>
            </div>
        );
    },
};

export const BookmarkDialog: Story = {
    args: {
        ...mockObjectActionData,
        object: mockProject,
        isBookmarkDialogOpen: true,
    },
    render: (args) => <ObjectActionDialogs {...args} />,
};

export const DeleteDialog: Story = {
    args: {
        ...mockObjectActionData,
        object: mockProject,
        isDeleteDialogOpen: true,
    },
    render: (args) => <ObjectActionDialogs {...args} />,
};

export const ReportDialog: Story = {
    args: {
        ...mockObjectActionData,
        object: mockProject,
        isReportDialogOpen: true,
    },
    render: (args) => <ObjectActionDialogs {...args} />,
};

export const ShareDialog: Story = {
    args: {
        ...mockObjectActionData,
        object: mockProject,
        isShareDialogOpen: true,
    },
    render: (args) => <ObjectActionDialogs {...args} />,
};

export const StatsDialog: Story = {
    args: {
        ...mockObjectActionData,
        object: mockProject,
        isStatsDialogOpen: true,
    },
    render: (args) => <ObjectActionDialogs {...args} />,
};

export const AllDialogsClosed: Story = {
    args: {
        ...mockObjectActionData,
        object: mockProject,
        // All dialogs are false by default
    },
    render: (args) => (
        <div style={{ padding: "2rem", textAlign: "center" }}>
            <h3>No Dialogs Open</h3>
            <p>All action dialogs are currently closed.</p>
            <p>Use the controls in the Showcase story to open specific dialogs.</p>
            <ObjectActionDialogs {...args} />
        </div>
    ),
};
