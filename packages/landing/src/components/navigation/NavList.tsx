import {
    ContactInfo,
    PopupMenu
} from 'components';
import { Action, actionsToMenu, ACTION_TAGS, getUserActions, openLink } from 'utils';
import { Button, Container, Theme } from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { useHistory } from 'react-router-dom';
import { useMemo } from 'react';

const useStyles = makeStyles((theme: Theme) => ({
    root: {
        display: 'flex',
        marginTop: '0px',
        marginBottom: '0px',
        right: '0px',
        padding: '0px',
    },
    navItem: {
        background: 'transparent',
        color: theme.palette.primary.contrastText,
        textTransform: 'none',
        fontSize: '1.5em',
        '&:hover': {
            color: theme.palette.secondary.light,
        },
    },
    button: {
        fontSize: '1.5em',
        marginLeft: '20px',
        borderRadius: '10px',
    },
    menuItem: {
        color: theme.palette.primary.contrastText,
    },
    menuIcon: {
        fill: theme.palette.primary.contrastText,
    },
    contact: {
        width: 'calc(min(100vw, 400px))',
        height: '300px',
    },
}));

export const NavList = () => {
    const classes = useStyles();
    const history = useHistory();

    const nav_actions = useMemo<Action[]>(() => getUserActions({ exclude: [ACTION_TAGS.Home] }), []);
    // Display button for entering main application
    const enter_action: Action | undefined = nav_actions.find((action: Action) => action.value === ACTION_TAGS.Start)
    const enter_button = useMemo(() => enter_action ? (
        <Button 
            className={classes.button} 
            onClick={() => openLink(history, enter_action.link)}
            >
                {enter_action.label}
        </Button>
    ) : null, [enter_action, classes.button, history])

    return (
        <Container className={classes.root}>
            <PopupMenu
                text="Contact"
                variant="text"
                size="large"
                className={classes.navItem}
            >
                <ContactInfo className={classes.contact} />
            </PopupMenu>
            {actionsToMenu({
                actions: nav_actions.filter((a: Action) => a.value !== ACTION_TAGS.Start),
                history,
                classes: { root: classes.navItem },
            })}
            {enter_button}
        </Container>
    );
}