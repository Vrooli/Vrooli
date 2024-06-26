import { Component } from "react";
import { ErrorBoundaryProps } from "views/types";

interface ErrorBoundaryState {
    hasError: boolean;
    error: Error | null;
}

type FormErrorBoundaryProps = ErrorBoundaryProps & {
    onError?: () => unknown;
};

/**
 * Displays an error message if a form throws an error.
 * 
 * NOTE: Cannot be a functional component. See https://legacy.reactjs.org/docs/error-boundaries.html#introducing-error-boundaries
 */
export class FormErrorBoundary extends Component<FormErrorBoundaryProps, ErrorBoundaryState> {
    constructor(props: FormErrorBoundaryProps) {
        super(props);
        this.state = { hasError: false, error: null };
    }

    static getDerivedStateFromError(error) {
        // Update state so the next render will show the fallback UI.
        return { hasError: true, error };
    }

    componentDidCatch(error, errorInfo) {
        // You can also log the error to an error reporting service
        console.error("Error caught by Error Boundary: ", error, errorInfo);
        // Call the provided function to close any open popovers or other cleanup tasks
        if (this.props.onError) {
            this.props.onError();
        }
    }

    render() {
        if (this.state.hasError) {
            // You can render any custom fallback UI
            return (
                <div style={{ color: "#f00", padding: "8px" }}>
                    <p>Something went wrong with the form :(</p>
                    <details style={{ whiteSpace: "pre-wrap" }}>
                        {this.state.error && this.state.error.toString()}
                        <br />
                    </details>
                </div>
            );
        }

        return this.props.children;
    }
}
