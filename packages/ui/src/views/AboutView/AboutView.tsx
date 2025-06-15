import { keyframes, styled, useTheme } from "@mui/material";
import Box from "@mui/material/Box";
import Fade from "@mui/material/Fade";
import Grow from "@mui/material/Grow";
import IconButton from "@mui/material/IconButton";
import Link from "@mui/material/Link";
import Stack from "@mui/material/Stack";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { SOCIALS } from "@vrooli/shared";
import { cloneElement, useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import MattProfilePic from "../../assets/img/profile-matt.webp";
import { PageContainer } from "../../components/Page/Page.js";
import { Button } from "../../components/buttons/Button.js";
import { Footer } from "../../components/navigation/Footer.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { IconCommon, IconRoutine, IconService } from "../../icons/Icons.js";
import { openLink } from "../../route/openLink.js";
import { useLocation } from "../../route/router.js";
import { ScrollBox } from "../../styles.js";
import { type ViewProps } from "../../types.js";

type MemberData = {
    fullName: string;
    role: string;
    photo: string;
    socials: {
        website?: string;
        x?: string;
        github?: string;
    }
}

// Hand wave animation
const wave = keyframes`
  0% {
    transform: rotate(0deg);
  }
  10% {
    transform: rotate(30deg);
  }
  20% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(0deg);
  }
`;

// Floating animation for feature cards
const float = keyframes`
  0%, 100% {
    transform: translateY(0px);
  }
  50% {
    transform: translateY(-6px);
  }
`;

// Gradient shimmer animation
const shimmer = keyframes`
  0% {
    background-position: -200% center;
  }
  100% {
    background-position: 200% center;
  }
`;

// Pulse glow animation
const pulseGlow = keyframes`
  0%, 100% {
    box-shadow: 0 0 20px rgba(255, 255, 255, 0.1);
  }
  50% {
    box-shadow: 0 0 40px rgba(255, 255, 255, 0.2), 0 0 60px rgba(0, 150, 255, 0.1);
  }
`;

const RotatedBox = styled("div")({
    display: "inline-block",
    animation: `${wave} 3s infinite ease`,
});

const FloatingCard = styled(Box)(({ theme }) => ({
    transition: "all 0.3s cubic-bezier(0.4, 0, 0.2, 1)",
    "&:hover": {
        animation: `${float} 2s ease-in-out infinite`,
        transform: "translateY(-4px)",
        boxShadow: `0 12px 40px ${theme.palette.mode === "dark" ? "rgba(0, 0, 0, 0.6)" : "rgba(0, 0, 0, 0.15)"}`,
    },
}));

const GradientText = styled(Typography)(({ theme }) => ({
    background: `linear-gradient(135deg, ${theme.palette.primary.main}, ${theme.palette.secondary.main}, ${theme.palette.primary.light})`,
    backgroundClip: "text",
    WebkitBackgroundClip: "text",
    WebkitTextFillColor: "transparent",
    backgroundSize: "200% 200%",
    animation: `${shimmer} 3s ease-in-out infinite`,
}));

const memberButtonProps = {
    background: "transparent",
    border: "0",
    "&:hover": {
        background: "transparent",
        filter: "brightness(1.2)",
        transform: "scale(1.2)",
    },
    transition: "all 0.2s ease",
};

const teamMembers: MemberData[] = [
    {
        fullName: "Matt Halloran",
        role: "Founder & Chief Architect",
        photo: MattProfilePic,
        socials: {
            website: "https://matthalloran.info",
            x: "https://x.com/mdhalloran",
            github: "https://github.com/MattHalloran",
        },
    },
];

const joinTeamLink = "https://github.com/Vrooli/Vrooli#-join-the-team";

export function AboutView(_props: ViewProps) {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const theme = useTheme();
    const { palette } = theme;
    const [visibleCards, setVisibleCards] = useState<number[]>([]);

    const handleJoinTeam = useCallback(function handleJoinTeamCallback(event: React.MouseEvent<HTMLButtonElement>) {
        event.preventDefault();
        openLink(setLocation, joinTeamLink);
    }, [setLocation]);

    // Staggered card animation on mount
    useEffect(() => {
        const timer = setTimeout(() => {
            setVisibleCards([0, 1, 2, 3, 4, 5, 6, 7]);
        }, 200);
        return () => clearTimeout(timer);
    }, []);

    return (
        <PageContainer size="fullSize">
            <ScrollBox>
                <Navbar title={t("AboutUs")} />
                <Box
                    display="flex"
                    flexDirection="column"
                    gap={6}
                    p={{ xs: 2, md: 3 }}
                    pt={2}
                    pb={2}
                    margin="auto"
                    maxWidth="min(900px, 100%)"
                >
                    <Fade in timeout={800}>
                        <Box
                            p={{ xs: 4, md: 6 }}
                            borderRadius={4}
                            sx={{
                                background: `linear-gradient(135deg, ${palette.primary.main}15, ${palette.secondary.main}10, ${palette.primary.light}05)`,
                                backdropFilter: "blur(10px)",
                                border: `1px solid ${palette.primary.main}20`,
                                position: "relative",
                                overflow: "hidden",
                                "&::before": {
                                    content: '""',
                                    position: "absolute",
                                    top: 0,
                                    left: 0,
                                    right: 0,
                                    bottom: 0,
                                    background: `linear-gradient(45deg, transparent 30%, ${palette.primary.main}08 50%, transparent 70%)`,
                                    animation: `${shimmer} 4s ease-in-out infinite`,
                                },
                            }}
                        >
                            <Box position="relative" zIndex={1}>
                                <GradientText variant="h3" component="h1" gutterBottom fontWeight="bold" textAlign="center">
                                    Vrooli: Where AI Builds Better AI <RotatedBox>🚀</RotatedBox>
                                </GradientText>
                                <Typography
                                    variant="h6"
                                    component="p"
                                    textAlign="center"
                                    sx={{
                                        color: palette.text.secondary,
                                        maxWidth: "700px",
                                        margin: "0 auto",
                                        lineHeight: 1.6,
                                        fontSize: { xs: "1.1rem", md: "1.25rem" },
                                    }}
                                >
                                    Experience the world's first <strong>recursive intelligence platform</strong>—where AI agents don't just execute tasks, they evolve. Through collaborative swarms and compound knowledge effects, we're building systems that improve themselves exponentially, making billionaire-level productivity accessible to everyone.
                                </Typography>
                            </Box>
                        </Box>
                    </Fade>

                    <Stack spacing={5}>
                        {[
                            { icon: <IconCommon name="Bot" />, title: "AI That Manages Itself", text: "Think of Vrooli as a self-organizing company. At the top, AI managers decide what needs to be done and assign the right agents. In the middle, AI supervisors break big goals into step-by-step plans. At the bottom, AI workers execute tasks and learn from experience. Just like a real company, but running 24/7 at the speed of thought. The best part? They get smarter and more efficient without you lifting a finger.", color: "#FF6B6B" },
                            { icon: <IconCommon name="Team" />, title: "Dynamic Swarm Intelligence", text: "Unlike rigid automation, our swarms assemble dynamically around objectives, then gracefully disband when complete. Using MOISE+ organizational modeling, agents form sophisticated hierarchies with specialized roles—from security monitoring to performance optimization. Teams provide strategic direction while swarms handle tactical execution with unprecedented flexibility.", color: "#4ECDC4" },
                            { icon: <IconRoutine name="Routine" />, title: "Evolutionary Routines", text: "Routines aren't just workflows—they're living intelligence that evolves. Starting as conversational interactions, they progress through reasoning frameworks to become deterministic automation. With recursive composition and unlimited nesting, every routine becomes a building block for more sophisticated capabilities. The result? Exponential growth in what your agents can accomplish.", color: "#45B7D1" },
                            { icon: <IconCommon name="Refresh" />, title: "Compound Knowledge Effects", text: "This is where the magic happens: every routine executed, every pattern learned, every improvement made compounds throughout the entire system. Optimization agents analyze execution patterns and propose enhancements. Success patterns propagate across teams. The system doesn't just get better—it gets better at getting better, creating true recursive self-improvement.", color: "#96CEB4" },
                            { icon: <IconCommon name="Comment" />, title: "Event-Driven Intelligence", text: "Instead of hard-coded features, specialized agent swarms subscribe to events and provide capabilities through reactive intelligence. Security agents monitor threats and evolve defenses. Quality agents detect bias and validate outputs. Performance agents identify bottlenecks and optimize resource usage. The system gains new capabilities simply by deploying new agent swarms.", color: "#FFEAA7" },
                            { icon: <IconCommon name="Wallet" />, title: "Democratizing Billionaire Productivity", text: "By enabling compound knowledge effects and collaborative improvement, we're making elite-level productivity accessible to everyone. Routines evolve from 450 credits and 5+ minutes to 75 credits and 45 seconds while improving quality. Teams share successful patterns. The cost of automation drops exponentially while capabilities grow. This is how we level the playing field.", color: "#DDA0DD" },
                            { icon: <IconCommon name="HeartFilled" />, title: "Built for Responsible AI", text: "Our multi-layered security architecture addresses both traditional and AI-specific threats. Synchronous guard rails provide sub-10ms safety checks. Adaptive security agents evolve defenses in real-time. Privacy preservation through sensitivity classification. Complete audit transparency. All open-source under AGPLv3, ensuring this technology serves humanity transparently.", color: "#FDA085" },
                            { icon: <IconService name="GitHub" />, title: "Join the Revolution", text: () => (<>Be part of building the future where AI systems enhance their own capabilities. Whether you're interested in Kubernetes orchestration, API development, or AI architecture, there's a place for you. Check out our <Link href={SOCIALS.GitHub} target="_blank" rel="noopener">GitHub</Link> and help us build the infrastructure for recursive intelligence!</>), color: "#A8E6CF" },
                        ].map((section, index) => (
                            <Grow
                                key={section.title}
                                in={visibleCards.includes(index)}
                                timeout={600 + index * 100}
                            >
                                <FloatingCard
                                    p={{ xs: 3, md: 4 }}
                                    borderRadius={3}
                                    sx={{
                                        background: `linear-gradient(135deg, ${palette.background.paper}, ${palette.background.default})`,
                                        border: `1px solid ${section.color}30`,
                                        position: "relative",
                                        overflow: "hidden",
                                        "&::before": {
                                            content: '""',
                                            position: "absolute",
                                            top: 0,
                                            left: 0,
                                            width: "4px",
                                            height: "100%",
                                            background: `linear-gradient(to bottom, ${section.color}, ${section.color}80)`,
                                        },
                                        "&:hover::after": {
                                            content: '""',
                                            position: "absolute",
                                            top: 0,
                                            left: 0,
                                            right: 0,
                                            bottom: 0,
                                            background: `linear-gradient(135deg, ${section.color}05, transparent)`,
                                        },
                                    }}
                                >
                                    <Stack direction="row" spacing={2} alignItems="flex-start" mb={2}>
                                        {section.icon && typeof section.icon !== "string" ? (
                                            <Box
                                                sx={{
                                                    color: section.color,
                                                    display: "flex",
                                                    alignItems: "center",
                                                    p: 1.5,
                                                    borderRadius: 2,
                                                    background: `${section.color}15`,
                                                    border: `1px solid ${section.color}30`,
                                                }}
                                            >
                                                {cloneElement(section.icon, { size: 32 })}
                                            </Box>
                                        ) : null}
                                        <Box flex={1}>
                                            <Typography
                                                variant="h5"
                                                component="h2"
                                                gutterBottom
                                                sx={{
                                                    fontWeight: 600,
                                                    color: palette.text.primary,
                                                }}
                                            >
                                                {section.title}
                                            </Typography>
                                        </Box>
                                    </Stack>
                                    <Typography
                                        variant="body1"
                                        sx={{
                                            lineHeight: 1.7,
                                            color: palette.text.secondary,
                                            fontSize: "1.05rem",
                                        }}
                                    >
                                        {typeof section.text === "function" ? section.text() : section.text}
                                    </Typography>
                                </FloatingCard>
                            </Grow>
                        ))}
                    </Stack>

                    <Fade in timeout={1000}>
                        <Box
                            p={{ xs: 4, md: 6 }}
                            borderRadius={4}
                            sx={{
                                background: `linear-gradient(135deg, ${palette.secondary.main}10, ${palette.primary.main}05)`,
                                border: `1px solid ${palette.secondary.main}20`,
                                textAlign: "center",
                                position: "relative",
                                overflow: "hidden",
                                "&::after": {
                                    content: '""',
                                    position: "absolute",
                                    top: 0,
                                    left: 0,
                                    right: 0,
                                    bottom: 0,
                                    background: `radial-gradient(circle at center, ${palette.secondary.main}05, transparent 70%)`,
                                    animation: `${pulseGlow} 4s ease-in-out infinite`,
                                },
                            }}
                        >
                            <Box position="relative" zIndex={1}>
                                <Typography variant="h4" component="h2" gutterBottom fontWeight="bold" color="primary">
                                    The Future Isn't Built—It Builds Itself
                                </Typography>
                                <Typography variant="h6" sx={{ color: palette.text.secondary, maxWidth: "800px", margin: "0 auto", lineHeight: 1.7 }}>
                                    Traditional AI gives you tools. Vrooli gives you <strong>evolving intelligence</strong>.
                                    Every day, our platform becomes more capable through compound knowledge effects and recursive improvement.
                                    We're not just automating tasks—we're creating the infrastructure for AI systems that enhance their own capabilities.
                                </Typography>
                                <Typography variant="h6" sx={{ color: palette.text.secondary, maxWidth: "800px", margin: "2rem auto 0", lineHeight: 1.7 }}>
                                    <em>"Big ideas deserve big teams"</em>—and now, everyone can have one.
                                    Launch your first swarm in under a minute. Watch it grow. See it evolve.
                                    Experience what happens when intelligence truly compounds.
                                </Typography>
                            </Box>
                        </Box>
                    </Fade>

                    <Fade in timeout={1200}>
                        <Box
                            mt={6}
                            p={{ xs: 4, md: 6 }}
                            borderRadius={4}
                            sx={{
                                background: `linear-gradient(145deg, ${palette.background.paper}, ${theme.palette.mode === "dark" ? palette.grey[900] : palette.grey[100]})`,
                                border: `1px solid ${palette.divider}`,
                                position: "relative",
                                overflow: "hidden",
                                "&::before": {
                                    content: '""',
                                    position: "absolute",
                                    top: "-50%",
                                    left: "-50%",
                                    width: "200%",
                                    height: "200%",
                                    background: `radial-gradient(circle, ${palette.primary.main}05 0%, transparent 70%)`,
                                    animation: `${float} 6s ease-in-out infinite`,
                                },
                            }}
                        >
                            <Box position="relative" zIndex={1}>
                                <GradientText variant='h3' component="h2" pb={2} textAlign="center" fontWeight="bold">
                                    Building the Future Together
                                </GradientText>
                                <Typography variant="h6" pb={4} textAlign="center" sx={{ color: palette.text.secondary, maxWidth: "600px", margin: "0 auto" }}>
                                    While AI builds better AI, humans guide the vision. Meet the team creating the infrastructure for recursive intelligence.
                                </Typography>
                                <Stack id="members-stack" direction="column" spacing={4}>
                                    {teamMembers.map((member, key) => {
                                        function openPersonalWebsite() {
                                            openLink(setLocation, member.socials.website as string);
                                        }
                                        function openX() {
                                            openLink(setLocation, member.socials.x as string);
                                        }
                                        function openGitHub() {
                                            openLink(setLocation, member.socials.github as string);
                                        }

                                        return (
                                            <Grow in timeout={1000} key={key}>
                                                <FloatingCard
                                                    key={key}
                                                    sx={{
                                                        display: "flex",
                                                        flexDirection: { xs: "column", sm: "row" },
                                                        alignItems: "center",
                                                        justifyContent: "flex-start",
                                                        background: `linear-gradient(135deg, ${palette.primary.dark}E6 0%, ${palette.primary.main}E6 50%, ${palette.secondary.main}E6 100%)`,
                                                        color: palette.primary.contrastText,
                                                        borderRadius: 4,
                                                        padding: { xs: 3, md: 4 },
                                                        overflow: "hidden",
                                                        gap: { xs: 3, md: 4 },
                                                        border: `1px solid ${palette.primary.light}40`,
                                                        position: "relative",
                                                        "&::before": {
                                                            content: '""',
                                                            position: "absolute",
                                                            top: 0,
                                                            left: 0,
                                                            right: 0,
                                                            bottom: 0,
                                                            background: `linear-gradient(45deg, transparent 40%, ${palette.secondary.light}10 50%, transparent 60%)`,
                                                            animation: `${shimmer} 5s ease-in-out infinite`,
                                                        },
                                                    }}
                                                >
                                                    <Box position="relative" zIndex={1} display="flex" flexDirection={{ xs: "column", sm: "row" }} alignItems="center" gap={{ xs: 3, md: 4 }} width="100%">
                                                        <Box
                                                            component="img"
                                                            src={member.photo}
                                                            alt={`${member.fullName} profile picture`}
                                                            sx={{
                                                                width: { xs: "150px", sm: "180px" },
                                                                height: { xs: "150px", sm: "180px" },
                                                                aspectRatio: "1 / 1",
                                                                objectFit: "cover",
                                                                borderRadius: "50%",
                                                                flexShrink: 0,
                                                                border: `4px solid ${palette.secondary.light}`,
                                                                boxShadow: `0 8px 32px ${palette.common.black}40`,
                                                                transition: "all 0.3s ease",
                                                                "&:hover": {
                                                                    transform: "scale(1.05)",
                                                                    boxShadow: `0 12px 48px ${palette.common.black}60`,
                                                                },
                                                            }}
                                                        />
                                                        <Box
                                                            width={{ xs: "100%", sm: "auto" }}
                                                            flexGrow={1}
                                                            textAlign={{ xs: "center", sm: "left" }}
                                                        >
                                                            <Typography variant='h4' component="h3" mb={1} fontWeight="bold" sx={{ color: "white" }}>
                                                                {member.fullName}
                                                            </Typography>
                                                            <Typography variant='h6' mb={3} sx={{ color: palette.grey[200], fontStyle: "italic" }}>
                                                                {member.role}
                                                            </Typography>
                                                            <Stack direction="row" alignItems="center" justifyContent={{ xs: "center", sm: "flex-start" }} spacing={2}>
                                                                {member.socials.website && (
                                                                    <Tooltip title="Personal website" placement="bottom">
                                                                        <IconButton
                                                                            aria-label="Personal website"
                                                                            onClick={openPersonalWebsite}
                                                                            sx={{
                                                                                ...memberButtonProps,
                                                                                color: palette.secondary.light,
                                                                                background: `${palette.secondary.light}20`,
                                                                                "&:hover": {
                                                                                    background: `${palette.secondary.light}30`,
                                                                                    transform: "scale(1.15) rotate(5deg)",
                                                                                },
                                                                            }}
                                                                        >
                                                                            <IconCommon decorative name="Website" size={40} />
                                                                        </IconButton>
                                                                    </Tooltip>
                                                                )}
                                                                {member.socials.x && (
                                                                    <Tooltip title="X/Twitter" placement="bottom">
                                                                        <IconButton
                                                                            aria-label="X/Twitter"
                                                                            onClick={openX}
                                                                            sx={{
                                                                                ...memberButtonProps,
                                                                                color: palette.secondary.light,
                                                                                background: `${palette.secondary.light}20`,
                                                                                "&:hover": {
                                                                                    background: `${palette.secondary.light}30`,
                                                                                    transform: "scale(1.15) rotate(-5deg)",
                                                                                },
                                                                            }}
                                                                        >
                                                                            <IconService decorative name="X" size={36} />
                                                                        </IconButton>
                                                                    </Tooltip>
                                                                )}
                                                                {member.socials.github && (
                                                                    <Tooltip title="GitHub" placement="bottom">
                                                                        <IconButton
                                                                            aria-label="GitHub"
                                                                            onClick={openGitHub}
                                                                            sx={{
                                                                                ...memberButtonProps,
                                                                                color: palette.secondary.light,
                                                                                background: `${palette.secondary.light}20`,
                                                                                "&:hover": {
                                                                                    background: `${palette.secondary.light}30`,
                                                                                    transform: "scale(1.15) rotate(5deg)",
                                                                                },
                                                                            }}
                                                                        >
                                                                            <IconService decorative name="GitHub" size={36} />
                                                                        </IconButton>
                                                                    </Tooltip>
                                                                )}
                                                            </Stack>
                                                        </Box>
                                                    </Box>
                                                </FloatingCard>
                                            </Grow>
                                        );
                                    })}
                                    <Stack direction="row" justifyContent="center" alignItems="center" pt={4}>
                                        <Grow in timeout={1400}>
                                            <Button
                                                fullWidth
                                                size="lg"
                                                href={joinTeamLink}
                                                onClick={handleJoinTeam}
                                                variant="space"
                                                sx={{
                                                    maxWidth: "500px",
                                                    py: 2,
                                                    px: 4,
                                                    fontSize: "1.2rem",
                                                    fontWeight: "bold",
                                                    borderRadius: 3,
                                                    color: "white",
                                                    position: "relative",
                                                    overflow: "hidden",
                                                    "&::before": {
                                                        content: '""',
                                                        position: "absolute",
                                                        top: 0,
                                                        left: "-100%",
                                                        width: "100%",
                                                        height: "100%",
                                                        background: `linear-gradient(90deg, transparent, ${palette.common.white}30, transparent)`,
                                                        transition: "left 0.5s ease",
                                                    },
                                                    "&:hover::before": {
                                                        left: "100%",
                                                    },
                                                }}
                                            >
                                                🚀 Join the Team
                                            </Button>
                                        </Grow>
                                    </Stack>
                                </Stack>
                            </Box>
                        </Box>
                    </Fade>
                </Box>
                <Footer />
            </ScrollBox>
        </PageContainer>
    );
}
