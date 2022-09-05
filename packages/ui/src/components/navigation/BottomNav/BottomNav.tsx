import { useLocation } from '@shared/route';
import { makeStyles } from '@mui/styles';
import { BottomNavigation, Theme } from '@mui/material';
import { actionsToBottomNav, ACTION_TAGS, getUserActions } from 'utils';
import { BottomNavProps } from '../types';

const useStyles = makeStyles((theme: Theme) => ({
    root: {
        background: theme.palette.primary.dark,
        position: 'fixed',
        zIndex: 5,
        bottom: 0,
        paddingBottom: 'env(safe-area-inset-bottom)',
        // safe-area-inset-bottom is the iOS navigation bar
        height: 'calc(56px + env(safe-area-inset-bottom))',
        width: '100%',
    },
    icon: {
        color: theme.palette.primary.contrastText,
        '&:hover': {
            color: theme.palette.secondary.light,
        },
    },
    [theme.breakpoints.up('md')]: {
        root: {
            display: 'none',
        }
    },
}));

export const BottomNav = ({
    session,
    ...props
}: BottomNavProps) => {
    const [, setLocation] = useLocation();
    const classes = useStyles();

    let actions = actionsToBottomNav({
        actions: getUserActions({ session, exclude: [ACTION_TAGS.Profile, ACTION_TAGS.LogIn] }),
        setLocation,
        classes: { root: classes.icon }
    });

    return (
        <BottomNavigation
            className={classes.root}
            showLabels
            {...props}
        >
            {actions}
        </BottomNavigation>
    );
}