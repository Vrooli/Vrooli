import { Box, Button, Divider, FormControlLabel, Switch, Typography } from "@mui/material";
import { action } from "@storybook/addon-actions";
import { Meta } from "@storybook/react";
import userEvent from "@testing-library/user-event";
import { Form, Formik } from "formik";
import { useState } from "react";
import { ScrollBox } from "../../../styles.js";
import { PageContainer } from "../../Page/Page.js";
import { AdvancedInput, AdvancedInputBase, TranslatedAdvancedInput } from "./AdvancedInput.js";
import { AdvancedInputFeatures, ContextItem, DEFAULT_FEATURES, Tool, ToolState, advancedInputTextareaClassName } from "./utils.js";

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

// Helper text style used in various places
const helperTextStyle = { mt: 1 } as const;

// Minimal props story styles
const minimalTitleStyle = { mb: 2 } as const;
const minimalDescriptionStyle = { mb: 2 } as const;

// Feature toggle UI styles
const featureBoxStyle = {
    display: "flex",
    flexDirection: "column" as const,
    gap: 1,
    mb: 4,
};

const featureControlsStyle = {
    display: "flex",
    flexWrap: "wrap" as const,
    gap: 1,
    mb: 2,
};

const featureToggleStyle = {
    display: "flex",
    flexWrap: "wrap" as const,
    gap: 1,
    mb: 3,
};

const featureButtonStyle = {
    minWidth: "120px",
};

const featureGroupStyle = {
    p: 2,
    mb: 2,
    border: "1px solid",
    borderColor: "divider",
    borderRadius: 1,
};

const featureGroupTitleStyle = {
    mb: 1,
    fontWeight: "bold",
};

const columnFlexStyle = {
    display: "flex",
    flexDirection: "column",
    gap: 1,
};

const dividerStyle = { my: 2 };

const preCodeStyle = {
    mt: 1,
    p: 2,
    backgroundColor: "background.paper",
    border: "1px solid",
    borderColor: "divider",
    borderRadius: 1,
    overflow: "auto",
    maxHeight: "300px",
    fontSize: "0.75rem",
};

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

const formikInitialValues = { message: "" };
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
};

const MAX_DISPLAY_LENGTH = 250;
const TRUNCATED_LENGTH = MAX_DISPLAY_LENGTH - "...".length;
const valueDisplayStyle = { mb: 1 } as const;

function handleFormikSubmit(values: typeof formikInitialValues) {
    action("onSubmit")(values);
    return new Promise((resolve) => {
        setTimeout(() => {
            resolve(formikInitialValues);
        }, 0);
    });
}

