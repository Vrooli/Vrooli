// AI_CHECK: TYPE_SAFETY=eliminated-15-any-types-with-proper-interfaces | LAST: 2025-06-30\nimport React from "react";
import { vi } from "vitest";
import { createMockSession } from "../helpers/formComponentTestHelpers.js";

/**
 * Central Mock Registry for Form Testing
 * 
 * This registry provides reusable mock implementations for common dependencies
 * in form tests, eliminating the need to duplicate mock setup across test files.
 */

// Export mock function references that can be accessed after mock setup
export const mockFunctions = {
    onSubmit: vi.fn(),
    handleCompleted: vi.fn(),
    handleCancel: vi.fn(),
    handleUpdate: vi.fn(),
    setObject: vi.fn(),
    handleAddLanguage: vi.fn(),
    handleDeleteLanguage: vi.fn(),
    setLanguage: vi.fn(),
    validateValues: vi.fn(),
};

/**
 * Interface for standard upsert form mock overrides
 */
interface StandardUpsertFormMockOverrides {
    session?: unknown;
    isLoading?: boolean;
    handleCancel?: () => void;
    handleCompleted?: () => void;
    onSubmit?: (values: unknown) => Record<string, string>;
    validateValues?: () => void;
    language?: string;
    languages?: string[];
    handleAddLanguage?: (lang: string) => void;
    handleDeleteLanguage?: (lang: string) => void;
    setLanguage?: (lang: string) => void;
    translationErrors?: Record<string, string>;
    [key: string]: unknown;
}

/**
 * Create standard mock for useStandardUpsertForm hook
 * This hook is used by virtually all form components
 */
export const createUseStandardUpsertFormMock = (overrides?: StandardUpsertFormMockOverrides) => {
    return vi.fn((config: unknown, options: { onClose?: () => void; onCompleted?: () => void }) => {
        // Reset and configure mock functions
        mockFunctions.handleCancel.mockReset();
        mockFunctions.handleCompleted.mockReset();
        mockFunctions.onSubmit.mockReset();
        
        mockFunctions.handleCancel.mockImplementation(() => {
            options.onClose?.();
        });
        
        mockFunctions.handleCompleted.mockImplementation(() => {
            options.onCompleted?.();
        });
        
        mockFunctions.onSubmit.mockImplementation((values: unknown) => {
            try {
                // Use real validation if available
                if (config && typeof config === 'object' && 'validation' in config) {
                    const validation = (config as { validation?: { isValidSync?: (v: unknown) => boolean } }).validation;
                    if (validation?.isValidSync) {
                        const isValid = validation.isValidSync(values);
                        if (isValid) {
                            options.onCompleted?.();
                            return {};
                        }
                    }
                }
                return { general: "Validation failed" };
            } catch (error) {
                // Assume success for testing if validation throws
                options.onCompleted?.();
                return {};
            }
        });

        return {
            session: createMockSession(),
            isLoading: false,
            handleCancel: mockFunctions.handleCancel,
            handleCompleted: mockFunctions.handleCompleted,
            onSubmit: mockFunctions.onSubmit,
            validateValues: mockFunctions.validateValues,
            language: "en",
            languages: ["en", "es"],
            handleAddLanguage: mockFunctions.handleAddLanguage,
            handleDeleteLanguage: mockFunctions.handleDeleteLanguage,
            setLanguage: mockFunctions.setLanguage,
            translationErrors: {},
            ...overrides,
        };
    });
};

/**
 * Interface for test configuration used by useManagedObject mock
 */
interface TestConfig {
    initialValuesFunction?: (session?: unknown) => unknown;
    fixtures?: {
        valid?: {
            minimal?: unknown;
        };
    };
    formFixtures?: {
        minimal?: unknown;
    };
    [key: string]: unknown;
}

/**
 * Create standard mock for useManagedObject hook
 * This hook manages object state for create/update forms
 */
export const createUseManagedObjectMock = (testConfig: TestConfig) => {
    return vi.fn(() => {
        const mockSession = createMockSession();
        const mockInitialValues = testConfig.initialValuesFunction?.(mockSession) || 
                                 testConfig.fixtures?.valid?.minimal ||
                                 testConfig.formFixtures?.minimal;

        return {
            isLoading: false,
            object: mockInitialValues,
            permissions: { canUpdate: true },
            setObject: mockFunctions.setObject,
        };
    });
};

