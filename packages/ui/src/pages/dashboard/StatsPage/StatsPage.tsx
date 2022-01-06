import { makeStyles } from '@mui/styles';
import { List, ListItem, ListItemIcon, Theme } from '@mui/material';
import { useCallback, useMemo, useState } from 'react';
import {
    ChevronLeft as ChevronLeftIcon,
    ChevronRight as ChevronRightIcon,
} from '@mui/icons-material';
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