function handleTranslatedSubmit(values: typeof translatedInitialValues) {
    action("onSubmit")(values);
    return new Promise((resolve) => {
        setTimeout(() => {
            resolve(translatedInitialValues);
        }, 0);
    });
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
    onTogglePlaceholder: () => void;
    onToggleHelperText: () => void;
    onClearAll: () => void;
    hasPlaceholder: boolean;
    hasHelperText: boolean;
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
    onTogglePlaceholder,
    onToggleHelperText,
    onClearAll,
    hasPlaceholder,
    hasHelperText,
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
                    <Button
                        onClick={onTogglePlaceholder}
                        variant={hasPlaceholder ? "contained" : "outlined"}
                        size="small"
                    >
                        Placeholder
                    </Button>
                    <Button
                        onClick={onToggleHelperText}
                        variant={hasHelperText ? "contained" : "outlined"}
                        size="small"
                    >
                        Helper Text
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
    const [placeholder, setPlaceholder] = useState<string | undefined>("Enter your message here...");
    const [helperText, setHelperText] = useState<string | undefined>(undefined);

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

    // Simulation handlers
    async function simulatePasteSingleLine() {
        const input = document.querySelector(`.${advancedInputTextareaClassName}`);
        if (!(input instanceof HTMLTextAreaElement)) {
            console.error("Could not find textarea element", input);
            return;
        }
        input.focus();
        await userEvent.paste(mockSingleLineText);
        action("simulatePasteSingleLine")(mockSingleLineText);
    }

    async function simulatePasteMultiLine() {
        const input = document.querySelector(`.${advancedInputTextareaClassName}`);
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
        const input = document.querySelector(`.${advancedInputTextareaClassName}`);
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
        const input = document.querySelector(`.${advancedInputTextareaClassName}`);
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

    function togglePlaceholder() {
        setPlaceholder(placeholder ? undefined : "Enter your message here...");
        action("togglePlaceholder")(!!placeholder);
    }

    function toggleHelperText() {
        setHelperText(helperText ? undefined : "This is a helpful message below the input");
        action("toggleHelperText")(!!helperText);
    }

    function clearAll() {
        onMessageChange("");
        setContextData([]);
        setTools([]);
        setMaxChars(undefined);
        setPlaceholder("Enter your message here...");
        setHelperText(undefined);
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
                onTogglePlaceholder={togglePlaceholder}
                onToggleHelperText={toggleHelperText}
                onClearAll={clearAll}
                hasPlaceholder={!!placeholder}
                hasHelperText={!!helperText}
            />
            <ValueDisplay value={message} />
            <AdvancedInputBase
                name="message"
                tools={tools}
                contextData={contextData}
                maxChars={maxChars}
                placeholder={placeholder}
                helperText={helperText}
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
    const [placeholder, setPlaceholder] = useState<string | undefined>("Enter your message here...");
    const [helperText, setHelperText] = useState<string | undefined>(undefined);
    const [title, setTitle] = useState<string | undefined>("Feedback Form");
    const [features] = useState<AdvancedInputFeatures>({
        ...DEFAULT_FEATURES,
        allowTools: false,
    });

    return (
        <Formik
            initialValues={formikInitialValues}
            onSubmit={handleFormikSubmit}
        >
            {({ values, setFieldValue, setValues }) => {
                // Simulation handlers
                async function simulatePasteSingleLine() {
                    const input = document.querySelector(`.${advancedInputTextareaClassName}`);
                    if (!(input instanceof HTMLTextAreaElement)) {
                        console.error("Could not find textarea element", input);
                        return;
                    }
                    input.focus();
                    await userEvent.paste(mockSingleLineText);
                    action("simulatePasteSingleLine")(mockSingleLineText);
                }

                async function simulatePasteMultiLine() {
                    const input = document.querySelector(`.${advancedInputTextareaClassName}`);
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
                    const input = document.querySelector(`.${advancedInputTextareaClassName}`);
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
                    const input = document.querySelector(`.${advancedInputTextareaClassName}`);
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

                function togglePlaceholder() {
                    setPlaceholder(placeholder ? undefined : "Enter your message here...");
                    action("togglePlaceholder")(!!placeholder);
                }

                function toggleHelperText() {
                    setHelperText(helperText ? undefined : "This is a helpful message below the input");
                    action("toggleHelperText")(!!helperText);
                }

                function toggleTitle() {
                    setTitle(title ? undefined : "Feedback Form");
                    action("toggleTitle")(!!title);
                }

                function clearAll() {
                    setValues({ message: "" });
                    setPlaceholder("Enter your message here...");
                    setHelperText(undefined);
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
                            onTogglePlaceholder={togglePlaceholder}
                            onToggleHelperText={toggleHelperText}
                            onClearAll={clearAll}
                            hasPlaceholder={!!placeholder}
                            hasHelperText={!!helperText}
                        />
                        <Box mb={2}>
                            <Button variant="outlined" onClick={toggleTitle} sx={{ mr: 1 }}>
                                {title ? "Remove Title" : "Add Title"}
                            </Button>
                        </Box>
                        <ValueDisplay value={values.message} />
                        <AdvancedInput
                            name="message"
                            tools={mockSomeTools}
                            contextData={mockContextData}
                            placeholder={placeholder}
                            title={title}
                            features={features}
                        />
                        {helperText && (
                            <Typography variant="caption" color="text.secondary" sx={helperTextStyle}>
                                {helperText}
                            </Typography>
                        )}
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
    const [placeholder, setPlaceholder] = useState<string | undefined>("Enter your message here...");
    const [helperText, setHelperText] = useState<string | undefined>(undefined);
    const [title, setTitle] = useState<string | undefined>("Translated Feedback");
    const [features] = useState<AdvancedInputFeatures>({
        ...DEFAULT_FEATURES,
        allowTools: false,
    });

    return (
        <Formik
            initialValues={translatedInitialValues}
            onSubmit={handleTranslatedSubmit}
        >
            {({ values, setFieldValue, setValues }) => {
                // Simulation handlers
                async function simulatePasteSingleLine() {
                    const input = document.querySelector(`.${advancedInputTextareaClassName}`);
                    if (!(input instanceof HTMLTextAreaElement)) {
                        console.error("Could not find textarea element", input);
                        return;
                    }
                    input.focus();
                    await userEvent.paste(mockSingleLineText);
                    action("simulatePasteSingleLine")(mockSingleLineText);
                }

                async function simulatePasteMultiLine() {
                    const input = document.querySelector(`.${advancedInputTextareaClassName}`);
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
                    const input = document.querySelector(`.${advancedInputTextareaClassName}`);
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
                    const input = document.querySelector(`.${advancedInputTextareaClassName}`);
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

                function togglePlaceholder() {
                    setPlaceholder(placeholder ? undefined : "Enter your message here...");
                    action("togglePlaceholder")(!!placeholder);
                }

                function toggleHelperText() {
                    setHelperText(helperText ? undefined : "This is a helpful message below the input");
                    action("toggleHelperText")(!!helperText);
                }

                function toggleTitle() {
                    setTitle(title ? undefined : "Translated Feedback");
                    action("toggleTitle")(!!title);
                }

                function clearAll() {
                    setValues({
                        translations: [
                            { ...values.translations[0], message: "" },
                            { ...values.translations[1], message: "" },
                        ],
                    });
                    setPlaceholder("Enter your message here...");
                    setHelperText(undefined);
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
                            onTogglePlaceholder={togglePlaceholder}
                            onToggleHelperText={toggleHelperText}
                            onClearAll={clearAll}
                            hasPlaceholder={!!placeholder}
                            hasHelperText={!!helperText}
                        />

                        <Box sx={{ mb: 2 }}>
                            <Button variant="outlined" onClick={toggleTitle} sx={{ mr: 1 }}>
                                {title ? "Remove Title" : "Add Title"}
                            </Button>
                        </Box>

                        <Box display="flex" flexDirection="column" gap={2}>
                            <Typography variant="h6" color="textSecondary">English</Typography>
                            <ValueDisplay value={values.translations[0].message} />
                            <TranslatedAdvancedInput
                                name="message"
                                language="en"
                                tools={mockSomeTools}
                                contextData={mockContextData}
                                placeholder={placeholder}
                                title={title}
                                features={features}
                            />
                            {helperText && (
                                <Typography variant="caption" color="text.secondary">
                                    {helperText}
                                </Typography>
                            )}
                            <Typography variant="h6" color="textSecondary">Spanish</Typography>
                            <ValueDisplay value={values.translations[1].message} />
                            <TranslatedAdvancedInput
                                name="message"
                                language="es"
                                tools={mockSomeTools}
                                contextData={mockContextData}
                                placeholder={placeholder}
                                title={title ? `${title} (Español)` : undefined}
                                features={features}
                            />
                            {helperText && (
                                <Typography variant="caption" color="text.secondary">
                                    {helperText}
                                </Typography>
                            )}
                        </Box>
                    </Form>
                );
            }}
        </Formik>
    );
}

/**
 * Features story: demonstrates toggling individual feature flags
 */
export function Features() {
    // State for message and component props
    const [message, setMessage] = useState("");
    const [contextData, setContextData] = useState<ContextItem[]>([]);
    const [tools, setTools] = useState<Tool[]>(mockSomeTools);
    // Feature toggles state - start with all features enabled
    const [features, setFeatures] = useState<AdvancedInputFeatures>({ ...DEFAULT_FEATURES });

    // Handle feature toggle
    function handleFeatureToggle(featureKey: keyof AdvancedInputFeatures) {
        setFeatures(prev => ({
            ...prev,
            [featureKey]: !prev[featureKey],
        }));
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

    function onContextDataChange(updated: ContextItem[]) {
        setContextData(updated);
        action("onContextDataChange")(updated);
    }

    // Reset features to default
    function resetFeatures() {
        setFeatures({ ...DEFAULT_FEATURES });
    }

    // Set minimal features (just text entry and submit)
    function setMinimalFeatures() {
        setFeatures({
            allowFormatting: false,
            allowExpand: false,
            allowFileAttachments: false,
            allowImageAttachments: false,
            allowTextAttachments: false,
            allowTools: false,
            allowCharacterLimit: true,
            allowVoiceInput: false,
            allowSubmit: true,
            allowSpellcheck: true,
            allowSettingsCustomization: false,
        });
    }

    // Set form-optimized features
    function setFormFeatures() {
        setFeatures({
            allowFormatting: true,
            allowExpand: true,
            allowFileAttachments: false,
            allowImageAttachments: false,
            allowTextAttachments: false,
            allowTools: false,
            allowCharacterLimit: true,
            allowVoiceInput: false,
            allowSubmit: false,
            allowSpellcheck: true,
            allowSettingsCustomization: false,
        });
    }

    // Set chat-optimized features
    function setChatFeatures() {
        setFeatures({
            ...DEFAULT_FEATURES,
            allowTools: true,
        });
    }

    // Helper function to create a callback for toggling features
    function handleFeatureToggleCb(key: keyof AdvancedInputFeatures) {
        return function toggleFeature() {
            handleFeatureToggle(key);
        };
    }

    return (
        <>
            <Box sx={featureBoxStyle}>
                <Typography variant="h6" mb={2}>Feature Configuration</Typography>

                <Box sx={featureControlsStyle}>
                    <Button
                        variant="outlined"
                        size="small"
                        onClick={resetFeatures}
                        sx={featureButtonStyle}
                    >
                        All Features
                    </Button>
                    <Button
                        variant="outlined"
                        size="small"
                        onClick={setMinimalFeatures}
                        sx={featureButtonStyle}
                    >
                        Minimal
                    </Button>
                    <Button
                        variant="outlined"
                        size="small"
                        onClick={setFormFeatures}
                        sx={featureButtonStyle}
                    >
                        Form Optimized
                    </Button>
                    <Button
                        variant="outlined"
                        size="small"
                        onClick={setChatFeatures}
                        sx={featureButtonStyle}
                    >
                        Chat Optimized
                    </Button>
                </Box>

                <Box sx={featureToggleStyle}>
                    {/* Input formatting features */}
                    <Box sx={featureGroupStyle}>
                        <Typography sx={featureGroupTitleStyle}>Formatting</Typography>
                        <Box sx={columnFlexStyle}>
                            <FormControlLabel
                                control={
                                    <Switch
                                        checked={features.allowFormatting}
                                        onChange={handleFeatureToggleCb("allowFormatting")}
                                        color="primary"
                                        size="small"
                                    />
                                }
                                label="Allow Formatting"
                            />
                        </Box>
                    </Box>

                    {/* Size and expansion features */}
                    <Box sx={featureGroupStyle}>
                        <Typography sx={featureGroupTitleStyle}>Size & Expansion</Typography>
                        <FormControlLabel
                            control={
                                <Switch
                                    checked={features.allowExpand}
                                    onChange={handleFeatureToggleCb("allowExpand")}
                                    color="primary"
                                    size="small"
                                />
                            }
                            label="Allow Expand"
                        />
                    </Box>

                    {/* Attachments and context */}
                    <Box sx={featureGroupStyle}>
                        <Typography sx={featureGroupTitleStyle}>Attachments</Typography>
                        <Box sx={columnFlexStyle}>
                            <FormControlLabel
                                control={
                                    <Switch
                                        checked={features.allowFileAttachments}
                                        onChange={handleFeatureToggleCb("allowFileAttachments")}
                                        color="primary"
                                        size="small"
                                    />
                                }
                                label="Allow File Attachments"
                            />
                            <FormControlLabel
                                control={
                                    <Switch
                                        checked={features.allowImageAttachments}
                                        onChange={handleFeatureToggleCb("allowImageAttachments")}
                                        color="primary"
                                        size="small"
                                    />
                                }
                                label="Allow Image Attachments"
                            />
                            <FormControlLabel
                                control={
                                    <Switch
                                        checked={features.allowTextAttachments}
                                        onChange={handleFeatureToggleCb("allowTextAttachments")}
                                        color="primary"
                                        size="small"
                                    />
                                }
                                label="Allow Text Attachments"
                            />
                            <FormControlLabel
                                control={
                                    <Switch
                                        checked={features.allowContextDropdown}
                                        onChange={handleFeatureToggleCb("allowContextDropdown")}
                                        color="primary"
                                        size="small"
                                    />
                                }
                                label="Allow Context Dropdown (@/Slash Commands)"
                            />
                        </Box>
                    </Box>

                    {/* Tool integration */}
                    <Box sx={featureGroupStyle}>
                        <Typography sx={featureGroupTitleStyle}>Tools</Typography>
                        <FormControlLabel
                            control={
                                <Switch
                                    checked={features.allowTools}
                                    onChange={handleFeatureToggleCb("allowTools")}
                                    color="primary"
                                    size="small"
                                />
                            }
                            label="Allow Tools"
                        />
                    </Box>

                    {/* Submission features */}
                    <Box sx={featureGroupStyle}>
                        <Typography sx={featureGroupTitleStyle}>Submission</Typography>
                        <Box sx={columnFlexStyle}>
                            <FormControlLabel
                                control={
                                    <Switch
                                        checked={features.allowCharacterLimit}
                                        onChange={handleFeatureToggleCb("allowCharacterLimit")}
                                        color="primary"
                                        size="small"
                                    />
                                }
                                label="Allow Character Limit"
                            />
                            <FormControlLabel
                                control={
                                    <Switch
                                        checked={features.allowVoiceInput}
                                        onChange={handleFeatureToggleCb("allowVoiceInput")}
                                        color="primary"
                                        size="small"
                                    />
                                }
                                label="Allow Voice Input"
                            />
                            <FormControlLabel
                                control={
                                    <Switch
                                        checked={features.allowSubmit}
                                        onChange={handleFeatureToggleCb("allowSubmit")}
                                        color="primary"
                                        size="small"
                                    />
                                }
                                label="Allow Submit"
                            />
                        </Box>
                    </Box>

                    {/* Settings */}
                    <Box sx={featureGroupStyle}>
                        <Typography sx={featureGroupTitleStyle}>Settings</Typography>
                        <FormControlLabel
                            control={
                                <Switch
                                    checked={features.allowSettingsCustomization}
                                    onChange={handleFeatureToggleCb("allowSettingsCustomization")}
                                    color="primary"
                                    size="small"
                                />
                            }
                            label="Allow Settings Customization"
                        />
                    </Box>

                    {/* Editor Behavior */}
                    <Box sx={featureGroupStyle}>
                        <Typography sx={featureGroupTitleStyle}>Editor Behavior</Typography>
                        <FormControlLabel
                            control={
                                <Switch
                                    checked={features.allowSpellcheck}
                                    onChange={handleFeatureToggleCb("allowSpellcheck")}
                                    color="primary"
                                    size="small"
                                />
                            }
                            label="Enable Spellchecking"
                        />
                    </Box>
                </Box>
            </Box>

            <Divider sx={dividerStyle} />

            <ValueDisplay value={message} />

            <AdvancedInputBase
                name="message"
                tools={tools}
                contextData={contextData}
                features={features}
                maxChars={100}
                placeholder="Enter your message here..."
                onChange={onMessageChange}
                onToolsChange={onToolsChange}
                onContextDataChange={onContextDataChange}
                onSubmit={onSubmit}
                value={message}
            />

            <Box mt={4}>
                <Typography variant="caption" color="text.secondary">
                    Current Features Configuration:
                </Typography>
                <Box
                    component="pre"
                    sx={preCodeStyle}
                >
                    {JSON.stringify(features, null, 2)}
                </Box>
            </Box>
        </>
    );
}

/**
 * MinimalProps: demonstrates using AdvancedInput with minimal required props
 */
export function MinimalProps() {
    const [message, setMessage] = useState("");

    function handleMessageChange(newMessage: string) {
        setMessage(newMessage);
        action("onChange")(newMessage);
    }

    function handleSubmit(msg: string) {
        action("onSubmit")(msg);
        setMessage("");
    }

    return (
        <>
            <Typography variant="h6" color="textSecondary" sx={minimalTitleStyle}>
                Minimal Props
            </Typography>

            <Typography variant="body2" color="textSecondary" sx={minimalDescriptionStyle}>
                This example demonstrates using AdvancedInput with only the required props,
                omitting optional props like tools and contextData.
            </Typography>

            <ValueDisplay value={message} />

            <AdvancedInputBase
                name="message"
                onChange={handleMessageChange}
                onSubmit={handleSubmit}
                value={message}
                placeholder="Type here..."
            />
        </>
    );
}

/**
 * Title Example: demonstrates using AdvancedInput with a title
 */
export function WithTitle() {
    const [message, setMessage] = useState("");
    const [title, setTitle] = useState("Enter your feedback below");

    function handleMessageChange(newMessage: string) {
        setMessage(newMessage);
        action("onChange")(newMessage);
    }

    function handleSubmit(msg: string) {
        action("onSubmit")(msg);
        setMessage("");
    }

    function toggleTitle() {
        setTitle(title ? "" : "Enter your feedback below");
    }

    return (
        <>
            <Typography variant="h6" color="textSecondary" sx={minimalTitleStyle}>
                AdvancedInput with Title
            </Typography>

            <Typography variant="body2" color="textSecondary" sx={minimalDescriptionStyle}>
                This example demonstrates using AdvancedInput with a title above the input field.
            </Typography>

            <Box sx={{ mb: 2 }}>
                <Button variant="outlined" onClick={toggleTitle}>
                    {title ? "Remove Title" : "Add Title"}
                </Button>
            </Box>

            <ValueDisplay value={message} />

            <AdvancedInputBase
                name="message"
                title={title}
                onChange={handleMessageChange}
                onSubmit={handleSubmit}
                value={message}
                placeholder="Type here..."
                features={{
                    ...DEFAULT_FEATURES,
                    allowTools: false,
                }}
            />
        </>
    );
}
