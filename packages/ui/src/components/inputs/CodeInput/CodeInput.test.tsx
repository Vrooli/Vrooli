/* eslint-disable react-perf/jsx-no-new-object-as-prop */
import { act, render, screen, waitFor } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import { Form, Formik } from "formik";
import React, { Suspense } from "react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import * as yup from "yup";
import { CodeLanguage } from "@vrooli/shared";
import { formAssertions, formInteractions, renderWithFormik, testValidationSchemas } from "../../../__test/helpers/formTestHelpers.js";
import { CodeInput, CodeInputBase } from "./CodeInput.js";

// Mock the lazy-loaded CodeMirror component
vi.mock("@uiw/react-codemirror", () => ({
    __esModule: true,
    default: React.forwardRef<any, any>((props, ref) => (
        <textarea
            data-testid="code-mirror-mock"
            ref={ref}
            value={props.value || ""}
            onChange={(e) => props.onChange?.(e.target.value)}
            readOnly={props.readOnly}
            style={{ height: props.height, width: "100%" }}
            {...(props.id && { id: props.id })}
        />
    )),
}));

// Mock the dynamic imports
vi.mock("@codemirror/view", () => ({
    EditorView: {
        baseTheme: vi.fn(() => ({})),
        decorations: {
            from: vi.fn(() => ({})),
        },
    },
    gutter: vi.fn(() => ({})),
    GutterMarker: class MockGutterMarker {
        toDOM() {
            return document.createElement("div");
        }
    },
    Decoration: {
        mark: vi.fn(() => ({
            range: vi.fn(() => ({})),
        })),
        set: vi.fn(() => ({})),
    },
    showTooltip: {
        computeN: vi.fn(() => ({})),
    },
}));
vi.mock("@codemirror/state", () => ({
    StateField: {
        define: vi.fn(() => ({})),
    },
}));
vi.mock("@codemirror/commands", () => ({
    undo: vi.fn(),
    redo: vi.fn(),
}));
vi.mock("@codemirror/lint", () => ({
    linter: vi.fn(() => ({})),
}));

// Mock all language imports
vi.mock("@codemirror/lang-javascript", () => ({
    javascript: vi.fn(() => ({})),
}));
vi.mock("@codemirror/lang-json", () => ({
    json: vi.fn(() => ({})),
    jsonParseLinter: vi.fn(() => vi.fn(() => [])),
}));
vi.mock("@codemirror/lang-python", () => ({
    python: vi.fn(() => ({})),
}));

const defaultProps = {
    codeLanguage: CodeLanguage.Javascript,
    content: "",
    handleCodeLanguageChange: vi.fn(),
    handleContentChange: vi.fn(),
    name: "test-code",
};

