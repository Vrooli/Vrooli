import Box from "@mui/material/Box";
import Chip from "@mui/material/Chip";
import Grid from "@mui/material/Grid";
import Typography from "@mui/material/Typography";
import { styled, keyframes } from "@mui/material/styles";
import { Icon, type IconInfo } from "../../icons/Icons.js";
import { ProfileAvatar } from "../../styles.js";
import { extractImageUrl } from "../../utils/display/imageTools.js";
import { getDisplay, placeholderColor } from "../../utils/display/listTools.js";
import { BotConfig } from "@vrooli/shared";

const fadeIn = keyframes`
    from {
        opacity: 0;
        transform: translateY(20px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
`;

const slideIn = keyframes`
    from {
        opacity: 0;
        transform: translateX(-20px);
    }
    to {
        opacity: 1;
        transform: translateX(0);
    }
`;

const EmptyStateContainer = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    justifyContent: "center",
    flex: 1,
    padding: theme.spacing(3),
    animation: `${fadeIn} 0.6s ease-out`,
}));

const WelcomeHeader = styled(Box)(({ theme }) => ({
    display: "flex",
    alignItems: "center",
    gap: theme.spacing(2),
    marginBottom: theme.spacing(2),
}));

const BotAvatar = styled(ProfileAvatar)(({ theme }) => ({
    width: 48,
    height: 48,
    "& svg": {
        fontSize: 28,
    },
}));


const FeatureCard = styled(Box)(({ theme }) => ({
    padding: theme.spacing(1),
    borderRadius: theme.shape.borderRadius,
    backgroundColor: theme.palette.mode === "dark" 
        ? "rgba(255, 255, 255, 0.05)" 
        : "rgba(0, 0, 0, 0.03)",
    border: `1px solid ${theme.palette.divider}`,
    cursor: "pointer",
    transition: "all 0.3s ease",
    animation: `${slideIn} 0.8s ease-out`,
    animationFillMode: "both",
    "&:hover": {
        transform: "translateY(-2px)",
        backgroundColor: theme.palette.mode === "dark" 
            ? "rgba(255, 255, 255, 0.08)" 
            : "rgba(0, 0, 0, 0.05)",
        boxShadow: `0 4px 12px ${theme.palette.secondary.main}22`,
        borderColor: theme.palette.secondary.main,
    },
    "& .feature-icon": {
        transition: "transform 0.3s ease",
    },
    "&:hover .feature-icon": {
        transform: "scale(1.05)",
    },
}));

const FeatureIcon = styled(Box)(({ theme }) => ({
    width: 28,
    height: 28,
    borderRadius: theme.shape.borderRadius,
    backgroundColor: theme.palette.secondary.main + "15",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    marginBottom: theme.spacing(0.5),
    "& svg": {
        fontSize: 18,
        color: theme.palette.secondary.main,
    },
}));

const SuggestionChip = styled(Chip)(({ theme }) => ({
    margin: theme.spacing(0.5),
    cursor: "pointer",
    transition: "all 0.2s ease",
    "&:hover": {
        transform: "scale(1.05)",
        boxShadow: `0 4px 12px ${theme.palette.primary.main}33`,
    },
}));

interface EmptyStateWelcomeProps {
    onSuggestionClick?: (suggestion: string) => void;
    bot?: {
        id: string;
        name: string;
        isBot: boolean;
        profileImage?: string;
        updatedAt?: string;
        botSettings?: any;
    } | null;
    isMultiParticipant?: boolean;
}

export function EmptyStateWelcome({ onSuggestionClick, bot, isMultiParticipant }: EmptyStateWelcomeProps) {
    // Parse bot configuration to get starting message
    const botConfig = bot?.botSettings ? BotConfig.parse({ botSettings: bot.botSettings }, console) : null;
    const startingMessage = botConfig?.persona?.startingMessage as string | undefined;
    
    // Determine if we should show bot info
    const showBotInfo = bot && !isMultiParticipant && bot.name !== "Valyxa";
    
    const profileColors = placeholderColor(bot?.id || "default");
    const features = [
        {
            iconInfo: { name: "Project", type: "Common" } as IconInfo,
            title: "Projects",
            description: "Organize collaborative work",
        },
        {
            iconInfo: { name: "Note", type: "Common" } as IconInfo,
            title: "Notes",
            description: "Capture ideas & thoughts",
        },
        {
            iconInfo: { name: "Routine", type: "Routine" } as IconInfo,
            title: "Routines",
            description: "Automate workflows",
        },
        {
            iconInfo: { name: "File", type: "Common" } as IconInfo,
            title: "Resources",
            description: "Manage materials",
        },
    ];

    const suggestions = [
        "How do I create my first project?",
        "Show me automation examples",
        "Help me organize my notes",
        "What can Vrooli do?",
    ];

    return (
        <EmptyStateContainer>
            {showBotInfo && (
                <WelcomeHeader>
                    <BotAvatar
                        isBot={true}
                        profileColors={profileColors}
                        src={extractImageUrl(bot.profileImage, bot.updatedAt, 100)}
                    >
                        <Icon info={{ name: "Bot", type: "Common" }} decorative />
                    </BotAvatar>
                    <Typography variant="h5" fontWeight={600}>
                        {getDisplay(bot).title}
                    </Typography>
                </WelcomeHeader>
            )}

            <Typography 
                variant="body1" 
                color="textSecondary" 
                textAlign="center" 
                maxWidth={500}
                mb={3}
            >
                {showBotInfo && startingMessage ? startingMessage : "What would you like to do today?"}
            </Typography>

            <Grid container spacing={1.5} maxWidth={500} mb={3}>
                {features.map((feature, index) => (
                    <Grid item xs={6} sm={3} key={feature.title}>
                        <FeatureCard 
                            sx={{ animationDelay: `${index * 0.1}s` }}
                            onClick={() => onSuggestionClick?.(feature.description)}
                        >
                            <FeatureIcon className="feature-icon">
                                <Icon info={feature.iconInfo} decorative />
                            </FeatureIcon>
                            <Typography variant="caption" fontWeight={600} sx={{ display: "block", marginBottom: 0.25 }}>
                                {feature.title}
                            </Typography>
                            <Typography variant="caption" color="textSecondary" sx={{ display: "block", fontSize: "0.7rem", lineHeight: 1.3 }}>
                                {feature.description}
                            </Typography>
                        </FeatureCard>
                    </Grid>
                ))}
            </Grid>

            <Box textAlign="center" mb={2}>
                <Typography variant="body2" color="textSecondary" mb={2}>
                    Try asking me:
                </Typography>
                <Box display="flex" flexWrap="wrap" justifyContent="center">
                    {suggestions.map((suggestion) => (
                        <SuggestionChip
                            key={suggestion}
                            label={suggestion}
                            size="medium"
                            variant="outlined"
                            onClick={() => onSuggestionClick?.(suggestion)}
                        />
                    ))}
                </Box>
            </Box>
        </EmptyStateContainer>
    );
}