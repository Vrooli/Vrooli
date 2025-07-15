import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material/styles";
import React, { Component, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../icons/Icons.js";

interface Props {
    children: ReactNode;
    onRetry?: () => void;
}

interface State {
    hasError: boolean;
    errorMessage: string;
}

/**
 * Error boundary for model selection components.
 * Shows a fallback UI when AI models fail to load.
 */
class ModelSelectionErrorBoundaryClass extends Component<Props, State> {
    constructor(props: Props) {
        super(props);
        this.state = { hasError: false, errorMessage: "" };
    }

    static getDerivedStateFromError(error: Error): State {
        return { 
            hasError: true, 
            errorMessage: error.message || "Failed to load AI models",
        };
    }

    componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
        // Log error to error reporting service
        // For now, we'll just use the error message in state
    }

    handleReset = () => {
        this.setState({ hasError: false, errorMessage: "" });
        this.props.onRetry?.();
    };

    render() {
        if (this.state.hasError) {
            return <ModelSelectionErrorFallback 
                errorMessage={this.state.errorMessage}
                onReset={this.handleReset}
            />;
        }

        return this.props.children;
    }
}

interface FallbackProps {
    errorMessage: string;
    onReset: () => void;
}

function ModelSelectionErrorFallback({ errorMessage, onReset }: FallbackProps) {
    const { t } = useTranslation();
    const theme = useTheme();

    return (
        <Box 
            sx={{ 
                display: "flex",
                flexDirection: "column",
                alignItems: "center",
                justifyContent: "center",
                gap: 2,
                p: 4,
                minHeight: 200,
                textAlign: "center",
            }}
        >
            <IconCommon 
                name="Warning" 
                size={48}
                fill={theme.palette.error.main}
            />
            <Typography variant="h6" color="textPrimary">
                {t("FailedToLoadModels", { ns: "error", defaultValue: "Failed to load AI models" })}
            </Typography>
            <Typography variant="body2" color="textSecondary" sx={{ maxWidth: 400 }}>
                {errorMessage || t("ModelsUnavailable", { 
                    ns: "error", 
                    defaultValue: "AI models are temporarily unavailable. Please check your connection and try again.",
                })}
            </Typography>
            <Button 
                variant="contained" 
                onClick={onReset}
                startIcon={<IconCommon name="Refresh" />}
            >
                {t("TryAgain", { ns: "ui", defaultValue: "Try Again" })}
            </Button>
        </Box>
    );
}

// Export wrapped component with hooks
export function ModelSelectionErrorBoundary(props: Props) {
    return <ModelSelectionErrorBoundaryClass {...props} />;
}
