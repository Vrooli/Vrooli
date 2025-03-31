import { Box, Button, Typography } from "@mui/material";
import { action } from "@storybook/addon-actions";
import { Meta } from "@storybook/react";
import userEvent from "@testing-library/user-event";
import { Form, Formik } from "formik";
import { useState } from "react";
import { ScrollBox } from "../../../styles.js";
import { PageContainer } from "../../Page/Page.js";
import { AdvancedInput, AdvancedInputBase, ContextItem, Tool, ToolState, TranslatedAdvancedInput } from "./AdvancedInput.js";

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
    gap: 1,
    marginBottom: 2,
} as const;

const sectionStyle = {
    display: "flex",
    alignItems: "center",
    gap: 2,
    marginBottom: 1,
} as const;

const buttonsContainerStyle = {
    display: "flex",
    gap: 1,
    flexWrap: "wrap",
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

const formikInitialValues = { message: "" } as const;
const translatedInitialValues = {
    translations: [
        {
            language: "en",
            message: "Hello world",
            description: "A test message",
        },
        {
            language: "es",
            message: "Hola mundo",
            description: "Un mensaje de prueba",
        },
    ],
} as const;

const MAX_DISPLAY_LENGTH = 250;
const TRUNCATED_LENGTH = MAX_DISPLAY_LENGTH - "...".length;
const valueDisplayStyle = { mb: 1 } as const;

function handleFormikSubmit(values: typeof formikInitialValues) {
    action("onSubmit")(values);
}

function handleTranslatedSubmit(values: typeof translatedInitialValues) {
    action("onSubmit")(values);
}

/** Displays the current value of the input, truncated if too long */
function ValueDisplay({ value }: { value: string }) {
    const displayValue = value.length > MAX_DISPLAY_LENGTH ? `${value.substring(0, TRUNCATED_LENGTH)}...` : value;
    return (
        <Typography variant="body2" color="text.secondary" sx={valueDisplayStyle}>
            Current Value ({value.length} chars): &ldquo;{displayValue}&rdquo;
        </Typography>
    );
}

interface SimulationButtonsProps {
    onPasteSingleLine: () => Promise<void>;
    onPasteMultiLine: () => Promise<void>;
    onSetSingleLine: () => void;
    onSetMultiLine: () => void;
    onImageDrop: () => Promise<void>;
    onFileDrop: () => Promise<void>;
    onSetNoTools: () => void;
    onSetSomeTools: () => void;
    onSetManyTools: () => void;
    onSetContextItems: () => void;
    onSetMaxChars: () => void;
    onClearAll: () => void;
}

function SimulationButtons({
    onPasteSingleLine,
    onPasteMultiLine,
    onSetSingleLine,
    onSetMultiLine,
    onImageDrop,
    onFileDrop,
    onSetNoTools,
    onSetSomeTools,
    onSetManyTools,
    onSetContextItems,
    onSetMaxChars,
    onClearAll,
}: SimulationButtonsProps) {
    return (
        <Box sx={simulationButtonsStyle}>
            <Box sx={sectionStyle}>
                <Typography variant="h6" color="textSecondary" minWidth={120}>Config</Typography>
                <Box sx={buttonsContainerStyle}>
                    <Button onClick={onSetNoTools} variant="outlined" size="small">
                        Basic Input
                    </Button>
                    <Button onClick={onSetSomeTools} variant="outlined" size="small">
                        Some Tools
                    </Button>
                    <Button onClick={onSetManyTools} variant="outlined" size="small">
                        Many Tools
                    </Button>
                    <Button onClick={onSetContextItems} variant="outlined" size="small">
                        Context Items
                    </Button>
                    <Button onClick={onSetMaxChars} variant="outlined" size="small">
                        Max Chars
                    </Button>
                </Box>
            </Box>

            <Box sx={sectionStyle}>
                <Typography variant="h6" color="textSecondary" minWidth={120}>Paste</Typography>
                <Box sx={buttonsContainerStyle}>
                    <Button onClick={onPasteSingleLine} variant="outlined" size="small">
                        Single Line
                    </Button>
                    <Button onClick={onPasteMultiLine} variant="outlined" size="small">
                        Multi Line
                    </Button>
                </Box>
            </Box>

            <Box sx={sectionStyle}>
                <Typography variant="h6" color="textSecondary" minWidth={120}>Set Value</Typography>
                <Box sx={buttonsContainerStyle}>
                    <Button onClick={onSetSingleLine} variant="outlined" size="small">
                        Single Line
                    </Button>
                    <Button onClick={onSetMultiLine} variant="outlined" size="small">
                        Multi Line
                    </Button>
                </Box>
            </Box>

            <Box sx={sectionStyle}>
                <Typography variant="h6" color="textSecondary" minWidth={120}>Context</Typography>
                <Box sx={buttonsContainerStyle}>
                    <Button onClick={onImageDrop} variant="outlined" size="small">
                        Drop Image
                    </Button>
                    <Button onClick={onFileDrop} variant="outlined" size="small">
                        Drop File
                    </Button>
                </Box>
            </Box>

            <Box sx={sectionStyle}>
                <Typography variant="h6" color="textSecondary" minWidth={120}>Actions</Typography>
                <Box sx={buttonsContainerStyle}>
                    <Button onClick={onClearAll} variant="outlined" color="error" size="small">
                        Clear All
                    </Button>
                </Box>
            </Box>
        </Box>
    );
}

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
    const [maxChars, setMaxChars] = useState<number | undefined>(undefined);

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
    }

    function onToolsChange(updated: Tool[]) {
        setTools(updated);
        action("onToolsChange")(updated);
    }

    // Simulation handlers
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
        onMessageChange(mockSingleLineText);
    }

    function simulateSetMultiLine() {
        onMessageChange(mockMultiLineText);
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

    // Configuration handlers
    function setNoTools() {
        setTools([]);
        setContextData([]);
        setMaxChars(undefined);
        action("setNoTools")();
    }

    function setSomeTools() {
        setTools(mockSomeTools);
        setContextData([]);
        setMaxChars(undefined);
        action("setSomeTools")();
    }

    function setManyTools() {
        setTools(mockManyTools);
        setContextData([]);
        setMaxChars(undefined);
        action("setManyTools")();
    }

    function setWithContextItems() {
        setTools([]);
        setContextData(mockContextData);
        setMaxChars(undefined);
        action("setWithContextItems")();
    }

    function setWithMaxChars() {
        setTools([]);
        setContextData([]);
        setMaxChars(100);
        action("setWithMaxChars")();
    }

    function clearAll() {
        onMessageChange("");
        setContextData([]);
        setTools([]);
        setMaxChars(undefined);
        action("clearAll")();
    }

    return (
        <>
            <SimulationButtons
                onPasteSingleLine={simulatePasteSingleLine}
                onPasteMultiLine={simulatePasteMultiLine}
                onSetSingleLine={simulateSetSingleLine}
                onSetMultiLine={simulateSetMultiLine}
                onImageDrop={simulateImageDrop}
                onFileDrop={simulateFileDrop}
                onSetNoTools={setNoTools}
                onSetSomeTools={setSomeTools}
                onSetManyTools={setManyTools}
                onSetContextItems={setWithContextItems}
                onSetMaxChars={setWithMaxChars}
                onClearAll={clearAll}
            />
            <ValueDisplay value={message} />
            <AdvancedInputBase
                tools={tools}
                contextData={contextData}
                maxChars={maxChars}
                onChange={onMessageChange}
                onToolsChange={onToolsChange}
                onContextDataChange={onContextDataChange}
                onSubmit={onSubmit}
                value={message}
            />
        </>
    );
}

