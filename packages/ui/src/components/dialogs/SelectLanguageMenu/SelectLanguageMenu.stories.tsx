import type { Meta, StoryObj } from "@storybook/react";
import { useState } from "react";
import { signedInNoPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { Button } from "../../buttons/Button.js";
import { SelectLanguageMenu } from "./SelectLanguageMenu.js";
import { centeredDecorator } from "../../../__test/helpers/storybookDecorators.tsx";

const meta: Meta<typeof SelectLanguageMenu> = {
    title: "Components/Dialogs/SelectLanguageMenu",
    component: SelectLanguageMenu,
    parameters: {
        layout: "fullscreen",
        backgrounds: { disable: true },
        docs: {
            story: {
                inline: false,
                iframeHeight: 600,
            },
        },
        session: signedInNoPremiumWithCreditsSession,
    },
    tags: ["autodocs"],
    argTypes: {
        currentLanguage: {
            control: { type: "select" },
            options: ["en", "es", "fr", "de", "ja", "zh", "ko", "pt", "ru", "ar"],
            description: "Currently selected language",
        },
        languages: {
            control: { type: "object" },
            description: "Array of available languages",
        },
        isEditing: {
            control: { type: "boolean" },
            description: "Whether in editing mode",
        },
        handleCurrent: {
            action: "language-selected",
            description: "Callback when language is selected",
        },
        handleDelete: {
            action: "language-deleted",
            description: "Callback when language is deleted",
        },
        sxs: {
            control: { type: "object" },
            description: "Custom styles",
        },
    },
    decorators: [centeredDecorator],
};

export default meta;
type Story = StoryObj<typeof meta>;

// Showcase with controls
export const Showcase: Story = {
    render: (args) => {
        const [currentLanguage, setCurrentLanguage] = useState(args.currentLanguage || "en");

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <div style={{ padding: "20px", backgroundColor: "var(--background-paper)", borderRadius: "8px" }}>
                    <h3 style={{ marginBottom: "10px" }}>Language Menu</h3>
                    <SelectLanguageMenu
                        {...args}
                        currentLanguage={currentLanguage}
                        handleCurrent={(lang) => {
                            setCurrentLanguage(lang);
                            args.handleCurrent?.(lang);
                        }}
                    />
                </div>
                <div style={{ textAlign: "center" }}>
                    <p>Click the language selector above to see the menu</p>
                    <p>Current: <strong>{currentLanguage}</strong></p>
                </div>
            </div>
        );
    },
    args: {
        currentLanguage: "en",
        languages: ["en", "es", "fr"],
        isEditing: false,
    },
};

// Single language (not editing)
export const SingleLanguage: Story = {
    render: () => {
        const [currentLanguage, setCurrentLanguage] = useState("en");

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <div style={{ padding: "20px", backgroundColor: "var(--background-paper)", borderRadius: "8px" }}>
                    <h3 style={{ marginBottom: "10px" }}>Single Language (Read-only)</h3>
                    <SelectLanguageMenu
                        currentLanguage={currentLanguage}
                        languages={["en"]}
                        isEditing={false}
                        handleCurrent={setCurrentLanguage}
                    />
                </div>
                <div style={{ textAlign: "center" }}>
                    <p>Only one language available, no editing</p>
                    <p>Current: <strong>{currentLanguage}</strong></p>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Language menu with only one language available and not in editing mode.",
            },
        },
    },
};

// Multiple languages (not editing)
export const MultipleLanguagesReadOnly: Story = {
    render: () => {
        const [currentLanguage, setCurrentLanguage] = useState("en");

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <div style={{ padding: "20px", backgroundColor: "var(--background-paper)", borderRadius: "8px" }}>
                    <h3 style={{ marginBottom: "10px" }}>Multiple Languages (Read-only)</h3>
                    <SelectLanguageMenu
                        currentLanguage={currentLanguage}
                        languages={["en", "es", "fr", "de", "ja"]}
                        isEditing={false}
                        handleCurrent={setCurrentLanguage}
                    />
                </div>
                <div style={{ textAlign: "center" }}>
                    <p>Multiple languages available, no editing</p>
                    <p>Current: <strong>{currentLanguage}</strong></p>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Language menu with multiple languages but not in editing mode - no delete buttons shown.",
            },
        },
    },
};

