import {
    Card,
    CardContent,
    Theme,
    Typography
} from '@mui/material';
import { makeStyles } from '@mui/styles';
import { BarGraph, Dimensions } from 'components';
import { useEffect, useRef, useState } from 'react';
import { combineStyles } from 'utils';
import { cardStyles } from '../styles';
import { StatCardProps } from '../types';

const componentStyles = (theme: Theme) => ({
    imageContainer: {
        display: 'contents',
        position: 0,
    },
    title: {
        textAlign: 'center',
        background: theme.palette.primary.light,
        color: theme.palette.primary.contrastText,
        marginBottom: '10px',
    },
    graph: {
        width: '90%',
        height: '85%',
        display: 'block',
        margin: 'auto',
    }
})

const useStyles = makeStyles(combineStyles(cardStyles, componentStyles));

export const StatCard = ({
    data,
    index,
}: StatCardProps) => {
    const classes = useStyles();
    const [dimensions, setDimensions] = useState<Dimensions>({ width: undefined, height: undefined });
    const ref = useRef<HTMLDivElement>(null);

    useEffect(() => {
        const width = ref.current?.clientWidth ? ref.current.clientWidth - 50 : 0;
        const height = ref.current?.clientHeight ? ref.current.clientHeight - 50 : 0;
        setDimensions({ width, height});
    }, [setDimensions])

    return (
        <Card className={classes.cardRoot} ref={ref}>
            <CardContent className={classes.imageContainer}>
                <Typography className={classes.title} variant="h6" component="h2">Daily active users</Typography>
                <BarGraph className={classes.graph} dimensions={dimensions} />
            </CardContent>
        </Card>
    );
}