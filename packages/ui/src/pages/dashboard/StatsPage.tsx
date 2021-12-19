import { makeStyles } from '@material-ui/styles';
import { List, ListItem, ListItemIcon, Theme } from '@material-ui/core';
import { useCallback, useMemo, useState } from 'react';
import {
    ChevronLeft as ChevronLeftIcon,
    ChevronRight as ChevronRightIcon,
} from '@material-ui/icons';
import { StatsList } from 'components';

const useStyles = makeStyles((theme: Theme) => ({
    root: {},
}));

export const StatsPage = () => {
    const classes = useStyles();

    return (
        <div id='page' className={classes.root}>
            <StatsList data={[{}, {}, {}, {}, {}, {}]} />
        </div>
    )
};
