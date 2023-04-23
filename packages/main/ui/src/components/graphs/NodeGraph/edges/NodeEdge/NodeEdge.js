import { Fragment as _Fragment, jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { AddIcon, BranchIcon, DeleteIcon, EditIcon } from "@local/icons";
import { Stack, Tooltip, useTheme } from "@mui/material";
import { useCallback, useMemo } from "react";
import { ColorIconButton } from "../../../../buttons/ColorIconButton/ColorIconButton";
import { calculateNodeSize } from "../../nodes";
import { BaseEdge } from "../BaseEdge/BaseEdge";
export const NodeEdge = ({ fastUpdate, handleAdd, handleBranch, handleDelete, handleEdit, isEditing, isFromRoutineList, isToRoutineList, link, scale, }) => {
    const { palette } = useTheme();
    const thiccness = useMemo(() => Math.ceil(calculateNodeSize(30, scale)), [scale]);
    const handleEditClick = useCallback(() => {
        handleEdit(link);
    }, [handleEdit, link]);
    const popoverT = useMemo(() => {
        let t = 0.5;
        if (isFromRoutineList && !isToRoutineList)
            t = 0.77;
        else if (!isFromRoutineList && isToRoutineList)
            t = 0.23;
        return t;
    }, [isFromRoutineList, isToRoutineList]);
    const popoverComponent = useMemo(() => {
        if (!isEditing)
            return _jsx(_Fragment, {});
        return (_jsxs(Stack, { direction: "row", spacing: 1, children: [_jsx(Tooltip, { title: 'Insert node', children: _jsx(ColorIconButton, { id: "insert-node-on-edge-button", background: palette.secondary.main, size: "small", onClick: () => { handleAdd(link); }, "aria-label": 'Insert node on edge', children: _jsx(AddIcon, { id: "insert-node-on-edge-button-icon", fill: "white" }) }) }), _jsx(Tooltip, { title: 'Insert branch', children: _jsx(ColorIconButton, { id: "insert-branch-on-edge-button", background: "#248791", size: "small", onClick: () => { handleBranch(link); }, "aria-label": 'Insert branch on edge', children: _jsx(BranchIcon, { id: "insert-branch-on-edge-button-icon", fill: 'white' }) }) }), _jsx(Tooltip, { title: 'Edit link', children: _jsx(ColorIconButton, { id: "edit-edge-button", background: '#c5ab17', size: "small", onClick: handleEditClick, "aria-label": 'Edit link', children: _jsx(EditIcon, { id: "insert-node-on-edge-button-icon", fill: "white" }) }) }), _jsx(Tooltip, { title: 'Delete link', children: _jsx(ColorIconButton, { id: "delete-link-on-edge-button", background: palette.error.main, size: "small", onClick: () => { handleDelete(link); }, "aria-label": 'Delete link button', children: _jsx(DeleteIcon, { id: "delete-link-on-edge-button-icon", fill: 'white' }) }) })] }));
    }, [isEditing, palette.secondary.main, palette.error.main, handleEditClick, handleAdd, link, handleBranch, handleDelete]);
    return _jsx(BaseEdge, { containerId: 'graph-root', fromId: `node-${link.from.id}`, isEditing: isEditing, popoverComponent: popoverComponent, popoverT: popoverT, thiccness: thiccness, timeBetweenDraws: fastUpdate ? 15 : 1000, toId: `node-${link.to.id}` });
};
//# sourceMappingURL=NodeEdge.js.map