import { LINKS } from "@local/shared";
import { Avatar, Box, BoxProps, Button, Palette, styled, useTheme } from "@mui/material";
import { PopupMenu } from "components/buttons/PopupMenu/PopupMenu";
import { SessionContext } from "contexts/SessionContext";
import { useIsLeftHanded } from "hooks/useIsLeftHanded";
import { useSideMenu } from "hooks/useSideMenu";
import { useWindowSize } from "hooks/useWindowSize";
import { LogInIcon, ProfileIcon } from "icons";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { openLink, useLocation } from "route";
import { checkIfLoggedIn, getCurrentUser } from "utils/authentication/session";
import { extractImageUrl } from "utils/display/imageTools";
import { NAV_ACTION_TAGS, NavAction, actionsToMenu, getUserActions } from "utils/navigation/userActions";
import { PubSub, SIDE_MENU_ID } from "utils/pubsub";
import { ContactInfo } from "../ContactInfo/ContactInfo";

function navItemStyle(palette: Palette) {
    return {
        background: "transparent",
        color: palette.primary.contrastText,
        textTransform: "none",
        fontSize: "1.4em",
        "&:hover": {
            color: palette.secondary.light,
        },
    } as const;
}

interface OuterBoxProps extends BoxProps {
    isLeftHanded: boolean;
}

const OuterBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isLeftHanded",
})<OuterBoxProps>(({ isLeftHanded }) => ({
    display: "flex",
    height: "100%",
    paddingBottom: "0",
    paddingRight: "0 !important",
    right: "0",
    // Reverse order on left handed mode
    flexDirection: isLeftHanded ? "row-reverse" : "row",
}));

const EnterButton = styled(Button)(({ theme }) => ({
    whiteSpace: "nowrap",
    // Hide text on small screens, and remove start icon's padding
    fontSize: "1.4em",
    [theme.breakpoints.down("md")]: {
        fontSize: "1em",
    },
    [theme.breakpoints.down("sm")]: {
        fontSize: "0px",
        "& .MuiButton-startIcon": {
            marginLeft: "0px",
            marginRight: "0px",
        },
    },
}));

const ProfileAvatar = styled(Avatar)(({ theme }) => ({
    background: theme.palette.primary.contrastText,
    width: "40px",
    height: "40px",
    margin: "auto",
    marginRight: theme.spacing(1),
    cursor: "pointer",
}));

export function NavList() {
    const session = useContext(SessionContext);
    const user = useMemo(() => getCurrentUser(session), [session]);
    const isLeftHanded = useIsLeftHanded();
    const { t } = useTranslation();
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();

    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const navActions = useMemo<NavAction[]>(() => getUserActions({ session, exclude: [NAV_ACTION_TAGS.Home, NAV_ACTION_TAGS.LogIn] }), [session]);

    const { isOpen: isSideMenuOpen } = useSideMenu({ id: SIDE_MENU_ID, isMobile });
    const openSideMenu = useCallback(() => { PubSub.get().publish("sideMenu", { id: SIDE_MENU_ID, isOpen: true }); }, []);

    return (
        <OuterBox isLeftHanded={isLeftHanded}>
            {/* Contact menu */}
            {!isMobile && !getCurrentUser(session).id && !isSideMenuOpen && <PopupMenu
                text={t("Contact")}
                variant="text"
                size="large"
                sx={navItemStyle(palette)}
            >
                <ContactInfo />
            </PopupMenu>}
            {/* List items displayed when on wide screen */}
            <Box>
                {!isMobile && !isSideMenuOpen && actionsToMenu({
                    actions: navActions,
                    setLocation,
                    sx: navItemStyle(palette),
                })}
            </Box>
            {/* Enter button displayed when not logged in */}
            {!checkIfLoggedIn(session) && (
                <EnterButton
                    href={LINKS.Login}
                    onClick={(e) => { e.preventDefault(); openLink(setLocation, LINKS.Login); }}
                    startIcon={<LogInIcon />}
                    variant="contained"
                >
                    {t("LogIn")}
                </EnterButton>
            )}
            {checkIfLoggedIn(session) && (
                <ProfileAvatar
                    id="side-menu-profile-icon"
                    src={extractImageUrl(user.profileImage, user.updated_at, 50)}
                    onClick={openSideMenu}
                >
                    <ProfileIcon fill={palette.primary.dark} width="100%" height="100%" />
                </ProfileAvatar>
            )}
        </OuterBox>
    );
}
