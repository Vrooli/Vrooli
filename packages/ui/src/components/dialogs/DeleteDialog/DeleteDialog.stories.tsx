import type { Meta, StoryObj } from "@storybook/react";
import React, { useState } from "react";
import { action } from "@storybook/addon-actions";
import { 
    Box, 
    Card, 
    CardContent, 
    Typography, 
    FormControl,
    InputLabel,
    Select,
    MenuItem,
    Alert,
    Stack,
} from "@mui/material";
import { Button } from "../../buttons/Button.js";
import { DeleteDialog } from "./DeleteDialog.js";
import { centeredDecorator } from "../../../__test/helpers/storybookDecorators.tsx";
import { getDisplay } from "../../../utils/display/listTools.js";
import { type IconInfo } from "../../../icons/Icons.js";
import { type ListObject } from "@vrooli/shared";

// Mock object data for different types
const mockObjects = {
    user: {
        __typename: "User",
        id: "user-1",
        handle: "johndoe",
        name: "John Doe",
        isBot: false,
        translations: [{ language: "en", name: "John Doe", bio: "Software developer" }],
    } as ListObject,
    botUser: {
        __typename: "User", 
        id: "bot-1",
        handle: "helpbot",
        name: "Help Bot",
        isBot: true,
        translations: [{ language: "en", name: "Help Bot", bio: "Automated assistant" }],
    } as ListObject,
    team: {
        __typename: "Team",
        id: "team-1",
        handle: "acme-corp",
        translations: [{ language: "en", name: "ACME Corporation", bio: "Leading software company" }],
    } as ListObject,
    project: {
        __typename: "Project",
        id: "project-1",
        handle: "todo-app",
        translations: [{ language: "en", name: "Todo Application", description: "A simple task management app" }],
    } as ListObject,
    routine: {
        __typename: "ResourceVersion",
        id: "routine-1",
        translations: [{ language: "en", name: "Email Automation", description: "Automate email responses" }],
        root: {
            __typename: "Resource",
            id: "resource-1",
        },
    } as ListObject,
    chat: {
        __typename: "Chat",
        id: "chat-1",
        translations: [{ language: "en", name: "Team Standup", description: "Daily standup discussion" }],
        participantsCount: 5,
    } as ListObject,
    meeting: {
        __typename: "Meeting",
        id: "meeting-1", 
        translations: [{ language: "en", name: "Quarterly Review", description: "Q4 performance review" }],
        attendeesCount: 8,
    } as ListObject,
    bookmarkList: {
        __typename: "BookmarkList",
        id: "bookmark-1",
        translations: [{ language: "en", name: "Favorite Resources", description: "My most useful tools" }],
    } as ListObject,
    schedule: {
        __typename: "Schedule",
        id: "schedule-1",
        translations: [{ language: "en", name: "Weekly Team Meeting", description: "Every Monday at 9 AM" }],
    } as ListObject,
    reminder: {
        __typename: "Reminder",
        id: "reminder-1",
        translations: [{ language: "en", name: "Project Deadline", description: "Submit final report" }],
    } as ListObject,
};

// Icon mapping for object types
const getObjectIcon = (objectType: string, isBot?: boolean): IconInfo => {
    const iconMap: Record<string, IconInfo> = {
        User: { name: isBot ? "Bot" : "User", type: "Common" },
        Team: { name: "Team", type: "Common" },
        Project: { name: "Project", type: "Common" },
        ResourceVersion: { name: "Routine", type: "Routine" },
        Chat: { name: "Comment", type: "Common" },
        Meeting: { name: "Team", type: "Common" },
        BookmarkList: { name: "BookmarkFilled", type: "Common" },
        Schedule: { name: "Schedule", type: "Common" },
        Reminder: { name: "Reminder", type: "Common" },
    };
    return iconMap[objectType] || { name: "Delete", type: "Common" };
};

// Helper to get display info for an object
const getObjectDisplayInfo = (object: ListObject) => {
    const display = getDisplay(object);
    const icon = getObjectIcon(object.__typename, (object as any).isBot);
    return { title: display.title, icon };
};

