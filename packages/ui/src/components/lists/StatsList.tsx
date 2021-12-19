import { makeStyles } from '@material-ui/styles';
import { Theme } from '@material-ui/core';
import { StatCard } from 'components';
import { useMemo } from 'react';

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

interface Props {
    data: Array<any>;
}

export const StatsList = ({
    data,
}: Props) => {
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