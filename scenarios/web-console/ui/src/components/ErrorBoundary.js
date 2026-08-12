import { jsx as _jsx } from "react/jsx-runtime";
// DOC: docs/concepts/ARCHITECTURE.md#file-map
// DOC: docs/internal/ERROR_SEMANTICS.md
import { Component } from "react";
import ErrorBoundaryFallback from "./ErrorBoundaryFallback";
/**
 * React Error Boundary that isolates runtime crashes to a UI region.
 *
 * Place around major sections (workspace, terminal panes, drawers) so that
 * a crash in one area does not take down the entire application.
 */
export default class ErrorBoundary extends Component {
    constructor() {
        super(...arguments);
        this.state = { error: null };
        this.handleReset = () => {
            this.setState({ error: null });
        };
    }
    static getDerivedStateFromError(error) {
        return { error };
    }
    componentDidCatch(error, info) {
        console.error(`[ErrorBoundary:${this.props.region}]`, error, info.componentStack);
    }
    render() {
        if (this.state.error) {
            if (this.props.fallback)
                return this.props.fallback;
            return (_jsx(ErrorBoundaryFallback, { region: this.props.region, message: this.state.error.message, onReset: this.handleReset }));
        }
        return this.props.children;
    }
}
