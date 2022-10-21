/**
 * Used to create/update a link between two routine nodes
 */
import {
    IconButton,
    Palette,
    Stack,
    Tooltip,
    useTheme
} from '@mui/material';
import { AddLinkIcon, CompressIcon, RedoIcon, UndoIcon } from '@shared/icons';
import { UnlinkedNodesDialog } from 'components/dialogs';
import { useCallback, useMemo, useState } from 'react';
import { useWindowSize } from 'utils';
import { GraphActionsProps } from '../types';

const commonButtonProps = (palette: Palette) => ({
    background: palette.secondary.main,
    marginRight: 1,
    transition: 'brightness 0.2s ease-in-out',
    '&:hover': {
        filter: `brightness(120%)`,
        background: palette.secondary.main,
    },
})

export const GraphActions = ({
    canRedo,
    canUndo,
    handleCleanUpGraph,
    handleNodeDelete,
    handleOpenLinkDialog,
    handleRedo,
    handleUndo,
    isEditing,
    language,
    nodesOffGraph,
    zIndex,
}: GraphActionsProps) => {
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width < breakpoints.values.sm);

    const [unlinkedNodesOpen, setIsUnlinkedNodesOpen] = useState(false);
    const handleUnlinkedToggle = useCallback(() => { setIsUnlinkedNodesOpen(!unlinkedNodesOpen); }, [unlinkedNodesOpen, setIsUnlinkedNodesOpen]);

    // If (not editing but there are unlinked nodes) OR (UnlinkedNodesDialog is open on mobile), only show UnlinkedNodesDialog
    const showAll = useMemo(() => (isEditing || nodesOffGraph.length > 0) && !(isMobile && unlinkedNodesOpen), [isEditing, isMobile, nodesOffGraph.length, unlinkedNodesOpen]);

    return (
        <Stack direction="row" spacing={1} sx={{
            zIndex: 2,
            height: '48px',
            background: 'transparent',
            color: palette.primary.contrastText,
            justifyContent: 'center',
            alignItems: 'center',
            paddingTop: '8px',
        }}>
            {showAll && <>
                <Tooltip title={canUndo ? 'Undo' : ''}>
                    <IconButton
                        id="undo-button"
                        disabled={!canUndo}
                        onClick={handleUndo}
                        aria-label="Undo"
                        sx={commonButtonProps(palette)}
                    >
                        <UndoIcon id="redo-button-icon" fill={palette.secondary.contrastText} />
                    </IconButton>
                </Tooltip>
                <Tooltip title={canRedo ? 'Redo' : ''}>
                    <IconButton
                        id="redo-button"
                        disabled={!canRedo}
                        onClick={handleRedo}
                        aria-label="Redo"
                        sx={commonButtonProps(palette)}
                    >
                        <RedoIcon id="redo-button-icon" fill={palette.secondary.contrastText} />
                    </IconButton>
                </Tooltip>
                <Tooltip title='Clean up graph'>
                    <IconButton
                        id="clean-graph-button"
                        onClick={handleCleanUpGraph}
                        aria-label='Clean up graph'
                        sx={commonButtonProps(palette)}
                    >
                        <CompressIcon id="clean-up-button-icon" fill={palette.secondary.contrastText} />
                    </IconButton>
                </Tooltip>
                <Tooltip title='Add new link'>
                    <IconButton
                        id="add-link-button"
                        onClick={handleOpenLinkDialog}
                        aria-label='Add link'
                        sx={commonButtonProps(palette)}
                    >
                        <AddLinkIcon id="add-link-button-icon" fill={palette.secondary.contrastText} />
                    </IconButton>
                </Tooltip>
            </>}
            {(isEditing || nodesOffGraph.length > 0) && <UnlinkedNodesDialog
                handleNodeDelete={handleNodeDelete}
                handleToggleOpen={handleUnlinkedToggle}
                language={language}
                nodes={nodesOffGraph}
                open={unlinkedNodesOpen}
                zIndex={zIndex + 3}
            />}
        </Stack>
    )
}