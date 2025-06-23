import Box from "@mui/material/Box";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import type { Meta, StoryObj } from "@storybook/react";
import { DUMMY_ID, LlmTask } from "@vrooli/shared";
import React, { useState } from "react";
import { ViewDisplayType } from "../../../types.js";
import { Switch } from "../Switch/Switch.js";
import { ChatMessageInput } from "./ChatMessageInput.js";
import { chatContainerDecorator } from "../../../__test/helpers/storybookDecorators.tsx";

const meta: Meta<typeof ChatMessageInput> = {
    title: "Components/Chat/ChatMessageInput",
    component: ChatMessageInput,
    parameters: {
        layout: "centered",
    },
    tags: ["autodocs"],
    argTypes: {
        display: {
            control: { type: "select" },
            options: Object.values(ViewDisplayType),
        },
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Mock participants typing
const mockParticipantsTyping = [
    {
        __typename: "ChatParticipant" as const,
        id: "participant-1",
        user: {
            __typename: "User" as const,
            id: "user-1",
            name: "AI Assistant",
            handle: "ai-assistant",
        },
    },
];

const mockParticipantsTypingMultiple = [
    {
        __typename: "ChatParticipant" as const,
        id: "participant-1",
        user: {
            __typename: "User" as const,
            id: "user-1",
            name: "Alice",
            handle: "alice",
        },
    },
    {
        __typename: "ChatParticipant" as const,
        id: "participant-2",
        user: {
            __typename: "User" as const,
            id: "user-2",
            name: "Bob",
            handle: "bob",
        },
    },
];

// Mock message being replied to
const mockMessageBeingRepliedTo = {
    __typename: "ChatMessage" as const,
    id: DUMMY_ID,
    user: {
        __typename: "User" as const,
        id: "user-1",
        name: "AI Assistant",
        handle: "ai-assistant",
    },
    translations: [
        {
            __typename: "ChatMessageTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            text: "This is the message being replied to. It can be quite long and will be displayed in the reply preview above the input.",
        },
    ],
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    versionIndex: 0,
    parent: null,
    children: [],
};

// Mock message being edited
const mockMessageBeingEdited = {
    __typename: "ChatMessage" as const,
    id: DUMMY_ID,
    user: {
        __typename: "User" as const,
        id: "user-2",
        name: "Current User",
        handle: "currentuser",
    },
    translations: [
        {
            __typename: "ChatMessageTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            text: "This message is being edited",
        },
    ],
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    versionIndex: 0,
    parent: null,
    children: [],
};

// Mock task info with different task states
const mockTaskInfoDefault = {
    activeTask: {
        taskId: "task-1",
        task: LlmTask.Start,
        label: "Start Conversation",
        status: "Suggested" as const,
        lastUpdated: new Date(),
    },
    inactiveTasks: [],
    contexts: {
        "task-1": [],
    },
    selectTask: (task: any) => console.log("selectTask", task),
    unselectTask: () => console.log("unselectTask"),
};

const mockTaskInfoWithTasks = {
    activeTask: {
        taskId: "task-1",
        task: LlmTask.ApiAdd,
        label: "Add API Endpoint",
        status: "Running" as const,
        lastUpdated: new Date(),
    },
    inactiveTasks: [
        {
            taskId: "task-2",
            task: LlmTask.NoteAdd,
            label: "Create Note",
            status: "Suggested" as const,
            lastUpdated: new Date(),
        },
        {
            taskId: "task-3",
            task: LlmTask.RoutineFind,
            label: "Find Routine",
            status: "Completed" as const,
            resultLabel: "Found 3 routines",
            lastUpdated: new Date(),
        },
        {
            taskId: "task-4",
            task: LlmTask.ProjectUpdate,
            label: "Update Project Settings",
            status: "Failed" as const,
            lastUpdated: new Date(),
        },
    ],
    contexts: {
        "task-1": [
            {
                id: "context-1",
                label: "User Documentation",
            },
            {
                id: "context-2",
                label: "API Schema",
            },
        ],
        "task-2": [],
        "task-3": [],
        "task-4": [],
    },
    selectTask: (task: any) => console.log("selectTask", task),
    unselectTask: () => console.log("unselectTask"),
};

// Interactive showcase with controls
export const Showcase: Story = {
    render: () => {
        const [message, setMessage] = useState("");
        const [disabled, setDisabled] = useState(false);
        const [isLoading, setIsLoading] = useState(false);
        const [showTyping, setShowTyping] = useState(false);
        const [showReply, setShowReply] = useState(false);
        const [showEdit, setShowEdit] = useState(false);
        const [showTasks, setShowTasks] = useState(false);
        const [placeholder, setPlaceholder] = useState("What would you like to do?");

        const participantsTyping = showTyping ? mockParticipantsTyping : [];
        const taskInfo = showTasks ? mockTaskInfoWithTasks : mockTaskInfoDefault;

        return (
            <Box p={3} bgcolor="background.paper" minHeight="500px">
                <Typography variant="h5" gutterBottom>
                    ChatMessageInput Showcase
                </Typography>

                {/* Controls */}
                <Box mb={3} display="flex" flexDirection="column" gap={2} maxWidth={400}>
                    <TextField
                        label="Placeholder"
                        value={placeholder}
                        onChange={(e) => setPlaceholder(e.target.value)}
                        size="small"
                    />
                    <TextField
                        label="Message"
                        value={message}
                        onChange={(e) => setMessage(e.target.value)}
                        size="small"
                    />
                    <Box display="flex" flexWrap="wrap" gap={1}>
                        <Switch 
                            checked={disabled} 
                            onChange={(checked) => setDisabled(checked)} 
                            label="Disabled"
                            labelPosition="right"
                        />
                        <Switch 
                            checked={isLoading} 
                            onChange={(checked) => setIsLoading(checked)} 
                            label="Loading"
                            labelPosition="right"
                        />
                        <Switch 
                            checked={showTyping} 
                            onChange={(checked) => setShowTyping(checked)} 
                            label="Show Typing"
                            labelPosition="right"
                        />
                        <Switch 
                            checked={showReply} 
                            onChange={(checked) => setShowReply(checked)} 
                            label="Reply Mode"
                            labelPosition="right"
                        />
                        <Switch 
                            checked={showEdit} 
                            onChange={(checked) => setShowEdit(checked)} 
                            label="Edit Mode"
                            labelPosition="right"
                        />
                        <Switch 
                            checked={showTasks} 
                            onChange={(checked) => setShowTasks(checked)} 
                            label="Show Tasks"
                            labelPosition="right"
                        />
                    </Box>
                </Box>

                {/* Component */}
                <Box bgcolor="background.default" border="1px solid" borderColor="divider" borderRadius={1}>
                    <ChatMessageInput
                        disabled={disabled}
                        display={ViewDisplayType.Page}
                        isLoading={isLoading}
                        message={message}
                        messageBeingEdited={showEdit ? mockMessageBeingEdited : null}
                        messageBeingRepliedTo={showReply ? mockMessageBeingRepliedTo : null}
                        participantsTyping={participantsTyping}
                        placeholder={placeholder}
                        setMessage={setMessage}
                        stopEditingMessage={() => console.log("stopEditingMessage")}
                        stopReplyingToMessage={() => console.log("stopReplyingToMessage")}
                        submitMessage={(msg) => console.log("submitMessage", msg)}
                        taskInfo={taskInfo}
                    />
                </Box>
            </Box>
        );
    },
};

// Specific focused scenarios
export const MultipleParticipantsTyping: Story = {
    args: {
        disabled: false,
        display: ViewDisplayType.Page,
        isLoading: false,
        message: "",
        messageBeingEdited: null,
        messageBeingRepliedTo: null,
        participantsTyping: mockParticipantsTypingMultiple,
        placeholder: "Alice and Bob are typing...",
        setMessage: (message: string) => console.log("setMessage", message),
        stopEditingMessage: () => console.log("stopEditingMessage"),
        stopReplyingToMessage: () => console.log("stopReplyingToMessage"),
        submitMessage: (message: string) => console.log("submitMessage", message),
        taskInfo: mockTaskInfoDefault,
    },
};

export const ReplyWithLongMessage: Story = {
    args: {
        disabled: false,
        display: ViewDisplayType.Page,
        isLoading: false,
        message: "Sure, I can help with that! Let me break down the implementation into several steps...",
        messageBeingEdited: null,
        messageBeingRepliedTo: {
            ...mockMessageBeingRepliedTo,
            translations: [
                {
                    __typename: "ChatMessageTranslation" as const,
                    id: DUMMY_ID,
                    language: "en",
                    text: "I need help implementing a complex feature that involves multiple components, database changes, API endpoints, and frontend updates. The feature should handle user authentication, data validation, real-time updates, and error handling. Can you help me plan this out step by step?",
                },
            ],
        },
        participantsTyping: [],
        placeholder: "Reply to message...",
        setMessage: (message: string) => console.log("setMessage", message),
        stopEditingMessage: () => console.log("stopEditingMessage"),
        stopReplyingToMessage: () => console.log("stopReplyingToMessage"),
        submitMessage: (message: string) => console.log("submitMessage", message),
        taskInfo: mockTaskInfoDefault,
    },
};

export const ActiveTasksWithContext: Story = {
    args: {
        disabled: false,
        display: ViewDisplayType.Page,
        isLoading: false,
        message: "I want to add error handling to the API endpoint and update the documentation",
        messageBeingEdited: null,
        messageBeingRepliedTo: null,
        participantsTyping: [],
        placeholder: "Describe what you want to accomplish...",
        setMessage: (message: string) => console.log("setMessage", message),
        stopEditingMessage: () => console.log("stopEditingMessage"),
        stopReplyingToMessage: () => console.log("stopReplyingToMessage"),
        submitMessage: (message: string) => console.log("submitMessage", message),
        taskInfo: mockTaskInfoWithTasks,
    },
};

export const MobileDialog: Story = {
    args: {
        disabled: false,
        display: ViewDisplayType.Dialog,
        isLoading: false,
        message: "",
        messageBeingEdited: null,
        messageBeingRepliedTo: null,
        participantsTyping: [],
        placeholder: "Type your message...",
        setMessage: (message: string) => console.log("setMessage", message),
        stopEditingMessage: () => console.log("stopEditingMessage"),
        stopReplyingToMessage: () => console.log("stopReplyingToMessage"),
        submitMessage: (message: string) => console.log("submitMessage", message),
        taskInfo: mockTaskInfoDefault,
    },
    decorators: [chatContainerDecorator(350, 400)],
};

export const LoadingWithTypingIndicator: Story = {
    args: {
        disabled: false,
        display: ViewDisplayType.Page,
        isLoading: true,
        message: "Help me create a new project with React and TypeScript",
        messageBeingEdited: null,
        messageBeingRepliedTo: null,
        participantsTyping: mockParticipantsTyping,
        placeholder: "AI is thinking...",
        setMessage: (message: string) => console.log("setMessage", message),
        stopEditingMessage: () => console.log("stopEditingMessage"),
        stopReplyingToMessage: () => console.log("stopReplyingToMessage"),
        submitMessage: (message: string) => console.log("submitMessage", message),
        taskInfo: mockTaskInfoWithTasks,
    },
};
