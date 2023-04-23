import { LINKS } from "@local/consts";
import { LogInIcon, ProfileIcon } from "@local/icons";
import { Button, Container, IconButton, Palette, useTheme } from "@mui/material";
import React, { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { checkIfLoggedIn, getCurrentUser } from "../../../utils/authentication/session";
import { useWindowSize } from "../../../utils/hooks/useWindowSize";
import { Action, actionsToMenu, ACTION_TAGS, getUserActions } from "../../../utils/navigation/userActions";
import { openLink, useLocation } from "../../../utils/route";
import { SessionContext } from "../../../utils/SessionContext";
import { PopupMenu } from "../../buttons/PopupMenu/PopupMenu";
import { AccountMenu } from "../../dialogs/AccountMenu/AccountMenu";
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

    // Handle account menu
    const [accountMenuAnchor, setAccountMenuAnchor] = useState<any>(null);
    const openAccountMenu = useCallback((ev: React.MouseEvent<any>) => {
        ev.stopPropagation();
        setAccountMenuAnchor(ev.currentTarget);
    }, [setAccountMenuAnchor]);
    const closeAccountMenu = useCallback((ev: React.MouseEvent<any>) => {
        ev.stopPropagation();
        setAccountMenuAnchor(null);
    }, []);

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
            {/* Account menu */}
            <AccountMenu
                anchorEl={accountMenuAnchor}
                onClose={closeAccountMenu}
            />
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
                    sx={{
                        background: "#387e30",
                        borderRadius: "10px",
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
                <IconButton
                    color="inherit"
                    onClick={openAccountMenu}
                    aria-label="profile"
                >
                    <ProfileIcon fill='white' width='40px' height='40px' />
                </IconButton>
            )}
        </Container>
    );
};