const meta: Meta<typeof DeleteDialog> = {
    title: "Components/Dialogs/DeleteDialog",
    component: DeleteDialog,
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
    decorators: [centeredDecorator],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Comprehensive showcase with object type selector
export const Showcase: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);
        const [selectedObjectType, setSelectedObjectType] = useState<keyof typeof mockObjects>("user");
        
        const selectedObject = mockObjects[selectedObjectType];
        const { title: objectName, icon } = getObjectDisplayInfo(selectedObject);

        const handleClose = (wasDeleted: boolean) => {
            action("delete-dialog-closed")({ wasDeleted, objectType: selectedObjectType, objectName });
            setIsOpen(false);
        };

        const handleDelete = () => {
            action("delete-confirmed")({ objectType: selectedObjectType, objectName });
            // Simulate deletion process
            setTimeout(() => {
                handleClose(true);
            }, 500);
        };

        return (
            <Stack spacing={3} sx={{ maxWidth: 600 }}>
                {/* Control Panel */}
                <Card>
                    <CardContent>
                        <Typography variant="h6" gutterBottom>
                            DeleteDialog Controls
                        </Typography>
                        
                        <FormControl fullWidth sx={{ mb: 2 }}>
                            <InputLabel>Object Type</InputLabel>
                            <Select
                                value={selectedObjectType}
                                label="Object Type"
                                onChange={(e) => setSelectedObjectType(e.target.value as keyof typeof mockObjects)}
                            >
                                <MenuItem value="user">User - Regular User</MenuItem>
                                <MenuItem value="botUser">User - Bot Account</MenuItem>
                                <MenuItem value="team">Team Organization</MenuItem>
                                <MenuItem value="project">Project</MenuItem>
                                <MenuItem value="routine">Routine (ResourceVersion)</MenuItem>
                                <MenuItem value="chat">Chat Conversation</MenuItem>
                                <MenuItem value="meeting">Meeting</MenuItem>
                                <MenuItem value="bookmarkList">Bookmark List</MenuItem>
                                <MenuItem value="schedule">Schedule</MenuItem>
                                <MenuItem value="reminder">Reminder</MenuItem>
                            </Select>
                        </FormControl>

                        <Alert severity="info" sx={{ mb: 2 }}>
                            <Typography variant="body2">
                                <strong>Selected Object:</strong><br />
                                <em>Type:</em> {selectedObject.__typename}<br />
                                <em>Name:</em> {objectName}<br />
                                <em>Icon:</em> {icon.name} ({icon.type})
                            </Typography>
                        </Alert>

                        <Alert severity="warning">
                            <Typography variant="body2">
                                <strong>Note:</strong> User must type the exact object name "{objectName}" to enable the delete button.
                            </Typography>
                        </Alert>
                    </CardContent>
                </Card>

                {/* Trigger Button */}
                <Box sx={{ textAlign: "center" }}>
                    <Button onClick={() => setIsOpen(true)} variant="danger" size="lg">
                        Delete {selectedObject.__typename}: "{objectName}"
                    </Button>
                </Box>

                {/* Feature highlights */}
                <Alert severity="success">
                    <Typography variant="body2">
                        <strong>Dialog Features:</strong>
                    </Typography>
                    <Box component="ul" sx={{ mt: 1, mb: 0, pl: 2 }}>
                        <li>Object-specific icons (User, Team, Project, etc.)</li>
                        <li>Confirmation by typing the exact object name</li>
                        <li>Delete button disabled until name matches</li>
                        <li>Real-time progress bar and validation feedback</li>
                        <li>Success animation before closing</li>
                        <li>FormTip warning with proper icon</li>
                        <li>Professional gradient header design</li>
                    </Box>
                </Alert>

                {/* Dialog Component */}
                <DeleteDialog
                    isOpen={isOpen}
                    handleClose={handleClose}
                    handleDelete={handleDelete}
                    objectName={objectName}
                    objectIcon={icon}
                />
            </Stack>
        );
    },
};

// Example with simple project name
export const SimpleProject: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);
        const objectName = "Todo App";
        const projectIcon = { name: "Project", type: "Common" } as const;

        const handleClose = (wasDeleted: boolean) => {
            action("delete-closed")({ wasDeleted, objectName });
            setIsOpen(false);
        };

        const handleDelete = () => {
            action("delete-confirmed")({ objectName });
            setTimeout(() => handleClose(true), 300);
        };

        return (
            <>
                <Button onClick={() => setIsOpen(true)} variant="danger">
                    Delete Project: "{objectName}"
                </Button>
                <DeleteDialog
                    isOpen={isOpen}
                    handleClose={handleClose}
                    handleDelete={handleDelete}
                    objectName={objectName}
                    objectIcon={projectIcon}
                />
            </>
        );
    },
};

// Example with complex name (spaces, special chars)
export const ComplexName: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);
        const objectName = "My Super-Important Project (v2.1)";
        const routineIcon = { name: "Routine", type: "Routine" } as const;

        const handleClose = (wasDeleted: boolean) => {
            action("delete-closed")({ wasDeleted, objectName });
            setIsOpen(false);
        };

        const handleDelete = () => {
            action("delete-confirmed")({ objectName });
            setTimeout(() => handleClose(true), 300);
        };

        return (
            <Stack spacing={2}>
                <Alert severity="info">
                    <Typography variant="body2">
                        <strong>Complex Name:</strong> This example shows how the dialog handles names with spaces,
                        special characters, and parentheses with a Routine icon.
                    </Typography>
                </Alert>
                <Box sx={{ textAlign: "center" }}>
                    <Button onClick={() => setIsOpen(true)} variant="danger">
                        Delete Routine: "{objectName}"
                    </Button>
                </Box>
                <DeleteDialog
                    isOpen={isOpen}
                    handleClose={handleClose}
                    handleDelete={handleDelete}
                    objectName={objectName}
                    objectIcon={routineIcon}
                />
            </Stack>
        );
    },
};

