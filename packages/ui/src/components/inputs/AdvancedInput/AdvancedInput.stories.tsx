import { noop } from "@local/shared";
import { Box } from "@mui/material";
import { action } from "@storybook/addon-actions";
import { Meta } from "@storybook/react";
import React, { useState } from "react";
import { SearchIcon, UploadIcon } from "../../../icons/common.js";
import { AdvancedInput, ContextItem, Tool } from "./AdvancedInput.js";

const outerBoxStyle = {
    maxWidth: "800px",
    padding: "20px",
    border: "1px solid #ccc",
    margin: "auto",
} as const;
function Outer({ children }: { children: React.ReactNode }) {
    return (
        <Box
            sx={outerBoxStyle}
        >
            {children}
        </Box>
    );
}

const emptyArray = [];

const mockSomeTools: Tool[] = [
    {
        enabled: true,
        displayName: "Find routine",
        icon: <SearchIcon />,
        type: "find",
        name: "findRoutine",
        arguments: {},
    },
    {
        enabled: false,
        displayName: "Publish inventory",
        icon: <UploadIcon />,
        type: "routine",
        name: "publishInventory",
        arguments: {},
    },
];
const mockManyTools: Tool[] = [
    ...mockSomeTools,
    ...mockSomeTools,
    ...mockSomeTools,
    ...mockSomeTools,
    ...mockSomeTools,
];

const mockContextData: ContextItem[] = [
    {
        id: "ctx-1",
        type: "text",
        label: "Current Form",
    },
    {
        id: "ctx-2",
        type: "file",
        label: "Budget.xlsx",
        // `src` is unused for file type, but included for consistency
    },
    {
        id: "ctx-3",
        type: "image",
        label: "Sample Image",
        src: "https://via.placeholder.com/80",
    },
];

// Default export for Storybook
export default {
    title: "Components/Inputs/AdvancedInput",
    component: AdvancedInput,
    decorators: [
        (Story) => (
            <Outer>
                <Story />
            </Outer>
        ),
    ],
} satisfies Meta<typeof AdvancedInput>;

/**
 * Default story: minimal usage, no tools or context items.
 */
export function Default() {
    const [message, setMessage] = useState("");

    function onSubmit(msg: string) {
        action("onSubmit")(msg);
        setMessage("");
    }

    return (
        <AdvancedInput
            enterWillSubmit={true}
            tools={emptyArray}
            contextData={emptyArray}
            onToolsChange={noop}
            onContextDataChange={noop}
            onSubmit={onSubmit}
        />
    );
}

export function WithSomeTools() {
    const [message, setMessage] = useState("");
    const [tools, setTools] = useState<Tool[]>(mockSomeTools);

    function onSubmit(msg: string) {
        action("onSubmit")(msg);
        setMessage("");
    }

    return (
        <AdvancedInput
            enterWillSubmit={true}
            tools={tools}
            contextData={emptyArray}
            onToolsChange={(updated) => setTools(updated)}
            onContextDataChange={noop}
            onSubmit={onSubmit}
        />
    );
}

export function WithManyTools() {
    const [message, setMessage] = useState("");
    const [tools, setTools] = useState<Tool[]>(mockManyTools);

    function onSubmit(msg: string) {
        action("onSubmit")(msg);
        setMessage("");
    }

    return (
        <AdvancedInput
            enterWillSubmit={true}
            tools={tools}
            contextData={emptyArray}
            onToolsChange={noop}
            onContextDataChange={noop}
            onSubmit={onSubmit}
        />
    );
}

/**
 * WithContextItems: demonstrates attaching files/images/text as context data.
 */
export function WithContextItems() {
    const [message, setMessage] = useState("");
    const [contextData, setContextData] = useState<ContextItem[]>(mockContextData);

    function onSubmit(msg: string) {
        action("onSubmit")(msg);
        setMessage("");
    }

    return (
        <AdvancedInput
            enterWillSubmit={true}
            tools={emptyArray}
            contextData={contextData}
            onToolsChange={noop}
            onContextDataChange={(updated) => setContextData(updated)}
            onSubmit={onSubmit}
        />
    );
}

/**
 * WithEnterKeyMode: toggles whether enter key submits or adds new lines.
 */
export function WithEnterKeyMode() {
    const [message, setMessage] = useState("");
    const [enterWillSubmit, setEnterWillSubmit] = useState(true);

    function onSubmit(msg: string) {
        action("onSubmit")(msg);
        setMessage("");
    }

    return (
        <AdvancedInput
            enterWillSubmit={enterWillSubmit}
            tools={emptyArray}
            contextData={emptyArray}
            onToolsChange={noop}
            onContextDataChange={noop}
            onSubmit={onSubmit}
        />
    );
}

/**
 * WithWysiwygAndToolbar: demonstrates toggling the WYSIWYG editor and toolbar
 * via the gear icon in the top left. (No direct props needed for demonstration;
 * toggling happens in the settings popover.)
 */
export function WithWysiwygAndToolbar() {
    const [message, setMessage] = useState("");

    function onSubmit(msg: string) {
        action("onSubmit")(msg);
        setMessage("");
    }

    return (
        <AdvancedInput
            enterWillSubmit={true}
            tools={emptyArray}
            contextData={emptyArray}
            onToolsChange={noop}
            onContextDataChange={noop}
            onSubmit={onSubmit}
        />
    );
}
