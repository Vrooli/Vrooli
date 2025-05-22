import { SOCIALS } from "@local/shared";
import { Box, Button, IconButton, Link, Stack, Tooltip, Typography, keyframes, styled, useTheme } from "@mui/material";
import { cloneElement, useCallback } from "react";
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
const RotatedBox = styled("div")({
    display: "inline-block",
    animation: `${wave} 3s infinite ease`,
});

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

    const handleJoinTeam = useCallback(function handleJoinTeamCallback(event: React.MouseEvent<HTMLButtonElement>) {
        event.preventDefault();
        openLink(setLocation, joinTeamLink);
    }, [setLocation]);

    return (
        <PageContainer size="fullSize">
            <ScrollBox>
                <Navbar title={t("AboutUs")} />
                <Box
                    display="flex"
                    flexDirection="column"
                    gap={4}
                    p={{ xs: 1, md: 2 }}
                    pt={2}
                    pb={2}
                    margin="auto"
                    maxWidth="min(800px, 100%)"
                >
                    <Box p={3} borderRadius={2} bgcolor={palette.background.paper}>
                        <Typography variant="h4" gutterBottom>
                            Vrooli: Automate, Collaborate, Evolve <RotatedBox>👋</RotatedBox>
                        </Typography>
                        <Typography variant="body1">
                            Welcome to Vrooli, the platform designed to delegate like a billionaire! Let our AI agents handle the details so you can focus on what truly matters. We empower users to create autonomous agents, build dynamic teams, and run powerful routines to automate workflows with unprecedented efficiency. Our goal is to foster collaboration between humans and AI, creating ethical and beneficial systems for society.
                        </Typography>
                    </Box>

                    <Stack spacing={4}>
                        {[
                            { icon: <IconCommon name="Bot" />, title: "Autonomous Agents", text: "Create AI-driven bots, known as autonomous agents, capable of performing tasks without constant human supervision. Interact with them through chat to complete various tasks, from managing projects to automating workflows. These agents provide reliable support around the clock, making your life easier and your operations smoother." },
                            { icon: <IconCommon name="Team" />, title: "Dynamic Teams", text: "Organize users and bots into teams with assigned roles. Teams collaborate towards specific goals, such as building a business, conducting research, or any project benefiting from collaboration. Add or remove bots anytime, providing flexibility to grow and reshape your initiatives without traditional overhead." },
                            { icon: <IconRoutine name="Routine" />, title: "Powerful Routines", text: "Accomplish tasks using routines—reusable building blocks combining standards, APIs, code, and more. Tailor routines for any purpose, like internal processes, tutorials, or cognitive architectures for bots. Routines can be run by users, bots, or both, enabling seamless collaboration and consistent, automated workflows. Routines are polymorphic, meaning their output adapts based on the executing agent's unique personality and role." },
                            { icon: <IconCommon name="Refresh" />, title: "Recursive Self-Improvement", text: "At the heart of Vrooli is recursive self-improvement. Bots utilize cognitive routines to continuously enhance their own capabilities and those of the platform. By analyzing metrics and identifying inefficiencies, routines (and the routines that improve routines) are constantly refined, enabling the platform to evolve and become more efficient over time." },
                            { icon: <IconCommon name="Comment" />, title: "Integrated Human and Bot Collaboration", text: "Vrooli's platform seamlessly integrates human and bot employees within teams. Engage with multiple bots and humans in the same conversation, facilitating smooth collaboration. Our upcoming messaging features will further enhance interaction, allowing users to guide, provide feedback to, and assign tasks to bots within their team environment." },
                            { icon: <IconCommon name="Wallet" />, title: "Reducing Costs Through Collaboration", text: "Vrooli tackles the high costs of starting and maintaining organizations. Since routines can be publicly shared and improved, humans and bots collaborate to enhance all teams simultaneously. As automation increases, the cost of running operations drops significantly, benefiting consumers, supporting growth, or enhancing employee benefits. Anyone can copy a public team, gaining access to refined processes and on-demand bot employees at no extra cost." },
                            { icon: <IconCommon name="HeartFilled" />, title: "Our Commitment", text: "At Vrooli, we're committed to ethical and beneficial outcomes. We prioritize responsible development and transparency. Our platform operates under the AGPLv3 license, ensuring that the software and its improvements remain free and open-source, fostering a collaborative ecosystem where technology serves society transparently and reliably." },
                            { icon: <IconService name="GitHub" />, title: "Get Involved", text: () => (<>Join us in building the future of automation! Explore Vrooli and see how our platform enables transparent and reliable autonomous systems. Visit our <Link href={SOCIALS.GitHub} target="_blank" rel="noopener">GitHub page</Link> to learn more, contribute, or join the team. We look forward to collaborating with you!</>) },
                        ].map((section, index) => (
                            <Box
                                key={section.title}
                                p={3}
                                borderRadius={2}
                                sx={{
                                    background: index % 2 === 0 ? palette.background.default : palette.background.paper,
                                }}
                            >
                                <Stack direction="row" spacing={1.5} alignItems="center" mb={1.5}>
                                    {section.icon && typeof section.icon !== "string" ? (
                                        <Box sx={{ color: palette.secondary.main, display: "flex", alignItems: "center" }}>
                                            {cloneElement(section.icon, { size: 28 })}
                                        </Box>
                                    ) : null}
                                    <Typography variant="h5" component="h2">{section.title}</Typography>
                                </Stack>
                                <Typography variant="body1">
                                    {typeof section.text === "function" ? section.text() : section.text}
                                </Typography>
                            </Box>
                        ))}
                    </Stack>

                    <Box
                        mt={4}
                        p={3}
                        borderRadius={2}
                        sx={{
                            background: `linear-gradient(to bottom, ${palette.background.paper}, ${theme.palette.mode === "dark" ? palette.grey[900] : palette.grey[100]})`,
                        }}
                    >
                        <Typography variant='h4' component="h2" pb={3} textAlign="start">The Team</Typography>
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
                                    <Box
                                        key={key}
                                        sx={{
                                            display: "flex",
                                            flexDirection: { xs: "column", sm: "row" },
                                            alignItems: "center",
                                            justifyContent: "flex-start",
                                            background: `linear-gradient(135deg, ${palette.primary.dark} 0%, ${palette.primary.main} 100%)`,
                                            color: palette.primary.contrastText,
                                            borderRadius: 3,
                                            padding: 3,
                                            overflow: "hidden",
                                            gap: 3,
                                            border: `1px solid ${palette.primary.light}20`,
                                        }}
                                    >
                                        <Box component="img" src={member.photo} alt={`${member.fullName} profile picture`} sx={{
                                            width: { xs: "50%", sm: "150px" },
                                            height: { xs: "auto", sm: "150px" },
                                            aspectRatio: "1 / 1",
                                            objectFit: "cover",
                                            borderRadius: "50%",
                                            flexShrink: 0,
                                            border: `3px solid ${palette.secondary.light}`,
                                        }} />
                                        <Box
                                            width={{ xs: "100%", sm: "auto" }}
                                            flexGrow={1}
                                            textAlign={{ xs: "center", sm: "left" }}
                                        >
                                            <Typography variant='h5' component="h3" mb={0.5} fontWeight="bold">{member.fullName}</Typography>
                                            <Typography variant='body1' mb={2} sx={{ color: palette.grey[300] }}>{member.role}</Typography>
                                            <Stack direction="row" alignItems="center" justifyContent={{ xs: "center", sm: "flex-start" }} spacing={1.5}>
                                                {member.socials.website && (
                                                    <Tooltip title="Personal website" placement="bottom">
                                                        <IconButton
                                                            aria-label="Personal website"
                                                            onClick={openPersonalWebsite}
                                                            sx={{ ...memberButtonProps, color: palette.secondary.light }}
                                                        >
                                                            <IconCommon decorative name="Website" size={36} />
                                                        </IconButton>
                                                    </Tooltip>
                                                )}
                                                {member.socials.x && (
                                                    <Tooltip title="X/Twitter" placement="bottom">
                                                        <IconButton
                                                            aria-label="X/Twitter"
                                                            onClick={openX}
                                                            sx={{ ...memberButtonProps, color: palette.secondary.light }}
                                                        >
                                                            <IconService decorative name="X" size={32} />
                                                        </IconButton>
                                                    </Tooltip>
                                                )}
                                                {member.socials.github && (
                                                    <Tooltip title="GitHub" placement="bottom">
                                                        <IconButton
                                                            aria-label="GitHub"
                                                            onClick={openGitHub}
                                                            sx={{ ...memberButtonProps, color: palette.secondary.light }}
                                                        >
                                                            <IconService decorative name="GitHub" size={32} />
                                                        </IconButton>
                                                    </Tooltip>
                                                )}
                                            </Stack>
                                        </Box>
                                    </Box>
                                );
                            })}
                            <Stack direction="row" justifyContent="center" alignItems="center" pt={2}>
                                <Button
                                    fullWidth
                                    size="large"
                                    color="secondary"
                                    href={joinTeamLink}
                                    onClick={handleJoinTeam}
                                    variant="contained"
                                    sx={{
                                        maxWidth: "400px",
                                        py: 1.5,
                                        fontSize: "1.1rem",
                                        "&:hover": {
                                            transform: "scale(1.03)",
                                            filter: "brightness(1.1)",
                                        },
                                        transition: "all 0.2s ease-in-out",
                                    }}
                                    startIcon={<IconCommon
                                        decorative
                                        name="Team"
                                        size={24}
                                    />}
                                >Join the Team</Button>
                            </Stack>
                        </Stack>
                    </Box>
                </Box>
                <Footer />
            </ScrollBox>
        </PageContainer>
    );
}
