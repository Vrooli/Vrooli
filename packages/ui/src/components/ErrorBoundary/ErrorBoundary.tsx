import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Checkbox from "@mui/material/Checkbox";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { styled } from "@mui/material";
import { stringifySearchParams } from "@vrooli/shared";
import { Component } from "react";
import BunnyCrash from "../../assets/img/BunnyCrash.svg";
import { IconCommon } from "../../icons/Icons.js";
import { type ErrorBoundaryProps } from "../../views/types.js";

interface ErrorBoundaryState {
    hasError: boolean;
}

interface ErrorBoundaryState {
    hasError: boolean;
    error: Error | null;
    mailToUrl: string;
    shouldSendReport: boolean;
    showDetails: boolean;
}

const OuterBox = styled(Box)(() => ({
    position: "fixed",
    overflow: "auto",
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    justifyContent: "center",
    minHeight: "100vh",
    backgroundColor: "#2b3539",
    color: "white",
    textAlign: "center",
    // Style visited, active, and hovered links
    "& span, p": {
        "& a": {
            color: "#dd86db",
            "&:visited": {
                color: "#f551ef",
            },
            "&:active": {
                color: "#f551ef",
            },
            "&:hover": {
                color: "#f3d4f2",
            },
        },
    },
}));

const InnerBox = styled(Box)(() => ({
    display: "flex",
    flexDirection: "column",
    gap: "1rem",
    maxWidth: "600px",
    width: "90%",
    maxHeight: "90vh",
    margin: "0 auto",
}));

const UhOhImage = styled("img")(() => ({
    maxWidth: "max(150px, 50%)",
    margin: "auto",
    marginBottom: "1rem",
}));

const ErrorMessageBox = styled(Box)(() => ({
    display: "flex",
    flexDirection: "column",
    justifyContent: "center",
    alignItems: "center",
    backgroundColor: "rgba(255, 0, 0, 0.1)",
    border: "1px solid #ff0000",
    borderRadius: "4px",
    padding: "1rem",
    marginBottom: "2rem",
    maxWidth: "100%",
    width: "auto",
    height: "100%",
    minHeight: "100px",
    overflowX: "auto",
}));

const CrashLogsBox = styled(Box)(() => ({
    display: "flex",
    justifyContent: "center",
    alignItems: "center",
}));

const CrashLogsCheckbox = styled(Checkbox)(() => ({
    color: "white",
    "&.Mui-checked": {
        color: "#42f9a3",
    },
}));

const ActionButton = styled(Button)(() => ({
    backgroundColor: "#16a361",
}));

const actionButtonDirection = { xs: "column", sm: "row" } as const;

