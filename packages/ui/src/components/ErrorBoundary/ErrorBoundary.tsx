import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Checkbox from "@mui/material/Checkbox";
import Container from "@mui/material/Container";
import FormControlLabel from "@mui/material/FormControlLabel";
import IconButton from "@mui/material/IconButton";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { styled, alpha } from "@mui/material";
import { stringifySearchParams } from "@vrooli/shared";
import { Component } from "react";
import BunnyCrash from "../../assets/img/BunnyCrash.svg";
import { IconCommon } from "../../icons/Icons.js";
import { type ErrorBoundaryProps } from "../../views/types.js";

interface ErrorBoundaryState {
    hasError: boolean;
    error: Error | null;
    mailToUrl: string;
    shouldSendReport: boolean;
    showDetails: boolean;
}

const RootContainer = styled(Box)(() => ({
    position: "fixed",
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    background: "linear-gradient(135deg, #1e2428 0%, #2b3539 100%)",
    backdropFilter: "blur(10px)",
    overflow: "auto",
}));

const ContentCard = styled(Paper)(({ theme }) => ({
    maxWidth: 680,
    width: "90%",
    maxHeight: "90vh",
    overflow: "auto",
    padding: theme.spacing(4),
    borderRadius: theme.spacing(2),
    boxShadow: "0 8px 32px rgba(0, 0, 0, 0.4)",
    background: "rgba(30, 36, 40, 0.95)",
    backdropFilter: "blur(10px)",
    border: "1px solid rgba(255, 255, 255, 0.1)",
    color: "#ffffff",
    [theme.breakpoints.down("sm")]: {
        padding: theme.spacing(3),
    },
}));

const ErrorIcon = styled("img")(({ theme }) => ({
    width: 120,
    height: 120,
    marginBottom: theme.spacing(3),
    opacity: 0.85,
    filter: "drop-shadow(0px 4px 12px rgba(0, 0, 0, 0.15))",
}));

const ErrorDetailsContainer = styled(Paper)(({ theme }) => ({
    padding: theme.spacing(2),
    borderRadius: theme.spacing(1),
    backgroundColor: "rgba(255, 82, 82, 0.08)",
    border: "1px solid rgba(255, 82, 82, 0.3)",
    marginTop: theme.spacing(2),
    marginBottom: theme.spacing(3),
    position: "relative",
    overflow: "hidden",
}));

const ErrorMessage = styled(Typography)(({ theme }) => ({
    fontFamily: "monospace",
    fontSize: "0.875rem",
    whiteSpace: "pre-wrap",
    wordBreak: "break-word",
    maxHeight: 300,
    overflow: "auto",
    padding: theme.spacing(1),
    color: "#e0e0e0",
    "&::-webkit-scrollbar": {
        width: 8,
        height: 8,
    },
    "&::-webkit-scrollbar-track": {
        background: "rgba(255, 255, 255, 0.05)",
        borderRadius: 4,
    },
    "&::-webkit-scrollbar-thumb": {
        background: "rgba(255, 255, 255, 0.2)",
        borderRadius: 4,
        "&:hover": {
            background: "rgba(255, 255, 255, 0.3)",
        },
    },
}));

const StyledFormControlLabel = styled(FormControlLabel)(({ theme }) => ({
    marginTop: theme.spacing(2),
    "& .MuiCheckbox-root": {
        color: "rgba(255, 255, 255, 0.7)",
        "&.Mui-checked": {
            color: "#42f9a3",
        },
    },
    "& .MuiTypography-root": {
        fontSize: "0.875rem",
        color: "rgba(255, 255, 255, 0.8)",
    },
}));

const ActionButton = styled(Button)(({ theme }) => ({
    minWidth: 140,
    textTransform: "none",
    fontWeight: 600,
    borderRadius: theme.spacing(1),
    padding: theme.spacing(1, 3),
    "&.MuiButton-contained": {
        backgroundColor: "#16a361",
        color: "#ffffff",
        "&:hover": {
            backgroundColor: "#1db870",
        },
    },
    "&.MuiButton-outlined": {
        borderColor: "rgba(255, 255, 255, 0.3)",
        color: "#ffffff",
        "&:hover": {
            borderColor: "rgba(255, 255, 255, 0.5)",
            backgroundColor: "rgba(255, 255, 255, 0.05)",
        },
    },
}));

