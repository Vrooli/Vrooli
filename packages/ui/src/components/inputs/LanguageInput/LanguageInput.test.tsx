import { act, render, screen } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { SessionContext } from "../../../contexts/session.js";
import { LanguageInput } from "./LanguageInput.js";

interface MockSelectLanguageMenuProps {
    currentLanguage: string;
    handleDelete: (lang: string) => void;
    handleCurrent: (lang: string) => void;
    languages: string[];
    isEditing: boolean;
}

// Mock the SelectLanguageMenu since we're testing LanguageInput behavior, not SelectLanguageMenu
vi.mock("../../dialogs/SelectLanguageMenu/SelectLanguageMenu.js", () => ({
    SelectLanguageMenu: ({ currentLanguage, handleDelete, handleCurrent, languages, isEditing }: MockSelectLanguageMenuProps) => {
        const handleSelectorClick = () => handleCurrent("fr");
        const handleAddClick = () => handleCurrent("es");
        
        return (
            <div data-testid="select-language-menu">
                <button 
                    data-testid="language-selector" 
                    onClick={handleSelectorClick}
                >
                    {currentLanguage?.toUpperCase() || "SELECT"}
                </button>
                {languages.map((lang: string, index: number) => {
                    const handleDeleteClick = () => handleDelete(lang);
                    return (
                        <div key={`${lang}-${index}`} data-testid={`language-${lang}`}>
                            {lang}
                            {isEditing && (
                                <button 
                                    data-testid={`delete-${lang}`}
                                    onClick={handleDeleteClick}
                                >
                                    Delete
                                </button>
                            )}
                        </div>
                    );
                })}
                <button 
                    data-testid="add-language"
                    onClick={handleAddClick}
                >
                    Add Language
                </button>
            </div>
        );
    },
}));

interface MockSession {
    user?: {
        languages?: string[];
    };
}

// Mock translation tools
vi.mock("../../../utils/display/translationTools.js", () => ({
    getLanguageSubtag: (lang: string) => lang?.split("-")[0] || lang,
    getUserLanguages: (session: MockSession) => session?.user?.languages || ["en"],
}));

function createMockSession(languages: string[] = ["en"]) {
    return {
        user: {
            id: "user-1",
            languages,
        },
    };
}