// Editing mode with multiple languages
export const EditingMode: Story = {
    render: () => {
        const [currentLanguage, setCurrentLanguage] = useState("en");
        const [languages, setLanguages] = useState(["en", "es", "fr", "de", "ja", "zh"]);

        const handleDelete = (languageToDelete: string) => {
            if (languages.length > 1) {
                const newLanguages = languages.filter(lang => lang !== languageToDelete);
                setLanguages(newLanguages);
                
                // If we deleted the current language, switch to the first available
                if (languageToDelete === currentLanguage) {
                    setCurrentLanguage(newLanguages[0]);
                }
            }
        };

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <div style={{ padding: "20px", backgroundColor: "var(--background-paper)", borderRadius: "8px" }}>
                    <h3 style={{ marginBottom: "10px" }}>Editing Mode</h3>
                    <SelectLanguageMenu
                        currentLanguage={currentLanguage}
                        languages={languages}
                        isEditing={true}
                        handleCurrent={setCurrentLanguage}
                        handleDelete={handleDelete}
                    />
                </div>
                <div style={{ textAlign: "center" }}>
                    <p>Click language selector to add/remove languages</p>
                    <p>Current: <strong>{currentLanguage}</strong></p>
                    <p>Available: {languages.join(", ")}</p>
                    <Button 
                        variant="outline" 
                        size="sm" 
                        onClick={() => setLanguages(["en", "es", "fr", "de", "ja", "zh"])}
                    >
                        Reset Languages
                    </Button>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Language menu in editing mode - shows delete buttons and allows adding new languages.",
            },
        },
    },
};

// Popular languages preset
export const PopularLanguages: Story = {
    render: () => {
        const [currentLanguage, setCurrentLanguage] = useState("en");
        const [languages, setLanguages] = useState(["en", "es", "fr", "de", "ja", "zh", "ko", "pt", "ru", "ar"]);

        const handleDelete = (languageToDelete: string) => {
            if (languages.length > 1) {
                const newLanguages = languages.filter(lang => lang !== languageToDelete);
                setLanguages(newLanguages);
                
                if (languageToDelete === currentLanguage) {
                    setCurrentLanguage(newLanguages[0]);
                }
            }
        };

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <div style={{ padding: "20px", backgroundColor: "var(--background-paper)", borderRadius: "8px" }}>
                    <h3 style={{ marginBottom: "10px" }}>Popular Languages</h3>
                    <SelectLanguageMenu
                        currentLanguage={currentLanguage}
                        languages={languages}
                        isEditing={true}
                        handleCurrent={setCurrentLanguage}
                        handleDelete={handleDelete}
                    />
                </div>
                <div style={{ textAlign: "center" }}>
                    <p>Menu with many popular languages</p>
                    <p>Current: <strong>{currentLanguage}</strong></p>
                    <p>Total languages: {languages.length}</p>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Language menu with many popular world languages for testing search and scroll functionality.",
            },
        },
    },
};

// Custom styling
export const CustomStyled: Story = {
    render: () => {
        const [currentLanguage, setCurrentLanguage] = useState("en");

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <div style={{ padding: "20px", backgroundColor: "var(--background-paper)", borderRadius: "8px" }}>
                    <h3 style={{ marginBottom: "10px" }}>Custom Styled</h3>
                    <SelectLanguageMenu
                        currentLanguage={currentLanguage}
                        languages={["en", "es", "fr", "de"]}
                        isEditing={true}
                        handleCurrent={setCurrentLanguage}
                        handleDelete={(lang) => console.log("Delete", lang)}
                        sxs={{
                            root: {
                                border: "2px solid #2196f3",
                                backgroundColor: "#e3f2fd",
                                "&:hover": {
                                    backgroundColor: "#bbdefb",
                                },
                            },
                        }}
                    />
                </div>
                <div style={{ textAlign: "center" }}>
                    <p>Custom styled language selector</p>
                    <p>Current: <strong>{currentLanguage}</strong></p>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Language menu with custom styling applied through the sxs prop.",
            },
        },
    },
};

// Minimal languages
export const MinimalLanguages: Story = {
    render: () => {
        const [currentLanguage, setCurrentLanguage] = useState("en");
        const [languages, setLanguages] = useState(["en", "es"]);

        const handleDelete = (languageToDelete: string) => {
            if (languages.length > 1) {
                const newLanguages = languages.filter(lang => lang !== languageToDelete);
                setLanguages(newLanguages);
                
                if (languageToDelete === currentLanguage) {
                    setCurrentLanguage(newLanguages[0]);
                }
            }
        };

        return (
            <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "20px" }}>
                <div style={{ padding: "20px", backgroundColor: "var(--background-paper)", borderRadius: "8px" }}>
                    <h3 style={{ marginBottom: "10px" }}>Minimal Languages</h3>
                    <SelectLanguageMenu
                        currentLanguage={currentLanguage}
                        languages={languages}
                        isEditing={true}
                        handleCurrent={setCurrentLanguage}
                        handleDelete={handleDelete}
                    />
                </div>
                <div style={{ textAlign: "center" }}>
                    <p>Only 2 languages - test edge cases</p>
                    <p>Current: <strong>{currentLanguage}</strong></p>
                    <p>Try to delete one language</p>
                </div>
            </div>
        );
    },
    parameters: {
        docs: {
            description: {
                story: "Language menu with minimal languages to test edge cases like preventing deletion of the last language.",
            },
        },
    },
};
