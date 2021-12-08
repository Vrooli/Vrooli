import { makeStyles } from '@material-ui/styles';
import { List, ListItem, ListItemIcon, Theme } from '@material-ui/core';
import { useCallback, useMemo, useState } from 'react';
import {
    ChevronLeft as ChevronLeftIcon,
    ChevronRight as ChevronRightIcon,
} from '@material-ui/icons';

const useStyles = makeStyles((theme: Theme) => ({
    root: {},
    sideNav: {
        position: 'absolute',
        width: '20%',
        height: '100%',
        float: 'left',
        background: theme.palette.primary.light,
        borderRight: `1px solid ${theme.palette.text.primary}`,
    },
    navList: {
        paddingTop: '10vh',
    },
    navListItem: {

    },
    toggleButton: {
    },
    content: {
        width: '80%',
        height: '100%',
        float: 'right',
        padding: theme.spacing(1),
        paddingBottom: '20vh',
    },
}));

export const StatsPage = () => {
    const classes = useStyles();
    const [sideOpen, setSideOpen] = useState(true);

    const openSideNav = useCallback(() => setSideOpen(true), []);
    const closeSideNav = useCallback(() => setSideOpen(false), []);
    const sideNavButton = useMemo(() => sideOpen ? 
        (<ChevronLeftIcon  className={classes.toggleButton} onClick={closeSideNav} />) : 
        (<ChevronRightIcon className={classes.toggleButton} onClick={openSideNav} />)
    , [classes.toggleButton, closeSideNav, openSideNav, sideOpen]);

    return (
        <div className={classes.root}>
            <div className={classes.sideNav}>
                <List className={classes.navList}>
                    <ListItem
                        className={classes.navListItem}
                        secondaryAction={sideNavButton}
                    >
                        {sideOpen ? 'Close' : 'Open'}
                    </ListItem>
                    <ListItem>
                        <ListItemIcon>
                            {sideNavButton}
                        </ListItemIcon>
                    </ListItem>
                </List>
            </div>
            <div className={classes.content}>

            </div>
        </div>
    )
};