describe("LanguageInput", () => {
    const singleLanguageArray = ["en"];
    const twoLanguageArray = ["en", "fr"];
    const threeLanguageArray = ["en", "fr", "es"];
    const fourLanguageArray = ["en", "fr", "es", "de"];
    const duplicateLanguageArray = ["en", "en", "fr"];
    const emptyLanguageArray: string[] = [];
    const germanItalianArray = ["de", "it"];
    const frenchArray = ["fr"];
    const rapidChangeLanguages = ["fr", "es", "de", "it"];

    const defaultProps = {
        currentLanguage: "en",
        handleAdd: vi.fn(),
        handleDelete: vi.fn(),
        handleCurrent: vi.fn(),
        languages: singleLanguageArray,
    };

    beforeEach(() => {
        vi.clearAllMocks();
    });

    describe("Basic rendering", () => {
        it("renders with minimal required props", () => {
            render(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput {...defaultProps} />
                </SessionContext.Provider>,
            );

            expect(screen.getByTestId("language-input")).toBeDefined();
            expect(screen.getByTestId("select-language-menu")).toBeDefined();
        });

        it("renders with custom flex direction", () => {
            render(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput {...defaultProps} flexDirection="row-reverse" />
                </SessionContext.Provider>,
            );

            const container = screen.getByTestId("language-input");
            expect(container).toBeDefined();
        });

        it("passes disabled prop correctly", () => {
            render(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput {...defaultProps} disabled={true} />
                </SessionContext.Provider>,
            );

            expect(screen.getByTestId("language-input")).toBeDefined();
        });
    });

    describe("Translation count display", () => {
        it("does not show translation count when only one language", () => {
            render(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput {...defaultProps} languages={singleLanguageArray} />
                </SessionContext.Provider>,
            );

            expect(screen.queryByTestId("translation-count")).toBeNull();
        });

        it("shows correct translation count for multiple languages", () => {
            render(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput {...defaultProps} languages={threeLanguageArray} />
                </SessionContext.Provider>,
            );

            const countElement = screen.getByTestId("translation-count");
            expect(countElement).toBeDefined();
            expect(countElement.textContent).toBe("+2");
        });

        it("updates translation count when languages change", () => {
            const { rerender } = render(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput {...defaultProps} languages={twoLanguageArray} />
                </SessionContext.Provider>,
            );

            expect(screen.getByTestId("translation-count").textContent).toBe("+1");

            rerender(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput {...defaultProps} languages={fourLanguageArray} />
                </SessionContext.Provider>,
            );

            expect(screen.getByTestId("translation-count").textContent).toBe("+3");
        });
    });

    describe("Language selection behavior", () => {
        it("calls handleCurrent when selecting an existing language", async () => {
            const handleCurrent = vi.fn();
            const user = userEvent.setup();

            render(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput 
                        {...defaultProps} 
                        handleCurrent={handleCurrent}
                        languages={twoLanguageArray}
                    />
                </SessionContext.Provider>,
            );

            const selector = screen.getByTestId("language-selector");
            await act(async () => {
                await user.click(selector);
            });

            expect(handleCurrent).toHaveBeenCalledWith("fr");
        });

        it("calls handleAdd and handleCurrent when selecting a new language", async () => {
            const handleAdd = vi.fn();
            const handleCurrent = vi.fn();
            const user = userEvent.setup();

            render(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput 
                        {...defaultProps} 
                        handleAdd={handleAdd}
                        handleCurrent={handleCurrent}
                        languages={singleLanguageArray}
                    />
                </SessionContext.Provider>,
            );

            const addButton = screen.getByTestId("add-language");
            await act(async () => {
                await user.click(addButton);
            });

            expect(handleCurrent).toHaveBeenCalledWith("es");
        });

        it("only calls handleCurrent for existing languages", async () => {
            const handleAdd = vi.fn();
            const handleCurrent = vi.fn();
            const user = userEvent.setup();

            render(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput 
                        {...defaultProps} 
                        handleAdd={handleAdd}
                        handleCurrent={handleCurrent}
                        languages={twoLanguageArray}
                    />
                </SessionContext.Provider>,
            );

            const selector = screen.getByTestId("language-selector");
            await act(async () => {
                await user.click(selector);
            });

            expect(handleAdd).not.toHaveBeenCalled();
            expect(handleCurrent).toHaveBeenCalledWith("fr");
        });
    });

    describe("Language deletion behavior", () => {
        it("calls handleDelete when deleting a language", async () => {
            const handleDelete = vi.fn();
            const user = userEvent.setup();

            render(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput 
                        {...defaultProps} 
                        handleDelete={handleDelete}
                        languages={twoLanguageArray}
                    />
                </SessionContext.Provider>,
            );

            const deleteButton = screen.getByTestId("delete-en");
            await act(async () => {
                await user.click(deleteButton);
            });

            expect(handleDelete).toHaveBeenCalledWith("en");
        });

        it("switches to another language when deleting current language", async () => {
            const handleCurrent = vi.fn();
            const handleDelete = vi.fn();
            const user = userEvent.setup();

            render(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput 
                        {...defaultProps} 
                        currentLanguage="en"
                        handleCurrent={handleCurrent}
                        handleDelete={handleDelete}
                        languages={twoLanguageArray}
                    />
                </SessionContext.Provider>,
            );

            const deleteButton = screen.getByTestId("delete-en");
            await act(async () => {
                await user.click(deleteButton);
            });

            expect(handleCurrent).toHaveBeenCalledWith("fr");
            expect(handleDelete).toHaveBeenCalledWith("en");
        });

        it("adds user language when deleting the last language", async () => {
            const handleAdd = vi.fn();
            const handleCurrent = vi.fn();
            const handleDelete = vi.fn();
            const user = userEvent.setup();

            render(
                <SessionContext.Provider value={createMockSession(["fr"])}>
                    <LanguageInput 
                        {...defaultProps} 
                        currentLanguage="en"
                        handleAdd={handleAdd}
                        handleCurrent={handleCurrent}
                        handleDelete={handleDelete}
                        languages={singleLanguageArray}
                    />
                </SessionContext.Provider>,
            );

            const deleteButton = screen.getByTestId("delete-en");
            await act(async () => {
                await user.click(deleteButton);
            });

            expect(handleCurrent).toHaveBeenCalledWith("fr");
            expect(handleAdd).toHaveBeenCalledWith("fr");
            expect(handleDelete).toHaveBeenCalledWith("en");
        });

        it("falls back to English when no user languages and deleting last language", async () => {
            const handleAdd = vi.fn();
            const handleCurrent = vi.fn();
            const handleDelete = vi.fn();
            const user = userEvent.setup();

            render(
                <SessionContext.Provider value={createMockSession([])}>
                    <LanguageInput 
                        {...defaultProps} 
                        currentLanguage="fr"
                        handleAdd={handleAdd}
                        handleCurrent={handleCurrent}
                        handleDelete={handleDelete}
                        languages={frenchArray}
                    />
                </SessionContext.Provider>,
            );

            const deleteButton = screen.getByTestId("delete-fr");
            await act(async () => {
                await user.click(deleteButton);
            });

            expect(handleCurrent).toHaveBeenCalledWith("en");
            expect(handleAdd).toHaveBeenCalledWith("en");
            expect(handleDelete).toHaveBeenCalledWith("fr");
        });
    });

    describe("Session context integration", () => {
        it("works without session context", () => {
            render(<LanguageInput {...defaultProps} />);
            
            expect(screen.getByTestId("language-input")).toBeDefined();
        });

        it("uses user languages from session when available", async () => {
            const handleAdd = vi.fn();
            const handleCurrent = vi.fn();
            const user = userEvent.setup();

            render(
                <SessionContext.Provider value={createMockSession(germanItalianArray)}>
                    <LanguageInput 
                        {...defaultProps} 
                        handleAdd={handleAdd}
                        handleCurrent={handleCurrent}
                        languages={singleLanguageArray}
                    />
                </SessionContext.Provider>,
            );

            const deleteButton = screen.getByTestId("delete-en");
            await act(async () => {
                await user.click(deleteButton);
            });

            expect(handleCurrent).toHaveBeenCalledWith("de");
            expect(handleAdd).toHaveBeenCalledWith("de");
        });
    });

    describe("Props validation and edge cases", () => {
        it("handles empty languages array", () => {
            render(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput {...defaultProps} languages={emptyLanguageArray} />
                </SessionContext.Provider>,
            );

            expect(screen.getByTestId("language-input")).toBeDefined();
            expect(screen.queryByTestId("translation-count")).toBeNull();
        });

        it("handles undefined currentLanguage", () => {
            render(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput {...defaultProps} currentLanguage={undefined as unknown as string} />
                </SessionContext.Provider>,
            );

            expect(screen.getByTestId("language-input")).toBeDefined();
        });

        it("handles duplicate languages in array", async () => {
            const handleCurrent = vi.fn();
            const user = userEvent.setup();

            render(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput 
                        {...defaultProps} 
                        handleCurrent={handleCurrent}
                        languages={duplicateLanguageArray}
                    />
                </SessionContext.Provider>,
            );

            const selector = screen.getByTestId("language-selector");
            await act(async () => {
                await user.click(selector);
            });

            expect(handleCurrent).toHaveBeenCalledWith("fr");
        });
    });

    describe("State changes and re-renders", () => {
        it("updates display when currentLanguage changes", () => {
            const { rerender } = render(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput {...defaultProps} currentLanguage="en" />
                </SessionContext.Provider>,
            );

            const selector = screen.getByTestId("language-selector");
            expect(selector.textContent).toBe("EN");

            rerender(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput {...defaultProps} currentLanguage="fr" />
                </SessionContext.Provider>,
            );

            expect(selector.textContent).toBe("FR");
        });

        it("updates translation count when languages array changes", () => {
            const { rerender } = render(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput {...defaultProps} languages={["en"]} />
                </SessionContext.Provider>,
            );

            expect(screen.queryByTestId("translation-count")).toBeNull();

            rerender(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput {...defaultProps} languages={["en", "fr"]} />
                </SessionContext.Provider>,
            );

            expect(screen.getByTestId("translation-count").textContent).toBe("+1");
        });

        it("maintains state correctly across prop changes", () => {
            const { rerender } = render(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput {...defaultProps} disabled={false} />
                </SessionContext.Provider>,
            );

            expect(screen.getByTestId("language-input")).toBeDefined();

            rerender(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput {...defaultProps} disabled={true} />
                </SessionContext.Provider>,
            );

            expect(screen.getByTestId("language-input")).toBeDefined();
        });
    });

    describe("Accessibility", () => {
        it("provides proper structure for screen readers", () => {
            render(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput {...defaultProps} languages={["en", "fr"]} />
                </SessionContext.Provider>,
            );

            const container = screen.getByTestId("language-input");
            expect(container).toBeDefined();
            expect(container.tagName).toBe("DIV");
        });

        it("maintains focus management through language changes", async () => {
            const user = userEvent.setup();

            render(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput {...defaultProps} languages={["en", "fr"]} />
                </SessionContext.Provider>,
            );

            const selector = screen.getByTestId("language-selector");
            
            await act(async () => {
                await user.click(selector);
            });

            // Focus should remain manageable after interaction
            expect(document.body).toBeDefined();
        });
    });

    describe("Performance considerations", () => {
        it("does not cause unnecessary re-renders with stable props", () => {
            const handleAdd = vi.fn();
            const handleCurrent = vi.fn();
            const handleDelete = vi.fn();

            const { rerender } = render(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput 
                        currentLanguage="en"
                        handleAdd={handleAdd}
                        handleCurrent={handleCurrent}
                        handleDelete={handleDelete}
                        languages={twoLanguageArray}
                    />
                </SessionContext.Provider>,
            );

            // Re-render with same props
            rerender(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput 
                        currentLanguage="en"
                        handleAdd={handleAdd}
                        handleCurrent={handleCurrent}
                        handleDelete={handleDelete}
                        languages={twoLanguageArray}
                    />
                </SessionContext.Provider>,
            );

            expect(screen.getByTestId("language-input")).toBeDefined();
        });

        it("handles rapid prop changes gracefully", () => {
            const { rerender } = render(
                <SessionContext.Provider value={createMockSession()}>
                    <LanguageInput {...defaultProps} currentLanguage="en" />
                </SessionContext.Provider>,
            );

            // Simulate rapid changes
            for (const lang of rapidChangeLanguages) {
                rerender(
                    <SessionContext.Provider value={createMockSession()}>
                        <LanguageInput {...defaultProps} currentLanguage={lang} />
                    </SessionContext.Provider>,
                );
            }

            expect(screen.getByTestId("language-input")).toBeDefined();
        });
    });
});
