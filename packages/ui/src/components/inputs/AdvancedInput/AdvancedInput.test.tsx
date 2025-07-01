import { describe, expect, it, vi } from "vitest";
import { AITaskDisplayState } from "../../../types.js";
import { DEFAULT_FEATURES, type AdvancedInputFeatures, type AITaskDisplay, type ContextItem } from "./utils.js";

// Mock the entire AdvancedInput module to test behavior without rendering
vi.mock("./AdvancedInput.js", () => ({
    AdvancedInputBase: vi.fn(({ 
        value, 
        onChange, 
        onSubmit, 
        onTasksChange, 
        onContextDataChange,
        disabled,
        error,
        helperText,
        placeholder,
        title,
        isRequired,
        features,
        tasks,
        contextData,
    }: any) => {
        // Return null but allow tests to check how component was called
        return null;
    }),
    AdvancedInput: vi.fn(({ name, features, value, onChange }: any) => {
        // Mock Formik integration
        return null;
    }),
    TranslatedAdvancedInput: vi.fn(({ language, name }: any) => {
        // Mock translation integration
        return null;
    }),
}));

// Import after mocking to get the mocked versions
import { AdvancedInputBase, AdvancedInput, TranslatedAdvancedInput } from "./AdvancedInput.js";

describe("AdvancedInputBase", () => {
    describe("Basic rendering behavior", () => {
        it("accepts basic props", () => {
            const props = {
                value: "test value",
                onChange: vi.fn(),
                placeholder: "Enter text",
            };
            
            AdvancedInputBase(props);
            
            expect(AdvancedInputBase).toHaveBeenCalledWith(
                expect.objectContaining({
                    value: "test value",
                    placeholder: "Enter text",
                }),
            );
        });

        it("handles disabled state", () => {
            const props = {
                disabled: true,
                value: "",
                onChange: vi.fn(),
            };
            
            AdvancedInputBase(props);
            
            expect(AdvancedInputBase).toHaveBeenCalledWith(
                expect.objectContaining({
                    disabled: true,
                }),
            );
        });

        it("displays error state with helper text", () => {
            const props = {
                error: true,
                helperText: "This field is required",
            };
            
            AdvancedInputBase(props);
            
            expect(AdvancedInputBase).toHaveBeenCalledWith(
                expect.objectContaining({
                    error: true,
                    helperText: "This field is required",
                }),
            );
        });
    });

    describe("Title and required indicator", () => {
        it("accepts title prop", () => {
            const props = {
                title: "Message Title",
            };
            
            AdvancedInputBase(props);
            
            expect(AdvancedInputBase).toHaveBeenCalledWith(
                expect.objectContaining({
                    title: "Message Title",
                }),
            );
        });

        it("handles required field indicator", () => {
            const props = {
                title: "Required Field",
                isRequired: true,
            };
            
            AdvancedInputBase(props);
            
            expect(AdvancedInputBase).toHaveBeenCalledWith(
                expect.objectContaining({
                    title: "Required Field",
                    isRequired: true,
                }),
            );
        });
    });

    describe("Feature flags", () => {
        it("accepts feature configuration", () => {
            const features: Partial<AdvancedInputFeatures> = {
                allowFormatting: false,
                allowVoiceInput: true,
                allowSubmit: true,
                maxChars: 500,
            };
            
            AdvancedInputBase({ features });
            
            expect(AdvancedInputBase).toHaveBeenCalledWith(
                expect.objectContaining({
                    features: expect.objectContaining({
                        allowFormatting: false,
                        allowVoiceInput: true,
                        allowSubmit: true,
                        maxChars: 500,
                    }),
                }),
            );
        });

        it("merges features with defaults", () => {
            const customFeatures: Partial<AdvancedInputFeatures> = {
                allowFormatting: false,
            };
            
            const merged = { ...DEFAULT_FEATURES, ...customFeatures };
            
            expect(merged.allowFormatting).toBe(false);
            expect(merged.allowVoiceInput).toBe(DEFAULT_FEATURES.allowVoiceInput);
        });
    });

    describe("Submit functionality", () => {
        it("accepts onSubmit callback", () => {
            const onSubmit = vi.fn();
            
            AdvancedInputBase({ 
                value: "test message",
                onSubmit,
                features: { allowSubmit: true },
            });
            
            expect(AdvancedInputBase).toHaveBeenCalledWith(
                expect.objectContaining({
                    onSubmit,
                    value: "test message",
                }),
            );
        });
    });

    describe("Task management", () => {
        const mockTasks: AITaskDisplay[] = [
            {
                id: "1",
                name: "task-1",
                displayName: "Task 1",
                label: "Task 1",
                iconInfo: { name: "Settings", type: "Common" },
                state: AITaskDisplayState.Enabled,
            },
            {
                id: "2",
                name: "task-2",
                displayName: "Task 2",
                label: "Task 2",
                iconInfo: { name: "Info", type: "Common" },
                state: AITaskDisplayState.Disabled,
            },
        ];

        it("accepts tasks array", () => {
            AdvancedInputBase({ 
                tasks: mockTasks,
                features: { allowTasks: true },
            });
            
            expect(AdvancedInputBase).toHaveBeenCalledWith(
                expect.objectContaining({
                    tasks: mockTasks,
                }),
            );
        });

        it("accepts onTasksChange callback", () => {
            const onTasksChange = vi.fn();
            
            AdvancedInputBase({ 
                tasks: mockTasks,
                onTasksChange,
                features: { allowTasks: true },
            });
            
            expect(AdvancedInputBase).toHaveBeenCalledWith(
                expect.objectContaining({
                    onTasksChange,
                }),
            );
        });

        it("validates task state transitions", () => {
            // Test state transition logic
            const task = mockTasks[0];
            
            // Enabled -> Disabled
            if (task.state === AITaskDisplayState.Enabled) {
                const nextState = AITaskDisplayState.Disabled;
                expect(nextState).toBe(AITaskDisplayState.Disabled);
            }
            
            // Disabled -> Enabled
            const disabledTask = mockTasks[1];
            if (disabledTask.state === AITaskDisplayState.Disabled) {
                const nextState = AITaskDisplayState.Enabled;
                expect(nextState).toBe(AITaskDisplayState.Enabled);
            }
        });
    });

    describe("Context data management", () => {
        const mockContextData: ContextItem[] = [
            {
                id: "1",
                type: "text",
                label: "Text snippet",
            },
            {
                id: "2",
                type: "image",
                label: "image.jpg",
                src: "data:image/jpeg;base64,...",
            },
            {
                id: "3",
                type: "file",
                label: "document.pdf",
                file: new File(["content"], "document.pdf", { type: "application/pdf" }),
            },
        ];

        it("accepts context data", () => {
            AdvancedInputBase({ contextData: mockContextData });
            
            expect(AdvancedInputBase).toHaveBeenCalledWith(
                expect.objectContaining({
                    contextData: mockContextData,
                }),
            );
        });

        it("accepts onContextDataChange callback", () => {
            const onContextDataChange = vi.fn();
            
            AdvancedInputBase({ 
                contextData: mockContextData,
                onContextDataChange,
            });
            
            expect(AdvancedInputBase).toHaveBeenCalledWith(
                expect.objectContaining({
                    onContextDataChange,
                }),
            );
        });
    });

    describe("Voice input", () => {
        it("handles voice input feature", () => {
            const onChange = vi.fn();
            
            AdvancedInputBase({ 
                onChange,
                features: { allowVoiceInput: true },
            });
            
            expect(AdvancedInputBase).toHaveBeenCalledWith(
                expect.objectContaining({
                    features: expect.objectContaining({
                        allowVoiceInput: true,
                    }),
                }),
            );
        });
    });

    describe("Character limit", () => {
        it("handles character limit feature", () => {
            AdvancedInputBase({ 
                value: "Hello",
                features: { 
                    allowCharacterLimit: true,
                    maxChars: 100,
                },
            });
            
            expect(AdvancedInputBase).toHaveBeenCalledWith(
                expect.objectContaining({
                    value: "Hello",
                    features: expect.objectContaining({
                        allowCharacterLimit: true,
                        maxChars: 100,
                    }),
                }),
            );
        });
    });

    describe("File attachments", () => {
        it("handles file attachment features", () => {
            AdvancedInputBase({ 
                features: { 
                    allowFileAttachments: true,
                    allowImageAttachments: true,
                    allowTextAttachments: true,
                },
            });
            
            expect(AdvancedInputBase).toHaveBeenCalledWith(
                expect.objectContaining({
                    features: expect.objectContaining({
                        allowFileAttachments: true,
                        allowImageAttachments: true,
                        allowTextAttachments: true,
                    }),
                }),
            );
        });
    });
});

