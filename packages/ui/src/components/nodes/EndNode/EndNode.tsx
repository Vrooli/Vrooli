import { makeStyles } from '@material-ui/styles';
import { Theme, Tooltip, Typography } from '@material-ui/core';
import { useMemo } from 'react';
import { EndNodeProps } from '../types';
import { nodeStyles } from '../styles';
import { combineStyles } from 'utils';

const componentStyles = (theme: Theme) => ({
    outerCirlce: {
        position: 'relative',
        display: 'block',
        width: '10vw',
        height: '10vw',
        backgroundColor: 'lightgray',
        color: 'white',
        borderRadius: '100%',
        boxShadow: '0px 0px 12px gray',
    },
    innerCirlce: {
        position: 'absolute',
        display: 'block',
        margin: '0',
        top: '50%',
        left: '50%',
        transform: 'translate(-50%, -50%)',
        width: '8vw',
        height: '8vw',
        borderRadius: '100%',
        border: '2px solid white',
    },
});

const useStyles = makeStyles(combineStyles(nodeStyles, componentStyles));

export const EndNode = ({
    label = 'End',
    labelVisible = true,
}: EndNodeProps) => {
    const classes = useStyles();

    const labelObject = useMemo(() => labelVisible ? (
        <Typography className={classes.label} variant="h6">{label}</Typography>
    ) : null, [label, labelVisible, classes.label]);

    return (
        <Tooltip placement={'top'} title={'Start'}>
            <div className={classes.outerCirlce}>
                <div className={classes.innerCirlce}>
                    {labelObject}
                </div>
            </div>
        </Tooltip>
    )
}