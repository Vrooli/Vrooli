import {
    ExitToApp as ExitToAppIcon,
    Person as PersonIcon,
    PersonAdd as PersonAddIcon,
    Settings as SettingsIcon,
} from '@material-ui/icons';
import { ROLES } from '@local/shared';
import { LINKS } from 'utils';
import _ from 'lodash';
import { initializeApollo } from 'graphql/utils/initialize';
import { logoutMutation } from 'graphql/mutation';
import {
    Badge,
    BottomNavigationAction,
    Button,
    IconButton,
    ListItem,
    ListItemIcon,
    ListItemText,
} from '@material-ui/core';

// Returns navigational actions available to the user
export function getUserActions({ session, userRoles, exclude = [] }) {
    let actions = [];

    // If someone is not logged in, display sign up/log in links
    if (!_.isObject(session) || Object.entries(session).length === 0) {
        actions.push(['Join Waitlist', 'waitlist', LINKS.Waitlist, null, PersonAddIcon, 0]);
    } else {
        // If an owner admin is logged in, display owner links
        const haveArray = Array.isArray(userRoles) ? userRoles : [userRoles];
        if (userRoles && haveArray.some(r => [ROLES.Owner, ROLES.Admin].includes(r?.role?.title))) {
            actions.push(['Manage Site', 'admin', LINKS.Admin, null, SettingsIcon, 0]);
        }
        actions.push(['Profile', 'profile', LINKS.Profile, null, PersonIcon, 0],
            ['Log out', 'logout', LINKS.Home, () => { const client = initializeApollo(); client.mutate({ mutation: logoutMutation }) }, ExitToAppIcon, 0]);
    }

    return actions.map(a => createAction(...a)).filter(a => !exclude.includes(a.value));
}

// Factory for creating action objects
export const createAction = (label, value, link, onClick, Icon, numNotifications) => ({
    label,
    value,
    link,
    onClick,
    Icon,
    numNotifications,
})

// Display actions as a list
export const actionsToList = ({ actions, history, classes = { listItem: {}, listItemIcon: {} }, showIcon = true, onAnyClick }) => {
    return actions.map(({ label, value, link, onClick, Icon, numNotifications }) => (
        <ListItem
            key={value}
            className={classes.listItem}
            button
            onClick={() => {
                history.push(link);
                if (onClick) onClick();
                if (onAnyClick) onAnyClick();
            }}>
            {showIcon && Icon ?
                (<ListItemIcon>
                    <Badge badgeContent={numNotifications} color="error">
                        <Icon className={classes.listItemIcon} />
                    </Badge>
                </ListItemIcon>) : null}
            <ListItemText primary={label} />
        </ListItem>
    ))
}

// Display actions in a horizontal menu
export const actionsToMenu = ({ actions, history, classes = { button: {} } }) => {
    return actions.map(({ label, value, link, onClick }) => (
        <Button
            key={value}
            variant="text"
            size="large"
            className={classes.button}
            onClick={() => { history.push(link); if (onClick) onClick() }}
        >
            {label}
        </Button>
    ));
}

// Display actions in a bottom navigation
export const actionsToBottomNav = ({ actions, history, classes = { action: {} } }) => {
    return actions.map(({ label, value, link, onClick, Icon, numNotifications }) => (
        <BottomNavigationAction
            key={value}
            className={classes.action}
            label={label}
            value={value}
            onClick={() => { history.push(link); if (onClick) onClick() }}
            icon={<Badge badgeContent={numNotifications} color="error"><Icon /></Badge>} />
    ))
}

// Display an action as an icon button
export const actionToIconButton = ({ action, history, classes = { button: {} } }) => {
    const { value, link, Icon, numNotifications } = action;
    return <IconButton className={classes.button} edge="start" color="inherit" aria-label={value} onClick={() => history.push(link)}>
        <Badge badgeContent={numNotifications} color="error">
            <Icon />
        </Badge>
    </IconButton>
}