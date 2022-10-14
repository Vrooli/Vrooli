/* eslint-disable @typescript-eslint/no-redeclare */
import { APP_LINKS as LINKS } from '@shared/consts';
import {
    Badge,
    BottomNavigationAction,
    Button,
    IconButton,
} from '@mui/material';
import { openLink } from 'utils';
import { Session, SetLocation } from 'types';
import { CreateAccountIcon, DevelopIcon, HomeIcon, LearnIcon, ProfileIcon, ResearchIcon, SearchIcon, SvgComponent } from '@shared/icons';
import { getCurrentUser, guestSession } from 'utils/authentication';

export enum ACTION_TAGS {
    Home = 'Home',
    Search = 'Search',
    Learn = 'Learn',
    Research = 'Research',
    Develop = 'Develop',
    Profile = 'Profile',
    LogIn = 'LogIn',
}

export type ActionArray = [string, any, string, (() => any) | null, any, number];
export interface Action {
    label: string;
    value: ACTION_TAGS;
    link: string;
    onClick: (() => any) | null;
    Icon: SvgComponent;
    numNotifications: number;
}

// Returns navigational actions available to the user
interface GetUserActionsProps {
    session?: Session | null | undefined;
    exclude?: ACTION_TAGS[] | null | undefined;
}
export function getUserActions({ session = guestSession, exclude = [] }: GetUserActionsProps): Action[] {
    const { id: userId } = getCurrentUser(session);
    // Home action always available
    let actions: ActionArray[] = [
        ['Home', ACTION_TAGS.Home, LINKS.Home, null, HomeIcon, 0],
        ['Search', ACTION_TAGS.Search, LINKS.Search, null, SearchIcon, 0],
    ];
    // Available for all users
    actions.push(
        ['Learn', ACTION_TAGS.Learn, LINKS.Learn, null, LearnIcon, 0],
        ['Research', ACTION_TAGS.Research, LINKS.Research, null, ResearchIcon, 0],
        ['Develop', ACTION_TAGS.Develop, LINKS.Develop, null, DevelopIcon, 0],
    );
    // Log in/out
    if (userId) {
        actions.push(['Profile', ACTION_TAGS.Profile, LINKS.Profile, null, ProfileIcon, 0])
    } else {
        actions.push(['Log In', ACTION_TAGS.LogIn, LINKS.Start, null, CreateAccountIcon, 0]);
    }

    return actions.map(a => createAction(a)).filter(a => !(exclude ?? []).includes(a.value));
}

// Factory for creating action objects
const createAction = (action: ActionArray): Action => {
    const keys = ['label', 'value', 'link', 'onClick', 'Icon', 'numNotifications'];
    return action.reduce((obj: {}, val: any, i: number) => { obj[keys[i]] = val; return obj }, {}) as Action;
}

// Factory for creating a list of action objects
export const createActions = (actions: ActionArray[]): Action[] => actions.map(a => createAction(a));

// Display actions in a horizontal menu
interface ActionsToMenuProps {
    actions: Action[];
    setLocation: SetLocation;
    sx?: { [key: string]: any };
}
export const actionsToMenu = ({ actions, setLocation, sx = {} }: ActionsToMenuProps) => {
    return actions.map(({ label, value, link, onClick }) => (
        <Button
            key={value}
            variant="text"
            size="large"
            onClick={() => { openLink(setLocation, link); if (onClick) onClick() }}
            sx={sx}
        >
            {label}
        </Button>
    ));
}

// Display actions in a bottom navigation
interface ActionsToBottomNavProps {
    actions: Action[];
    setLocation: SetLocation;
}
export const actionsToBottomNav = ({ actions, setLocation }: ActionsToBottomNavProps) => {
    return actions.map(({ label, value, link, onClick, Icon, numNotifications }) => (
        <BottomNavigationAction
            key={value}
            label={label}
            value={value}
            onClick={() => { openLink(setLocation, link); if (onClick) onClick() }}
            icon={<Badge badgeContent={numNotifications} color="error"><Icon /></Badge>}
        />
    ));
}

// Display an action as an icon button
interface ActionToIconButtonProps {
    action: Action;
    setLocation: SetLocation;
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