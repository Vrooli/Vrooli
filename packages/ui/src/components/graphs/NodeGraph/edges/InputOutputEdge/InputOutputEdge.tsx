import { useMemo } from 'react';
import { InputOutputEdgeProps } from '../types';
import {
    Add as AddIcon,
} from '@mui/icons-material';
import { IconButton, Stack, Tooltip, useTheme } from '@mui/material';
import { BaseEdge } from '../BaseEdge/BaseEdge';

/**
 * Displays a line between two nodes of a routine graph
 * If in editing mode, displays a clickable button to edit the link or inserting a node
 */
export const InputOutputEdge = ({
    handleAdd,
    index,
    isEditing,
    isInput,
    isFromOpen,
}: InputOutputEdgeProps) => {
    const { palette } = useTheme();

    /**
     * Place button along bezier to display "Add" button. 
     * Ideally should be calculated based on height of "From" item, since most of the 
     * edge will be behind the "From" item.
     */
    const popoverT = useMemo(() => isFromOpen ? 0.8 : 0.5, [isFromOpen]);

    /**
     * If isEditable, displays a clickable button for editing the edge or inserting a node
     */
    const popoverComponent = useMemo(() => {
        if (!isEditing) return <></>;
        return (
            <Stack direction="row" spacing={1}>
                {/* Insert new input/output item */}
                <Tooltip title='Insert'>
                    <IconButton
                        id="insert-on-edge-button"
                        size="small"
                        onClick={() => { handleAdd(index) }}
                        aria-label='Insert new here'
                        sx={{
                            background: palette.secondary.main,
                            transition: 'brightness 0.2s ease-in-out',
                            '&:hover': {
                                filter: `brightness(105%)`,
                                background: palette.secondary.main,
                            },
                        }}
                    >
                        <AddIcon id="insert-on-edge-button-icon" sx={{ fill: 'white' }} />
                    </IconButton>
                </Tooltip>
            </Stack>
        );
    }, [isEditing, palette.secondary.main, handleAdd, index]);

    return <BaseEdge
        containerId={`${isInput ? 'input' : 'output'}-container`}
        fromId={`${isInput ? 'input' : 'output'}-item-${index}`}
        isEditing={isEditing}
        popoverComponent={popoverComponent}
        popoverT={popoverT}
        thiccness={3}
        timeBetweenDraws={1000}
        toId={`${isInput ? 'input' : 'output'}-item-${index + 1}`}
    />
}