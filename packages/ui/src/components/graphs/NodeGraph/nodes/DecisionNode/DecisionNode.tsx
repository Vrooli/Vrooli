import { makeStyles } from '@mui/styles';
import { Theme, Tooltip, Typography } from '@mui/material';
import { MouseEvent, useCallback, useMemo, useState } from 'react';
import { DecisionNodeProps } from '../types';
import { nodeStyles } from '../styles';
import { combineStyles } from 'utils';
import { NodeContextMenu } from '../..';

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
    node,
    scale = 1,
    label = 'Continue?',
    text = 'Would you like to continue?',
    labelVisible = true,
}: DecisionNodeProps) => {
    const classes = useStyles();
    console.log('CLASSESSSSS', classes)

    const labelObject = useMemo(() => labelVisible ? (
        <Typography className={`${classes.label} ${classes.noSelect}`} variant="h6">{label}</Typography>
    ) : null, [labelVisible, classes.label, classes.noSelect, label]);

    const nodeSize = useMemo(() => `${100 * scale}px`, [scale]);
    const fontSize = useMemo(() => `min(${100 * scale / 5}px, 2em)`, [scale]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const contextId = useMemo(() => `node-context-menu-${node.id}`, [node]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((ev: MouseEvent<HTMLDivElement>) => {
        setContextAnchor(ev.currentTarget)
        ev.preventDefault();
    }, []);
    const closeContext = useCallback(() => setContextAnchor(null), []);

    return (
        <div>
            <NodeContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                open={contextOpen}
                node={node}
                onClose={closeContext}
                onAddBefore={() => { }}
                onAddAfter={() => { }}
                onDelete={() => { }}
                onEdit={() => { }}
                onMove={() => { }}
            />
            <Tooltip placement={'top'} title={text}>
                <div 
                    className={classes.root} 
                    style={{ width: nodeSize, height: nodeSize, lineHeight: nodeSize, fontSize: fontSize }}
                    aria-owns={contextOpen ? contextId : undefined}
                    onContextMenu={openContext}
                    onClick={() => {}}
                >
                    {labelObject}
                </div>
            </Tooltip>
        </div>
    )
}