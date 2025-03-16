import { Box } from "@mui/material";
import { action } from "@storybook/addon-actions";
import { Meta } from "@storybook/react";
import { useState } from "react";
import { SearchIcon, UploadIcon } from "../../../icons/common.js";
import { ScrollBox } from "../../../styles.js";
import { PageContainer } from "../../Page/Page.js";
import { AdvancedInput, ContextItem, Tool } from "./AdvancedInput.js";

const outerBoxStyle = {
    display: "flex",
    flexDirection: "column",
    justifyContent: "flex-end",
    minHeight: "100vh",
} as const;
const innerBoxStyle = {
    maxWidth: "800px",
    padding: "20px",
    border: "1px solid #ccc",
    paddingBottom: "100px",
} as const;

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
            <PageContainer>
                <ScrollBox>
                    <Box sx={outerBoxStyle}>
                        <Box sx={innerBoxStyle}>
                            <Story />
                        </Box>
                    </Box>
                </ScrollBox>
            </PageContainer>
        ),
    ],
} satisfies Meta<typeof AdvancedInput>;

/**
 * Default story: minimal usage, no tools or context items.
 */
export function Default() {
    const [contextData, setContextData] = useState<ContextItem[]>([]);
    const [message, setMessage] = useState("");
    const [tools, setTools] = useState<Tool[]>([]);

    function onContextDataChange(updated: ContextItem[]) {
        setContextData(updated);
    }
    function onSubmit(msg: string) {
        action("onSubmit")(msg);
        setMessage("");
    }
    function onToolsChange(updated: Tool[]) {
        setTools(updated);
    }

    return (
        <AdvancedInput
            enterWillSubmit={true}
            tools={tools}
            contextData={contextData}
            onToolsChange={onToolsChange}
            onContextDataChange={onContextDataChange}
            onSubmit={onSubmit}
        />
    );
}

export function WithSomeTools() {
    const [contextData, setContextData] = useState<ContextItem[]>([]);
    const [message, setMessage] = useState("");
    const [tools, setTools] = useState<Tool[]>(mockSomeTools);

    function onContextDataChange(updated: ContextItem[]) {
        setContextData(updated);
    }
    function onSubmit(msg: string) {
        action("onSubmit")(msg);
        setMessage("");
    }
    function onToolsChange(updated: Tool[]) {
        setTools(updated);
    }

    return (
        <AdvancedInput
            enterWillSubmit={true}
            tools={tools}
            contextData={contextData}
            onToolsChange={onToolsChange}
            onContextDataChange={onContextDataChange}
            onSubmit={onSubmit}
        />
    );
}

export function WithManyTools() {
    const [contextData, setContextData] = useState<ContextItem[]>([]);
    const [message, setMessage] = useState("");
    const [tools, setTools] = useState<Tool[]>(mockManyTools);

    function onContextDataChange(updated: ContextItem[]) {
        setContextData(updated);
    }
    function onSubmit(msg: string) {
        action("onSubmit")(msg);
        setMessage("");
    }
    function onToolsChange(updated: Tool[]) {
        setTools(updated);
    }

    return (
        <AdvancedInput
            enterWillSubmit={true}
            tools={tools}
            contextData={contextData}
            onToolsChange={onToolsChange}
            onContextDataChange={onContextDataChange}
            onSubmit={onSubmit}
        />
    );
}

/**
 * WithContextItems: demonstrates attaching files/images/text as context data.
 */
export function WithContextItems() {
    const [contextData, setContextData] = useState<ContextItem[]>(mockContextData);
    const [message, setMessage] = useState("");
    const [tools, setTools] = useState<Tool[]>([]);

    function onContextDataChange(updated: ContextItem[]) {
        setContextData(updated);
    }
    function onSubmit(msg: string) {
        action("onSubmit")(msg);
        setMessage("");
    }
    function onToolsChange(updated: Tool[]) {
        setTools(updated);
    }

    return (
        <AdvancedInput
            enterWillSubmit={true}
            tools={tools}
            contextData={contextData}
            onToolsChange={onToolsChange}
            onContextDataChange={onContextDataChange}
            onSubmit={onSubmit}
        />
    );
}

/**
 * WithEnterKeyMode: toggles whether enter key submits or adds new lines.
 */
export function WithEnterKeyMode() {
    const [contextData, setContextData] = useState<ContextItem[]>([]);
    const [enterWillSubmit, setEnterWillSubmit] = useState(true);
    const [message, setMessage] = useState("");
    const [tools, setTools] = useState<Tool[]>([]);

    function onContextDataChange(updated: ContextItem[]) {
        setContextData(updated);
    }
    function onSubmit(msg: string) {
        action("onSubmit")(msg);
        setMessage("");
    }
    function onToolsChange(updated: Tool[]) {
        setTools(updated);
    }

    return (
        <AdvancedInput
            enterWillSubmit={enterWillSubmit}
            tools={tools}
            contextData={contextData}
            onToolsChange={onToolsChange}
            onContextDataChange={onContextDataChange}
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
    const [contextData, setContextData] = useState<ContextItem[]>([]);
    const [message, setMessage] = useState("");
    const [tools, setTools] = useState<Tool[]>([]);

    function onContextDataChange(updated: ContextItem[]) {
        setContextData(updated);
    }
    function onSubmit(msg: string) {
        action("onSubmit")(msg);
        setMessage("");
    }
    function onToolsChange(updated: Tool[]) {
        setTools(updated);
    }

    return (
        <AdvancedInput
            enterWillSubmit={true}
            tools={tools}
            contextData={contextData}
            // isWysiwyg={true}
            onToolsChange={onToolsChange}
            onContextDataChange={onContextDataChange}
            onSubmit={onSubmit}
        // showToolbar={true}
        />
    );
}

