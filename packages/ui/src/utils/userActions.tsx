import {
    Apps as ProjectsIcon,
    ExitToApp as LogOutIcon,
    Flag as MissionIcon,
    Home as HomeIcon,
    Info as AboutIcon,
    PersonAdd as LogInIcon,
    PlayCircle as DevelopIcon,
    School as LearnIcon,
    Science as ResearchIcon,
} from '@material-ui/icons';
import { LINKS } from 'utils';
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
import { UserRoles } from 'types';

export type ActionArray = [string, any, string, (() => any) | null, any, number];
interface Action {
    label: string;
    value: any;
    link: string;
    onClick: (() => any) | null;
    Icon: any;
    numNotifications: number;
}

// Returns navigational actions available to the user
interface GetUserActionsProps {
    userRoles: UserRoles;
    exclude?: string[] | undefined;
}
export function getUserActions({ userRoles, exclude = [] }: GetUserActionsProps): Action[] {
    let actions: ActionArray[] = [];

    // If someone is not logged in, display sign up/log in links
    if (!userRoles) {
        actions.push(
            ['Home', 'home', LINKS.Landing, null, HomeIcon, 0],
            ['Mission', 'mission', LINKS.Mission, null, MissionIcon, 0],
            ['About Us', 'about', LINKS.About, null, AboutIcon, 0],
            ['Log In', 'waitlist', LINKS.LogIn, null, LogInIcon, 0],
            );
    } else {
        actions.push(
            ['Home', 'home', LINKS.Home, null, HomeIcon, 0],
            ['Projects', 'projects', LINKS.Projects, null, ProjectsIcon, 0],
            ['Learn', 'learn', LINKS.Learn, null, LearnIcon, 0],
            ['Research', 'research', LINKS.Research, null, ResearchIcon, 0],
            ['Develop', 'develop', LINKS.Develop, null, DevelopIcon, 0],
            ['Log out', 'logout', LINKS.Home, () => { const client = initializeApollo(); client.mutate({ mutation: logoutMutation }) }, LogOutIcon, 0]
            );
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
    history: any;
    classes?: {[key: string]: string};
    showIcon?: boolean;
    onAnyClick?: () => any;
}
export const actionsToList = ({ actions, history, classes = { listItem: '', listItemIcon: '' }, showIcon = true, onAnyClick = () => {} }: ActionsToListProps) => {
    return actions.map(({ label, value, link, onClick, Icon, numNotifications }) => (
        <ListItem
            key={value}
            classes={{ root: classes.listItem }}
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
                </ListItemIcon>) : ''}
            <ListItemText primary={label} />
        </ListItem>
    ))
}

// Display actions in a horizontal menu
interface ActionsToMenuProps {
    actions: Action[];
    history: any;
    classes?: {[key: string]: string};
}
export const actionsToMenu = ({ actions, history, classes = { root: '' } }: ActionsToMenuProps) => {
    return actions.map(({ label, value, link, onClick }) => (
        <Button
            key={value}
            variant="text"
            size="large"
            classes={classes}
            onClick={() => { history.push(link); if (onClick) onClick() }}
        >
            {label}
        </Button>
    ));
}

// Display actions in a bottom navigation
interface ActionsToBottomNavProps {
    actions: Action[];
    history: any;
    classes?: {[key: string]: string};
}
export const actionsToBottomNav = ({ actions, history, classes = { root: '' } }: ActionsToBottomNavProps) => {
    return actions.map(({ label, value, link, onClick, Icon, numNotifications }) => (
        <BottomNavigationAction
            key={value}
            classes={classes}
            label={label}
            value={value}
            onClick={() => { history.push(link); if (onClick) onClick() }}
            icon={<Badge badgeContent={numNotifications} color="error"><Icon /></Badge>} />
    ))
}

// Display an action as an icon button
interface ActionToIconButtonProps {
    action: Action;
    history: any;
    classes?: {[key: string]: string};
}
export const actionToIconButton = ({ action, history, classes = { root: '' } }: ActionToIconButtonProps) => {
    const { value, link, Icon, numNotifications } = action;
    return <IconButton  classes={classes} edge="start" color="inherit" aria-label={value} onClick={() => history.push(link)}>
        <Badge badgeContent={numNotifications} color="error">
            <Icon />
        </Badge>
    </IconButton>
}