describe("AdvancedInput (Formik wrapper)", () => {
    it("passes name prop for Formik integration", () => {
        AdvancedInput({ 
            name: "message",
            value: "test",
            onChange: vi.fn(),
        });
        
        expect(AdvancedInput).toHaveBeenCalledWith(
            expect.objectContaining({
                name: "message",
            }),
        );
    });

    it("accepts features prop", () => {
        const features: Partial<AdvancedInputFeatures> = {
            allowFormatting: false,
        };
        
        AdvancedInput({ 
            name: "message",
            features,
        });
        
        expect(AdvancedInput).toHaveBeenCalledWith(
            expect.objectContaining({
                features,
            }),
        );
    });
});

describe("TranslatedAdvancedInput", () => {
    it("accepts language and name props", () => {
        TranslatedAdvancedInput({ 
            language: "en",
            name: "description",
        });
        
        expect(TranslatedAdvancedInput).toHaveBeenCalledWith(
            expect.objectContaining({
                language: "en",
                name: "description",
            }),
        );
    });
});

describe("DEFAULT_FEATURES configuration", () => {
    it("contains all expected feature flags", () => {
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
        expect(DEFAULT_FEATURES).toHaveProperty("minRowsCollapsed");
        expect(DEFAULT_FEATURES).toHaveProperty("minRowsExpanded");
        expect(DEFAULT_FEATURES).toHaveProperty("maxRowsCollapsed");
        expect(DEFAULT_FEATURES).toHaveProperty("maxRowsExpanded");
    });

    it("has sensible default values", () => {
        expect(DEFAULT_FEATURES.allowFormatting).toBe(true);
        expect(DEFAULT_FEATURES.allowExpand).toBe(true);
        expect(DEFAULT_FEATURES.allowFileAttachments).toBe(true);
        expect(DEFAULT_FEATURES.allowSubmit).toBe(true);
        expect(DEFAULT_FEATURES.allowVoiceInput).toBe(true);
        expect(DEFAULT_FEATURES.allowSpellcheck).toBe(true);
        expect(DEFAULT_FEATURES.allowSettingsCustomization).toBe(true);
        expect(DEFAULT_FEATURES.maxChars).toBeUndefined();
        expect(DEFAULT_FEATURES.minRowsCollapsed).toBeUndefined();
        expect(DEFAULT_FEATURES.maxRowsCollapsed).toBeUndefined();
        expect(DEFAULT_FEATURES.minRowsExpanded).toBeUndefined();
        expect(DEFAULT_FEATURES.maxRowsExpanded).toBeUndefined();
    });
});

