import { makeStyles } from '@mui/styles';
import { Theme, Tooltip, Typography } from '@mui/material';
import { MouseEvent, useCallback, useMemo, useState } from 'react';
import { EndNodeProps } from '../types';
import { nodeStyles } from '../styles';
import { combineStyles } from 'utils';
import { NodeContextMenu } from '../..';

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
    node,
    scale = 1,
    label = 'End',
    labelVisible = true,
}: EndNodeProps) => {
    const classes = useStyles();

    const labelObject = useMemo(() => labelVisible ? (
        <Typography className={`${classes.label} ${classes.noSelect}`} variant="h6">{label}</Typography>
    ) : null, [labelVisible, classes.label, classes.noSelect, label]);

    const outerCircleSize = useMemo(() => `${100 * scale}px`, [scale]);
    const innerCircleSize = useMemo(() => `${100 * scale / 1.5}px`, [scale]);
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
                node={node}
                onClose={closeContext}
                onAddBefore={() => { }}
                onAddAfter={() => { }}
                onDelete={() => { }}
                onEdit={() => { }}
                onMove={() => { }}
            />
            <Tooltip placement={'top'} title={'End'}>
                <div
                    className={classes.outerCirlce}
                    style={{ width: outerCircleSize, height: outerCircleSize, fontSize: fontSize }}
                    aria-owns={contextOpen ? contextId : undefined}
                    onContextMenu={openContext}
                    onClick={() => { }}
                >
                    <div className={classes.innerCirlce} style={{ width: innerCircleSize, height: innerCircleSize }}>
                        {labelObject}
                    </div>
                </div>
            </Tooltip>
        </div>
    )
}