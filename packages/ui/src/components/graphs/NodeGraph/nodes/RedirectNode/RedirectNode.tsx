import { makeStyles } from '@mui/styles';
import { IconButton, Theme, Tooltip, Typography } from '@mui/material';
import { MouseEvent, useCallback, useMemo, useState } from 'react';
import { RedirectNodeProps } from '../types';
import { nodeStyles } from '../styles';
import { combineStyles } from 'utils';
import { UTurnLeft as RedirectIcon } from '@mui/icons-material';
import { NodeContextMenu } from '../..';

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
            transform: 'scale(1.2)',
            transition: 'scale .2s ease-in-out',
        }
    },
});

const useStyles = makeStyles(combineStyles(nodeStyles, componentStyles));

export const RedirectNode = ({
    node,
    scale = 1,
    label = 'Redirect',
    labelVisible = true,
}: RedirectNodeProps) => {
    const classes = useStyles();
    const [dialogOpen, setDialogOpen] = useState(false);
    const openDialog = () => setDialogOpen(true);
    const closeDialog = () => setDialogOpen(false);
    const dialog = useMemo(() => dialogOpen ? (
        <div>TODO</div>
    ) : null, [dialogOpen])

    const labelObject = useMemo(() => labelVisible ? (
        <Typography className={`${classes.label} ${classes.noSelect} ${classes.ignoreHover}`} variant="h6">{label}</Typography>
    ) : null, [labelVisible, classes.label, classes.noSelect, classes.ignoreHover, label]);

    const nodeSize = useMemo(() => `${100 * scale}px`, [scale]);
    const fontSize = useMemo(() => `min(${100 * scale / 5}px, 2em)`, [scale]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const contextId = useMemo(() => `node-context-menu-${node.id}`, [node]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((ev: MouseEvent<HTMLButtonElement>) => {
        setContextAnchor(ev.currentTarget)
        ev.preventDefault();
    }, []);
    const closeContext = useCallback(() => setContextAnchor(null), []);

    return (
        <div>
            {dialog}
            <NodeContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                node={node}
                onClose={closeContext}
                onAddBefore={() => { }}
                onAddAfter={() => { }}
                onDelete={() => { }}
                onEdit={() => { }}
                onMove={() => { }}
            />
            <Tooltip placement={'top'} title='Redirect'>
                <IconButton 
                    className={classes.root} 
                    style={{width: nodeSize, height: nodeSize, fontSize: fontSize}} 
                    onClick={openDialog}
                    aria-owns={contextOpen ? contextId : undefined}
                    onContextMenu={openContext}
                >
                    <RedirectIcon className={classes.icon} />
                    {labelObject}
                </IconButton>
            </Tooltip>
        </div>
    )
}