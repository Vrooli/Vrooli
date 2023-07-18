import { LINKS } from "@local/shared";
import { Avatar, Button, Container, Palette, useTheme } from "@mui/material";
import { PopupMenu } from "components/buttons/PopupMenu/PopupMenu";
import { SideMenu } from "components/dialogs/SideMenu/SideMenu";
import { LogInIcon, ProfileIcon } from "icons";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { openLink, useLocation } from "route";
import { checkIfLoggedIn, getCurrentUser } from "utils/authentication/session";
import { useWindowSize } from "utils/hooks/useWindowSize";
import { Action, actionsToMenu, ACTION_TAGS, getUserActions } from "utils/navigation/userActions";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
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
    const { t } = useTranslation();
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();

    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const nav_actions = useMemo<Action[]>(() => getUserActions({ session, exclude: [ACTION_TAGS.Home, ACTION_TAGS.LogIn] }), [session]);

    const toggleSideMenu = useCallback(() => { PubSub.get().publishSideMenu(); }, []);

    return (
        <Container sx={{
            display: "flex",
            height: "100%",
            paddingBottom: "0",
            paddingRight: "0 !important",
            right: "0",
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
            {/* Side menu */}
            <SideMenu />
            {/* List items displayed when on wide screen */}
            {!isMobile && actionsToMenu({
                actions: nav_actions,
                setLocation,
                sx: navItemStyle(palette),
            })}
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
                    src="/broken-image.jpg" //TODO
                    onClick={toggleSideMenu}
                    sx={{
                        background: palette.primary.contrastText,
                        width: "40px",
                        height: "40px",
                        margin: "auto",
                        cursor: "pointer",
                    }}
                >
                    <ProfileIcon fill={palette.primary.dark} width="100%" height="100%" />
                </Avatar>
            )}
        </Container>
    );
};
