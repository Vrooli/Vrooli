import {
    Card,
    CardContent,
    Theme,
    Typography
} from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { BarGraph, Dimensions } from 'components';
import { useEffect, useRef, useState } from 'react';
import { combineStyles } from 'utils';
import { cardStyles } from './styles';

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

interface Props {
    data: any;
    index: number;
}

export const StatCard = ({
    data,
    index,
}: Props) => {
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