const CopyButton = styled(IconButton)(({ theme }) => ({
    position: "absolute",
    top: theme.spacing(1),
    right: theme.spacing(1),
    padding: theme.spacing(0.5),
    color: "rgba(255, 255, 255, 0.6)",
    "&:hover": {
        color: "rgba(255, 255, 255, 0.9)",
        backgroundColor: "rgba(255, 255, 255, 0.1)",
    },
}));

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
            const userAgent = navigator?.userAgent || "Unknown";
            const currentUrl = window?.location?.href || "Unknown";
            const errorMessage = this.state.error.toString() || "Unknown error";
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
                await fetch("/api/error-reports", {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
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
            console.error("Failed to send error report:", reportError);
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
        // Optional: Show a toast or some feedback that it was copied
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
        const { hasError, error, mailToUrl, showDetails, shouldSendReport } = this.state;
        if (!hasError) return this.props.children;
        
        return (
            <RootContainer id="errorBoundary">
                <Container maxWidth="md">
                    <ContentCard elevation={24}>
                        <Stack spacing={3} alignItems="center">
                            {/* Error Icon */}
                            <ErrorIcon
                                src={BunnyCrash}
                                alt="Error illustration"
                            />
                            
                            {/* Error Title */}
                            <Typography variant="h4" component="h1" fontWeight="bold" textAlign="center" color="white">
                                Something went wrong
                            </Typography>
                            
                            {/* Error Description */}
                            <Typography variant="body1" textAlign="center" sx={{ color: "rgba(255, 255, 255, 0.8)" }}>
                                We encountered an unexpected error. The application may need to be refreshed.
                            </Typography>
                            
                            {/* Error Details */}
                            <ErrorDetailsContainer elevation={0}>
                                <Tooltip title="Copy error details">
                                    <CopyButton 
                                        size="small" 
                                        onClick={this.copyError}
                                        aria-label="Copy error details"
                                    >
                                        <IconCommon name="Copy" decorative />
                                    </CopyButton>
                                </Tooltip>
                                
                                <ErrorMessage variant="body2" component="pre">
                                    {(showDetails ? error?.stack : error?.message) || "An unexpected error occurred."}
                                </ErrorMessage>
                                
                                <Box display="flex" justifyContent="center" mt={1}>
                                    <Button 
                                        size="small" 
                                        onClick={this.toggleDetails}
                                        startIcon={<IconCommon name={showDetails ? "ChevronUp" : "ChevronDown"} decorative />}
                                        sx={{ 
                                            color: "rgba(255, 255, 255, 0.8)", 
                                            "&:hover": { 
                                                backgroundColor: "rgba(255, 255, 255, 0.05)" 
                                            } 
                                        }}
                                    >
                                        {showDetails ? "Show Less" : "Show More"}
                                    </Button>
                                </Box>
                            </ErrorDetailsContainer>
                            
                            {/* Support Text */}
                            <Typography variant="body2" textAlign="center" sx={{ color: "rgba(255, 255, 255, 0.7)" }}>
                                If the problem persists, please{" "}
                                <Button
                                    component="a"
                                    href={mailToUrl}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    size="small"
                                    sx={{ 
                                        textTransform: "none", 
                                        verticalAlign: "baseline",
                                        color: "#dd86db",
                                        "&:hover": {
                                            color: "#f3d4f2",
                                            backgroundColor: "transparent",
                                        },
                                    }}
                                >
                                    contact support
                                </Button>
                            </Typography>
                            
                            {/* Error Reporting Checkbox */}
                            <StyledFormControlLabel
                                control={
                                    <Checkbox
                                        checked={shouldSendReport}
                                        onChange={this.toggleSendReport}
                                        size="small"
                                    />
                                }
                                label="Automatically send error reports to help us improve"
                            />
                            
                            {/* Action Buttons */}
                            <Stack 
                                direction={{ xs: "column", sm: "row" }} 
                                spacing={2} 
                                width="100%"
                                justifyContent="center"
                            >
                                <ActionButton
                                    variant="outlined"
                                    color="primary"
                                    startIcon={<IconCommon name="Home" decorative />}
                                    onClick={this.toHome}
                                >
                                    Go Home
                                </ActionButton>
                                <ActionButton
                                    variant="contained"
                                    color="primary"
                                    startIcon={<IconCommon name="Refresh" decorative />}
                                    onClick={this.handleRefresh}
                                >
                                    Refresh Page
                                </ActionButton>
                            </Stack>
                        </Stack>
                    </ContentCard>
                </Container>
            </RootContainer>
        );
    }
}
