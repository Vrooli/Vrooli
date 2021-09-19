import React from 'react';
import PropTypes from 'prop-types';
import { useHistory } from 'react-router-dom';
import { makeStyles } from '@material-ui/styles';
import { BottomNavigation, BottomNavigationAction, Badge } from '@material-ui/core';
import { getUserActions } from 'utils';

const useStyles = makeStyles((theme) => ({
    root: {
        background: theme.palette.primary.dark,
        position: 'fixed',
        zIndex: 5,
        bottom: '0',
        width: '100%',
    },
    icon: {
        color: theme.palette.primary.contrastText,
    },
    [theme.breakpoints.up(960)]: {
        root: {
            display: 'none',
        }
    },
}));

function BottomNav({
    session,
    userRoles,
    cart,
    ...props
}) {
    let history = useHistory();
    const classes = useStyles();

    let actions = getUserActions(session, userRoles, cart);

    return (
        <BottomNavigation 
            className={classes.root} 
            showLabels
            {...props}
        >
            {actions.map(([label, value, link, onClick, Icon, badgeNum], index) => (
                        <BottomNavigationAction 
                            key={index} 
                            className={classes.icon} 
                            label={label} 
                            value={value} 
                            onClick={() => {history.push(link); if(onClick) onClick()}}
                            icon={<Badge badgeContent={badgeNum} color="error"><Icon /></Badge>} />
                    ))}
        </BottomNavigation>
    );
}

BottomNav.propTypes = {
    session: PropTypes.object,
    userRoles: PropTypes.array,
    cart: PropTypes.object,
}

export { BottomNav };