import { makeStyles } from '@mui/styles';
import { Theme, Tooltip, Typography } from '@mui/material';
import { useMemo } from 'react';
import { EndNodeProps } from '../types';
import { nodeStyles } from '../styles';
import { combineStyles } from 'utils';

const componentStyles = (theme: Theme) => ({
    outerCirlce: {
        position: 'relative',
        display: 'block',
        backgroundColor: '#979696',
        color: 'white',
        borderRadius: '100%',
        boxShadow: '0px 0px 12px gray',
        '&:hover': {
            filter: `brightness(120%)`,
            transition: 'filter 0.2s',
        },
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
    scale = 1,
    label = 'End',
    labelVisible = true,
}: EndNodeProps) => {
    const classes = useStyles();

    const labelObject = useMemo(() => labelVisible ? (
        <Typography className={classes.label} variant="h6">{label}</Typography>
    ) : null, [label, labelVisible, classes.label]);

    const outerCircleSize = useMemo(() => `${100 * scale}px`, [scale]);
    const innerCircleSize = useMemo(() => `${100 * scale / 1.5}px`, [scale]);
    const fontSize = useMemo(() => `min(${100 * scale / 5}px, 2em)`, [scale]);

    return (
        <Tooltip placement={'top'} title={'End'}>
            <div className={classes.outerCirlce} style={{width: outerCircleSize, height: outerCircleSize, fontSize: fontSize}}>
                <div className={classes.innerCirlce} style={{width: innerCircleSize, height: innerCircleSize}}>
                    {labelObject}
                </div>
            </div>
        </Tooltip>
    )
}