import { Box, Button, IconButton, Link, Stack, Tooltip, Typography, keyframes, styled, useTheme } from "@mui/material";
import MattProfilePic from "assets/img/profile-matt.webp";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { PageContainer } from "../../components/Page/Page.js";
import { Footer } from "../../components/navigation/Footer/Footer.js";
import { TopBar } from "../../components/navigation/TopBar/TopBar.js";
import { GitHubIcon, TeamIcon, WebsiteIcon, XIcon } from "../../icons/common.js";
import { openLink } from "../../route/openLink.js";
import { useLocation } from "../../route/router.js";
import { ScrollBox } from "../../styles.js";
import { AboutViewProps } from "../types.js";

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

export function AboutView({
    display,
    onClose,
}: AboutViewProps) {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const { palette, transitions } = useTheme();

    const handleJoinTeam = useCallback(function handleJoinTeamCallback(event: React.MouseEvent<HTMLButtonElement>) {
        event.preventDefault();
        openLink(setLocation, joinTeamLink);
    }, [setLocation]);

    return (
        <PageContainer size="fullSize">
            <ScrollBox>
                <TopBar
                    display={display}
                    onClose={onClose}
                    title={t("AboutUs")}
                    titleBehaviorDesktop="ShowIn"
                />
                <Box
                    display="flex"
                    flexDirection="column"
                    gap={6}
                    p={1}
                    pt={2}
                    pb={2}
                    margin="auto"
                    maxWidth="min(800px, 100%)"
                >
                    <Box>
                        <Typography variant="h4" gutterBottom>
                            Hello there! <RotatedBox sx={{ display: "inline-block" }}>👋</RotatedBox>
                        </Typography>
                        <Typography variant="body1">
                            Welcome to Vrooli, a platform designed to tackle the challenges of transparency and reliability in autonomous systems. We&apos;re passionate about creating a cooperative organizational layer that fosters collaboration between humans and digital actors in a decentralized manner. Our top priority is developing systems that are both ethical and beneficial to society, instead of just chasing profit or power.
                        </Typography>
                    </Box>
                    <Box>
                        <Typography variant="h5" gutterBottom>
                            The Hypergraph
                        </Typography>
                        <Typography variant="body1">
                            At the heart of Vrooli is the concept of a hypergraph of routines. This innovative structure represents an automated economy through interconnected routines, with each routine linked to a specific task or function. By embracing a flexible and adaptable approach to building autonomous systems, we can meet the ever-changing needs of various industries while keeping our focus on transparency and reliability.
                        </Typography>
                    </Box>
                    <Box>
                        <Typography variant="h5" gutterBottom>
                            Streamlining Organizational Processes
                        </Typography>
                        <Typography variant="body1">
                            Leveraging advanced language models, Vrooli will soon be able to generate standards and routines that automate intra- and inter-organizational processes. We stand out from traditional automation tools by allowing processes to be built hierarchically, reducing the complexity of automating a business. For instance, a routine could be created to handle customer support inquiries, which then branches out into subroutines for processing refunds, answering frequently asked questions, or escalating issues to the right team members. With our program, you can group these processes into a team, making it easy to create your own business by simply copying a team template and accessing its internal processes.
                        </Typography>
                    </Box>
                    <Box>
                        <Typography variant="h5" gutterBottom>
                            Integrating Human and Bot Employees
                        </Typography>
                        <Typography variant="body1">
                            Vrooli&apos;s platform is designed with both human and bot employees in mind, ensuring seamless integration within a team. We&apos;re excited to be launching a messaging feature later this year that allows users to interact with bots for guidance, feedback, and task assignments within your team. This feature will promote better communication and collaboration between human employees and bots, leading to a more efficient and cohesive work environment.
                        </Typography>
                    </Box>
                    <Box>
                        <Typography variant="h5" gutterBottom>
                            Reducing Costs
                        </Typography>
                        <Typography variant="body1">
                            Starting and maintaining an organization can be expensive, but Vrooli has a solution. Since the hypergraph is publicly defined and shared, humans and bots can collaborate to improve all teams simultaneously. As teams become more automated, the cost of running businesses that don&apos;t rely heavily on physical labor will drop significantly. These savings can benefit consumers, support business growth and development, or even enhance employee benefits. The upcoming adoption of humanoid robots and autonomous vehicles will further unlock the potential of this system and increase individual productivity.
                        </Typography>
                    </Box>
                    <Box>
                        <Typography variant="h5" gutterBottom>
                            Our Commitment
                        </Typography>
                        <Typography variant="body1">
                            At Vrooli, we&apos;re committed to ethical and beneficial outcomes in the development of autonomous systems. By concentrating on responsible and sustainable development, our platform champions a collaborative and distributed approach to promoting transparency and reliability across diverse industries.
                        </Typography>
                    </Box>
                    <Box>
                        <Typography variant="h5" gutterBottom>
                            Get Involved
                        </Typography>
                        <Typography variant="body1">
                            We&apos;d love for you to join us in learning more about Vrooli and how our platform can contribute to creating transparent and reliable autonomous systems. Together, we can build a future where technology serves society in the most ethical and transparent way possible. To get started, visit our <Link href="https://github.com/Vrooli/Vrooli#-join-the-team" target="_blank" rel="noopener">GitHub page</Link>. We&apos;re looking forward to embarking on this exciting journey with you!
                        </Typography>
                    </Box>
                    <Box>
                        <Typography variant='h4' pb={2} textAlign="start">The Team</Typography>
                        {/* Vertical stack of cards, one for each member */}
                        <Stack id="members-stack" direction="column" spacing={4}>
                            {teamMembers.map((member, key) => (
                                <Box
                                    key={key}
                                    sx={{
                                        display: "flex",
                                        alignItems: "center",
                                        justifyContent: "space-between",
                                        backgroundColor: palette.primary.dark,
                                        color: palette.primary.contrastText,
                                        borderRadius: 2,
                                        padding: 2,
                                        overflow: "overlay",
                                    }}
                                >
                                    {/* Image, positioned to left */}
                                    <Box component="img" src={member.photo} alt={`${member.fullName} profile picture`} sx={{
                                        maxWidth: "30%",
                                        maxHeight: "150px",
                                        objectFit: "contain",
                                        borderRadius: "100%",
                                    }} />
                                    {/* Name, role, and links */}
                                    <Box sx={{
                                        width: "70%",
                                        height: "fit-content",
                                    }}>
                                        <Typography variant='h4' component="h5" mb={1} textAlign="center">{member.fullName}</Typography>
                                        <Typography variant='body1' mb={2} textAlign="center">{member.role}</Typography>
                                        <Stack direction="row" alignItems="center" justifyContent="center">
                                            {member.socials.website && (
                                                <Tooltip title="Personal website" placement="bottom">
                                                    <IconButton onClick={() => openLink(setLocation, member.socials.website as string)} sx={memberButtonProps}>
                                                        <WebsiteIcon fill={palette.secondary.light} width="42px" height="42px" />
                                                    </IconButton>
                                                </Tooltip>
                                            )}
                                            {member.socials.x && (
                                                <Tooltip title="X/Twitter" placement="bottom">
                                                    <IconButton onClick={() => openLink(setLocation, member.socials.x as string)} sx={memberButtonProps}>
                                                        <XIcon fill={palette.secondary.light} width="36px" height="36px" />
                                                    </IconButton>
                                                </Tooltip>
                                            )}
                                            {member.socials.github && (
                                                <Tooltip title="GitHub" placement="bottom">
                                                    <IconButton onClick={() => openLink(setLocation, member.socials.github as string)} sx={memberButtonProps}>
                                                        <GitHubIcon fill={palette.secondary.light} width="36px" height="36px" />
                                                    </IconButton>
                                                </Tooltip>
                                            )}
                                        </Stack>
                                    </Box>
                                </Box>
                            ))}
                            <Stack direction="row" justifyContent="center" alignItems="center">
                                <Button
                                    fullWidth
                                    size="large"
                                    color="secondary"
                                    href={joinTeamLink}
                                    onClick={handleJoinTeam}
                                    variant="contained"
                                    startIcon={<TeamIcon />}
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
