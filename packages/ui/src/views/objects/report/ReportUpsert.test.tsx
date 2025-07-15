// AI_CHECK: TEST_QUALITY=4 | LAST: 2025-06-24
// AI_CHECK: TYPE_SAFETY=eliminated-9-any-types-in-mock-interfaces | LAST: 2025-06-28
import { DUMMY_ID, reportFormConfig } from "@vrooli/shared";
import { render, screen, waitFor } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import React from "react";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { SessionContext } from "../../../contexts/session.js";
import { renderWithProviders } from "../../../__test/testUtils.js";
import { ReportUpsert } from "./ReportUpsert.js";

// Mock dependencies that are not the focus of this behavioral test
vi.mock("../../../hooks/useManagedObject.js", () => ({
    useManagedObject: vi.fn(() => ({
        isLoading: false,
        object: null,
        permissions: { canUpdate: true },
        setObject: vi.fn(),
    })),
}));

const mockHandleCancel = vi.fn();
const mockHandleCompleted = vi.fn();
const mockOnSubmit = vi.fn();

vi.mock("../../../hooks/useStandardUpsertForm.js", () => ({
    useStandardUpsertForm: vi.fn(() => ({
        session: { id: DUMMY_ID, users: [{ id: DUMMY_ID, languages: ["en", "es"] }] },
        isLoading: false,
        handleCancel: mockHandleCancel,
        handleCompleted: mockHandleCompleted,
        onSubmit: mockOnSubmit,
        validateValues: vi.fn(),
        language: "en",
        languages: ["en", "es"],
        handleAddLanguage: vi.fn(),
        handleDeleteLanguage: vi.fn(),
        setLanguage: vi.fn(),
        translationErrors: {},
    })),
}));

// Mock Formik's useField hook
vi.mock("formik", async () => {
    const actual = await vi.importActual("formik");
    return {
        ...actual,
        useField: vi.fn((name: string) => [
            { value: "", name }, // field props
            { error: undefined, touched: false }, // meta props  
            { setValue: vi.fn() }, // helpers
        ]),
    };
});

interface SearchExistingButtonProps {
    children?: React.ReactNode;
    onClick?: () => void;
}

vi.mock("../../../components/buttons/SearchExistingButton/SearchExistingButton.js", () => ({
    SearchExistingButton: ({ children, onClick }: SearchExistingButtonProps) => (
        <button data-testid="search-existing-button" onClick={onClick}>
            {children || "Search Existing"}
        </button>
    ),
}));

interface TranslatedAdvancedInputProps {
    name: string;
    placeholder?: string;
    label?: string;
    isRequired?: boolean;
    disabled?: boolean;
    features?: unknown;
    language?: string;
    limitTo?: unknown;
    onTasksChange?: unknown;
    onContextDataChange?: unknown;
    onSubmit?: unknown;
    tabIndex?: number;
    tasks?: unknown;
    contextData?: unknown;
    [key: string]: unknown;
}

vi.mock("../../../components/inputs/AdvancedInput/AdvancedInput.js", () => ({
    TranslatedAdvancedInput: ({ name, placeholder, label, isRequired, disabled, ...domProps }: TranslatedAdvancedInputProps) => {
        // Filter out React-specific props that shouldn't go to DOM elements
        const { features, language, limitTo, onTasksChange, onContextDataChange, onSubmit, tabIndex, tasks, contextData, ...filteredProps } = domProps;
        
        return (
            <div data-testid={`translated-input-${name}`}>
                <label htmlFor={name}>{label || name}</label>
                <textarea 
                    id={name} 
                    name={name} 
                    placeholder={placeholder}
                    disabled={disabled}
                    data-testid={`input-${name}`}
                    {...filteredProps}
                />
            </div>
        );
    },
}));

interface LanguageInputProps {
    currentLanguage: string;
    handleAdd?: (language: string) => void;
    handleDelete?: (language: string) => void;
    handleCurrent?: (language: string) => void;
    languages?: string[];
}

