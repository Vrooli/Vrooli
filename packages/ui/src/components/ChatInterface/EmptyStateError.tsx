import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import { styled } from "@mui/material/styles";
import { Icon } from "../../icons/Icons.js";

const ErrorContainer = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    justifyContent: "center",
    padding: theme.spacing(4),
    minHeight: "300px",
}));

const ErrorIcon = styled(Box)(({ theme }) => ({
    width: 64,
    height: 64,
    borderRadius: "50%",
    backgroundColor: theme.palette.error.main + "15",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    marginBottom: theme.spacing(3),
    "& svg": {
        fontSize: 36,
        color: theme.palette.error.main,
    },
}));

interface EmptyStateErrorProps {
    errorMessage?: string;
    onRetry?: () => void;
    onGoBack?: () => void;
}

export function EmptyStateError({ 
    errorMessage = "Something went wrong",
    onRetry,
    onGoBack,
}: EmptyStateErrorProps) {
    return (
        <ErrorContainer>
            <ErrorIcon>
                <Icon info={{ name: "Error", type: "Common" }} decorative />
            </ErrorIcon>
            
            <Typography variant="h6" gutterBottom>
                Unable to Load Chat
            </Typography>
            
            <Typography 
                variant="body2" 
                color="textSecondary" 
                textAlign="center" 
                mb={3}
                maxWidth={400}
            >
                {errorMessage}. Please try again or contact support if the problem persists.
            </Typography>
            
            <Box display="flex" gap={2}>
                {onRetry && (
                    <Button 
                        variant="contained" 
                        color="primary"
                        startIcon={<Icon info={{ name: "Refresh", type: "Common" }} />}
                        onClick={onRetry}
                    >
                        Try Again
                    </Button>
                )}
                {onGoBack && (
                    <Button 
                        variant="outlined"
                        onClick={onGoBack}
                    >
                        Go Back
                    </Button>
                )}
            </Box>
        </ErrorContainer>
    );
}