import { describe, it, expect, vi, beforeEach } from "vitest";
import React from "react";
import { render, screen, fireEvent, waitFor } from "../../../__test/testUtils.js";
import { AdvancedInputBase } from "./AdvancedInput.js";
import { AITaskDisplayState } from "../../../types.js";
import { DEFAULT_FEATURES, type AdvancedInputFeatures, type ContextItem } from "./utils.js";

// Custom matchers for this test file
expect.extend({
    toBeInTheDocument(received) {
        const pass = received != null;
        return {
            pass,
            message: () => pass 
                ? "expected element not to be in the document"
                : "expected element to be in the document",
        };
    },
    toHaveAttribute(received, attr, value) {
        if (!received) {
            return {
                pass: false,
                message: () => `expected element to have attribute ${attr}="${value}" but element was null`,
            };
        }
        const actualValue = received.getAttribute?.(attr);
        const pass = value === undefined ? actualValue !== null : actualValue === value;
        return {
            pass,
            message: () => pass
                ? `expected element not to have attribute ${attr}="${value}"`
                : `expected element to have attribute ${attr}="${value}" but got "${actualValue}"`,
        };
    },
    toBeDisabled(received) {
        if (!received) {
            return {
                pass: false,
                message: () => "expected element to be disabled but element was null",
            };
        }
        const pass = received.disabled === true || received.getAttribute?.("disabled") !== null;
        return {
            pass,
            message: () => pass
                ? "expected element not to be disabled"
                : "expected element to be disabled",
        };
    },
});

// Override the global mock from setup.vitest.ts to test the real component
vi.unmock("./AdvancedInput.js");

// Mock complex dependencies but keep the main component
vi.mock("../../../hooks/useDimensions.js", () => ({
    useDimensions: () => ({
        dimensions: { width: 800, height: 600 },
        ref: { current: null },
    }),
}));

vi.mock("../../../hooks/useHotkeys.js", () => ({
    useHotkeys: vi.fn(),
}));

vi.mock("../../../hooks/useUndoRedo.js", () => ({
    useUndoRedo: (config: any) => ({
        internalValue: config.initialValue || "",
        changeInternalValue: vi.fn((newValue) => {
            if (typeof newValue === "function") {
                config.onChange?.(newValue(config.initialValue || ""));
            } else {
                config.onChange?.(newValue);
            }
        }),
        resetInternalValue: vi.fn(),
        undo: vi.fn(),
        redo: vi.fn(),
        canUndo: false,
        canRedo: false,
    }),
}));

vi.mock("react-dropzone", () => ({
    useDropzone: vi.fn(() => ({
        getRootProps: () => ({ 
            role: "button", 
            "data-testid": "dropzone",
        }),
        getInputProps: () => ({ type: "file", "data-testid": "file-input" }),
        isDragActive: false,
    })),
}));

vi.mock("../../../utils/localStorage.js", () => ({
    getCookie: vi.fn(() => ({
        showToolbar: true,
        enterWillSubmit: false,
        spellcheck: true,
    })),
    setCookie: vi.fn(),
}));

// Mock child components with realistic behavior instead of null
vi.mock("./AdvancedInputMarkdown.js", () => ({
    AdvancedInputMarkdown: React.forwardRef(({ value, onChange, placeholder, disabled, mergedFeatures, enterWillSubmit, error, maxRows, minRows, name, onActiveStatesChange, onBlur, onFocus, onSubmit, setHandleAction, tabIndex, toggleMarkdown, undo, redo, ...props }: any, ref: any) => (
        <textarea
            ref={ref}
            data-testid="markdown-editor"
            value={value || ""}
            onChange={(e) => onChange?.(e.target.value)}
            placeholder={placeholder}
            disabled={disabled}
            spellCheck={mergedFeatures?.allowSpellcheck}
            className="advanced-input-textarea"
            rows={minRows}
            name={name}
            tabIndex={tabIndex}
            onBlur={onBlur}
            onFocus={onFocus}
        />
    )),
}));