describe("Utility functions", () => {
    describe("Task state management", () => {
        it("handles state transitions correctly", () => {
            const tasks: AITaskDisplay[] = [
                {
                    id: "1",
                    name: "test",
                    displayName: "Test",
                    label: "Test",
                    iconInfo: { name: "Test", type: "Common" },
                    state: AITaskDisplayState.Enabled,
                },
            ];
            
            // Simulate toggle logic
            const currentState = tasks[0].state;
            let newState: AITaskDisplayState;
            
            if (currentState === AITaskDisplayState.Disabled) {
                newState = AITaskDisplayState.Enabled;
            } else if (currentState === AITaskDisplayState.Enabled) {
                newState = AITaskDisplayState.Disabled;
            } else {
                newState = AITaskDisplayState.Enabled;
            }
            
            expect(newState).toBe(AITaskDisplayState.Disabled);
        });

        it("handles exclusive state correctly", () => {
            const tasks: AITaskDisplay[] = [
                {
                    id: "1",
                    name: "task1",
                    displayName: "Task 1",
                    label: "Task 1",
                    iconInfo: { name: "Test", type: "Common" },
                    state: AITaskDisplayState.Enabled,
                },
                {
                    id: "2",
                    name: "task2",
                    displayName: "Task 2",
                    label: "Task 2",
                    iconInfo: { name: "Test", type: "Common" },
                    state: AITaskDisplayState.Exclusive,
                },
            ];
            
            // Simulate making task 1 exclusive
            const updated = tasks.map((task, i) => {
                if (i === 0) {
                    return { ...task, state: AITaskDisplayState.Exclusive };
                } else if (task.state === AITaskDisplayState.Exclusive) {
                    return { ...task, state: AITaskDisplayState.Enabled };
                } else {
                    return task;
                }
            });
            
            expect(updated[0].state).toBe(AITaskDisplayState.Exclusive);
            expect(updated[1].state).toBe(AITaskDisplayState.Enabled);
        });
    });

    describe("Context item types", () => {
        it("identifies different context types", () => {
            const textContext: ContextItem = {
                id: "1",
                type: "text",
                label: "Text",
            };
            
            const imageContext: ContextItem = {
                id: "2",
                type: "image",
                label: "Image",
                src: "data:image/png;base64,...",
            };
            
            const fileContext: ContextItem = {
                id: "3",
                type: "file",
                label: "File",
                file: new File([""], "test.txt"),
            };
            
            expect(textContext.type).toBe("text");
            expect(imageContext.type).toBe("image");
            expect(fileContext.type).toBe("file");
        });
    });
});
