import { Box, Button, Checkbox, IconButton, Stack, Tooltip, Typography } from "@mui/material";
import BunnyCrash from "assets/img/BunnyCrash.svg";
import { ArrowDropDownIcon, ArrowDropUpIcon, CopyIcon, HomeIcon, RefreshIcon } from "icons";
import { Component } from "react";
import { stringifySearchParams } from "route";
import { SlideImage, SlideImageContainer } from "styles";
import { ErrorBoundaryProps } from "../../views/types";

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

    handleRefresh = () => {
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

    // TODO send report if true and either button is pressed
    toggleSendReport = () => {
        this.setState(prevState => {
            // Update localStorage when state changes
            const updatedValue = !prevState.shouldSendReport;
            localStorage.setItem("shouldSendReport", String(updatedValue));
            return { shouldSendReport: updatedValue };
        });
    };

    render() {
        const { hasError, error, mailToUrl, showDetails } = this.state;
        if (hasError) {
            return (
                <Box
                    sx={{
                        display: "flex",
                        overflowY: "scroll",
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
                        backgroundColor: "#2b3539",
                        maxWidth: "100vw",
                        maxHeight: "100vh",
                        color: "white",
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
                    }}
                >
                    <Stack direction="column" spacing={4} style={{ textAlign: "center", maxWidth: "100%" }}>
                        <SlideImageContainer>
                            <SlideImage
                                alt="A lop-eared bunny calling for tech support."
                                src={BunnyCrash}
                            />
                        </SlideImageContainer>
                        <Typography variant="h4">Something went wrong 😔</Typography>
                        <Box sx={{
                            border: "1px solid red",
                            borderRadius: "8px",
                            background: "#480202",
                            color: "white",
                            maxWidth: "100%",
                            maxHeight: "50vh",
                            overflow: "auto",
                        }}>
                            <Stack
                                direction="row"
                                spacing={1}
                                justifyContent="center"
                                alignItems="center"
                            >
                                <Tooltip title="Copy">
                                    <IconButton onClick={this.copyError}>
                                        <CopyIcon fill="#42f9a3" />
                                    </IconButton>
                                </Tooltip>
                                <Typography variant="body1">
                                    {error?.toString() ?? ""}
                                </Typography>
                                <IconButton onClick={this.toggleDetails}>
                                    {showDetails ? <ArrowDropUpIcon fill="#42f9a3" /> : <ArrowDropDownIcon fill="#42f9a3" />}
                                </IconButton>
                            </Stack>
                            {showDetails && (
                                <Typography variant="body2" style={{ whiteSpace: "pre-wrap", lineHeight: "2", paddingBottom: "8px", paddingTop: "8px" }}>
                                    {error?.stack ?? ""}
                                </Typography>
                            )}
                        </Box>
                        <Typography variant="body1">
                            Please try refreshing the page. If the problem persists, you can{" "}
                            <a href={mailToUrl} target="_blank" rel="noopener noreferrer">
                                contact us
                            </a>{" "}
                            and we will try to help you as soon as possible.
                        </Typography>
                        <Box sx={{ display: "flex", justifyContent: "center", alignItems: "center" }}>
                            <Checkbox
                                checked={this.state.shouldSendReport}
                                onChange={this.toggleSendReport}
                                sx={{
                                    color: "white",
                                    "&.Mui-checked": {
                                        color: "#42f9a3",
                                    },
                                }}
                            />
                            <Typography>Send crash logs</Typography>
                        </Box>
                        <Stack direction="row" spacing={2} pt={2} pb={8} justifyContent="center" alignItems="center">
                            <Button
                                variant="contained"
                                startIcon={<RefreshIcon />}
                                onClick={this.handleRefresh}
                                sx={{ marginTop: "16px", backgroundColor: "#16a361" }}
                            >
                                Refresh
                            </Button>
                            <Button
                                variant="contained"
                                startIcon={<HomeIcon />}
                                onClick={() => window.location.assign("/")}
                                sx={{ marginTop: "16px", backgroundColor: "#16a361" }}
                            >
                                Go to Home
                            </Button>
                        </Stack>
                    </Stack>
                </Box>
            );
        }
        return this.props.children;
    }
}
