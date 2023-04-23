import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { AddLinkIcon, CompressIcon, RedoIcon, UndoIcon } from "@local/icons";
import { Stack, Tooltip, useTheme } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { useWindowSize } from "../../../../utils/hooks/useWindowSize";
import { ColorIconButton } from "../../../buttons/ColorIconButton/ColorIconButton";
import { UnlinkedNodesDialog } from "../../../dialogs/UnlinkedNodesDialog/UnlinkedNodesDialog";
export const GraphActions = ({ canRedo, canUndo, handleCleanUpGraph, handleNodeDelete, handleOpenLinkDialog, handleRedo, handleUndo, isEditing, language, nodesOffGraph, zIndex, }) => {
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width < breakpoints.values.sm);
    const [unlinkedNodesOpen, setIsUnlinkedNodesOpen] = useState(false);
    const handleUnlinkedToggle = useCallback(() => { setIsUnlinkedNodesOpen(!unlinkedNodesOpen); }, [unlinkedNodesOpen, setIsUnlinkedNodesOpen]);
    const showAll = useMemo(() => (isEditing || nodesOffGraph.length > 0) && !(isMobile && unlinkedNodesOpen), [isEditing, isMobile, nodesOffGraph.length, unlinkedNodesOpen]);
    return (_jsxs(Stack, { direction: "row", spacing: 1, sx: {
            zIndex: 2,
            height: "48px",
            background: "transparent",
            color: palette.primary.contrastText,
            justifyContent: "center",
            alignItems: "center",
            paddingTop: "8px",
        }, children: [showAll && _jsxs(_Fragment, { children: [_jsx(Tooltip, { title: canUndo ? "Undo" : "", children: _jsx(ColorIconButton, { id: "undo-button", disabled: !canUndo, onClick: handleUndo, "aria-label": "Undo", background: palette.secondary.main, children: _jsx(UndoIcon, { id: "redo-button-icon", fill: palette.secondary.contrastText }) }) }), _jsx(Tooltip, { title: canRedo ? "Redo" : "", children: _jsx(ColorIconButton, { id: "redo-button", disabled: !canRedo, onClick: handleRedo, "aria-label": "Redo", background: palette.secondary.main, children: _jsx(RedoIcon, { id: "redo-button-icon", fill: palette.secondary.contrastText }) }) }), _jsx(Tooltip, { title: 'Clean up graph', children: _jsx(ColorIconButton, { id: "clean-graph-button", onClick: handleCleanUpGraph, "aria-label": 'Clean up graph', background: palette.secondary.main, children: _jsx(CompressIcon, { id: "clean-up-button-icon", fill: palette.secondary.contrastText }) }) }), _jsx(Tooltip, { title: 'Add new link', children: _jsx(ColorIconButton, { id: "add-link-button", onClick: handleOpenLinkDialog, "aria-label": 'Add link', background: palette.secondary.main, children: _jsx(AddLinkIcon, { id: "add-link-button-icon", fill: palette.secondary.contrastText }) }) })] }), (isEditing || nodesOffGraph.length > 0) && _jsx(UnlinkedNodesDialog, { handleNodeDelete: handleNodeDelete, handleToggleOpen: handleUnlinkedToggle, language: language, nodes: nodesOffGraph, open: unlinkedNodesOpen, zIndex: zIndex + 3 })] }));
};
//# sourceMappingURL=GraphActions.js.map