vi.mock("../../../components/inputs/LanguageInput/LanguageInput.js", () => ({
    LanguageInput: ({ currentLanguage, handleAdd, handleDelete, handleCurrent, languages }: LanguageInputProps) => (
        <div data-testid="language-input">
            <select 
                data-testid="language-selector" 
                value={currentLanguage}
                onChange={(e) => handleCurrent?.(e.target.value)}
            >
                {languages?.map((lang) => (
                    <option key={lang} value={lang}>{lang}</option>
                )) || <option value="en">en</option>}
            </select>
            <button onClick={() => handleAdd?.("es")}>Add Language</button>
            <button onClick={() => handleDelete?.(currentLanguage)}>Delete Language</button>
        </div>
    ),
}));

interface SelectorProps {
    name: string;
    options?: Array<{ value: string; label: string } | string>;
    value?: string;
    onChange?: (event: React.ChangeEvent<HTMLSelectElement>) => void;
    label?: string;
    getOptionLabel?: (option: unknown) => string;
    disabled?: boolean;
    fullWidth?: boolean;
    [key: string]: unknown;
}

vi.mock("../../../components/inputs/Selector/Selector.js", () => ({
    Selector: ({ name, options, value, onChange, label, getOptionLabel, disabled, fullWidth, ...props }: SelectorProps) => {
        // Filter out non-DOM props
        const { ...domProps } = props;
        return (
            <div data-testid={`selector-${name}`}>
                <label htmlFor={name}>{label || name}</label>
                <select
                    id={name}
                    name={name}
                    value={value}
                    onChange={onChange}
                    data-testid={`select-${name}`}
                    disabled={disabled}
                    {...domProps}
                >
                    <option value="">Select an option</option>
                    {options?.map((option) => {
                        const optionValue = typeof option === "string" ? option : option.value;
                        const optionLabel = typeof option === "string" ? option : (getOptionLabel ? getOptionLabel(option) : option.label);
                        return (
                            <option key={optionValue} value={optionValue}>
                                {optionLabel}
                            </option>
                        );
                    })}
                </select>
            </div>
        );
    },
}));

// Mock dialog and form containers
interface DialogProps {
    children: React.ReactNode;
    isOpen: boolean;
    onClose: () => void;
}

vi.mock("../../../components/dialogs/Dialog/Dialog.js", () => ({
    Dialog: ({ children, isOpen, onClose }: DialogProps) => 
        isOpen ? <div data-testid="report-dialog">{children}</div> : null,
}));

interface BaseFormProps {
    children: React.ReactNode;
}

vi.mock("../../../forms/BaseForm/BaseForm.js", () => ({
    BaseForm: ({ children }: BaseFormProps) => <form data-testid="base-form">{children}</form>,
}));

interface BottomActionsButtonsProps {
    onCancel: () => void;
    onSubmit: () => void;
    isCreate: boolean;
    isLoading: boolean;
}

vi.mock("../../../components/buttons/BottomActionsButtons.js", () => ({
    BottomActionsButtons: ({ onCancel, onSubmit, isCreate, isLoading }: BottomActionsButtonsProps) => (
        <div data-testid="bottom-actions">
            <button type="button" onClick={onCancel} data-testid="cancel-button">
                Cancel
            </button>
            <button 
                type="submit" 
                onClick={onSubmit} 
                disabled={isLoading}
                data-testid="submit-button"
            >
                {isCreate ? "Create Report" : "Update Report"}
            </button>
        </div>
    ),
}));

interface TopBarProps {
    display: { title: string };
    onClose: () => void;
}

vi.mock("../../../components/navigation/TopBar.js", () => ({
    TopBar: ({ display, onClose }: TopBarProps) => (
        <div data-testid="top-bar">
            <h1>{display.title}</h1>
            <button onClick={onClose} data-testid="close-button">✕</button>
        </div>
    ),
}));

// Create mock session
const mockSession = {
    id: DUMMY_ID,
    users: [{ 
        id: DUMMY_ID,
        languages: ["en", "es"],
    }],
};