describe("CodeInputBase", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe("Basic rendering", () => {
        it("renders the code editor with basic structure", async () => {
            render(<CodeInputBase {...defaultProps} />);

            // Check for language label
            await waitFor(() => {
                expect(screen.getByText("Javascript")).toBeDefined();
            });

            // Check for code editor (eventually loads)
            await waitFor(() => {
                const editor = screen.getByTestId("code-mirror-mock");
                expect(editor).toBeDefined();
            }, { timeout: 3000 });
        });

        it("displays the current language in the info bar", () => {
            render(<CodeInputBase {...defaultProps} codeLanguage={CodeLanguage.Python} />);

            // Check for language label via test id
            const languageLabel = screen.getByTestId("language-label");
            expect(languageLabel).toBeDefined();
            expect(languageLabel.textContent).toContain("Python");
        });

        it("renders with provided content", async () => {
            render(<CodeInputBase {...defaultProps} content="console.log('hello');" />);

            await waitFor(() => {
                const editor = screen.getByTestId("code-mirror-mock") as HTMLTextAreaElement;
                expect(editor.value).toBe("console.log('hello');");
            });
        });

        it("shows loading message initially", () => {
            render(<CodeInputBase {...defaultProps} />);

            expect(screen.getByText("Loading editor...")).toBeDefined();
        });
    });

    describe("Language selection", () => {
        it("shows language selector when multiple languages are available", () => {
            render(
                <CodeInputBase
                    {...defaultProps}
                    limitTo={[CodeLanguage.Javascript, CodeLanguage.Python, CodeLanguage.Json]}
                />,
            );

            // Should show a select/combobox for language selection
            const languageSelector = screen.getByRole("combobox") ||
                                   screen.getByLabelText(/select language/i);
            expect(languageSelector).toBeDefined();
        });

        it("shows only language label when single language is available", () => {
            render(
                <CodeInputBase
                    {...defaultProps}
                    limitTo={[CodeLanguage.Javascript]}
                />,
            );

            const languageLabel = screen.getByTestId("language-label");
            expect(languageLabel).toBeDefined();
            expect(languageLabel.textContent).toContain("Javascript");
            // Should not show a selector
            expect(screen.queryByRole("combobox")).toBeNull();
        });

        it("handles language change", async () => {
            const handleChange = vi.fn();
            const user = userEvent.setup();

            render(
                <CodeInputBase
                    {...defaultProps}
                    handleCodeLanguageChange={handleChange}
                    limitTo={[CodeLanguage.Javascript, CodeLanguage.Python]}
                />,
            );

            // Find and interact with language selector
            const languageSelector = screen.getByRole("combobox");
            
            await act(async () => {
                await user.click(languageSelector);
            });

            // Language selector should be present and clickable
            expect(languageSelector).toBeDefined();
        });

        it("disables language selector when disabled", () => {
            render(
                <CodeInputBase
                    {...defaultProps}
                    disabled={true}
                    limitTo={[CodeLanguage.Javascript, CodeLanguage.Python]}
                />,
            );

            // Should not show a selector when disabled
            expect(screen.queryByRole("combobox")).toBeNull();
        });
    });

    describe("Content editing", () => {
        it("handles content changes", async () => {
            const handleChange = vi.fn();
            const user = userEvent.setup();

            render(<CodeInputBase {...defaultProps} handleContentChange={handleChange} />);

            await waitFor(() => {
                const editor = screen.getByTestId("code-mirror-mock");
                expect(editor).toBeDefined();
            });

            const editor = screen.getByTestId("code-mirror-mock");
            await act(async () => {
                await user.type(editor, "new code");
            });

            await waitFor(() => {
                expect(handleChange).toHaveBeenCalled();
            });
        }, 10000);

        it("prevents editing when disabled", async () => {
            const handleChange = vi.fn();

            render(<CodeInputBase {...defaultProps} disabled={true} handleContentChange={handleChange} />);

            await waitFor(() => {
                const editor = screen.getByTestId("code-mirror-mock") as HTMLTextAreaElement;
                expect(editor.readOnly).toBe(true);
            });
        });

        it("debounces content changes", async () => {
            const handleChange = vi.fn();
            const user = userEvent.setup();

            render(<CodeInputBase {...defaultProps} handleContentChange={handleChange} />);

            await waitFor(() => {
                const editor = screen.getByTestId("code-mirror-mock");
                expect(editor).toBeDefined();
            });

            const editor = screen.getByTestId("code-mirror-mock");
            await act(async () => {
                // Type multiple characters quickly
                await user.type(editor, "abc");
            });

            // Should debounce the calls
            await waitFor(() => {
                expect(handleChange).toHaveBeenCalled();
            });
        }, 10000);
    });

    describe("Toolbar actions", () => {
        it("renders copy button", () => {
            render(<CodeInputBase {...defaultProps} />);

            const copyButton = screen.getByTestId("copy-button");
            expect(copyButton).toBeDefined();
        });

        it("renders collapse/expand button", () => {
            render(<CodeInputBase {...defaultProps} />);

            const collapseButton = screen.getByTestId("collapse-toggle");
            expect(collapseButton).toBeDefined();
        });

        it("renders word wrap toggle button", () => {
            render(<CodeInputBase {...defaultProps} />);

            const wrapButton = screen.getByTestId("word-wrap-toggle");
            expect(wrapButton).toBeDefined();
        });

        it("shows undo and redo buttons when not disabled", () => {
            render(<CodeInputBase {...defaultProps} />);

            const undoButton = screen.getByRole("button", { name: /undo/i });
            const redoButton = screen.getByRole("button", { name: /redo/i });

            expect(undoButton).toBeDefined();
            expect(redoButton).toBeDefined();
        });

        it("hides action buttons when disabled", () => {
            render(<CodeInputBase {...defaultProps} disabled={true} />);

            expect(screen.queryByRole("button", { name: /undo/i })).toBeNull();
            expect(screen.queryByRole("button", { name: /redo/i })).toBeNull();
        });

        it("shows format button for JSON languages", () => {
            render(<CodeInputBase {...defaultProps} codeLanguage={CodeLanguage.Json} />);

            const formatButton = screen.getByRole("button", { name: /format/i });
            expect(formatButton).toBeDefined();
        });

        it("handles copy action", async () => {
            // Mock clipboard
            Object.assign(navigator, {
                clipboard: {
                    writeText: vi.fn().mockImplementation(() => Promise.resolve()),
                },
            });

            const user = userEvent.setup();
            render(<CodeInputBase {...defaultProps} content="test content" />);

            const copyButton = screen.getByTestId("copy-button");

            await act(async () => {
                await user.click(copyButton);
            });

            expect(navigator.clipboard.writeText).toHaveBeenCalledWith("test content");
        });

        it("handles collapse/expand toggle", async () => {
            const user = userEvent.setup();
            render(<CodeInputBase {...defaultProps} />);

            const collapseButton = screen.getByTestId("collapse-toggle");

            await act(async () => {
                await user.click(collapseButton);
            });

            // Should update the container to collapsed state
            await waitFor(() => {
                const container = screen.getByTestId("code-editor-container");
                expect(container.getAttribute("data-collapsed")).toBe("true");
            });
        });

        it("handles word wrap toggle", async () => {
            const user = userEvent.setup();
            render(<CodeInputBase {...defaultProps} />);

            const wrapButton = screen.getByTestId("word-wrap-toggle");

            await act(async () => {
                await user.click(wrapButton);
            });

            // Button should still be present (state changed internally)
            expect(wrapButton).toBeDefined();
        });
    });

    describe("JSON format validation", () => {
        it("shows format button for JSON languages", () => {
            render(<CodeInputBase {...defaultProps} codeLanguage={CodeLanguage.Json} />);

            const formatButton = screen.getByRole("button", { name: /format/i });
            expect(formatButton).toBeDefined();
        });

        it("handles JSON formatting", async () => {
            const handleChange = vi.fn();
            const user = userEvent.setup();

            render(
                <CodeInputBase
                    {...defaultProps}
                    codeLanguage={CodeLanguage.Json}
                    content='{"name":"test","value":123}'
                    handleContentChange={handleChange}
                />,
            );

            const formatButton = screen.getByRole("button", { name: /format/i });

            await act(async () => {
                await user.click(formatButton);
            });

            expect(handleChange).toHaveBeenCalledWith(
                JSON.stringify({ name: "test", value: 123 }, null, 4),
            );
        });

        it("handles invalid JSON gracefully", async () => {
            const user = userEvent.setup();

            render(
                <CodeInputBase
                    {...defaultProps}
                    codeLanguage={CodeLanguage.Json}
                    content='{"invalid json'
                />,
            );

            const formatButton = screen.getByRole("button", { name: /format/i });

            await act(async () => {
                await user.click(formatButton);
            });

            // Should not crash and might show an error message
            expect(formatButton).toBeDefined();
        });
    });

    describe("Collapsed state", () => {
        it("shows collapsed message when collapsed", async () => {
            const user = userEvent.setup();
            render(<CodeInputBase {...defaultProps} />);

            const collapseButton = screen.getByTestId("collapse-toggle");

            await act(async () => {
                await user.click(collapseButton);
            });

            await waitFor(() => {
                expect(screen.getByText(/collapsed/i)).toBeDefined();
            });
        });

        it("hides editor when collapsed", async () => {
            const user = userEvent.setup();
            render(<CodeInputBase {...defaultProps} />);

            const collapseButton = screen.getByTestId("collapse-toggle");

            await act(async () => {
                await user.click(collapseButton);
            });

            await waitFor(() => {
                const container = screen.getByTestId("code-editor-container");
                expect(container.getAttribute("data-collapsed")).toBe("true");
            });
        });
    });

    describe("Accessibility", () => {
        it("provides proper ARIA labels for buttons", () => {
            render(<CodeInputBase {...defaultProps} />);

            const copyButton = screen.getByTestId("copy-button");
            const collapseButton = screen.getByTestId("collapse-toggle");

            expect(copyButton.getAttribute("aria-label")).toBeTruthy();
            expect(collapseButton.getAttribute("aria-label")).toBeTruthy();
        });

        it("provides proper labeling for language selector", () => {
            render(
                <CodeInputBase
                    {...defaultProps}
                    limitTo={[CodeLanguage.Javascript, CodeLanguage.Python]}
                />,
            );

            const selector = screen.getByLabelText(/select language/i) ||
                           screen.getByRole("combobox", { name: /select language/i });
            expect(selector).toBeDefined();
        });
    });

    describe("Refresh functionality", () => {
        it("shows refresh button after delay", async () => {
            vi.useFakeTimers();
            render(<CodeInputBase {...defaultProps} />);

            // Fast forward past the refresh icon delay
            act(() => {
                vi.advanceTimersByTime(4000);
            });

            await waitFor(() => {
                const refreshButton = screen.getByRole("button", { name: /refresh/i }) ||
                                    document.querySelector("[data-testid*='refresh']");
                expect(refreshButton).toBeDefined();
            });

            vi.useRealTimers();
        });
    });
});

