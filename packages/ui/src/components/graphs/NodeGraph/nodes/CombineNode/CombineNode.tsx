import { makeStyles } from '@mui/styles';
import { Theme, Tooltip, Typography } from '@mui/material';
import { useMemo, useState } from 'react';
import { CombineNodeProps } from '../types';
import { nodeStyles } from '../styles';
import { combineStyles } from 'utils';
import { ArrowRightIcon } from 'assets/img';

const componentStyles = (theme: Theme) => ({
    root: {
        position: 'relative',
        display: 'block',
        color: 'white',
        '&:hover': {
            filter: `brightness(120%)`,
            transition: 'filter 0.2s',
        },
    },
    triangle: {
        width: '100%',
        height: '100%',
        fill: '#6daf72',
        filter: 'drop-shadow(0px 0px 12px gray)'
    }
});

const useStyles = makeStyles(combineStyles(nodeStyles, componentStyles));

export const CombineNode = ({
    scale = 1,
    label = 'Combine',
    labelVisible = true,
}: CombineNodeProps) => {
    const classes = useStyles();
    const [dialogOpen, setDialogOpen] = useState(false);
    const openDialog = () => setDialogOpen(true);
    const closeDialog = () => setDialogOpen(false);
    const dialog = useMemo(() => dialogOpen ? (
        <div>TODO</div>
    ) : null, [dialogOpen])

    const labelObject = useMemo(() => labelVisible ? (
        <Typography className={`${classes.label} ${classes.noSelect} ${classes.ignoreHover}`} style={{ marginLeft: '-20px' }} variant="h6">{label}</Typography>
    ) : null, [labelVisible, classes.label, classes.noSelect, classes.ignoreHover, label]);

    const nodeSize = useMemo(() => `${100 * scale}px`, [scale]);
    const fontSize = useMemo(() => `min(${100 * scale / 5}px, 2em)`, [scale]);

    return (
        <div>
            {dialog}
            <Tooltip placement={'top'} title='Combine'>
                <div className={classes.root} style={{ width: nodeSize, height: nodeSize, fontSize: fontSize }}>
                    <ArrowRightIcon
                        className={classes.triangle}
                        onClick={openDialog}
                    />
                    {labelObject}
                </div>
            </Tooltip>
        </div>
    )
}