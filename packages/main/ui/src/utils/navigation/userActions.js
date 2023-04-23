import { jsx as _jsx } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { CreateAccountIcon, CreateIcon, GridIcon, HelpIcon, HomeIcon, NotificationsAllIcon, PremiumIcon, SearchIcon } from "@local/icons";
import { Badge, BottomNavigationAction, Button, IconButton } from "@mui/material";
import i18next from "i18next";
import { checkIfLoggedIn } from "../authentication/session";
import { openLink } from "../route";
export var ACTION_TAGS;
(function (ACTION_TAGS) {
    ACTION_TAGS["Home"] = "Home";
    ACTION_TAGS["Search"] = "Search";
    ACTION_TAGS["Create"] = "Create";
    ACTION_TAGS["Notifications"] = "Notifications";
    ACTION_TAGS["About"] = "About";
    ACTION_TAGS["Pricing"] = "Pricing";
    ACTION_TAGS["LogIn"] = "LogIn";
    ACTION_TAGS["MyStuff"] = "MyStuff";
})(ACTION_TAGS || (ACTION_TAGS = {}));
export function getUserActions({ session, exclude = [] }) {
    const isLoggedIn = checkIfLoggedIn(session);
    const actions = [];
    actions.push(["Home", ACTION_TAGS.Home, LINKS.Home, HomeIcon, 0]);
    actions.push(["Search", ACTION_TAGS.Search, LINKS.Search, SearchIcon, 0]);
    if (isLoggedIn) {
        actions.push(["Create", ACTION_TAGS.Create, LINKS.Create, CreateIcon, 0], ["Inbox", ACTION_TAGS.Notifications, LINKS.Notifications, NotificationsAllIcon, 0], ["MyStuff", ACTION_TAGS.MyStuff, LINKS.MyStuff, GridIcon, 0]);
    }
    else {
        actions.push(["About", ACTION_TAGS.About, LINKS.About, HelpIcon, 0]);
        actions.push(["Pricing", ACTION_TAGS.Pricing, LINKS.Premium, PremiumIcon, 0]);
        actions.push(["Log In", ACTION_TAGS.LogIn, LINKS.Start, CreateAccountIcon, 0]);
    }
    return actions.map(a => createAction(a)).filter(a => !(exclude ?? []).includes(a.value));
}
const createAction = (action) => {
    const keys = ["label", "value", "link", "Icon", "numNotifications"];
    return action.reduce((obj, val, i) => { obj[keys[i]] = val; return obj; }, {});
};
export const createActions = (actions) => actions.map(a => createAction(a));
export const actionsToMenu = ({ actions, setLocation, sx = {} }) => {
    return actions.map(({ label, value, link }) => (_jsx(Button, { variant: "text", size: "large", href: link, onClick: (e) => { e.preventDefault(); openLink(setLocation, link); }, sx: sx, children: i18next.t(label, { count: 2 }) }, value)));
};
export const actionsToBottomNav = ({ actions, setLocation }) => {
    return actions.map(({ label, value, link, Icon, numNotifications }) => (_jsx(BottomNavigationAction, { label: i18next.t(label, { count: 2 }), value: value, href: link, onClick: (e) => {
            e.preventDefault();
            const shouldScroll = link === window.location.pathname;
            if (shouldScroll)
                window.scrollTo({ top: 0, behavior: "smooth" });
            else
                openLink(setLocation, link);
        }, icon: _jsx(Badge, { badgeContent: numNotifications, color: "error", children: _jsx(Icon, {}) }), sx: { color: "white" } }, value)));
};
export const actionToIconButton = ({ action, setLocation, classes = { root: "" } }) => {
    const { value, link, Icon, numNotifications } = action;
    return _jsx(IconButton, { classes: classes, edge: "start", color: "inherit", "aria-label": value, onClick: () => openLink(setLocation, link), children: _jsx(Badge, { badgeContent: numNotifications, color: "error", children: _jsx(Icon, {}) }) });
};
//# sourceMappingURL=userActions.js.map