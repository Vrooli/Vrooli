/* eslint-disable @typescript-eslint/no-redeclare */
import {
    Flag as MissionIcon,
    Home as HomeIcon,
    Info as AboutIcon,
    PlayCircle as DevelopIcon,
} from '@material-ui/icons';
import { APP_LINKS, LANDING_LINKS, WEBSITE } from '@local/shared';
import {
    Badge,
    BottomNavigationAction,
    Button,
    IconButton,
    ListItem,
    ListItemIcon,
    ListItemText,
} from '@material-ui/core';
import { ValueOf } from '@local/shared';
import { openLink } from 'utils';

export const ACTION_TAGS = {
    Home: 'home',
    Mission: 'mission',
    About: 'about',
    Start: 'start',
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
    exclude?: ACTION_TAGS[] | undefined;
}
export function getUserActions({ exclude = [] }: GetUserActionsProps): Action[] {
    let actions: ActionArray[] = [
        ['Home', ACTION_TAGS.Home, LANDING_LINKS.Home, null, HomeIcon, 0],
        ['Mission', ACTION_TAGS.Mission, LANDING_LINKS.Mission, null, MissionIcon, 0],
        ['About Us', ACTION_TAGS.About, LANDING_LINKS.About, null, AboutIcon, 0],
        ['Start', ACTION_TAGS.Start, `app.${WEBSITE}${APP_LINKS.Start}`, null, DevelopIcon, 0],
    ];

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
    classes?: { [key: string]: string };
    showIcon?: boolean;
    onAnyClick?: () => any;
}
export const actionsToList = ({ actions, history, classes = { listItem: '', listItemIcon: '' }, showIcon = true, onAnyClick = () => { } }: ActionsToListProps) => {
    return actions.map(({ label, value, link, onClick, Icon, numNotifications }) => (
        <ListItem
            key={value}
            classes={{ root: classes.listItem }}
            onClick={() => {
                openLink(history, link);
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
    classes?: { [key: string]: string };
}
export const actionsToMenu = ({ actions, history, classes = { root: '' } }: ActionsToMenuProps) => {
    return actions.map(({ label, value, link, onClick }) => (
        <Button
            key={value}
            variant="text"
            size="large"
            classes={classes}
            onClick={() => { openLink(history, link); if (onClick) onClick() }}
        >
            {label}
        </Button>
    ));
}

// Display actions in a bottom navigation
interface ActionsToBottomNavProps {
    actions: Action[];
    history: any;
    classes?: { [key: string]: string };
}
export const actionsToBottomNav = ({ actions, history, classes = { root: '' } }: ActionsToBottomNavProps) => {
    return actions.map(({ label, value, link, onClick, Icon, numNotifications }) => (
        <BottomNavigationAction
            key={value}
            classes={classes}
            label={label}
            value={value}
            onClick={() => { openLink(history, link); if (onClick) onClick() }}
            icon={<Badge badgeContent={numNotifications} color="error"><Icon /></Badge>} />
    ))
}

// Display an action as an icon button
interface ActionToIconButtonProps {
    action: Action;
    history: any;
    classes?: { [key: string]: string };
}
export const actionToIconButton = ({ action, history, classes = { root: '' } }: ActionToIconButtonProps) => {
    const { value, link, Icon, numNotifications } = action;
    return <IconButton classes={classes} edge="start" color="inherit" aria-label={value} onClick={() => openLink(history, link)}>
        <Badge badgeContent={numNotifications} color="error">
            <Icon />
        </Badge>
    </IconButton>
}