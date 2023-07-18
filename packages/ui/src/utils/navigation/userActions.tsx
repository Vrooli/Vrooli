/* eslint-disable @typescript-eslint/no-redeclare */
import { CommonKey, LINKS, Session } from "@local/shared";
import { Badge, BottomNavigationAction, Button, IconButton, SxProps, Theme } from "@mui/material";
import i18next from "i18next";
import { CreateAccountIcon, CreateIcon, GridIcon, HelpIcon, HomeIcon, NotificationsAllIcon, PremiumIcon, SearchIcon } from "icons";
import { openLink, SetLocation } from "route";
import { SvgComponent } from "types";
import { checkIfLoggedIn } from "utils/authentication/session";

export enum ACTION_TAGS {
    Home = "Home",
    Search = "Search",
    Create = "Create",
    Inbox = "Inbox",
    About = "About",
    Pricing = "Pricing",
    LogIn = "LogIn",
    MyStuff = "MyStuff",
}

export type ActionArray = [string, any, string, any, number];
export interface Action {
    label: CommonKey;
    value: ACTION_TAGS;
    link: string;
    Icon: SvgComponent;
    numNotifications: number;
}

// Returns navigational actions available to the user
interface GetUserActionsProps {
    session?: Session | null | undefined;
    exclude?: ACTION_TAGS[] | null | undefined;
}
export function getUserActions({ session, exclude = [] }: GetUserActionsProps): Action[] {
    // Check if user is logged in using session
    const isLoggedIn = checkIfLoggedIn(session);
    const actions: ActionArray[] = [];
    // Home always available. Page changes based on login status, 
    // but we don't worry about that here.
    actions.push(
        ["Home", ACTION_TAGS.Home, LINKS.Home, HomeIcon, 0],
    );
    // Search always available
    actions.push(
        ["Search", ACTION_TAGS.Search, LINKS.Search, SearchIcon, 0],
    );
    // Actions for logged in users
    if (isLoggedIn) {
        actions.push(
            ["Create", ACTION_TAGS.Create, LINKS.Create, CreateIcon, 0],
            ["Inbox", ACTION_TAGS.Inbox, LINKS.Inbox, NotificationsAllIcon, 0],
            ["MyStuff", ACTION_TAGS.MyStuff, LINKS.MyStuff, GridIcon, 0],
        );
    }
    // Display about, pricing, and login for logged out users 
    else {
        actions.push(["About", ACTION_TAGS.About, LINKS.About, HelpIcon, 0]);
        actions.push(["Pricing", ACTION_TAGS.Pricing, LINKS.Premium, PremiumIcon, 0]);
        actions.push(["Log In", ACTION_TAGS.LogIn, LINKS.Start, CreateAccountIcon, 0]);
    }
    return actions.map(a => createAction(a)).filter(a => !(exclude ?? []).includes(a.value));
}

// Factory for creating action objects
const createAction = (action: ActionArray): Action => {
    const keys = ["label", "value", "link", "Icon", "numNotifications"];
    return action.reduce((obj: object, val: any, i: number) => { obj[keys[i]] = val; return obj; }, {}) as Action;
};

// Factory for creating a list of action objects
export const createActions = (actions: ActionArray[]): Action[] => actions.map(a => createAction(a));

// Display actions in a horizontal menu
interface ActionsToMenuProps {
    actions: Action[];
    setLocation: SetLocation;
    sx?: SxProps<Theme>;
}
export const actionsToMenu = ({ actions, setLocation, sx = {} }: ActionsToMenuProps) => {
    return actions.map(({ label, value, link }) => (
        <Button
            key={value}
            variant="text"
            size="large"
            href={link}
            onClick={(e) => { e.preventDefault(); openLink(setLocation, link); }}
            sx={sx}
        >
            {i18next.t(label, { count: 2 })}
        </Button>
    ));
};

// Display actions in a bottom navigation
interface ActionsToBottomNavProps {
    actions: Action[];
    setLocation: SetLocation;
}
export const actionsToBottomNav = ({ actions, setLocation }: ActionsToBottomNavProps) => {
    return actions.map(({ label, value, link, Icon, numNotifications }) => (
        <BottomNavigationAction
            key={value}
            label={i18next.t(label, { count: 2 })}
            value={value}
            href={link}
            onClick={(e: any) => {
                e.preventDefault();
                // Check if link is different from current location
                const shouldScroll = link === window.location.pathname;
                // If same, scroll to top of page instead of navigating
                if (shouldScroll) window.scrollTo({ top: 0, behavior: "smooth" });
                // Otherwise, navigate to link
                else openLink(setLocation, link);
            }}
            icon={<Badge badgeContent={numNotifications} color="error"><Icon /></Badge>}
            sx={{ color: "white" }}
        />
    ));
};

// Display an action as an icon button
interface ActionToIconButtonProps {
    action: Action;
    setLocation: SetLocation;
    classes?: { [key: string]: string };
}
export const actionToIconButton = ({ action, setLocation, classes = { root: "" } }: ActionToIconButtonProps) => {
    const { value, link, Icon, numNotifications } = action;
    return <IconButton classes={classes} edge="start" color="inherit" aria-label={value} onClick={() => openLink(setLocation, link)}>
        <Badge badgeContent={numNotifications} color="error">
            <Icon />
        </Badge>
    </IconButton>;
};
