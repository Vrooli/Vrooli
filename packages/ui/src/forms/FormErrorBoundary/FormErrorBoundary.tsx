import { Component } from "react";
import { ErrorBoundaryProps } from "../../views/types.js";

interface ErrorBoundaryState {
    hasError: boolean;
    error: Error | null;
    componentStack?: string;
}

type FormErrorBoundaryProps = ErrorBoundaryProps & {
    onError?: () => unknown;
};

const outerBoxStyle = {
    backgroundColor: "#a00",
    border: "1px solid #ffcccc",
    borderRadius: "4px",
    padding: "16px",
    margin: "16px 0",
    color: "white",
} as const;
const warningBoxStyle = {
    display: "flex",
    alignItems: "center",
    marginBottom: "8px",
} as const;
const warningSpanStyle = {
    fontSize: "24px",
    marginRight: "8px",
} as const;
const errorHeaderStyle = {
    margin: 0,
} as const;
const errorMessageStyle = {
    margin: "0 0 8px 0",
} as const;
const errorDetailsStyle = {
    whiteSpace: "pre-wrap",
    fontFamily: "monospace",
    fontSize: "12px",
} as const;

/**
 * Displays an error message if a form throws an error.
 * 
 * NOTE: Cannot be a functional component. See https://legacy.reactjs.org/docs/error-boundaries.html#introducing-error-boundaries
 */
export class FormErrorBoundary extends Component<FormErrorBoundaryProps, ErrorBoundaryState> {
    constructor(props: FormErrorBoundaryProps) {
        super(props);
        this.state = { hasError: false, error: null, componentStack: undefined };
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
                    role="alert"
                    style={outerBoxStyle}
                >
                    <div
                        style={warningBoxStyle}
                    >
                        <span
                            style={warningSpanStyle}
                        >
                            ⚠️
                        </span>
                        <h2 style={errorHeaderStyle}>An error occurred</h2>
                    </div>
                    <p style={errorMessageStyle}>
                        There was a problem rendering the form. Please try refreshing the page or contact support if the issue persists.
                    </p>
                    <details
                        style={errorDetailsStyle}
                    >
                        {this.state.error && this.state.error.toString()}
                        <br />
                        {this.state.componentStack}
                    </details>
                </div>
            );
        }

        return this.props.children;
    }
}