// Example with very long name
export const LongName: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);
        const objectName = "This is a very long object name that might wrap to multiple lines in the dialog";
        const teamIcon = { name: "Team", type: "Common" } as const;

        const handleClose = (wasDeleted: boolean) => {
            action("delete-closed")({ wasDeleted, objectName });
            setIsOpen(false);
        };

        const handleDelete = () => {
            action("delete-confirmed")({ objectName });
            setTimeout(() => handleClose(true), 300);
        };

        return (
            <Stack spacing={2}>
                <Alert severity="info">
                    <Typography variant="body2">
                        <strong>Long Name:</strong> This example tests how the dialog handles very long object names
                        that might wrap to multiple lines with a Team icon.
                    </Typography>
                </Alert>
                <Box sx={{ textAlign: "center" }}>
                    <Button onClick={() => setIsOpen(true)} variant="danger">
                        Delete Team
                    </Button>
                </Box>
                <DeleteDialog
                    isOpen={isOpen}
                    handleClose={handleClose}
                    handleDelete={handleDelete}
                    objectName={objectName}
                    objectIcon={teamIcon}
                />
            </Stack>
        );
    },
};

// Example showing empty name edge case  
export const EmptyName: Story = {
    render: () => {
        const [isOpen, setIsOpen] = useState(false);
        const objectName = "";

        const handleClose = (wasDeleted: boolean) => {
            action("delete-closed")({ wasDeleted, objectName });
            setIsOpen(false);
        };

        const handleDelete = () => {
            action("delete-confirmed")({ objectName });
            setTimeout(() => handleClose(true), 300);
        };

        return (
            <Stack spacing={2}>
                <Alert severity="warning">
                    <Typography variant="body2">
                        <strong>Edge Case:</strong> This example shows what happens when the object name is empty.
                        The delete button should be enabled immediately since empty string matches empty string.
                    </Typography>
                </Alert>
                <Box sx={{ textAlign: "center" }}>
                    <Button onClick={() => setIsOpen(true)} variant="danger">
                        Delete Empty Name
                    </Button>
                </Box>
                <DeleteDialog
                    isOpen={isOpen}
                    handleClose={handleClose}
                    handleDelete={handleDelete}
                    objectName={objectName}
                />
            </Stack>
        );
    },
};

// Gallery of different object types
export const ObjectTypeGallery: Story = {
    render: () => {
        const [openDialogs, setOpenDialogs] = useState<Record<string, boolean>>({});
        
        const toggleDialog = (key: string, isOpen: boolean) => {
            setOpenDialogs(prev => ({ ...prev, [key]: isOpen }));
        };

        const objectTypes = [
            { key: "user", object: mockObjects.user, label: "Regular User" },
            { key: "botUser", object: mockObjects.botUser, label: "Bot User" },
            { key: "team", object: mockObjects.team, label: "Team" },
            { key: "project", object: mockObjects.project, label: "Project" },
            { key: "routine", object: mockObjects.routine, label: "Routine" },
            { key: "chat", object: mockObjects.chat, label: "Chat" },
            { key: "meeting", object: mockObjects.meeting, label: "Meeting" },
            { key: "bookmarkList", object: mockObjects.bookmarkList, label: "Bookmark List" },
            { key: "schedule", object: mockObjects.schedule, label: "Schedule" },
            { key: "reminder", object: mockObjects.reminder, label: "Reminder" },
        ];

        return (
            <Box sx={{
                display: "grid",
                gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))",
                gap: 2,
                maxWidth: 1200,
            }}>
                {objectTypes.map(({ key, object, label }) => {
                    const { title, icon } = getObjectDisplayInfo(object);
                    
                    return (
                        <Card key={key}>
                            <CardContent sx={{ textAlign: "center" }}>
                                <Typography variant="subtitle2" gutterBottom>
                                    {label}
                                </Typography>
                                <Typography variant="caption" color="text.secondary" display="block">
                                    {object.__typename}
                                </Typography>
                                <Typography variant="caption" color="text.secondary" display="block" fontStyle="italic" sx={{ mb: 2 }}>
                                    "{title}"
                                </Typography>
                                <Button 
                                    onClick={() => toggleDialog(key, true)} 
                                    variant="danger" 
                                    size="sm"
                                    fullWidth
                                >
                                    Delete
                                </Button>
                                
                                <DeleteDialog
                                    isOpen={!!openDialogs[key]}
                                    handleClose={(wasDeleted) => {
                                        action(`delete-${key}`)({ wasDeleted, title });
                                        toggleDialog(key, false);
                                    }}
                                    handleDelete={() => {
                                        action(`confirm-delete-${key}`)({ title });
                                        setTimeout(() => toggleDialog(key, false), 300);
                                    }}
                                    objectName={title}
                                    objectIcon={icon}
                                />
                            </CardContent>
                        </Card>
                    );
                })}
            </Box>
        );
    },
};
