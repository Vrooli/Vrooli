/* eslint-disable @typescript-eslint/no-redeclare */
import { LINKS, type Session, type TranslationKeyCommon } from "@vrooli/shared";
import { type IconInfo } from "../../icons/Icons.js";
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
    const keys = ["label", "value", "link", "iconInfo", "numNotifications"];
    return action.reduce((obj, val, i) => { obj[keys[i]] = val; return obj; }, {}) as NavAction;
}
