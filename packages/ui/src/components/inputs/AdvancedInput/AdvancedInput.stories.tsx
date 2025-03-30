import { Box, Button, Typography } from "@mui/material";
import { action } from "@storybook/addon-actions";
import { Meta } from "@storybook/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { ScrollBox } from "../../../styles.js";
import { PageContainer } from "../../Page/Page.js";
import { AdvancedInput, ContextItem, Tool, ToolState } from "./AdvancedInput.js";

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

const simulationButtonsStyle = {
    display: "flex",
    flexDirection: "column",
    gap: 2,
    marginBottom: 2,
} as const;

const mockSingleLineText = "This is a single line of text that simulates pasting content.";
const mockMultiLineText = `First line of text
Second line that shows multiple lines
Third line to demonstrate scrolling
Fourth line of content
Fifth line to make it longer`;

const mockSomeTools: Tool[] = [
    {
        state: ToolState.Enabled,
        displayName: "Find routine",
        iconInfo: { name: "Search", type: "Common" },
        type: "find",
        name: "findRoutine",
        arguments: {},
    },
    {
        state: ToolState.Disabled,
        displayName: "Publish inventory",
        iconInfo: { name: "Upload", type: "Common" },
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
 * Default story: demonstrates various input scenarios with simulation buttons.
 */
export function Default() {
    const [contextData, setContextData] = useState<ContextItem[]>([]);
    const [message, setMessage] = useState("");
    const [tools, setTools] = useState<Tool[]>([]);

    function onContextDataChange(updated: ContextItem[]) {
        setContextData(updated);
        action("onContextDataChange")(updated);
    }

    function onMessageChange(newMessage: string) {
        setMessage(newMessage);
        action("onMessageChange")(newMessage);
    }

    function onSubmit(msg: string) {
        action("onSubmit")(msg);
        setMessage("");
    }

    function onToolsChange(updated: Tool[]) {
        setTools(updated);
        action("onToolsChange")(updated);
    }

    async function simulatePasteSingleLine() {
        const input = document.querySelector(".advanced-input-field");
        if (!(input instanceof HTMLTextAreaElement)) {
            console.error("Could not find textarea element", input);
            return;
        }
        input.focus();
        await userEvent.paste(mockSingleLineText);
        action("simulatePasteSingleLine")(mockSingleLineText);
    }

    async function simulatePasteMultiLine() {
        const input = document.querySelector(".advanced-input-field");
        if (!(input instanceof HTMLTextAreaElement)) {
            console.error("Could not find textarea element", input);
            return;
        }
        input.focus();
        await userEvent.paste(mockMultiLineText);
    }

    function simulateSetSingleLine() {
        setMessage(mockSingleLineText);
        action("simulateSetSingleLine")(mockSingleLineText);
    }

    function simulateSetMultiLine() {
        setMessage(mockMultiLineText);
        action("simulateSetMultiLine")(mockMultiLineText);
    }

    async function simulateImageDrop() {
        const input = document.querySelector(".advanced-input-field");
        if (!(input instanceof HTMLTextAreaElement)) {
            console.error("Could not find textarea element", input);
            return;
        }

        const file = new File(["mock image content"], "image.png", { type: "image/png" });
        const dataTransfer = new DataTransfer();
        dataTransfer.items.add(file);

        const dragOverEvent = new DragEvent("dragover", {
            bubbles: true,
            cancelable: true,
            dataTransfer,
        });
        input.dispatchEvent(dragOverEvent);

        const dropEvent = new DragEvent("drop", {
            bubbles: true,
            cancelable: true,
            dataTransfer,
        });
        input.dispatchEvent(dropEvent);

        action("simulateImageDrop")(file);
    }

    async function simulateFileDrop() {
        const input = document.querySelector(".advanced-input-field");
        if (!(input instanceof HTMLTextAreaElement)) {
            console.error("Could not find textarea element", input);
            return;
        }

        const file = new File(["mock file content"], "file.txt", { type: "text/plain" });
        const dataTransfer = new DataTransfer();
        dataTransfer.items.add(file);

        const dragOverEvent = new DragEvent("dragover", {
            bubbles: true,
            cancelable: true,
            dataTransfer,
        });
        input.dispatchEvent(dragOverEvent);

        const dropEvent = new DragEvent("drop", {
            bubbles: true,
            cancelable: true,
            dataTransfer,
        });
        input.dispatchEvent(dropEvent);

        action("simulateFileDrop")(file);
    }

    function clearAll() {
        setMessage("");
        setContextData([]);
        action("clearAll")();
    }

    return (
        <>
            <Box sx={simulationButtonsStyle}>
                <Typography variant="h6" color="textSecondary">Paste Simulation</Typography>
                <Box display="flex" flexDirection="row" gap={2}>
                    <Button onClick={simulatePasteSingleLine} variant="outlined">
                        Paste Single Line
                    </Button>
                    <Button onClick={simulatePasteMultiLine} variant="outlined">
                        Paste Multi Line
                    </Button>
                </Box>

                <Typography variant="h6" color="textSecondary">Direct Message Changes</Typography>
                <Box display="flex" flexDirection="row" gap={2}>
                    <Button onClick={simulateSetSingleLine} variant="outlined">
                        Set Single Line
                    </Button>
                    <Button onClick={simulateSetMultiLine} variant="outlined">
                        Set Multi Line
                    </Button>
                </Box>

                <Typography variant="h6" color="textSecondary">Context Changes</Typography>
                <Box display="flex" flexDirection="row" gap={2}>
                    <Button onClick={simulateImageDrop} variant="outlined">
                        Drop Image
                    </Button>
                    <Button onClick={simulateFileDrop} variant="outlined">
                        Drop File
                    </Button>
                </Box>

                <Typography variant="h6" color="textSecondary">
                    Clear All
                </Typography>
                <Box display="flex" flexDirection="row" gap={2}>
                    <Button onClick={clearAll} variant="outlined" color="error">
                        Clear All
                    </Button>
                </Box>
            </Box>
            <AdvancedInput
                tools={tools}
                contextData={contextData}
                message={message}
                onMessageChange={onMessageChange}
                onToolsChange={onToolsChange}
                onContextDataChange={onContextDataChange}
                onSubmit={onSubmit}
            />
        </>
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
            tools={tools}
            contextData={contextData}
            message={message}
            onMessageChange={setMessage}
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
            tools={tools}
            contextData={contextData}
            onToolsChange={onToolsChange}
            onContextDataChange={onContextDataChange}
            onSubmit={onSubmit}
        />
    );
}

export function WithMaxChars() {
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
            tools={tools}
            contextData={contextData}
            maxChars={100}
            onToolsChange={onToolsChange}
            onContextDataChange={onContextDataChange}
            onSubmit={onSubmit}
        />
    );
}