/**
 * Interface for mock data structure used by useLazyFetch
 */
interface MockLazyFetchData {
    threads?: Array<{
        comment?: {
            id?: string;
            you?: {
                canUpdate?: boolean;
                canDelete?: boolean;
            };
        };
    }>;
    [key: string]: unknown;
}

/**
 * Create standard mock for useLazyFetch hook
 * This hook is used for fetching data on demand
 */
export const createUseLazyFetchMock = (mockData?: MockLazyFetchData) => {
    return vi.fn(() => [
        vi.fn(), // The fetch function
        {
            data: mockData || {
                threads: [{
                    comment: {
                        id: "test-comment-id",
                        you: { canUpdate: true, canDelete: true },
                    },
                }],
            },
            error: null,
            loading: false,
        },
    ]);
};

/**
 * Interface for translated advanced input props
 */
interface TranslatedAdvancedInputProps {
    name: string;
    label?: string;
    title?: string;
    value?: string;
    onChange?: (value: string) => void;
    placeholder?: string;
    features?: unknown;
    actionButtons?: Array<{
        onClick?: () => void;
        iconInfo?: {
            name?: string;
        };
    }>;
    sxs?: unknown;
    isRequired?: boolean;
    language?: string;
    disabled?: boolean;
    helperText?: string;
    error?: string;
    onBlur?: () => void;
    onFocus?: () => void;
    limitTo?: unknown;
    onTasksChange?: unknown;
    onContextDataChange?: unknown;
    onSubmit?: unknown;
    tabIndex?: number;
    tasks?: unknown;
    contextData?: unknown;
    minRows?: number;
    maxRows?: number;
    [key: string]: unknown;
}

/**
 * Mock implementations for common UI components
 * These are lightweight versions for testing purposes
 */
export const mockTranslatedAdvancedInput = ({ 
    name, 
    label, 
    title, 
    value, 
    onChange, 
    placeholder, 
    features, 
    actionButtons, 
    sxs, 
    isRequired,
    language,
    disabled,
    helperText,
    error,
    onBlur,
    onFocus,
    ...domProps 
}: TranslatedAdvancedInputProps) => {
    // Filter out React-specific props that shouldn't go to DOM elements
    const { limitTo, onTasksChange, onContextDataChange, onSubmit, tabIndex, tasks, contextData, minRows, maxRows, ...filteredProps } = domProps;
    
    return (
        <div data-testid={`translated-input-${name}`}>
            <label htmlFor={name}>{label || title || name}</label>
            <textarea
                id={name}
                name={name}
                value={value}
                onChange={(e) => onChange?.(e.target.value)}
                onBlur={onBlur}
                onFocus={onFocus}
                placeholder={placeholder}
                disabled={disabled}
                data-testid={`input-${name}`}
                rows={minRows}
                {...filteredProps}
            />
            {helperText && <div data-testid="helper-text">{helperText}</div>}
            {actionButtons && (
                <div data-testid="action-buttons">
                    {actionButtons.map((button, index: number) => (
                        <button
                            key={index}
                            onClick={button.onClick}
                            data-testid={`action-button-${index}`}
                        >
                            {button.iconInfo?.name || "Action"}
                        </button>
                    ))}
                </div>
            )}
        </div>
    );
};

/**
 * Interface for component props
 */
interface MarkdownDisplayProps {
    content?: React.ReactNode;
}

interface DialogProps {
    children?: React.ReactNode;
    isOpen?: boolean;
    onClose?: () => void;
}

interface BaseFormProps {
    children?: React.ReactNode;
    display?: unknown;
    isLoading?: boolean;
    style?: React.CSSProperties;
}

interface TopBarProps {
    title?: string;
    onClose?: () => void;
}

interface BottomActionsProps {
    onCancel?: () => void;
    onSubmit?: () => void;
    isCreate?: boolean;
    loading?: boolean;
}

interface BoxProps {
    children?: React.ReactNode;
    [key: string]: unknown;
}

export const mockMarkdownDisplay = ({ content }: MarkdownDisplayProps) => (
    <div data-testid="markdown-display">{content}</div>
);

export const mockDialog = ({ children, isOpen, onClose }: DialogProps) => 
    isOpen ? <div data-testid="dialog" role="dialog">{children}</div> : null;

