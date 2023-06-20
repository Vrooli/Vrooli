/**
 * Used to create/update a link between two routine nodes
 */
import { AddLinkIcon, CompressIcon, RedoIcon, UndoIcon } from "@local/shared";
import { Stack, Tooltip, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { UnlinkedNodesDialog } from "components/dialogs/UnlinkedNodesDialog/UnlinkedNodesDialog";
import { useCallback, useMemo, useState } from "react";
import { useWindowSize } from "utils/hooks/useWindowSize";
import { GraphActionsProps } from "../types";

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
            height: "48px",
            background: "transparent",
            color: palette.primary.contrastText,
            justifyContent: "center",
            alignItems: "center",
            paddingTop: "8px",
            "@media print": {
                display: "none",
            },
        }}>
            {showAll && <>
                <Tooltip title={canUndo ? "Undo" : ""}>
                    <ColorIconButton
                        id="undo-button"
                        disabled={!canUndo}
                        onClick={handleUndo}
                        aria-label="Undo"
                        background={palette.secondary.main}
                    >
                        <UndoIcon id="redo-button-icon" fill={palette.secondary.contrastText} />
                    </ColorIconButton>
                </Tooltip>
                <Tooltip title={canRedo ? "Redo" : ""}>
                    <ColorIconButton
                        id="redo-button"
                        disabled={!canRedo}
                        onClick={handleRedo}
                        aria-label="Redo"
                        background={palette.secondary.main}
                    >
                        <RedoIcon id="redo-button-icon" fill={palette.secondary.contrastText} />
                    </ColorIconButton>
                </Tooltip>
                <Tooltip title='Clean up graph'>
                    <ColorIconButton
                        id="clean-graph-button"
                        onClick={handleCleanUpGraph}
                        aria-label='Clean up graph'
                        background={palette.secondary.main}
                    >
                        <CompressIcon id="clean-up-button-icon" fill={palette.secondary.contrastText} />
                    </ColorIconButton>
                </Tooltip>
                <Tooltip title='Add new link'>
                    <ColorIconButton
                        id="add-link-button"
                        onClick={handleOpenLinkDialog}
                        aria-label='Add link'
                        background={palette.secondary.main}
                    >
                        <AddLinkIcon id="add-link-button-icon" fill={palette.secondary.contrastText} />
                    </ColorIconButton>
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
    );
};
