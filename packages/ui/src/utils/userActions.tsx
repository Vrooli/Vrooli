/* eslint-disable @typescript-eslint/no-redeclare */
import {
    Apps as ProjectsIcon,
    ExitToApp as LogOutIcon,
    Home as HomeIcon,
    PersonAdd as RegisterIcon,
    PlayCircle as DevelopIcon,
    School as LearnIcon,
    Science as ResearchIcon,
} from '@mui/icons-material';
import { APP_LINKS as LINKS } from '@local/shared';
import { initializeApollo } from 'graphql/utils/initialize';
import { logOutMutation } from 'graphql/mutation';
import {
    Badge,
    BottomNavigationAction,
    Button,
    IconButton,
    ListItem,
    ListItemIcon,
    ListItemText,
} from '@mui/material';
import { UserRoles } from 'types';
import { ROLES, ValueOf } from '@local/shared';
import { openLink } from 'utils';
import { Path } from 'wouter';

export const ACTION_TAGS = {
    Home: 'home',
    Projects: 'projects',
    Learn: 'learn',
    Research: 'research',
    Develop: 'develop',
    LogIn: 'logIn',
    LogOut: 'logOut',
}
export type ACTION_TAGS = ValueOf<typeof ACTION_TAGS>;

export type ActionArray = [string, any, string, (() => any) | null, any, number];
export interface Action {
    label: string;
    value: ACTION_TAGS;
    link: string;
    onClick: (() => any) | null;
    Icon: any;
    numNotifications: number;
}

// Returns navigational actions available to the user
interface GetUserActionsProps {
    userRoles: UserRoles;
    exclude?: ACTION_TAGS[] | undefined;
}
export function getUserActions({ userRoles, exclude = [] }: GetUserActionsProps): Action[] {
    // Home action always available
    let actions: ActionArray[] = [['Home', ACTION_TAGS.Home, LINKS.Home, null, HomeIcon, 0]];
    // Map user roles to string array
    // Projets action is only for logged-in users (since it uses personal data)
    if (userRoles?.includes(ROLES.Actor)) {
        actions.push(['Projects', ACTION_TAGS.Projects, LINKS.Projects, null, ProjectsIcon, 0]);
    }
    // Available for all users
    actions.push(
        ['Learn', ACTION_TAGS.Learn, LINKS.Learn, null, LearnIcon, 0],
        ['Research', ACTION_TAGS.Research, LINKS.Research, null, ResearchIcon, 0],
        ['Develop', ACTION_TAGS.Develop, LINKS.Develop, null, DevelopIcon, 0],
    );
    // Log in/out
    if (!userRoles?.includes(ROLES.Actor)) {
        actions.push(['Log In', ACTION_TAGS.LogIn, LINKS.Start, null, RegisterIcon, 0]);
    } else {
        actions.push(['Log Out', ACTION_TAGS.LogOut, LINKS.Home, () => { const client = initializeApollo(); client.mutate({ mutation: logOutMutation }) }, LogOutIcon, 0]);
    }

    return actions.map(a => createAction(a)).filter(a => !exclude.includes(a.value));
}

// Factory for creating action objects
const createAction = (action: ActionArray): Action => {
    const keys = ['label', 'value', 'link', 'onClick', 'Icon', 'numNotifications'];
    return action.reduce((obj: {}, val: any, i: number) => { obj[keys[i]] = val; return obj }, {}) as Action;
}

// Factory for creating a list of action objects
export const createActions = (actions: ActionArray[]): Action[] => actions.map(a => createAction(a));

// Display actions as a list
interface ActionsToListProps {
    actions: Action[];
    setLocation: (to: Path, options?: { replace?: boolean }) => void;
    classes?: { [key: string]: string };
    showIcon?: boolean;
    onAnyClick?: () => any;
}
export const actionsToList = ({ actions, setLocation, classes = { listItem: '', listItemIcon: '' }, showIcon = true, onAnyClick = () => { } }: ActionsToListProps) => {
    return actions.map(({ label, value, link, onClick, Icon, numNotifications }) => (
        <ListItem
            key={value}
            classes={{ root: classes.listItem }}
            onClick={() => {
                openLink(setLocation, link);
                if (onClick) onClick();
                if (onAnyClick) onAnyClick();
            }}>
            {showIcon && Icon ?
                (<ListItemIcon>
                    <Badge badgeContent={numNotifications} color="error">
                        <Icon className={classes.listItemIcon} />
                    </Badge>
                </ListItemIcon>) : ''}
            <ListItemText primary={label} />
        </ListItem>
    ))
}

// Display actions in a horizontal menu
interface ActionsToMenuProps {
    actions: Action[];
    setLocation: (to: Path, options?: { replace?: boolean }) => void;
    classes?: { [key: string]: string };
}
export const actionsToMenu = ({ actions, setLocation, classes = { root: '' } }: ActionsToMenuProps) => {
    return actions.map(({ label, value, link, onClick }) => (
        <Button
            key={value}
            variant="text"
            size="large"
            classes={classes}
            onClick={() => { openLink(setLocation, link); if (onClick) onClick() }}
        >
            {label}
        </Button>
    ));
}

// Display actions in a bottom navigation
interface ActionsToBottomNavProps {
    actions: Action[];
    setLocation: (to: Path, options?: { replace?: boolean }) => void;
    classes?: { [key: string]: string };
}
export const actionsToBottomNav = ({ actions, setLocation, classes = { root: '' } }: ActionsToBottomNavProps) => {
    return actions.map(({ label, value, link, onClick, Icon, numNotifications }) => (
        <BottomNavigationAction
            key={value}
            classes={classes}
            label={label}
            value={value}
            onClick={() => { openLink(setLocation, link); if (onClick) onClick() }}
            icon={<Badge badgeContent={numNotifications} color="error"><Icon /></Badge>} />
    ))
}

// Display an action as an icon button
interface ActionToIconButtonProps {
    action: Action;
    setLocation: (to: Path, options?: { replace?: boolean }) => void;
    classes?: { [key: string]: string };
}
export const actionToIconButton = ({ action, setLocation, classes = { root: '' } }: ActionToIconButtonProps) => {
    const { value, link, Icon, numNotifications } = action;
    return <IconButton classes={classes} edge="start" color="inherit" aria-label={value} onClick={() => openLink(setLocation, link)}>
        <Badge badgeContent={numNotifications} color="error">
            <Icon />
        </Badge>
    </IconButton>
}