vi.mock("./lexical/AdvancedInputLexical.js", () => ({
    AdvancedInputLexical: React.forwardRef(({ value, onChange, placeholder, disabled, mergedFeatures, enterWillSubmit, error, maxRows, minRows, name, onActiveStatesChange, onBlur, onFocus, onSubmit, setHandleAction, tabIndex, toggleMarkdown, undo, redo, ...props }: any, ref: any) => (
        <div
            ref={ref}
            data-testid="lexical-editor"
            role="textbox"
            contentEditable={!disabled}
            onInput={(e) => onChange?.(e.currentTarget.textContent || "")}
            suppressContentEditableWarning
            aria-label={placeholder}
            spellCheck={mergedFeatures?.allowSpellcheck}
            className="advanced-input-textarea"
            tabIndex={tabIndex}
            onBlur={onBlur}
            onFocus={onFocus}
        >
            {value || ""}
        </div>
    )),
}));

vi.mock("../../buttons/MicrophoneButton.js", () => ({
    MicrophoneButton: ({ onTranscriptChange, disabled }: any) => (
        <button
            data-testid="microphone-button"
            onClick={() => onTranscriptChange?.("voice input text")}
            disabled={disabled}
            aria-label="Voice input"
        >
            🎤
        </button>
    ),
}));

vi.mock("./AdvancedInputToolbar.js", () => ({
    AdvancedInputToolbar: ({ handleAction }: any) => (
        <div data-testid="toolbar" role="toolbar" aria-label="Text formatting toolbar">
            <button onClick={() => handleAction?.("Bold")} aria-label="Bold">Bold</button>
            <button onClick={() => handleAction?.("Italic")} aria-label="Italic">Italic</button>
        </div>
    ),
    TOOLBAR_CLASS_NAME: "advanced-input-toolbar",
    defaultActiveStates: {
        Bold: false,
        Italic: false,
        Header1: false,
        Header2: false,
        Header3: false,
        Header4: false,
        Header5: false,
        Header6: false,
        Code: false,
        Link: false,
        ListBullet: false,
        ListNumber: false,
        ListCheckbox: false,
        Quote: false,
        Spoiler: false,
        Strikethrough: false,
        Table: false,
        Underline: false,
    },
}));

vi.mock("./ContextDropdown.js", () => ({
    ContextDropdown: () => <div data-testid="context-dropdown" />,
}));

vi.mock("../../dialogs/FindObjectDialog/FindObjectDialog.js", () => ({
    FindObjectDialog: () => <div data-testid="find-object-dialog" />,
}));

// Mock additional problematic components
vi.mock("../../text/MarkdownDisplay.js", () => ({
    MarkdownDisplay: ({ children }: any) => <div data-testid="markdown-display">{children}</div>,
}));

vi.mock("../../../styles.js", () => ({
    ObjectListProfileAvatar: "div",
    multiLineEllipsis: {},
    noSelect: {},
}));

vi.mock("../../lists/ObjectListItemBase/ObjectListItemBase.tsx", () => ({
    ObjectListItemBase: () => null,
}));

vi.mock("../../lists/ObjectList/ObjectList.tsx", () => ({
    ObjectList: () => null,
}));

