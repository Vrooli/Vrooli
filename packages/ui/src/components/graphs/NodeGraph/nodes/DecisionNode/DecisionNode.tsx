import { makeStyles } from '@mui/styles';
import { Theme, Tooltip, Typography } from '@mui/material';
import { useMemo } from 'react';
import { DecisionNodeProps } from '../types';
import { nodeStyles } from '../styles';
import { combineStyles } from 'utils';

const componentStyles = (theme: Theme) => ({
    root: {
        position: 'relative',
        textAlign: 'center',
        margin: '10px 40px',
        color: 'white',
        '&:before': {
            position: 'absolute',
            content: '""',
            top: '0',
            left: '0',
            width: '100%',
            height: '100%',
            backgroundColor: '#68b4c5',
            transform: 'rotateX(45deg) rotateZ(45deg)',
            boxShadow: '0px 0px 12px gray',
        },
        '&:after': {
            position: 'absolute',
            content: '""',
            top: '10px',
            left: '10px',
            height: 'calc(100% - 22px)',
            width: 'calc(100% - 22px)',
            border: '1px solid organge',
            transform: 'rotateX(45deg) rotateZ(45deg)',
        },
        '&:hover': {
            filter: `brightness(120%)`,
            transition: 'filter 0.2s',
        },
    },
});

const useStyles = makeStyles(combineStyles(nodeStyles, componentStyles));

export const DecisionNode = ({
    scale = 1,
    label = 'Continue?',
    text = 'Would you like to continue?',
    labelVisible = true,
}: DecisionNodeProps) => {
    const classes = useStyles();
    console.log('CLASSESSSSS', classes)

    const labelObject = useMemo(() => labelVisible ? (
        <Typography className={`${classes.label} ${classes.noSelect}`} variant="h6">{label}</Typography>
    ): null, [labelVisible, classes.label, classes.noSelect, label]);

    const nodeSize = useMemo(() => `${100 * scale}px`, [scale]);
    const fontSize = useMemo(() => `min(${100 * scale / 5}px, 2em)`, [scale]);

    return (
        <Tooltip placement={'top'} title={text}>
            <div className={classes.root} style={{width: nodeSize, height: nodeSize, lineHeight: nodeSize, fontSize: fontSize}}>
                {labelObject}
            </div>
        </Tooltip>
    )
}