/**
 * FormikExample: demonstrates usage with Formik form handling
 */
export function FormikExample() {
    return (
        <Formik
            initialValues={formikInitialValues}
            onSubmit={handleFormikSubmit}
        >
            {({ values, setFieldValue, setValues }) => {
                // Simulation handlers
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
                    setFieldValue("message", mockSingleLineText);
                    action("simulateSetSingleLine")(mockSingleLineText);
                }

                function simulateSetMultiLine() {
                    setFieldValue("message", mockMultiLineText);
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

                // Configuration handlers
                function setNoTools() {
                    setValues({ message: values.message });
                    action("setNoTools")();
                }

                function setSomeTools() {
                    setValues({ message: values.message });
                    action("setSomeTools")();
                }

                function setManyTools() {
                    setValues({ message: values.message });
                    action("setManyTools")();
                }

                function setWithContextItems() {
                    setValues({ message: values.message });
                    action("setWithContextItems")();
                }

                function setWithMaxChars() {
                    setValues({ message: values.message });
                    action("setWithMaxChars")();
                }

                function clearAll() {
                    setValues({ message: "" });
                    action("clearAll")();
                }

                return (
                    <Form>
                        <SimulationButtons
                            onPasteSingleLine={simulatePasteSingleLine}
                            onPasteMultiLine={simulatePasteMultiLine}
                            onSetSingleLine={simulateSetSingleLine}
                            onSetMultiLine={simulateSetMultiLine}
                            onImageDrop={simulateImageDrop}
                            onFileDrop={simulateFileDrop}
                            onSetNoTools={setNoTools}
                            onSetSomeTools={setSomeTools}
                            onSetManyTools={setManyTools}
                            onSetContextItems={setWithContextItems}
                            onSetMaxChars={setWithMaxChars}
                            onClearAll={clearAll}
                        />
                        <ValueDisplay value={values.message} />
                        <AdvancedInput
                            name="message"
                            tools={mockSomeTools}
                            contextData={mockContextData}
                        />
                    </Form>
                );
            }}
        </Formik>
    );
}

