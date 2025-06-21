import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { styled, keyframes } from "@mui/material/styles";
import { Icon } from "../../icons/Icons.js";

const pulse = keyframes`
    0% {
        opacity: 0.6;
        transform: scale(0.95);
    }
    50% {
        opacity: 1;
        transform: scale(1);
    }
    100% {
        opacity: 0.6;
        transform: scale(0.95);
    }
`;

const slideUp = keyframes`
    from {
        opacity: 0;
        transform: translateY(10px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
`;

const CompactContainer = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    justifyContent: "center",
    padding: theme.spacing(3),
    minHeight: "200px",
}));

const BotIcon = styled(Box)(({ theme }) => ({
    width: 48,
    height: 48,
    borderRadius: "50%",
    backgroundColor: theme.palette.primary.main + "15",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    marginBottom: theme.spacing(2),
    animation: `${pulse} 3s ease-in-out infinite`,
    "& svg": {
        fontSize: 28,
        color: theme.palette.primary.main,
    },
}));

const SuggestionPill = styled(Box)(({ theme }) => ({
    padding: theme.spacing(0.75, 2),
    borderRadius: theme.spacing(3),
    backgroundColor: theme.palette.mode === "dark" 
        ? "rgba(255, 255, 255, 0.05)" 
        : "rgba(0, 0, 0, 0.03)",
    border: `1px solid ${theme.palette.divider}`,
    cursor: "pointer",
    margin: theme.spacing(0.5),
    fontSize: "0.875rem",
    transition: "all 0.2s ease",
    animation: `${slideUp} 0.5s ease-out`,
    animationFillMode: "both",
    "&:hover": {
        backgroundColor: theme.palette.primary.main + "15",
        borderColor: theme.palette.primary.main,
        transform: "translateY(-2px)",
    },
}));

interface EmptyStateCompactProps {
    onSuggestionClick?: (suggestion: string) => void;
}

export function EmptyStateCompact({ onSuggestionClick }: EmptyStateCompactProps) {
    const quickStarts = [
        "Quick help",
        "Show features",
        "Get started",
    ];

    return (
        <CompactContainer>
            <BotIcon>
                <Icon info={{ name: "Bot", type: "Common" }} decorative />
            </BotIcon>
            
            <Typography variant="h6" gutterBottom>
                How can I help?
            </Typography>
            
            <Typography variant="body2" color="textSecondary" textAlign="center" mb={2}>
                Ask me anything or choose:
            </Typography>
            
            <Box display="flex" flexWrap="wrap" justifyContent="center">
                {quickStarts.map((suggestion, index) => (
                    <SuggestionPill
                        key={suggestion}
                        onClick={() => onSuggestionClick?.(suggestion)}
                        sx={{ animationDelay: `${index * 0.1}s` }}
                    >
                        {suggestion}
                    </SuggestionPill>
                ))}
            </Box>
        </CompactContainer>
    );
}