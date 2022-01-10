import { Box, Tooltip, Typography } from '@mui/material';
import { useMemo } from 'react';
import { StartNodeProps } from '../types';
import { containerShadow, noSelect } from 'styles';
import { combineStyles } from 'utils';

export const StartNode = ({
    node,
    scale = 1,
    label = 'Start',
    labelVisible = true,
}: StartNodeProps) => {

    const labelObject = useMemo(() => labelVisible ? (
        <Typography
            variant="h6"
            sx={{
                ...noSelect,
                ...nodeLabel,
            }}
        >
            {label}
        </Typography>
    ) : null, [labelVisible, label]);

    const nodeSize = useMemo(() => `${100 * scale}px`, [scale]);
    const fontSize = useMemo(() => `min(${100 * scale / 5}px, 2em)`, [scale]);

    return (
        <Tooltip placement={'top'} title={label ?? ''}>
            <Box
                sx={{
                    ...containerShadow,
                    width: nodeSize,
                    height: nodeSize,
                    fontSize: fontSize,
                    position: 'relative',
                    display: 'block',
                    backgroundColor: '#6daf72',
                    color: 'white',
                    borderRadius: '100%',
                    '&:hover': {
                        filter: `brightness(120%)`,
                        transition: 'filter 0.2s',
                    },
                }}
            >
                {labelObject}
            </Box>
        </Tooltip>
    )
}