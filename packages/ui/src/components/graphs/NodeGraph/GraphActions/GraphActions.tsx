/**
 * Used to create/update a link between two routine nodes
 */
import { IconButton, Stack, Tooltip, useTheme } from "@mui/material";
import { UnlinkedNodesDialog } from "components/dialogs/UnlinkedNodesDialog/UnlinkedNodesDialog";
import { useWindowSize } from "hooks/useWindowSize";
import { AddLinkIcon, CompressIcon, RedoIcon, UndoIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
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
}: GraphActionsProps) => {
    const { t } = useTranslation();
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
                <Tooltip title={canUndo ? t("Undo") : ""}>
                    <IconButton
                        id="undo-button"
                        disabled={!canUndo}
                        onClick={handleUndo}
                        aria-label={t("Undo")}
                        sx={{ background: palette.secondary.main }}
                    >
                        <UndoIcon id="redo-button-icon" fill={palette.secondary.contrastText} />
                    </IconButton>
                </Tooltip>
                <Tooltip title={canRedo ? t("Redo") : ""}>
                    <IconButton
                        id="redo-button"
                        disabled={!canRedo}
                        onClick={handleRedo}
                        aria-label={t("Redo")}
                        sx={{ background: palette.secondary.main }}
                    >
                        <RedoIcon id="redo-button-icon" fill={palette.secondary.contrastText} />
                    </IconButton>
                </Tooltip>
                <Tooltip title={t("CleanGraph")}>
                    <IconButton
                        id="clean-graph-button"
                        onClick={handleCleanUpGraph}
                        aria-label={t("CleanGraph")}
                        sx={{ background: palette.secondary.main }}
                    >
                        <CompressIcon id="clean-up-button-icon" fill={palette.secondary.contrastText} />
                    </IconButton>
                </Tooltip>
                <Tooltip title={t("AddNewLink")}>
                    <IconButton
                        id="add-link-button"
                        onClick={handleOpenLinkDialog}
                        aria-label={t("AddNewLink")}
                        sx={{ background: palette.secondary.main }}
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
            />}
        </Stack>
    );
};
