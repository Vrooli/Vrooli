import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Fade from "@mui/material/Fade";
import Grow from "@mui/material/Grow";
import IconButton from "@mui/material/IconButton";
import Link from "@mui/material/Link";
import Stack from "@mui/material/Stack";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { keyframes } from "@mui/material";
import { styled, useTheme } from "@mui/material";
import { SOCIALS } from "@vrooli/shared";
import { cloneElement, useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import MattProfilePic from "../../assets/img/profile-matt.webp";
import { PageContainer } from "../../components/Page/Page.js";
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

const GlowingButton = styled(Button)(({ theme }) => ({
    background: `linear-gradient(135deg, ${theme.palette.secondary.main}, ${theme.palette.secondary.dark})`,
    animation: `${pulseGlow} 2s ease-in-out infinite`,
    "&:hover": {
        background: `linear-gradient(135deg, ${theme.palette.secondary.light}, ${theme.palette.secondary.main})`,
        transform: "scale(1.05) translateY(-2px)",
        boxShadow: `0 8px 25px ${theme.palette.secondary.main}40`,
    },
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
        role: "Leader/developer",
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
                                    Vrooli: Automate, Collaborate, Evolve <RotatedBox>👋</RotatedBox>
                                </GradientText>
                                <Typography 
                                    variant="h6" 
                                    component="p"
                                    textAlign="center"
                                    sx={{ 
                                        color: palette.text.secondary,
                                        maxWidth: "600px",
                                        margin: "0 auto",
                                        lineHeight: 1.6,
                                        fontSize: { xs: "1.1rem", md: "1.25rem" },
                                    }}
                                >
                                    Welcome to Vrooli, the platform designed to delegate like a billionaire! Let our AI agents handle the details so you can focus on what truly matters. We empower users to create autonomous agents, build dynamic teams, and run powerful routines to automate workflows with unprecedented efficiency.
                                </Typography>
                            </Box>
                        </Box>
                    </Fade>

                    <Stack spacing={5}>
                        {[
                            { icon: <IconCommon name="Bot" />, title: "Autonomous Agents", text: "Create AI-driven bots, known as autonomous agents, capable of performing tasks without constant human supervision. Interact with them through chat to complete various tasks, from managing projects to automating workflows. These agents provide reliable support around the clock, making your life easier and your operations smoother.", color: "#FF6B6B" },
                            { icon: <IconCommon name="Team" />, title: "Dynamic Teams", text: "Organize users and bots into teams with assigned roles. Teams collaborate towards specific goals, such as building a business, conducting research, or any project benefiting from collaboration. Add or remove bots anytime, providing flexibility to grow and reshape your initiatives without traditional overhead.", color: "#4ECDC4" },
                            { icon: <IconRoutine name="Routine" />, title: "Powerful Routines", text: "Accomplish tasks using routines—reusable building blocks combining standards, APIs, code, and more. Tailor routines for any purpose, like internal processes, tutorials, or cognitive architectures for bots. Routines can be run by users, bots, or both, enabling seamless collaboration and consistent, automated workflows. Routines are polymorphic, meaning their output adapts based on the executing agent's unique personality and role.", color: "#45B7D1" },
                            { icon: <IconCommon name="Refresh" />, title: "Recursive Self-Improvement", text: "At the heart of Vrooli is recursive self-improvement. Bots utilize cognitive routines to continuously enhance their own capabilities and those of the platform. By analyzing metrics and identifying inefficiencies, routines (and the routines that improve routines) are constantly refined, enabling the platform to evolve and become more efficient over time.", color: "#96CEB4" },
                            { icon: <IconCommon name="Comment" />, title: "Integrated Human and Bot Collaboration", text: "Vrooli's platform seamlessly integrates human and bot employees within teams. Engage with multiple bots and humans in the same conversation, facilitating smooth collaboration. Our upcoming messaging features will further enhance interaction, allowing users to guide, provide feedback to, and assign tasks to bots within their team environment.", color: "#FFEAA7" },
                            { icon: <IconCommon name="Wallet" />, title: "Reducing Costs Through Collaboration", text: "Vrooli tackles the high costs of starting and maintaining organizations. Since routines can be publicly shared and improved, humans and bots collaborate to enhance all teams simultaneously. As automation increases, the cost of running operations drops significantly, benefiting consumers, supporting growth, or enhancing employee benefits. Anyone can copy a public team, gaining access to refined processes and on-demand bot employees at no extra cost.", color: "#DDA0DD" },
                            { icon: <IconCommon name="HeartFilled" />, title: "Our Commitment", text: "At Vrooli, we're committed to ethical and beneficial outcomes. We prioritize responsible development and transparency. Our platform operates under the AGPLv3 license, ensuring that the software and its improvements remain free and open-source, fostering a collaborative ecosystem where technology serves society transparently and reliably.", color: "#FDA085" },
                            { icon: <IconService name="GitHub" />, title: "Get Involved", text: () => (<>Join us in building the future of automation! Explore Vrooli and see how our platform enables transparent and reliable autonomous systems. Visit our <Link href={SOCIALS.GitHub} target="_blank" rel="noopener">GitHub page</Link> to learn more, contribute, or join the team. We look forward to collaborating with you!</>), color: "#A8E6CF" },
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
                                <GradientText variant='h3' component="h2" pb={4} textAlign="center" fontWeight="bold">
                                    Meet The Team
                                </GradientText>
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
                                        <GlowingButton
                                            fullWidth
                                            size="large"
                                            href={joinTeamLink}
                                            onClick={handleJoinTeam}
                                            variant="contained"
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
                                            startIcon={<IconCommon
                                                decorative
                                                name="Team"
                                                size={28}
                                            />}
                                        >
                                            🚀 Join the Team
                                        </GlowingButton>
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
