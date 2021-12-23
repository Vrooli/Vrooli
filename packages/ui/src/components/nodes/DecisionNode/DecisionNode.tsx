import { makeStyles } from '@material-ui/styles';
import { Theme, Tooltip, Typography } from '@material-ui/core';
import { useMemo } from 'react';
import { DecisionNodeProps } from '../types';
import { nodeStyles } from '../styles';
import { combineStyles } from 'utils';

const componentStyles = (theme: Theme) => ({
    root: {
        position: 'relative',
        height: '10vw',
        width: '10vw',
        lineHeight: '10vw',
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
    },
});

const useStyles = makeStyles(combineStyles(nodeStyles, componentStyles));

export const DecisionNode = ({
    label = 'Continue?',
    text = 'Would you like to continue?',
    labelVisible = true,
}: DecisionNodeProps) => {
    const classes = useStyles();
    console.log('CLASSESSSSS', classes)

    const labelObject = useMemo(() => labelVisible ? (
        <Typography className={classes.label} variant="h6">{label}</Typography>
    ): null, [label, labelVisible, classes.label]);

    return (
        <Tooltip placement={'top'} title={text}>
            <div className={classes.root}>
                {labelObject}
            </div>
        </Tooltip>
    )
}