const mockCreatedFor = {
    __typename: "User" as const,
    id: DUMMY_ID,
};

describe("ReportUpsert Form Behavior", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    const defaultProps = {
        isCreate: true,
        isOpen: true,
        display: "Dialog" as const,
        values: undefined,
        createdFor: mockCreatedFor,
        onCancel: vi.fn(),
        onClose: vi.fn(),
        onCompleted: vi.fn(),
        onDeleted: vi.fn(),
    };

    beforeEach(() => {
        vi.clearAllMocks();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it("should display initial form correctly", () => {
        render(
            <SessionContext.Provider value={{ session: mockSession }}>
                <ReportUpsert {...defaultProps} />
            </SessionContext.Provider>,
        );

        // Check that form elements appear
        expect(screen.getByTestId("report-dialog")).toBeTruthy();
        expect(screen.getByTestId("base-form")).toBeTruthy();
        expect(screen.getByTestId("bottom-actions")).toBeTruthy();
        expect(screen.getByTestId("submit-button")).toBeTruthy();
        expect(screen.getByTestId("cancel-button")).toBeTruthy();
    });

    it("should display report reason selector", () => {
        render(
            <SessionContext.Provider value={{ session: mockSession }}>
                <ReportUpsert {...defaultProps} />
            </SessionContext.Provider>,
        );

        // Check that reason selector appears
        const reasonSelector = screen.getByTestId("selector-reason");
        expect(reasonSelector).toBeTruthy();
        
        const selectElement = screen.getByTestId("select-reason");
        expect(selectElement).toBeTruthy();
    });

    it("should display language input", () => {
        render(
            <SessionContext.Provider value={{ session: mockSession }}>
                <ReportUpsert {...defaultProps} />
            </SessionContext.Provider>,
        );

        expect(screen.getByTestId("language-input")).toBeTruthy();
        expect(screen.getByTestId("language-selector")).toBeTruthy();
    });

    it("should display details input field", () => {
        render(
            <SessionContext.Provider value={{ session: mockSession }}>
                <ReportUpsert {...defaultProps} />
            </SessionContext.Provider>,
        );

        // Check that details input appears
        const detailsInput = screen.getByTestId("translated-input-details");
        expect(detailsInput).toBeTruthy();
        
        const textareaElement = screen.getByTestId("input-details");
        expect(textareaElement).toBeTruthy();
    });

    it("should handle reason selection", async () => {
        const user = userEvent.setup();
        
        render(
            <SessionContext.Provider value={{ session: mockSession }}>
                <ReportUpsert {...defaultProps} />
            </SessionContext.Provider>,
        );

        const reasonSelect = screen.getByTestId("select-reason");
        
        // Change reason to "Spam"
        await user.selectOptions(reasonSelect, "Spam");
        
        expect(reasonSelect.value).toBe("Spam");
    });

    it("should handle details input", async () => {
        const user = userEvent.setup();
        
        render(
            <SessionContext.Provider value={{ session: mockSession }}>
                <ReportUpsert {...defaultProps} />
            </SessionContext.Provider>,
        );

        const detailsInput = screen.getByTestId("input-details");
        
        await user.type(detailsInput, "This content contains spam links");
        
        expect(detailsInput.value).toBe("This content contains spam links");
    });

    it("should handle language switching", async () => {
        const user = userEvent.setup();
        
        render(
            <SessionContext.Provider value={{ session: mockSession }}>
                <ReportUpsert {...defaultProps} />
            </SessionContext.Provider>,
        );

        const languageSelector = screen.getByTestId("language-selector");
        
        // Should start with first language from user languages
        expect(languageSelector.value).toBe("en");
    });

    it("should handle cancel action", async () => {
        const user = userEvent.setup();
        
        render(
            <SessionContext.Provider value={{ session: mockSession }}>
                <ReportUpsert {...defaultProps} />
            </SessionContext.Provider>,
        );

        const cancelButton = screen.getByTestId("cancel-button");
        await user.click(cancelButton);
        
        expect(mockHandleCancel).toHaveBeenCalled();
    });
});
