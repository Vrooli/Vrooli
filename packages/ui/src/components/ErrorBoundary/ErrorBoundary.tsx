import { HomeIcon, RefreshIcon, stringifySearchParams } from "@local/shared";
import { Button, Stack, Typography } from "@mui/material";
import { Component } from "react";
import { ErrorBoundaryProps } from "../../views/types";

interface ErrorBoundaryState {
    hasError: boolean;
}

interface ErrorBoundaryState {
    hasError: boolean;
    error: Error | null;
    mailToUrl: string;
}

/**
 * Displays an error message if a child component throws an error.
 * 
 * NOTE: Cannot be a functional component. See https://legacy.reactjs.org/docs/error-boundaries.html#introducing-error-boundaries
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
    constructor(props: ErrorBoundaryProps) {
        super(props);
        this.state = { hasError: false, error: null, mailToUrl: "mailto:official@vrooli.com" };
    }

    static getDerivedStateFromError(error: Error) {
        const subject = `Error Report: ${error?.name ?? ""}`;
        const body = `Error message: ${error?.toString() ?? ""}`;
        const mailToUrl = `mailto:official@vrooli.com${stringifySearchParams({ body, subject })}`;
        return { hasError: true, error, mailToUrl };
    }

    handleRefresh = () => {
        window.location.reload();
    };

    render() {
        const { hasError, error, mailToUrl } = this.state;
        if (hasError) {
            return (
                <div
                    style={{
                        display: "flex",
                        justifyContent: "center",
                        alignItems: "center",
                        height: "100%",
                        position: "absolute",
                        top: 0,
                        left: 0,
                        right: 0,
                        bottom: 0,
                        paddingLeft: "16px",
                        paddingRight: "16px",
                        backgroundColor: "white",
                        color: "black",
                    }}
                >
                    <Stack direction="column" spacing={2} style={{ textAlign: "center" }}>
                        <Typography variant="h4">Something went wrong 😔</Typography>
                        <Typography variant="body1" style={{ color: "red" }}>
                            {error?.toString() ?? ""}
                        </Typography>
                        <Typography variant="body1">
                            Please try refreshing the page. If the problem persists, you can{" "}
                            <a href={mailToUrl} target="_blank" rel="noopener noreferrer">
                                contact us
                            </a>{" "}
                            and we will try to help you as soon as possible.
                        </Typography>
                        <Button
                            variant="contained"
                            startIcon={<RefreshIcon />}
                            onClick={this.handleRefresh}
                            sx={{ marginTop: "16px" }}
                        >
                            Refresh
                        </Button>
                        <Button
                            variant="contained"
                            startIcon={<HomeIcon />}
                            onClick={() => window.location.assign("/")}
                            sx={{ marginTop: "16px" }}
                        >
                            Go to Home
                        </Button>
                    </Stack>
                </div>
            );
        }
        return this.props.children;
    }
}
