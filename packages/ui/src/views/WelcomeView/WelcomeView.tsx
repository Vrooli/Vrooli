import { ArticleIcon, LearnIcon, LINKS, openLink, PlayIcon, ProfileIcon, useLocation, WHITE_PAPER_URL } from "@local/shared";
import { Box, Button, Link, Stack, Typography, useTheme } from "@mui/material";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useContext, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { clickSize } from "styles";
import { checkIfLoggedIn } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { WelcomeViewProps } from "../types";

const buttonProps = {
    height: "48px",
    background: "white",
    color: "black",
    borderRadius: "10px",
    width: "20em",
    display: "flex",
    marginBottom: "5px",
    transition: "0.3s ease-in-out",
    "&:hover": {
        filter: "brightness(120%)",
        border: "1px solid white",
    },
};

export const WelcomeView = ({
    display = "page",
}: WelcomeViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    // Show confetti on page load, if it's the user's first time
    useEffect(() => {
        // Check storage for first time
        const firstTime = localStorage.getItem("firstTime");
        if (firstTime === null) {
            PubSub.get().publishCelebration();
            localStorage.setItem("firstTime", "false");
        }
    }, []);

    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: "WelcomeToVrooli",
                }}
            />
            <Box sx={{
                display: "flex",
                justifyContent: "center",
                alignItems: "center",
                textAlign: "center",
                marginTop: 4,
            }}>
                <Box>
                    <Typography component="h2" variant="h4" mb={3}>{t("NotSureWhereToStart")}</Typography>
                    <Stack direction="column" spacing={1} mb={2} sx={{ alignItems: "center" }}>
                        <Button
                            onClick={() => setLocation(LINKS.Tutorial)}
                            startIcon={<LearnIcon fill="black" />}
                            sx={{ ...buttonProps, marginBottom: 0 }}
                        >{t("Tutorial")}</Button>
                        <Button
                            onClick={() => setLocation(LINKS.Example)}
                            startIcon={<PlayIcon fill="black" />}
                            sx={{ ...buttonProps, marginBottom: 0 }}
                        >{t("RunExample")}</Button>
                        {checkIfLoggedIn(session) && <Button
                            onClick={() => setLocation(LINKS.SettingsProfile)}
                            startIcon={<ProfileIcon fill="black" />}
                            sx={{ ...buttonProps, marginBottom: 0 }}
                        >{t("SetUpProfile")}</Button>}
                        <Button
                            onClick={() => openLink(setLocation, WHITE_PAPER_URL)}
                            startIcon={<ArticleIcon fill="black" />}
                            sx={{ ...buttonProps, marginBottom: 0 }}
                        >{t("ReadWhitePaper")}</Button>
                    </Stack>
                    <Box sx={{
                        ...clickSize,
                        justifyContent: "end",
                    }}
                    >
                        <Link onClick={() => setLocation(LINKS.Home)} sx={{
                            cursor: "pointer",
                            "&:hover": {
                                brightness: "120%",
                            },
                        }}>
                            <Typography sx={{ marginRight: 2, color: palette.secondary.dark }}>{t("IKnowWhatImDoing")}</Typography>
                        </Link>
                    </Box>
                </Box>
            </Box>
        </>
    );
};