/**
 * Displays an error message if a child component throws an error.
 * 
 * NOTE: Cannot be a functional component. See https://legacy.reactjs.org/docs/error-boundaries.html#introducing-error-boundaries
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
    constructor(props: ErrorBoundaryProps) {
        super(props);

        const storedSendReport = localStorage.getItem("shouldSendReport");
        const sendReportInitialValue = storedSendReport === null ? true : storedSendReport === "true";

        this.state = {
            hasError: false,
            error: null,
            mailToUrl: "mailto:official@vrooli.com",
            shouldSendReport: sendReportInitialValue,
            showDetails: false,
        };
    }

    static getDerivedStateFromError(error: Error) {
        const subject = `Error Report: ${error?.name ?? ""}`;
        const body = `Error message: ${error?.toString() ?? ""}`;
        const mailToUrl = `mailto:official@vrooli.com${stringifySearchParams({ body, subject })}`;
        return { hasError: true, error, mailToUrl };
    }

    sendErrorReport = async () => {
        if (!this.state.shouldSendReport || !this.state.error) return;
        
        try {
            // Validate required fields before sending
            const userAgent = navigator?.userAgent || 'Unknown';
            const currentUrl = window?.location?.href || 'Unknown';
            const errorMessage = this.state.error.toString() || 'Unknown error';
            const errorStack = this.state.error.stack || undefined;
            
            // Apply size limits to prevent oversized payloads
            const MAX_ERROR_LENGTH = 2000;
            const MAX_STACK_LENGTH = 5000;
            const MAX_URL_LENGTH = 500;
            const MAX_USER_AGENT_LENGTH = 500;
            
            const errorReport = {
                error: errorMessage.substring(0, MAX_ERROR_LENGTH),
                stack: errorStack ? errorStack.substring(0, MAX_STACK_LENGTH) : undefined,
                userAgent: userAgent.substring(0, MAX_USER_AGENT_LENGTH),
                url: currentUrl.substring(0, MAX_URL_LENGTH),
                timestamp: new Date().toISOString(),
            };
            
            // Send to backend with timeout to prevent blocking user actions
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), 3000); // 3 second timeout
            
            try {
                await fetch('/api/error-reports', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(errorReport),
                    signal: controller.signal,
                });
                clearTimeout(timeoutId);
            } catch (fetchError) {
                clearTimeout(timeoutId);
                throw fetchError;
            }
        } catch (reportError) {
            // Fail silently - don't block user actions due to error reporting issues
            console.error('Failed to send error report:', reportError);
        }
    };

    handleRefresh = async () => {
        await this.sendErrorReport();
        window.location.reload();
    };

    toggleDetails = () => {
        this.setState(prevState => ({ showDetails: !prevState.showDetails }));
    };

    copyError = () => {
        if (!this.state.error) return;
        // Copy full trace if showDetails is true
        const text = this.state.showDetails ? this.state.error.stack : this.state.error.toString();
        navigator.clipboard.writeText(text ?? "");
    };

    // Error reporting implemented - sends report when shouldSendReport is true and buttons are pressed
    toggleSendReport = () => {
        this.setState(prevState => {
            // Update localStorage when state changes
            const updatedValue = !prevState.shouldSendReport;
            localStorage.setItem("shouldSendReport", String(updatedValue));
            return { shouldSendReport: updatedValue };
        });
    };

    toHome = async () => {
        await this.sendErrorReport();
        window.location.assign("/");
    };

    render() {
        const { hasError, error, mailToUrl, showDetails } = this.state;
        if (!hasError) return this.props.children;
        return (
            <OuterBox id="errorBoundary">
                <InnerBox>
                    <UhOhImage
                        src={BunnyCrash}
                        alt="Cute bunny with a hard hat and phone"
                    />
                    <Typography variant="h5" gutterBottom>
                        Uh oh! Something went wrong 😔
                    </Typography>
                    <ErrorMessageBox>
                        <Typography variant="body1">
                            {(showDetails ? error?.stack : error?.message) || "An unexpected error occurred."}
                        </Typography>
                        <Button variant="text" onClick={this.toggleDetails}>
                            {showDetails ? "Hide" : "Show"} Details
                        </Button>
                    </ErrorMessageBox>
                    <Typography variant="body1" gutterBottom>
                        If the problem persists, {" "}
                        <a href={mailToUrl} target="_blank" rel="noopener noreferrer">
                            email us
                        </a>{" "}
                        and we will try to help as soon as possible.
                    </Typography>
                    <CrashLogsBox>
                        <CrashLogsCheckbox
                            checked={this.state.shouldSendReport}
                            onChange={this.toggleSendReport}
                        />
                        <Typography variant="body2">Send crash logs</Typography>
                    </CrashLogsBox>
                    <Stack direction={actionButtonDirection} spacing={2} pb={4} justifyContent="center" alignItems="center">
                        <ActionButton
                            aria-label="Refresh"
                            fullWidth
                            variant="contained"
                            startIcon={<IconCommon
                                decorative
                                name="Refresh"
                            />}
                            onClick={this.handleRefresh}
                        >
                            Refresh
                        </ActionButton>
                        <ActionButton
                            aria-label="Go to Home"
                            fullWidth
                            variant="contained"
                            startIcon={<IconCommon
                                decorative
                                name="Home"
                            />}
                            onClick={this.toHome}
                        >
                            Go to Home
                        </ActionButton>
                    </Stack>
                </InnerBox>
            </OuterBox>
        );
    }
}