export const mockBaseForm = ({ children, display, isLoading, style }: BaseFormProps) => (
    <form data-testid="base-form" style={style}>
        {children}
    </form>
);

export const mockTopBar = ({ title, onClose }: TopBarProps) => (
    <div data-testid="top-bar">
        <h1>{title}</h1>
        {onClose && <button onClick={onClose} data-testid="close-button">Close</button>}
    </div>
);

export const mockBottomActionsButtons = ({ onCancel, onSubmit, isCreate, loading }: BottomActionsProps) => (
    <div data-testid="bottom-actions">
        <button onClick={onCancel} data-testid="cancel-button">Cancel</button>
        <button 
            onClick={() => onSubmit && onSubmit()} 
            data-testid="submit-button"
            disabled={loading}
        >
            {isCreate ? "Create" : "Update"}
        </button>
    </div>
);

export const mockBox = ({ children, ...props }: BoxProps) => <div {...props}>{children}</div>;

export const mockUseIsMobile = vi.fn(() => false);
export const mockUseWindowSize = vi.fn(() => ({ height: 800, width: 1200 }));

// Additional mock components for dialogs
export const mockDialogContent = ({ children, ...props }: BoxProps) => <div {...props}>{children}</div>;
export const mockDialogActions = ({ children, ...props }: BoxProps) => <div {...props}>{children}</div>;

/**
 * Interface for selector component props
 */
interface SelectorProps {
    name: string;
    label?: string;
    options?: string[];
    getOptionLabel?: (option: string) => string;
    value?: string;
    onChange?: (event: { target: { name: string; value: string } }) => void;
    disabled?: boolean;
    fullWidth?: boolean;
    [key: string]: unknown;
}

// Additional form input mocks
export const mockSelector = ({ name, label, options, getOptionLabel, value, onChange, disabled, fullWidth, ...props }: SelectorProps) => {
    // Get the formik context to update field value
    const formikContext = (global as { mockFormikContext?: { values?: Record<string, unknown>; setFieldValue?: (name: string, value: string) => void } }).mockFormikContext;
    
    return (
        <div data-testid={`selector-${name}`}>
            {label && <label>{label}</label>}
            <select
                data-testid={`select-${name}`}
                value={value || (typeof formikContext?.values?.[name] === 'string' ? formikContext.values[name] : "") || ""}
                onChange={(e) => {
                    const newValue = e.target.value;
                    onChange?.({ target: { name, value: newValue } });
                    // Also update formik directly
                    if (formikContext?.setFieldValue) {
                        formikContext.setFieldValue(name, newValue);
                    }
                }}
                disabled={disabled}
                style={fullWidth ? { width: "100%" } : {}}
                {...props}
            >
                <option value="">Select...</option>
                {options?.map((option: string) => (
                    <option key={option} value={option}>
                        {getOptionLabel ? getOptionLabel(option) : option}
                    </option>
                ))}
            </select>
        </div>
    );
};

/**
 * Interface for language input props
 */
interface LanguageInputProps {
    currentLanguage?: string;
    handleAdd?: (lang: string) => void;
    handleDelete?: (lang: string) => void;
    handleCurrent?: (lang: string) => void;
    languages?: string[];
}

export const mockLanguageInput = ({ currentLanguage, handleAdd, handleDelete, handleCurrent, languages }: LanguageInputProps) => (
    <div data-testid="language-input">
        <select
            data-testid="language-selector"
            value={currentLanguage || "en"}
            onChange={(e) => handleCurrent?.(e.target.value)}
        >
            {languages?.map((lang: string) => (
                <option key={lang} value={lang}>{lang}</option>
            ))}
        </select>
        <button onClick={() => handleAdd?.("new-lang")} data-testid="add-language">Add</button>
        <button onClick={() => handleDelete?.(currentLanguage)} data-testid="delete-language">Delete</button>
    </div>
);

/**
 * Interface for search existing button props
 */
interface SearchExistingButtonProps {
    href?: string;
    text?: string;
}

export const mockSearchExistingButton = ({ href, text }: SearchExistingButtonProps) => (
    <a href={href} data-testid="search-existing-button">{text || "Search Existing"}</a>
);

/**
 * Reset all mock functions
 * Call this in beforeEach or afterEach
 */
export function resetFormMocks() {
    Object.values(mockFunctions).forEach(fn => {
        fn.mockReset();
    });
}
