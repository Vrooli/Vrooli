import { Component } from "react";
import { Icon } from "../../icons/Icons.js";
import { type ErrorBoundaryProps } from "../../views/types.js";

interface ErrorBoundaryState {
    hasError: boolean;
    error: Error | null;
    componentStack?: string;
}

type FormErrorBoundaryProps = ErrorBoundaryProps & {
    onError?: () => unknown;
};

// Base classes that don't depend on theme
const errorBoundaryBaseClasses = [
    "tw-rounded-lg",
    "tw-p-4",
    "tw-my-4",
    "tw-shadow-sm",
    "tw-transition-all",
    "tw-duration-200",
    "tw-max-w-full",
    "tw-overflow-hidden",
    "tw-border",
].join(" ");

const headerClasses = [
    "tw-flex",
    "tw-items-center",
    "tw-mb-3",
    "tw-gap-2",
].join(" ");

const iconClasses = [
    "tw-text-xl",
    "tw-flex-shrink-0",
].join(" ");

const titleClasses = [
    "tw-text-lg",
    "tw-font-medium",
    "tw-m-0",
].join(" ");

const messageClasses = [
    "tw-mb-3",
    "tw-leading-relaxed",
    "tw-text-sm",
].join(" ");

const detailsBaseClasses = [
    "tw-rounded",
    "tw-text-xs",
    "tw-font-mono",
    "tw-overflow-hidden",
    "tw-border",
].join(" ");

const summaryBaseClasses = [
    "tw-cursor-pointer",
    "tw-text-sm",
    "tw-font-medium",
    "tw-select-none",
    "tw-transition-colors",
    "tw-p-2",
    "tw-rounded",
    "tw-inline-block",
].join(" ");

const detailsContentBaseClasses = [
    "tw-p-3",
    "tw-overflow-auto",
    "tw-max-h-40",
    "tw-break-words",
    "tw-whitespace-pre-wrap",
].join(" ");

// Add a style element to handle theme-aware colors
const themeStyles = `
    .error-boundary-container {
        background-color: var(--background-paper);
        border-color: var(--danger-main);
        box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12), 0 1px 2px rgba(0, 0, 0, 0.24);
    }
    
    .error-boundary-container[data-has-error="true"] {
        background-color: color-mix(in srgb, var(--danger-main) 10%, var(--background-paper));
    }
    
    .error-boundary-icon {
        color: var(--danger-main);
    }
    
    .error-boundary-title {
        color: var(--text-primary);
    }
    
    .error-boundary-message {
        color: var(--text-secondary);
    }
    
    .error-boundary-details {
        background-color: color-mix(in srgb, var(--danger-main) 5%, var(--background-paper));
        border-color: color-mix(in srgb, var(--danger-main) 20%, var(--background-paper));
    }
    
    .error-boundary-summary {
        color: var(--danger-dark, var(--danger-main));
    }
    
    .error-boundary-summary:hover {
        background-color: color-mix(in srgb, var(--danger-main) 10%, var(--background-paper));
    }
    
    .error-boundary-details-content {
        color: var(--text-secondary);
    }
    
    /* Fallback for browsers that don't support color-mix */
    @supports not (background-color: color-mix(in srgb, red 50%, blue)) {
        .error-boundary-container[data-has-error="true"] {
            background-color: var(--danger-light);
        }
        
        .error-boundary-details {
            background-color: var(--danger-light);
            opacity: 0.3;
        }
        
        .error-boundary-summary:hover {
            background-color: var(--danger-light);
            opacity: 0.5;
        }
    }
`;

/**
 * Displays an error message if a form throws an error.
 * 
 * NOTE: Cannot be a functional component. See https://legacy.reactjs.org/docs/error-boundaries.html#introducing-error-boundaries
 */
export class FormErrorBoundary extends Component<FormErrorBoundaryProps, ErrorBoundaryState> {
    private static stylesInjected = false;

    constructor(props: FormErrorBoundaryProps) {
        super(props);
        this.state = { hasError: false, error: null, componentStack: undefined };
    }

    componentDidMount() {
        // Inject styles only once
        if (!FormErrorBoundary.stylesInjected) {
            const styleElement = document.getElementById("form-error-boundary-styles");
            if (!styleElement) {
                const style = document.createElement("style");
                style.id = "form-error-boundary-styles";
                style.innerHTML = themeStyles;
                document.head.appendChild(style);
                FormErrorBoundary.stylesInjected = true;
            }
        }
    }

    static getDerivedStateFromError(error: Error) {
        // Update state so the next render will show the fallback UI.
        return { hasError: true, error };
    }

    componentDidCatch(error: Error, errorInfo: React.ErrorInfo): void {
        // You can also log the error to an error reporting service
        console.error("Error caught by Error Boundary: ", error, errorInfo);
        // Store componentStack from errorInfo
        this.setState({ componentStack: errorInfo.componentStack ?? undefined });
        // Call the provided function to close any open popovers or other cleanup tasks
        if (this.props.onError) {
            this.props.onError();
        }
    }

    render() {
        if (this.state.hasError) {
            return (
                <div
                    data-testid="form-error-boundary"
                    data-has-error="true"
                    role="alert"
                    className={`${errorBoundaryBaseClasses} error-boundary-container`}
                >
                        <div
                            data-testid="error-header"
                            className={headerClasses}
                        >
                            <Icon
                                data-testid="error-icon"
                                className={`${iconClasses} error-boundary-icon`}
                                info={{ name: "Error", type: "Common" }}
                                decorative="false"
                                fill="currentColor"
                                size={20}
                            />
                            <h2 className={`${titleClasses} error-boundary-title`}>
                                An error occurred
                            </h2>
                        </div>
                        <p 
                            data-testid="error-message"
                            className={`${messageClasses} error-boundary-message`}
                        >
                            There was a problem rendering the form. Please try refreshing the page or contact support if the issue persists.
                        </p>
                        <details
                            data-testid="error-details"
                            className={`${detailsBaseClasses} error-boundary-details`}
                        >
                            <summary 
                                className={`${summaryBaseClasses} error-boundary-summary`}
                            >
                                View Technical Details
                            </summary>
                            <div 
                                className={`${detailsContentBaseClasses} error-boundary-details-content`}
                            >
                                <strong>Error:</strong>
                                <div className="tw-mt-1 tw-mb-3">
                                    {this.state.error && this.state.error.toString()}
                                </div>
                                {this.state.componentStack && (
                                    <>
                                        <strong>Component Stack:</strong>
                                        <div className="tw-mt-1">
                                            {this.state.componentStack}
                                        </div>
                                    </>
                                )}
                            </div>
                        </details>
                </div>
            );
        }

        return (
            <div 
                data-testid="form-error-boundary" 
                data-has-error="false"
                className="tw-contents"
            >
                {this.props.children}
            </div>
        );
    }
}
