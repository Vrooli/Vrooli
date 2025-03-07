import { action } from "@storybook/addon-actions";
import { useMemo, useState } from "react";
import { ActiveChatContext, SessionContext } from "../../../contexts.js";
import { BoldIcon, ItalicIcon } from "../../../icons/common.js";
import { RichInputBase } from "./RichInput.js";

// Mock contexts
const mockSession = { user: { id: "mock-user-id" } };
const mockChat = { id: "mock-chat-id" };

// Mock taskInfo for contexts
const mockTaskInfo = {
    contexts: {
        "task-1": [
            { id: "context-1", label: "Context 1" },
            { id: "context-2", label: "Context 2" },
        ],
    },
    activeTask: { taskId: "task-1" },
};

const outerStyle = {
    maxWidth: "800px",
    padding: "20px",
    border: "1px solid #ccc",
} as const;

function Outer({ children }: { children: React.ReactNode }) {
    return (
        <div style={outerStyle}>
            {children}
        </div>
    );
}

/**
 * Storybook configuration for RichInputBase
 */
export default {
    title: "Components/Inputs/RichInputBase",
    component: RichInputBase,
    decorators: [
        (Story) => (
            <Outer>
                <SessionContext.Provider value={mockSession}>
                    <ActiveChatContext.Provider value={{ chat: mockChat }}>
                        <Story />
                    </ActiveChatContext.Provider>
                </SessionContext.Provider>
            </Outer>
        ),
    ],
};

/**
 * Default story: Basic interactive input
 */
export function Default() {
    const [value, setValue] = useState("");
    return (
        <RichInputBase
            name="default"
            value={value}
            onChange={setValue}
            placeholder="Type something..."
        />
    );
}

/**
 * With Action Buttons: Demonstrates custom action buttons
 */
export function WithActionButtons() {
    const [value, setValue] = useState("");
    const actionButtons = useMemo(() => [
        {
            Icon: BoldIcon,
            onClick: action("bold-clicked"),
            tooltip: "Bold",
        },
        {
            Icon: ItalicIcon,
            onClick: action("italic-clicked"),
            tooltip: "Italic",
        },
    ], []);

    return (
        <RichInputBase
            name="with-action-buttons"
            value={value}
            onChange={setValue}
            placeholder="Type something..."
            actionButtons={actionButtons}
        />
    );
}

/**
 * With Error: Shows error state and helper text
 */
export function WithError() {
    const [value, setValue] = useState("");
    return (
        <RichInputBase
            name="with-error"
            value={value}
            onChange={setValue}
            placeholder="Type something..."
            error={true}
            helperText="This is an error message"
        />
    );
}

/**
 * With Character Limit: Displays character limit indicator
 */
export function WithCharacterLimit() {
    const [value, setValue] = useState("");
    return (
        <RichInputBase
            name="with-character-limit"
            value={value}
            onChange={setValue}
            placeholder="Type something..."
            maxChars={100}
        />
    );
}

/**
 * Disabled: Shows the component in a disabled state
 */
export function Disabled() {
    const [value, setValue] = useState("Some text");
    return (
        <RichInputBase
            name="disabled"
            value={value}
            onChange={setValue}
            placeholder="Type something..."
            disabled={true}
        />
    );
}

/**
 * With Contexts: Demonstrates the contexts row with taskInfo
 */
export function WithContexts() {
    const [value, setValue] = useState("");
    return (
        <RichInputBase
            name="with-contexts"
            value={value}
            onChange={setValue}
            placeholder="Type something..."
            taskInfo={mockTaskInfo}
        />
    );
}
