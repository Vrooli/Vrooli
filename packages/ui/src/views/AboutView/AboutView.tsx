import { Box, Button, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { GitHubIcon, OrganizationIcon, TwitterIcon, WebsiteIcon } from "@shared/icons";
import { openLink, useLocation } from "@shared/route";
import MattProfilePic from 'assets/img/profile-matt.jpg';
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Slide } from "components/slides";
import { slideTitle, textPop } from "styles";
import { AboutViewProps } from "views/types";

type MemberData = {
    fullName: string;
    role: string;
    photo: string;
    socials: {
        website?: string;
        twitter?: string;
        github?: string;
    }
}

const memberButtonProps = {
    background: 'transparent',
    border: '0',
    '&:hover': {
        background: 'transparent',
        filter: 'brightness(1.2)',
        transform: 'scale(1.2)',
    },
    transition: 'all 0.2s ease',
}

const teamMembers: MemberData[] = [
    {
        fullName: 'Matt Halloran',
        role: 'Leader/developer',
        photo: MattProfilePic,
        socials: {
            website: 'https://matthalloran.info',
            twitter: 'https://twitter.com/mdhalloran',
            github: 'https://github.com/MattHalloran',
        }
    },
]

const joinTeamLink = 'https://github.com/Vrooli/Vrooli#-join-the-team';

export const AboutView = ({
    display = 'page',
}: AboutViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: 'AboutUs',
                    hideOnDesktop: true,
                }}
            />
            <Slide id="about-the-team">
                <Typography variant='h2' component="h1" pb={4} sx={{ ...slideTitle }}>The Team</Typography>
                {/* Vertical stack of cards, one for each member */}
                <Stack id="members-stack" direction="column" spacing={10}>
                    {teamMembers.map((member, key) => (
                        <Box
                            sx={{
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'space-between',
                                height: '300px',
                                boxShadow: 12,
                                backgroundColor: palette.primary.dark,
                                borderRadius: 2,
                                padding: 2,
                                overflow: 'overlay',
                            }}
                        >
                            {/* Image, positioned to left */}
                            <Box component="img" src={member.photo} alt={`${member.fullName} profile picture`} sx={{
                                maxWidth: 'min(300px, 40%)',
                                maxHeight: '200px',
                                objectFit: 'contain',
                                borderRadius: '100%',
                            }} />
                            {/* Name, role, and links */}
                            <Box sx={{
                                width: 'min(300px, 50%)',
                                height: 'fit-content',
                            }}>
                                <Typography variant='h4' mb={1} sx={{ ...textPop }}>{member.fullName}</Typography>
                                <Typography variant='h6' mb={2} sx={{ ...textPop }}>{member.role}</Typography>
                                <Stack direction="row" alignItems="center" justifyContent="center">
                                    {member.socials.website && (
                                        <Tooltip title="Personal website" placement="bottom">
                                            <IconButton onClick={() => openLink(setLocation, member.socials.website as string)} sx={memberButtonProps}>
                                                <WebsiteIcon fill={palette.secondary.light} width="50px" height="50px" />
                                            </IconButton>
                                        </Tooltip>
                                    )}
                                    {member.socials.twitter && (
                                        <Tooltip title="Twitter" placement="bottom">
                                            <IconButton onClick={() => openLink(setLocation, member.socials.twitter as string)} sx={memberButtonProps}>
                                                <TwitterIcon fill={palette.secondary.light} width="42px" height="42px" />
                                            </IconButton>
                                        </Tooltip>
                                    )}
                                    {member.socials.github && (
                                        <Tooltip title="GitHub" placement="bottom">
                                            <IconButton onClick={() => openLink(setLocation, member.socials.github as string)} sx={memberButtonProps}>
                                                <GitHubIcon fill={palette.secondary.light} width="42px" height="42px" />
                                            </IconButton>
                                        </Tooltip>
                                    )}
                                </Stack>
                            </Box>
                        </Box>
                    ))}
                    <Stack direction="row" justifyContent="center" alignItems="center" pt={4} spacing={2}>
                        <Button
                            size="large"
                            color="secondary"
                            href={joinTeamLink}
                            onClick={(e) => {
                                e.preventDefault();
                                openLink(setLocation, joinTeamLink);
                            }}
                            startIcon={<OrganizationIcon />}
                        >Join the Team</Button>
                    </Stack>
                </Stack>
            </Slide>
        </>
    )
}