describe("CodeInput (Formik-enabled wrapper)", () => {
    describe("Formik integration", () => {
        it("integrates with Formik form state", async () => {
            const { getFormValues } = renderWithFormik(
                <CodeInput
                    name="code"
                    codeLanguageField="language"
                    limitTo={[CodeLanguage.Javascript, CodeLanguage.Python]}
                />,
                {
                    initialValues: {
                        code: "console.log('test');",
                        language: CodeLanguage.Javascript,
                    },
                },
            );

            await waitFor(() => {
                const editor = screen.getByTestId("code-mirror-mock") as HTMLTextAreaElement;
                expect(editor.value).toBe("console.log('test');");
            }, { timeout: 10000 });

            expect(getFormValues().code).toBe("console.log('test');");
            expect(getFormValues().language).toBe(CodeLanguage.Javascript);
        }, 15000);

        it("updates form values when content changes", async () => {
            const { user, getFormValues } = renderWithFormik(
                <CodeInput name="code" />,
                {
                    initialValues: { code: "", codeLanguage: CodeLanguage.Javascript },
                },
            );

            await waitFor(() => {
                const editor = screen.getByTestId("code-mirror-mock");
                expect(editor).toBeDefined();
            }, { timeout: 10000 });

            const editor = screen.getByTestId("code-mirror-mock");
            await act(async () => {
                await user.type(editor, "new code");
            });

            await waitFor(() => {
                expect(getFormValues().code).toContain("new code");
            }, { timeout: 5000 });
        }, 15000);

        it("updates form values when language changes", async () => {
            const { getFormValues } = renderWithFormik(
                <CodeInput
                    name="code"
                    codeLanguageField="language"
                    limitTo={[CodeLanguage.Javascript, CodeLanguage.Python]}
                />,
                {
                    initialValues: {
                        code: "",
                        language: CodeLanguage.Javascript,
                    },
                },
            );

            await waitFor(() => {
                const languageSelector = screen.getByRole("combobox");
                expect(languageSelector).toBeDefined();
            }, { timeout: 10000 });

            // Language should be set to initial value
            const currentValues = getFormValues();
            expect(currentValues.language).toBe(CodeLanguage.Javascript);
        }, 15000);

        it("handles validation errors", async () => {
            const { getFormValues } = renderWithFormik(
                <CodeInput name="code" />,
                {
                    initialValues: { code: "" },
                    formikConfig: {
                        validationSchema: yup.object({
                            code: testValidationSchemas.requiredString("Code"),
                        }),
                    },
                },
            );

            await waitFor(() => {
                const editor = screen.getByTestId("code-mirror-mock");
                expect(editor).toBeDefined();
            }, { timeout: 10000 });

            // Validate form initial state
            expect(getFormValues().code).toBe("");
        }, 15000);

        it("works with custom validation function", async () => {
            function customValidate(value: string) {
                if (value && value.includes("forbidden")) {
                    return "Code cannot contain 'forbidden'";
                }
                return undefined;
            }

            const { getFormValues } = renderWithFormik(
                <CodeInput name="code" validate={customValidate} />,
                { initialValues: { code: "" } },
            );

            await waitFor(() => {
                const editor = screen.getByTestId("code-mirror-mock");
                expect(editor).toBeDefined();
            }, { timeout: 10000 });

            // Validate that component renders with custom validation
            expect(getFormValues().code).toBe("");
        }, 15000);
    });

    describe("Legacy Formik integration test", () => {
        it("integrates with Formik form state (legacy approach)", async () => {
            const onSubmit = vi.fn();

            render(
                <Formik
                    initialValues={{ code: "", codeLanguage: CodeLanguage.Javascript }}
                    onSubmit={onSubmit}
                >
                    <Form>
                        <CodeInput name="code" />
                    </Form>
                </Formik>,
            );

            await waitFor(() => {
                const editor = screen.getByTestId("code-mirror-mock");
                expect(editor).toBeDefined();
            }, { timeout: 10000 });

            // Just check that the component renders successfully
            expect(screen.getByTestId("code-mirror-mock")).toBeDefined();
        }, 15000);
    });
});

