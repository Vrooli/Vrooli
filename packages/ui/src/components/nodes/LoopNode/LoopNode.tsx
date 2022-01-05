import { makeStyles } from '@mui/styles';
import { IconButton, Theme, Tooltip, Typography } from '@mui/material';
import { useMemo, useState } from 'react';
import { LoopNodeProps } from '../types';
import { nodeStyles } from '../styles';
import { combineStyles } from 'utils';
import { Loop as LoopIcon } from '@mui/icons-material';

const componentStyles = (theme: Theme) => ({
    root: {
        position: 'relative',
        display: 'block',
        backgroundColor: '#6daf72',
        color: 'white',
        boxShadow: '0px 0px 12px gray',
        '&:hover': {
            backgroundColor: '#6daf72',
            filter: `brightness(120%)`,
            transition: 'filter 0.2s',
        },
    },
    icon: {
        width: '100%',
        height: '100%',
        color: '#00000044',
        '&:hover': {
            transform: 'rotate(-180deg)',
            transition: 'transform .2s ease-in-out',
        }
    },
});

const useStyles = makeStyles(combineStyles(nodeStyles, componentStyles));

export const LoopNode = ({
    scale = 1,
    label = 'Loop',
    labelVisible = true,
}: LoopNodeProps) => {
    const classes = useStyles();
    const [dialogOpen, setDialogOpen] = useState(false);
    const openDialog = () => setDialogOpen(true);
    const closeDialog = () => setDialogOpen(false);
    const dialog = useMemo(() => dialogOpen ? (
        <div>TODO</div>
    ) : null, [dialogOpen])

    const labelObject = useMemo(() => labelVisible ? (
        <Typography className={`${classes.label} ${classes.ignoreHover}`} variant="h6">{label}</Typography>
    ) : null, [labelVisible, classes.label, classes.ignoreHover, label]);

    const nodeSize = useMemo(() => `${100 * scale}px`, [scale]);
    const fontSize = useMemo(() => `min(${100 * scale / 5}px, 2em)`, [scale]);

    return (
        <div>
            {dialog}
            <Tooltip placement={'top'} title={label ?? ''}>
                <IconButton className={classes.root} style={{width: nodeSize, height: nodeSize, fontSize: fontSize}} onClick={openDialog}>
                    <LoopIcon className={classes.icon} />
                    {labelObject}
                </IconButton>
            </Tooltip>
        </div>
    )
}