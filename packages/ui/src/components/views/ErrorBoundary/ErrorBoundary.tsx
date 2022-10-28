import { Component } from "react";
import { ErrorBoundaryProps } from "../types";

export interface ErrorBoundaryState {
    hasError: boolean;
}

/**
 * Displays an error message if a child component throws an error.
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
    constructor(props: ErrorBoundaryProps) {
        super(props);
        this.state = { hasError: false };
    }

    static getDerivedStateFromError() {
        return { hasError: true };
    }

    render() {
        if (this.state.hasError) {
            // Show centered error message
            return (
                <div style={{
                    display: 'flex',
                    justifyContent: 'center',
                    alignItems: 'center',
                    height: '100%',
                    position: 'absolute',
                    top: 0,
                    left: 0,
                    right: 0,
                    bottom: 0,
                    paddingLeft: '16px',
                    paddingRight: '16px',
                    background: 'white',
                }}
                >
                    <div style={{ textAlign: 'center', color: 'black' }}>
                        <h1>Something went wrong 😔</h1>
                        <p>Try refreshing the page, or closing and reopening the application. If the problem persists, 
                            you may contact us at <a href="mailto:official@vrooli.com">official@vrooli.com</a> and we will try to help you as soon as possible.
                        </p>
                    </div>
                </div>
            );
        }
        return this.props.children;
    }
}