import { makeStyles } from '@material-ui/styles';
import { Theme, Tooltip, Typography } from '@material-ui/core';
import { useMemo } from 'react';
import { StartNodeProps } from '../types';
import { nodeStyles } from '../styles';
import { combineStyles } from 'utils';

const componentStyles = (theme: Theme) => ({
    root: {
        position: 'relative',
        display: 'block',
        width: '10vw',
        height: '10vw',
        backgroundColor: '#6daf72',
        color: 'white',
        borderRadius: '100%',
        boxShadow: '0px 0px 12px gray',
    },
});

const useStyles = makeStyles(combineStyles(nodeStyles, componentStyles));

export const StartNode = ({
    label = 'Start',
    labelVisible = true,
}: StartNodeProps) => {
    const classes = useStyles();

    const labelObject = useMemo(() => labelVisible ? (
        <Typography className={classes.label} variant="h6">{label}</Typography>
    ): null, [label, labelVisible, classes.label]);

    return (
        <Tooltip placement={'top'} title={label}>
            <div className={classes.root}>
                {labelObject}
            </div>
        </Tooltip>
    )
}