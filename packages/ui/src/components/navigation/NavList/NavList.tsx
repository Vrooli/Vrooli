import { LINKS } from "@local/shared";
import { Avatar, Box, Button, Container, Palette, useTheme } from "@mui/material";
import { PopupMenu } from "components/buttons/PopupMenu/PopupMenu";
import { SessionContext } from "contexts/SessionContext";
import { useIsLeftHanded } from "hooks/useIsLeftHanded";
import { useWindowSize } from "hooks/useWindowSize";
import { LogInIcon, ProfileIcon } from "icons";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { openLink, useLocation } from "route";
import { checkIfLoggedIn, getCurrentUser } from "utils/authentication/session";
import { extractImageUrl } from "utils/display/imageTools";
import { actionsToMenu, getUserActions, NavAction, NAV_ACTION_TAGS } from "utils/navigation/userActions";
import { PubSub } from "utils/pubsub";
import { ContactInfo } from "../ContactInfo/ContactInfo";

const navItemStyle = (palette: Palette) => ({
    background: "transparent",
    color: palette.primary.contrastText,
    textTransform: "none",
    fontSize: "1.4em",
    "&:hover": {
        color: palette.secondary.light,
    },
});

export const NavList = () => {
    const session = useContext(SessionContext);
    const user = useMemo(() => getCurrentUser(session), [session]);
    const isLeftHanded = useIsLeftHanded();
    const { t } = useTranslation();
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();

    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const navActions = useMemo<NavAction[]>(() => getUserActions({ session, exclude: [NAV_ACTION_TAGS.Home, NAV_ACTION_TAGS.LogIn] }), [session]);

    const toggleSideMenu = useCallback(() => { PubSub.get().publishSideMenu({ id: "side-menu", isOpen: true }); }, []);

    return (
        <Container sx={{
            display: "flex",
            height: "100%",
            paddingBottom: "0",
            paddingRight: "0 !important",
            right: "0",
            // Reverse order on left handed mode
            flexDirection: isLeftHanded ? "row-reverse" : "row",
        }}>
            {/* Contact menu */}
            {!isMobile && !getCurrentUser(session).id && <PopupMenu
                text={t("Contact")}
                variant="text"
                size="large"
                sx={navItemStyle(palette)}
            >
                <ContactInfo />
            </PopupMenu>}
            {/* List items displayed when on wide screen */}
            <Box>
                {!isMobile && actionsToMenu({
                    actions: navActions,
                    setLocation,
                    sx: navItemStyle(palette),
                })}
            </Box>
            {/* Enter button displayed when not logged in */}
            {!checkIfLoggedIn(session) && (
                <Button
                    href={LINKS.Start}
                    onClick={(e) => { e.preventDefault(); openLink(setLocation, LINKS.Start); }}
                    startIcon={<LogInIcon />}
                    variant="contained"
                    sx={{
                        whiteSpace: "nowrap",
                        // Hide text on small screens, and remove start icon's padding
                        fontSize: { xs: "0px", sm: "1em", md: "1.4em" },
                        "& .MuiButton-startIcon": {
                            marginLeft: { xs: "0px", sm: "-4px" },
                            marginRight: { xs: "0px", sm: "8px" },
                        },
                    }}
                >
                    {t("LogIn")}
                </Button>
            )}
            {/* Profile icon */}
            {checkIfLoggedIn(session) && (
                <Avatar
                    id="side-menu-profile-icon"
                    src={extractImageUrl(user.profileImage, user.updated_at, 50)}
                    onClick={toggleSideMenu}
                    sx={{
                        background: palette.primary.contrastText,
                        width: "40px",
                        height: "40px",
                        margin: "auto",
                        marginRight: 1,
                        cursor: "pointer",
                    }}
                >
                    <ProfileIcon fill={palette.primary.dark} width="100%" height="100%" />
                </Avatar>
            )}
        </Container>
    );
};