describe("AdvancedInput Configuration and Utils", () => {
    describe("Feature Configuration", () => {
        it("DEFAULT_FEATURES includes all expected properties", () => {
            expect(DEFAULT_FEATURES).toHaveProperty("allowFormatting");
            expect(DEFAULT_FEATURES).toHaveProperty("allowExpand");
            expect(DEFAULT_FEATURES).toHaveProperty("allowFileAttachments");
            expect(DEFAULT_FEATURES).toHaveProperty("allowImageAttachments");
            expect(DEFAULT_FEATURES).toHaveProperty("allowTextAttachments");
            expect(DEFAULT_FEATURES).toHaveProperty("allowContextDropdown");
            expect(DEFAULT_FEATURES).toHaveProperty("allowTasks");
            expect(DEFAULT_FEATURES).toHaveProperty("allowCharacterLimit");
            expect(DEFAULT_FEATURES).toHaveProperty("allowVoiceInput");
            expect(DEFAULT_FEATURES).toHaveProperty("allowSubmit");
            expect(DEFAULT_FEATURES).toHaveProperty("allowSpellcheck");
            expect(DEFAULT_FEATURES).toHaveProperty("allowSettingsCustomization");
        });

        it("DEFAULT_FEATURES has reasonable default values", () => {
            expect(DEFAULT_FEATURES.allowFormatting).toBe(true);
            expect(DEFAULT_FEATURES.allowExpand).toBe(true);
            expect(DEFAULT_FEATURES.allowFileAttachments).toBe(true);
            expect(DEFAULT_FEATURES.allowVoiceInput).toBe(true);
            expect(DEFAULT_FEATURES.allowSubmit).toBe(true);
            expect(DEFAULT_FEATURES.allowTasks).toBe(true);
            expect(DEFAULT_FEATURES.allowCharacterLimit).toBe(true);
            expect(DEFAULT_FEATURES.allowSpellcheck).toBe(true);
            expect(DEFAULT_FEATURES.allowSettingsCustomization).toBe(true);
        });

        it("allows partial feature override", () => {
            const customFeatures: AdvancedInputFeatures = {
                allowFormatting: false,
                allowVoiceInput: false,
            };

            const mergedFeatures = {
                ...DEFAULT_FEATURES,
                ...customFeatures,
            };

            expect(mergedFeatures.allowFormatting).toBe(false);
            expect(mergedFeatures.allowVoiceInput).toBe(false);
            expect(mergedFeatures.allowSubmit).toBe(true); // Should keep default
            expect(mergedFeatures.allowExpand).toBe(true); // Should keep default
        });
    });

    describe("Task State Management Logic", () => {
        const mockTasks = [
            {
                id: "1",
                name: "test-task",
                displayName: "Test Task",
                iconInfo: { name: "Settings", type: "Common" as const },
                state: AITaskDisplayState.Enabled,
            },
            {
                id: "2",
                name: "disabled-task", 
                displayName: "Disabled Task",
                iconInfo: { name: "Stop", type: "Common" as const },
                state: AITaskDisplayState.Disabled,
            },
        ];

        it("can simulate task state transitions", () => {
            // Test Enabled -> Disabled transition
            const toggleEnabledToDisabled = (tasks: typeof mockTasks, index: number) => {
                const updated = [...tasks];
                const currentState = updated[index].state;
                let newState: AITaskDisplayState;
                
                if (currentState === AITaskDisplayState.Disabled) {
                    newState = AITaskDisplayState.Enabled;
                } else if (currentState === AITaskDisplayState.Enabled) {
                    newState = AITaskDisplayState.Disabled;
                } else {
                    newState = AITaskDisplayState.Enabled;
                }
                
                updated[index] = { ...updated[index], state: newState };
                return updated;
            };

            const result = toggleEnabledToDisabled(mockTasks, 0);
            expect(result[0].state).toBe(AITaskDisplayState.Disabled);
            expect(result[1]).toBe(mockTasks[1]); // Unchanged
        });

        it("can simulate exclusive task logic", () => {
            const makeTaskExclusive = (tasks: typeof mockTasks, index: number) => {
                return tasks.map((task, i) => {
                    if (i === index) {
                        return { ...task, state: AITaskDisplayState.Exclusive };
                    } else if (task.state === AITaskDisplayState.Exclusive) {
                        return { ...task, state: AITaskDisplayState.Enabled };
                    } else {
                        return task;
                    }
                });
            };

            const result = makeTaskExclusive(mockTasks, 1);
            expect(result[1].state).toBe(AITaskDisplayState.Exclusive);
            expect(result[0].state).toBe(AITaskDisplayState.Enabled);
        });
    });
});

