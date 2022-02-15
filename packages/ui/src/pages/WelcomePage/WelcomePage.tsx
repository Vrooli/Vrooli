import { APP_LINKS } from '@local/shared';
import { Box, Button, Link, Stack, Tooltip, Typography } from '@mui/material';
import {
    Help as FAQIcon,
    Article as WhitePaperIcon,
    AccountCircle as ProfileIcon,
    PlayCircle as ExampleIcon,
} from '@mui/icons-material';
import { useLocation } from 'wouter';
import { clickSize } from 'styles';

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
    const [, setLocation] = useLocation();
    const openLink = (link: string) => window.open(link, '_blank', 'noopener,noreferrer');

    return (
        <Box
            id="page"
            sx={{
                padding: 2,
                background: `linear-gradient(-46deg, #d5aa75, #d264b5, #0f65d1, #69d7d2) center center / cover no-repeat fixed`,
                backgroundSize: '400% 400%',
                animation: 'gradient 10s ease infinite',
                overflowX: 'hidden',
            }}
        >
            <Box sx={{
                padding: 2,
                background: "#072781",
                color: 'white',
            }}>
                <Typography component="h1" mb={1}>Welcome to Vrooli</Typography>
                <Typography variant="h4" component="h1" mb={1}>Not sure where to start?</Typography>
                <Stack direction="column" spacing={1} mb={2} sx={{ alignItems: 'center' }}>
                    <Tooltip title="View Frequently Asked Questions">
                        <Button
                            onClick={() => setLocation(APP_LINKS.FAQ)}
                            startIcon={<FAQIcon />}
                            sx={{ ...buttonProps, marginBottom: 0 }}
                        >Read the FAQ</Button>
                    </Tooltip>
                    <Tooltip title="Read the white paper">
                        <Button
                            onClick={() => openLink("https://docs.google.com/document/d/1zHYdjAyy01SSFZX0O-YnZicef7t6sr1leOFnynQQOx4/edit?usp=sharing")}
                            startIcon={<WhitePaperIcon />}
                            sx={{ ...buttonProps, marginBottom: 0 }}
                        >Read the White Paper</Button>
                    </Tooltip>
                    <Tooltip title="Set up your profile and display settings">
                        <Button
                            onClick={() => setLocation(APP_LINKS.Profile)}
                            startIcon={<ProfileIcon />}
                            sx={{ ...buttonProps, marginBottom: 0 }}
                        >Set Up Profile</Button>
                    </Tooltip>
                    <Tooltip title="Run your first routine">
                        <Button
                            onClick={() => setLocation(APP_LINKS.Example)}
                            startIcon={<ExampleIcon />}
                            sx={{ ...buttonProps, marginBottom: 0 }}
                        >Run Example</Button>
                    </Tooltip>
                </Stack>
                <Box sx={{
                    ...clickSize,
                    justifyContent: 'end',
                }}
                >
                    <Link onClick={() => setLocation(APP_LINKS.Home)}>
                        <Typography sx={{ marginRight: 2, color: (t) => t.palette.secondary.dark }}>I know what I'm doing</Typography>
                    </Link>
                </Box>
            </Box>
        </Box>
    )
}