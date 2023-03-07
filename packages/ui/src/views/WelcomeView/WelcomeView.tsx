import { APP_LINKS, WHITE_PAPER_URL } from '@shared/consts';
import { Box, Button, Link, Stack, Typography, useTheme } from '@mui/material';
import { openLink, useLocation } from '@shared/route';
import { clickSize } from 'styles';
import { useEffect } from 'react';
import { PubSub } from 'utils';
import { ArticleIcon, LearnIcon, PlayIcon, ProfileIcon } from '@shared/icons';
import { checkIfLoggedIn } from 'utils/authentication';
import { useTranslation } from 'react-i18next';
import { WelcomeViewProps } from '../types';
import { TopBar } from 'components';

const buttonProps = {
    height: "48px",
    background: "white",
    color: 'black',
    borderRadius: "10px",
    width: "20em",
    display: "flex",
    marginBottom: "5px",
    transition: "0.3s ease-in-out",
    '&:hover': {
        filter: `brightness(120%)`,
        border: '1px solid white',
    }
}

export const WelcomeView = ({
    display = 'page',
    session
}: WelcomeViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

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
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                session={session}
                titleData={{
                    titleKey: 'WelcomeToVrooli',
                }}
            />
            <Box sx={{
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
                textAlign: 'center',
                marginTop: 4,
            }}>
                <Box>
                    <Typography component="h2" variant="h4" mb={3}>{t(`NotSureWhereToStart`)}</Typography>
                    <Stack direction="column" spacing={1} mb={2} sx={{ alignItems: 'center' }}>
                        <Button
                            onClick={() => setLocation(APP_LINKS.Tutorial)}
                            startIcon={<LearnIcon fill="black" />}
                            sx={{ ...buttonProps, marginBottom: 0 }}
                        >{t(`Tutorial`)}</Button>
                        <Button
                            onClick={() => setLocation(APP_LINKS.Example)}
                            startIcon={<PlayIcon fill="black" />}
                            sx={{ ...buttonProps, marginBottom: 0 }}
                        >{t(`RunExample`)}</Button>
                        {checkIfLoggedIn(session) && <Button
                            onClick={() => setLocation(`${APP_LINKS.Settings}?page="profile"`)}
                            startIcon={<ProfileIcon fill="black" />}
                            sx={{ ...buttonProps, marginBottom: 0 }}
                        >{t(`SetUpProfile`)}</Button>}
                        <Button
                            onClick={() => openLink(setLocation, WHITE_PAPER_URL)}
                            startIcon={<ArticleIcon fill="black" />}
                            sx={{ ...buttonProps, marginBottom: 0 }}
                        >{t(`ReadWhitePaper`)}</Button>
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
                            <Typography sx={{ marginRight: 2, color: palette.secondary.dark }}>{t(`IKnowWhatImDoing`)}</Typography>
                        </Link>
                    </Box>
                </Box>
            </Box>
        </>
    )
}