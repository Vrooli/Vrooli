import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import { action } from "@storybook/addon-actions";
import {
    Box,
    Card,
    CardContent,
    Typography,
    TextField,
    Alert,
    Stack,
} from "@mui/material";
import { Button } from "../../buttons/Button.js";
import { DeleteAccountDialog } from "./DeleteAccountDialog.js";
import { SessionContext } from "../../../contexts/session.js";
import type { Session } from "@vrooli/shared";
import { centeredDecorator } from "../../../__test/helpers/storybookDecorators.tsx";

// Mock session data for the story
const createMockSession = (userName = "john_doe"): Session => ({
    id: "session-123",
    theme: "light",
    users: [{
        id: "user-123",
        name: userName,
        handle: userName.toLowerCase(),
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        bannerImage: null,
        bio: null,
        botSettings: null,
        emails: [],
        focusModes: [],
        hasPremium: false,
        isBot: false,
        isPrivate: false,
        isVerified: false,
        languages: ["en"],
        profileImage: null,
        reportsReceivedCount: 0,
        resourceLists: [],
        stats: [],
        translations: [],
        wallets: [],
        you: {
            canDelete: true,
            canReport: false,
            canSupport: false,
            canUpdate: true,
            isBlocked: false,
            isBookmarked: false,
            isViewed: true,
        },
        __typename: "User" as const,
    }],
    __typename: "Session" as const,
});

const meta: Meta<typeof DeleteAccountDialog> = {
    title: "Components/Dialogs/DeleteAccountDialog",
    component: DeleteAccountDialog,
    parameters: {
        layout: "fullscreen",
        backgrounds: { disable: true },
        docs: {
            story: {
                inline: false,
                iframeHeight: 700,
            },
        },
    },
    tags: ["autodocs"],
    decorators: [centeredDecorator],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Showcase story with controls
export const Showcase: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);
        const [userName, setUserName] = useState("john_doe");
        const mockSession = createMockSession(userName);

        const handleClose = (wasDeleted: boolean) => {
            action("delete-account-dialog-closed")({ wasDeleted, userName });
            setIsOpen(false);
        };

        return (
            <SessionContext.Provider value={mockSession}>
                <Stack spacing={3} sx={{ maxWidth: 600 }}>
                    {/* Control Panel */}
                    <Card>
                        <CardContent>
                            <Typography variant="h6" gutterBottom>
                                DeleteAccountDialog Controls
                            </Typography>
                            
                            <TextField
                                label="User Name"
                                value={userName}
                                onChange={(e) => setUserName(e.target.value)}
                                placeholder="Enter username for mock session"
                                fullWidth
                                sx={{ mb: 2 }}
                            />

                            <Alert severity="info">
                                <Typography variant="body2">
                                    <strong>Note:</strong> This is a mock dialog. In the real app, it would actually delete the user's account.
                                    Password validation is simulated - any password will work in Storybook.
                                </Typography>
                            </Alert>
                        </CardContent>
                    </Card>

                    {/* Trigger Button */}
                    <Box sx={{ textAlign: "center" }}>
                        <Button onClick={() => setIsOpen(true)} variant="danger" size="lg">
                            Delete Account "{userName}"
                        </Button>
                    </Box>

                    {/* Feature highlights */}
                    <Alert severity="success">
                        <Typography variant="body2">
                            <strong>Dialog Features:</strong>
                        </Typography>
                        <Box component="ul" sx={{ mt: 1, mb: 0, pl: 2 }}>
                            <li>Password confirmation required</li>
                            <li>Option to delete public data</li>
                            <li>Warning about action being irreversible</li>
                            <li>Formik form validation</li>
                            <li>Danger variant styling (red theme)</li>
                            <li>Returns boolean indicating if deletion occurred</li>
                        </Box>
                    </Alert>

                    {/* Warning Notice */}
                    <Alert severity="error">
                        <Typography variant="body2">
                            <strong>⚠️ Important:</strong> In a real application, this dialog would permanently delete the user's account.
                            This is a destructive action that cannot be undone.
                        </Typography>
                    </Alert>

                    {/* Dialog Component */}
                    <DeleteAccountDialog
                        isOpen={isOpen}
                        handleClose={handleClose}
                    />
                </Stack>
            </SessionContext.Provider>
        );
    },
};

// Example with different user names
export const DifferentUsers: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);
        const [currentUser, setCurrentUser] = useState("alice_smith");
        const users = ["alice_smith", "bob_jones", "charlie_brown", "diana_prince"];

        const mockSession = createMockSession(currentUser);

        const handleClose = (wasDeleted: boolean) => {
            action("delete-account-closed")({ wasDeleted, userName: currentUser });
            setIsOpen(false);
        };

        return (
            <SessionContext.Provider value={mockSession}>
                <Stack spacing={2}>
                    <Box sx={{ 
                        display: "flex", 
                        gap: 1, 
                        flexWrap: "wrap",
                    }}>
                        {users.map(user => (
                            <Button
                                key={user}
                                variant={currentUser === user ? "primary" : "outline"}
                                size="sm"
                                onClick={() => setCurrentUser(user)}
                            >
                                {user}
                            </Button>
                        ))}
                    </Box>
                    
                    <Box sx={{ textAlign: "center" }}>
                        <Button onClick={() => setIsOpen(true)} variant="danger">
                            Delete Account "{currentUser}"
                        </Button>
                    </Box>
                    
                    <DeleteAccountDialog
                        isOpen={isOpen}
                        handleClose={handleClose}
                    />
                </Stack>
            </SessionContext.Provider>
        );
    },
};

// Example showing the public data checkbox
export const PublicDataOption: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);
        const mockSession = createMockSession("content_creator");

        const handleClose = (wasDeleted: boolean) => {
            action("delete-account-closed")({ wasDeleted, userName: "content_creator" });
            setIsOpen(false);
        };

        return (
            <SessionContext.Provider value={mockSession}>
                <Stack spacing={2}>
                    <Alert severity="info">
                        <Typography variant="body2">
                            <strong>Public Data Checkbox:</strong> This dialog includes a checkbox option to delete public data.
                            When checked, all public content created by the user will also be deleted.
                            This is especially important for content creators.
                        </Typography>
                    </Alert>
                    
                    <Box sx={{ textAlign: "center" }}>
                        <Button onClick={() => setIsOpen(true)} variant="danger">
                            Delete Content Creator Account
                        </Button>
                    </Box>
                    
                    <DeleteAccountDialog
                        isOpen={isOpen}
                        handleClose={handleClose}
                    />
                </Stack>
            </SessionContext.Provider>
        );
    },
};

// Example with empty/minimal session
export const MinimalSession: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);
        const mockSession = createMockSession("");

        const handleClose = (wasDeleted: boolean) => {
            action("delete-account-closed")({ wasDeleted, userName: "" });
            setIsOpen(false);
        };

        return (
            <SessionContext.Provider value={mockSession}>
                <Stack spacing={2}>
                    <Alert severity="warning">
                        <Typography variant="body2">
                            <strong>Edge Case:</strong> This example shows what happens when the user has no name or minimal session data.
                        </Typography>
                    </Alert>
                    
                    <Box sx={{ textAlign: "center" }}>
                        <Button onClick={() => setIsOpen(true)} variant="danger">
                            Delete Account (No Name)
                        </Button>
                    </Box>
                    
                    <DeleteAccountDialog
                        isOpen={isOpen}
                        handleClose={handleClose}
                    />
                </Stack>
            </SessionContext.Provider>
        );
    },
};