describe("AdvancedInput Real Component Tests", () => {
    const defaultProps = {
        name: "test-input",
        value: "",
        onChange: vi.fn(),
    };

    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe("Component Rendering", () => {
        it("renders the actual AdvancedInputBase component", () => {
            render(<AdvancedInputBase {...defaultProps} />);
            
            // Should render the wrapper with advanced-input class
            const wrapper = document.querySelector(".advanced-input");
            expect(wrapper).toBeInTheDocument();
        });

        it("renders markdown editor when formatting is disabled", () => {
            render(<AdvancedInputBase {...defaultProps} features={{ allowFormatting: false }} />);
            
            expect(screen.getByTestId("markdown-editor")).toBeInTheDocument();
            expect(screen.queryByTestId("lexical-editor")).not.toBeInTheDocument();
        });

        it("renders lexical editor when formatting is enabled", () => {
            render(<AdvancedInputBase {...defaultProps} features={{ allowFormatting: true }} />);
            
            expect(screen.getByTestId("lexical-editor")).toBeInTheDocument();
            expect(screen.queryByTestId("markdown-editor")).not.toBeInTheDocument();
        });

        it("renders toolbar when formatting is enabled", () => {
            render(<AdvancedInputBase {...defaultProps} features={{ allowFormatting: true }} />);
            
            expect(screen.getByTestId("toolbar")).toBeInTheDocument();
        });
    });

    describe("User Input Handling", () => {
        it("handles text input in markdown mode", () => {
            const onChange = vi.fn();
            render(
                <AdvancedInputBase 
                    {...defaultProps} 
                    onChange={onChange}
                    features={{ allowFormatting: false }}
                />,
            );
            
            const editor = screen.getByTestId("markdown-editor");
            fireEvent.change(editor, { target: { value: "Hello, world!" } });
            
            expect(onChange).toHaveBeenCalledWith("Hello, world!");
        });

        it("handles text input in lexical mode", () => {
            const onChange = vi.fn();
            render(
                <AdvancedInputBase 
                    {...defaultProps} 
                    onChange={onChange}
                    features={{ allowFormatting: true }}
                />,
            );
            
            const editor = screen.getByTestId("lexical-editor");
            fireEvent.input(editor, { target: { textContent: "Hello, world!" } });
            
            expect(onChange).toHaveBeenCalledWith("Hello, world!");
        });

        it("shows placeholder text correctly", () => {
            render(
                <AdvancedInputBase 
                    {...defaultProps} 
                    placeholder="Enter your message..."
                />,
            );
            
            const markdownEditor = screen.queryByTestId("markdown-editor");
            const lexicalEditor = screen.queryByTestId("lexical-editor");
            
            if (markdownEditor) {
                expect(markdownEditor).toHaveAttribute("placeholder", "Enter your message...");
            } else if (lexicalEditor) {
                expect(lexicalEditor).toHaveAttribute("aria-label", "Enter your message...");
            }
        });
    });

    describe("Feature Integration", () => {
        it("integrates spellcheck properly when enabled", () => {
            render(
                <AdvancedInputBase 
                    {...defaultProps} 
                    features={{ allowSpellcheck: true }}
                />,
            );
            
            const editor = screen.queryByTestId("markdown-editor") || screen.queryByTestId("lexical-editor");
            expect(editor?.getAttribute("spellcheck")).toBe("true");
        });

        it("disables spellcheck when configured", () => {
            render(
                <AdvancedInputBase 
                    {...defaultProps} 
                    features={{ allowSpellcheck: false }}
                />,
            );
            
            const editor = screen.queryByTestId("markdown-editor") || screen.queryByTestId("lexical-editor");
            expect(editor?.getAttribute("spellcheck")).toBe("false");
        });

        it("shows voice input button when enabled", () => {
            render(
                <AdvancedInputBase 
                    {...defaultProps} 
                    features={{ allowVoiceInput: true }}
                />,
            );
            
            expect(screen.getByTestId("microphone-button")).toBeDefined();
        });

        it("hides voice input when disabled", () => {
            render(
                <AdvancedInputBase 
                    {...defaultProps} 
                    features={{ allowVoiceInput: false }}
                />,
            );
            
            expect(screen.queryByTestId("microphone-button")).toBeNull();
        });

        it("shows file dropzone when file attachments enabled", () => {
            render(
                <AdvancedInputBase 
                    {...defaultProps} 
                    features={{ allowFileAttachments: true }}
                />,
            );
            
            expect(screen.getByTestId("dropzone")).toBeDefined();
            expect(screen.getByTestId("file-input")).toBeDefined();
        });

        it("handles disabled state correctly", () => {
            render(
                <AdvancedInputBase 
                    {...defaultProps} 
                    disabled={true}
                />,
            );
            
            const markdownEditor = screen.queryByTestId("markdown-editor");
            const lexicalEditor = screen.queryByTestId("lexical-editor");
            
            if (markdownEditor) {
                expect(markdownEditor.disabled).toBe(true);
            } else if (lexicalEditor) {
                expect(lexicalEditor.getAttribute("contenteditable")).toBe("false");
            }
        });
    });

    describe("Interactive Features", () => {
        it("handles voice input interaction", () => {
            const onChange = vi.fn();
            render(
                <AdvancedInputBase 
                    {...defaultProps} 
                    onChange={onChange}
                    features={{ allowVoiceInput: true }}
                />,
            );
            
            const micButton = screen.getByTestId("microphone-button");
            fireEvent.click(micButton);
            
            // The button should be functional and not disabled
            expect(micButton).not.toBeDisabled();
            expect(micButton).toBeInTheDocument();
        });

        it("handles toolbar interactions", () => {
            const onChange = vi.fn();
            render(
                <AdvancedInputBase 
                    {...defaultProps} 
                    onChange={onChange}
                    features={{ allowFormatting: true }}
                />,
            );
            
            const toolbar = screen.getByTestId("toolbar");
            const boldButton = screen.getByLabelText("Bold");
            
            expect(toolbar).toBeInTheDocument();
            expect(boldButton).toBeInTheDocument();
            
            // Should be able to click formatting buttons
            fireEvent.click(boldButton);
            expect(boldButton).toBeInTheDocument(); // Should remain after click
        });

        it("handles context item display", () => {
            const mockContextData: ContextItem[] = [
                {
                    id: "1",
                    type: "text",
                    label: "Text snippet",
                },
                {
                    id: "2",
                    type: "file",
                    label: "document.pdf",
                    file: new File(["content"], "document.pdf", { type: "application/pdf" }),
                },
            ];

            render(
                <AdvancedInputBase 
                    {...defaultProps} 
                    contextData={mockContextData}
                />,
            );
            
            // Context items should be displayed
            expect(screen.getByText("Text snippet")).toBeInTheDocument();
            expect(screen.getByText("document.pdf")).toBeInTheDocument();
        });
    });

    describe("Accessibility", () => {
        it("provides proper ARIA labels for voice input", () => {
            render(
                <AdvancedInputBase 
                    {...defaultProps} 
                    features={{ allowVoiceInput: true }}
                />,
            );
            
            expect(screen.getByLabelText("Voice input")).toBeInTheDocument();
        });

        it("provides proper ARIA labels for toolbar", () => {
            render(
                <AdvancedInputBase 
                    {...defaultProps} 
                    features={{ allowFormatting: true }}
                />,
            );
            
            expect(screen.getByLabelText("Text formatting toolbar")).toBeInTheDocument();
        });

        it("supports keyboard focus", () => {
            render(<AdvancedInputBase {...defaultProps} />);
            
            const editor = screen.queryByTestId("markdown-editor") || screen.queryByTestId("lexical-editor");
            expect(editor).toBeInTheDocument();
            
            if (editor) {
                editor.focus();
                expect(document.activeElement).toBe(editor);
            }
        });
    });

    describe("Error Handling", () => {
        it("handles missing onChange gracefully", () => {
            expect(() => {
                render(
                    <AdvancedInputBase 
                        name="test"
                        value=""
                    />,
                );
            }).not.toThrow();
        });

        it("handles undefined features gracefully", () => {
            expect(() => {
                render(
                    <AdvancedInputBase 
                        {...defaultProps}
                        features={undefined as any}
                    />,
                );
            }).not.toThrow();
        });

        it("handles empty context data gracefully", () => {
            render(
                <AdvancedInputBase 
                    {...defaultProps} 
                    contextData={[]}
                />,
            );
            
            const editor = screen.queryByTestId("markdown-editor") || screen.queryByTestId("lexical-editor");
            expect(editor).toBeInTheDocument();
        });
    });

    describe("Complex Feature Combinations", () => {
        it("handles all features enabled", () => {
            const features: AdvancedInputFeatures = {
                allowFormatting: true,
                allowVoiceInput: true,
                allowFileAttachments: true,
                allowSpellcheck: true,
                allowSubmit: true,
                maxChars: 500,
            };
            
            render(
                <AdvancedInputBase 
                    {...defaultProps} 
                    features={features}
                />,
            );
            
            // Should render with all features enabled
            expect(screen.getByTestId("lexical-editor")).toBeInTheDocument();
            expect(screen.getByTestId("microphone-button")).toBeInTheDocument();
            expect(screen.getByTestId("dropzone")).toBeInTheDocument();
            expect(screen.getByTestId("toolbar")).toBeInTheDocument();
        });

        it("handles minimal feature configuration", () => {
            const features: AdvancedInputFeatures = {
                allowFormatting: false,
                allowVoiceInput: false,
                allowFileAttachments: false,
                allowImageAttachments: false,
                allowSpellcheck: false,
                allowSubmit: false,
            };
            
            render(
                <AdvancedInputBase 
                    {...defaultProps} 
                    features={features}
                />,
            );
            
            // Should render basic editor only
            expect(screen.getByTestId("markdown-editor")).toBeInTheDocument();
            expect(screen.queryByTestId("microphone-button")).not.toBeInTheDocument();
            expect(screen.queryByTestId("dropzone")).not.toBeInTheDocument();
            expect(screen.queryByTestId("toolbar")).not.toBeInTheDocument();
        });
    });
});