/**
 * TranslatedExample: demonstrates usage with translations
 */
export function TranslatedExample() {
    return (
        <Formik
            initialValues={translatedInitialValues}
            onSubmit={handleTranslatedSubmit}
        >
            {({ values, setFieldValue, setValues }) => {
                // Simulation handlers
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
                    setFieldValue("translations[0].message", mockSingleLineText);
                    setFieldValue("translations[1].message", mockSingleLineText);
                    action("simulateSetSingleLine")(mockSingleLineText);
                }

                function simulateSetMultiLine() {
                    setFieldValue("translations[0].message", mockMultiLineText);
                    setFieldValue("translations[1].message", mockMultiLineText);
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

                // Configuration handlers
                function setNoTools() {
                    setValues({ translations: values.translations });
                    action("setNoTools")();
                }

                function setSomeTools() {
                    setValues({ translations: values.translations });
                    action("setSomeTools")();
                }

                function setManyTools() {
                    setValues({ translations: values.translations });
                    action("setManyTools")();
                }

                function setWithContextItems() {
                    setValues({ translations: values.translations });
                    action("setWithContextItems")();
                }

                function setWithMaxChars() {
                    setValues({ translations: values.translations });
                    action("setWithMaxChars")();
                }

                function clearAll() {
                    setValues({
                        translations: [
                            { ...values.translations[0], message: "" },
                            { ...values.translations[1], message: "" },
                        ],
                    });
                    action("clearAll")();
                }

                return (
                    <Form>
                        <SimulationButtons
                            onPasteSingleLine={simulatePasteSingleLine}
                            onPasteMultiLine={simulatePasteMultiLine}
                            onSetSingleLine={simulateSetSingleLine}
                            onSetMultiLine={simulateSetMultiLine}
                            onImageDrop={simulateImageDrop}
                            onFileDrop={simulateFileDrop}
                            onSetNoTools={setNoTools}
                            onSetSomeTools={setSomeTools}
                            onSetManyTools={setManyTools}
                            onSetContextItems={setWithContextItems}
                            onSetMaxChars={setWithMaxChars}
                            onClearAll={clearAll}
                        />
                        <Box display="flex" flexDirection="column" gap={2}>
                            <Typography variant="h6" color="textSecondary">English</Typography>
                            <ValueDisplay value={values.translations[0].message} />
                            <TranslatedAdvancedInput
                                name="message"
                                language="en"
                                tools={mockSomeTools}
                                contextData={mockContextData}
                            />
                            <Typography variant="h6" color="textSecondary">Spanish</Typography>
                            <ValueDisplay value={values.translations[1].message} />
                            <TranslatedAdvancedInput
                                name="message"
                                language="es"
                                tools={mockSomeTools}
                                contextData={mockContextData}
                            />
                        </Box>
                    </Form>
                );
            }}
        </Formik>
    );
}
