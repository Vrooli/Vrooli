import { APP_LINKS } from '@shared/consts';
import { Box, Button, Link, Stack, Typography, useTheme } from '@mui/material';
import {
    Help as FAQIcon,
    Article as WhitePaperIcon,
    AccountCircle as ProfileIcon,
    PlayCircle as ExampleIcon,
    YouTube as VideoIcon,
} from '@mui/icons-material';
import { useLocation } from '@shared/route';
import { clickSize } from 'styles';
import { useEffect } from 'react';
import { PubSub } from 'utils';

const buttonProps = {
    height: "48px",
    background: "white",
    color: "black",
    borderRadius: "10px",
    width: "20em",
    display: "flex",
    marginBottom: "5px",
    transition: "0.3s ease-in-out",
    '&:hover': {
        filter: `brightness(120%)`,
        color: 'white',
        border: '1px solid white',
    }
}

export const WelcomePage = () => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const openLink = (link: string) => window.open(link, '_blank', 'noopener,noreferrer');

    // Show confetti on page load, if it's the user's first time
    useEffect(() => {
        // Check storage for first time
        const firstTime = localStorage.getItem('firstTime');
        if (firstTime === null) {
            PubSub.get().publishCelebration();
            localStorage.setItem('firstTime', 'false');
        }
    }, []);

    return (
        <Box
            sx={{
                minHeight: '100vh',
                width: '100vw',
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
                textAlign: 'center',
                animation: 'gradient 15s ease infinite',
                overflowX: 'hidden',
                paddingTop: { xs: '64px', md: '80px' },
            }}
        >
            <Box sx={{
                boxShadow: `rgb(0 0 0 / 50%) 0px 0px 35px 0px`,
                padding: 2,
                borderRadius: 2,
                overflow: 'overlay',
                marginTop: '-5vh',
                background: palette.mode === 'light' ? palette.primary.dark : palette.background.paper,
                color: palette.mode === 'light' ? palette.primary.contrastText : palette.background.textPrimary,
            }}>
                <Typography component="h1" variant="h2" mb={1}>Welcome to Vrooli!</Typography>
                <Typography component="h2" variant="h4" mb={3}>Not sure where to start?</Typography>
                <Stack direction="column" spacing={1} mb={2} sx={{ alignItems: 'center' }}>
                    <Button
                        onClick={() => openLink("https://www.youtube.com/watch?v=hBHaPYi5esQ")}
                        startIcon={<VideoIcon />}
                        sx={{ ...buttonProps, marginBottom: 0 }}
                    >Understand the vision</Button>
                    <Button
                        onClick={() => setLocation(APP_LINKS.FAQ)}
                        startIcon={<FAQIcon />}
                        sx={{ ...buttonProps, marginBottom: 0 }}
                    >Read the FAQ</Button>
                    <Button
                        onClick={() => openLink("https://docs.google.com/document/d/1zHYdjAyy01SSFZX0O-YnZicef7t6sr1leOFnynQQOx4?usp=sharing")}
                        startIcon={<WhitePaperIcon />}
                        sx={{ ...buttonProps, marginBottom: 0 }}
                    >Read the White Paper</Button>
                    <Button
                        onClick={() => setLocation(`${APP_LINKS.Settings}?page="profile"`)}
                        startIcon={<ProfileIcon />}
                        sx={{ ...buttonProps, marginBottom: 0 }}
                    >Set Up Profile</Button>
                    <Button
                        onClick={() => setLocation(APP_LINKS.Example)}
                        startIcon={<ExampleIcon />}
                        sx={{ ...buttonProps, marginBottom: 0 }}
                    >Run Example</Button>
                </Stack>
                <Box sx={{
                    ...clickSize,
                    justifyContent: 'end',
                }}
                >
                    <Link onClick={() => setLocation(APP_LINKS.Home)} sx={{
                        cursor: 'pointer',
                        '&:hover': {
                            brightness: '120%',
                        }
                    }}>
                        <Typography sx={{ marginRight: 2, color: palette.secondary.light }}>I know what I'm doing</Typography>
                    </Link>
                </Box>
            </Box>
        </Box>
    )
}