/* eslint-disable @typescript-eslint/no-redeclare */
import { LINKS, Session, TranslationKeyCommon } from "@local/shared";
import { Badge, BottomNavigationAction, Button, IconButton, SxProps, Theme } from "@mui/material";
import i18next from "i18next";
import { Icon, IconInfo } from "../../icons/Icons.js";
import { openLink } from "../../route/openLink.js";
import { SetLocation } from "../../route/types.js";
import { checkIfLoggedIn } from "../../utils/authentication/session.js";

export enum NAV_ACTION_TAGS {
    Home = "Home",
    Search = "Search",
    Create = "Create",
    Inbox = "Inbox",
    About = "About",
    Pricing = "Pricing",
    LogIn = "LogIn",
    MyStuff = "MyStuff",
}

export type NavActionArray = [string, NAV_ACTION_TAGS, LINKS, IconInfo, number];
export type NavAction = {
    label: TranslationKeyCommon;
    value: NAV_ACTION_TAGS;
    iconInfo: IconInfo;
    link: string;
    numNotifications: number;
}

// Returns navigational actions available to the user
interface GetUserActionsProps {
    session?: Session | null | undefined;
    exclude?: NAV_ACTION_TAGS[] | null | undefined;
}
export function getUserActions({ session, exclude = [] }: GetUserActionsProps): NavAction[] {
    // Check if user is logged in using session
    const isLoggedIn = checkIfLoggedIn(session);
    const actions: NavActionArray[] = [];
    // Home always available. Page changes based on login status, 
    // but we don't worry about that here.
    actions.push(
        ["Home", NAV_ACTION_TAGS.Home, LINKS.Home, { name: "Home", type: "Common" }, 0],
    );
    // Search always available
    actions.push(
        ["Search", NAV_ACTION_TAGS.Search, LINKS.Search, { name: "Search", type: "Common" }, 0],
    );
    // Actions for logged in users
    if (isLoggedIn) {
        actions.push(
            ["Create", NAV_ACTION_TAGS.Create, LINKS.Create, { name: "Create", type: "Common" }, 0],
            ["Inbox", NAV_ACTION_TAGS.Inbox, LINKS.Inbox, { name: "NotificationsAll", type: "Common" }, 0],
            ["MyStuff", NAV_ACTION_TAGS.MyStuff, LINKS.MyStuff, { name: "Grid", type: "Common" }, 0],
        );
    }
    // Display about, pricing, and login for logged out users 
    else {
        actions.push(["About", NAV_ACTION_TAGS.About, LINKS.About, { name: "Help", type: "Common" }, 0]);
        actions.push(["Pricing", NAV_ACTION_TAGS.Pricing, LINKS.Pro, { name: "Premium", type: "Common" }, 0]);
        actions.push(["Log In", NAV_ACTION_TAGS.LogIn, LINKS.Login, { name: "CreateAccount", type: "Common" }, 0]);
    }
    return actions.map(a => createAction(a)).filter(a => !(exclude ?? []).includes(a.value));
}

/** Factory for creating action objects */
function createAction(action: NavActionArray): NavAction {
    const keys = ["label", "value", "link", "Icon", "numNotifications"];
    return action.reduce((obj, val, i) => { obj[keys[i]] = val; return obj; }, {}) as NavAction;
}

// Factory for creating a list of action objects
export function createActions(actions: NavActionArray[]): NavAction[] {
    return actions.map(a => createAction(a));
}

/** Display actions in a horizontal menu */
interface ActionsToMenuProps {
    actions: NavAction[];
    setLocation: SetLocation;
    sx?: SxProps<Theme>;
}
export function actionsToMenu({ actions, setLocation, sx = {} }: ActionsToMenuProps) {
    return actions.map(({ label, value, link }) => {
        function handleClick(event: React.MouseEvent) {
            event.preventDefault();
            openLink(setLocation, link);
        }

        return (
            <Button
                key={value}
                variant="text"
                size="large"
                href={link}
                onClick={handleClick}
                sx={sx}
            >
                {i18next.t(label, { count: 2 })}
            </Button>
        );
    });
}

// Display actions in a bottom navigation
interface ActionsToBottomNavProps {
    actions: NavAction[];
    setLocation: SetLocation;
}
const navActionStyle = {
    color: "white",
    minWidth: "58px", // Default min width is too big for some screens, like closed Galaxy Fold 
};
export function actionsToBottomNav({ actions, setLocation }: ActionsToBottomNavProps) {
    return actions.map(({ label, value, iconInfo, link, numNotifications }) => {
        function handleNavActionClick(e: React.MouseEvent) {
            e.preventDefault();
            // Check if link is different from current location
            const shouldScroll = link === window.location.pathname;
            // If same, scroll to top of page instead of navigating
            if (shouldScroll) window.scrollTo({ top: 0, behavior: "smooth" });
            // Otherwise, navigate to link
            else openLink(setLocation, link);
        }
        return (
            <BottomNavigationAction
                key={value}
                label={i18next.t(label, { count: 2 })}
                value={value}
                href={link}
                onClick={handleNavActionClick}
                icon={<Badge badgeContent={numNotifications} color="error">
                    <Icon
                        decorative
                        info={iconInfo}
                    />
                </Badge>}
                sx={navActionStyle}
            />
        );
    });
}

// Display an action as an icon button
interface ActionToIconButtonProps {
    action: NavAction;
    setLocation: SetLocation;
    classes?: { [key: string]: string };
}
export function actionToIconButton({ action, setLocation, classes = { root: "" } }: ActionToIconButtonProps) {
    const { value, iconInfo, link, numNotifications } = action;

    function handleClick(event: React.MouseEvent) {
        event.preventDefault();
        openLink(setLocation, link);
    }

    return <IconButton classes={classes} edge="start" color="inherit" aria-label={value} onClick={handleClick}>
        <Badge badgeContent={numNotifications} color="error">
            <Icon
                decorative
                info={iconInfo}
            />
        </Badge>
    </IconButton>;
}
