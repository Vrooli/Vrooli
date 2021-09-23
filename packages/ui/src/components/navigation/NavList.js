import React from 'react';
import PropTypes from 'prop-types';
import {
    ContactInfo,
    PopupMenu
} from 'components';
import { actionsToMenu, getUserActions } from 'utils';
import { Container } from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { useHistory } from 'react-router-dom';

const useStyles = makeStyles((theme) => ({
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

function NavList({
    session,
    business,
    roles,
}) {
    const classes = useStyles();
    const history = useHistory();

    const nav_actions = getUserActions({ session, userRoles: roles });

    return (
        <Container className={classes.root}>
            <PopupMenu 
                text="Contact"
                variant="text"
                size="large"
                className={classes.navItem}
            >
                <ContactInfo className={classes.contact} business={business} />
            </PopupMenu>
            {actionsToMenu({
                actions: nav_actions,
                history,
                classes: { button: classes.navItem },
            })}
        </Container>
    );
}

NavList.propTypes = {
    session: PropTypes.object,
    logout: PropTypes.func.isRequired,
    roles: PropTypes.array,
}

export { NavList };