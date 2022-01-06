import { makeStyles } from '@mui/styles';
import { Theme, Tooltip, Typography } from '@mui/material';
import { useMemo } from 'react';
import { StartNodeProps } from '../types';
import { nodeStyles } from '../styles';
import { combineStyles } from 'utils';

const componentStyles = (theme: Theme) => ({
    root: {
        position: 'relative',
        display: 'block',
        backgroundColor: '#6daf72',
        color: 'white',
        borderRadius: '100%',
        boxShadow: '0px 0px 12px gray',
        '&:hover': {
            filter: `brightness(120%)`,
            transition: 'filter 0.2s',
        },
    },
});

const useStyles = makeStyles(combineStyles(nodeStyles, componentStyles));

export const StartNode = ({
    scale = 1,
    label = 'Start',
    labelVisible = true,
}: StartNodeProps) => {
    const classes = useStyles();

    const labelObject = useMemo(() => labelVisible ? (
        <Typography className={`${classes.label} ${classes.noSelect}`} variant="h6">{label}</Typography>
    ): null, [label, labelVisible, classes.label]);

    const nodeSize = useMemo(() => `${100 * scale}px`, [scale]);
    const fontSize = useMemo(() => `min(${100 * scale / 5}px, 2em)`, [scale]);

    return (
        <Tooltip placement={'top'} title={label ?? ''}>
            <div className={classes.root} style={{width: nodeSize, height: nodeSize, fontSize: fontSize}}>
                {labelObject}
            </div>
        </Tooltip>
    )
}