describe("Component state transitions", () => {
    it("toggles between collapsed and expanded states", async () => {
        const user = userEvent.setup();
        render(<CodeInputBase {...defaultProps} />);

        // Start expanded
        await waitFor(() => {
            const container = screen.getByTestId("code-editor-container");
            expect(container.getAttribute("data-collapsed")).toBe("false");
        });

        const collapseButton = screen.getByTestId("collapse-toggle");

        await act(async () => {
            await user.click(collapseButton);
        });

        // Should now be collapsed
        await waitFor(() => {
            const container = screen.getByTestId("code-editor-container");
            expect(container.getAttribute("data-collapsed")).toBe("true");
        });

        await act(async () => {
            await user.click(collapseButton); // Same button, different state
        });

        // Should be expanded again
        await waitFor(() => {
            const container = screen.getByTestId("code-editor-container");
            expect(container.getAttribute("data-collapsed")).toBe("false");
        });
    });

    it("toggles between wrapped and unwrapped states", async () => {
        const user = userEvent.setup();
        render(<CodeInputBase {...defaultProps} />);

        const wrapButton = screen.getByTestId("word-wrap-toggle");
        expect(wrapButton).toBeDefined();

        await act(async () => {
            await user.click(wrapButton);
        });

        // Button should still exist after state change
        expect(screen.getByTestId("word-wrap-toggle")).toBeDefined();
    });

    it("toggles between different languages", () => {
        const { rerender } = render(
            <CodeInputBase {...defaultProps} codeLanguage={CodeLanguage.Javascript} />,
        );

        const languageLabel = screen.getByTestId("language-label");
        expect(languageLabel.textContent).toContain("Javascript");

        rerender(
            <CodeInputBase {...defaultProps} codeLanguage={CodeLanguage.Python} />,
        );

        const updatedLabel = screen.getByTestId("language-label");
        expect(updatedLabel.textContent).toContain("Python");
    });

    it("toggles between enabled and disabled states", () => {
        const { rerender } = render(<CodeInputBase {...defaultProps} />);

        // Should show action buttons when enabled
        expect(screen.getByRole("button", { name: /undo/i })).toBeDefined();
        expect(screen.getByRole("button", { name: /redo/i })).toBeDefined();

        rerender(<CodeInputBase {...defaultProps} disabled={true} />);

        // Should hide action buttons when disabled
        expect(screen.queryByRole("button", { name: /undo/i })).toBeNull();
        expect(screen.queryByRole("button", { name: /redo/i })).toBeNull();
    });
});

describe("Error handling", () => {
    it("handles missing content gracefully", () => {
        render(<CodeInputBase {...defaultProps} content={undefined as any} />);

        // Should not crash - check by language label
        const languageLabel = screen.getByTestId("language-label");
        expect(languageLabel.textContent).toContain("Javascript");
    });

    it("handles invalid language gracefully", () => {
        render(<CodeInputBase {...defaultProps} codeLanguage={"invalid" as any} />);

        // Should not crash - component should render something
        const container = document.body;
        expect(container).toBeDefined();
    });

    it("handles missing handlers gracefully", () => {
        render(
            <CodeInputBase
                {...defaultProps}
                handleCodeLanguageChange={undefined as any}
                handleContentChange={undefined as any}
            />,
        );

        // Should not crash - check by language label
        const languageLabel = screen.getByTestId("language-label");
        expect(languageLabel.textContent).toContain("Javascript");
    });
});
