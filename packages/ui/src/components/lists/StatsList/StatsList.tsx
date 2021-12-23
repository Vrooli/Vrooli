import { makeStyles } from '@mui/styles';
import { Theme } from '@mui/material';
import { StatCard } from 'components';
import { useMemo } from 'react';
import { StatsListProps } from '../types';

const useStyles = makeStyles((theme: Theme) => ({
    root: {
        border: `2px dashed ${theme.palette.text.primary}`,
        borderRadius: '10px',
        padding: '0',
    },
    flexed: {
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(400px, 1fr))',
        gridAutoRows: '400px',
        alignItems: 'stretch',
    },
}));

export const StatsList = ({
    data,
}: StatsListProps) => {
    const classes = useStyles();

    const statCards = useMemo(() => (
        data?.map((item, index) => (
            <StatCard
                key={index}
                index={index}
                data={item}
            />
        ))
    ), [data]);

    return (
        <div className={classes.flexed}>
            {statCards}
        </div>
    )
}