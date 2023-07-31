import { Stack, Tooltip, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { AddIcon, BranchIcon, DeleteIcon, EditIcon, LinkIcon } from "icons";
import { useCallback, useMemo } from "react";
import { calculateNodeSize } from "../../nodes";
import { BaseEdge } from "../BaseEdge/BaseEdge";
import { NodeEdgeProps } from "../types";

/**
 * Displays a line between two nodes of a routine graph
 * If in editing mode, displays a clickable button to edit the link or inserting a node
 */
export const NodeEdge = ({
    fastUpdate,
    handleAdd,
    handleBranch,
    handleDelete,
    handleEdit,
    isEditing,
    isFromRoutineList,
    isToRoutineList,
    link,
    scale,
}: NodeEdgeProps) => {
    const { palette } = useTheme();

    // Store dimensions of edge
    const thiccness = useMemo(() => Math.max(Math.min(Math.ceil(calculateNodeSize(30, scale)), 50), 10), [scale]);

    // Triggers edit menu in parent. This is needed because the link's from and to nodes
    // can be updated, and the edge doesn't have this information
    const handleEditClick = useCallback(() => {
        handleEdit(link);
    }, [handleEdit, link]);

    /**
     * Place button along bezier to display "Add Node" button
     */
    const popoverT = useMemo(() => {
        // If from and to are both routine lists (or both NOT routine lists), then use bezier midpoint.
        // If one is a routine list and the other is not, use a point further from the routine list
        let t = 0.5;
        if (isFromRoutineList && !isToRoutineList) t = 0.77;
        else if (!isFromRoutineList && isToRoutineList) t = 0.23;
        return t;
    }, [isFromRoutineList, isToRoutineList]);

    /**
     * If isEditable, displays a clickable button for editing the edge or inserting a node
     */
    const popoverComponent = useMemo(() => {
        if (!isEditing) return <></>;
        return (
            <Stack direction="row" spacing={1}>
                {/* Insert Node */}
                <Tooltip title='Insert node'>
                    <ColorIconButton
                        id="insert-node-on-edge-button"
                        background={palette.secondary.main}
                        size="small"
                        onClick={() => { handleAdd(link); }}
                        aria-label='Insert node on edge'
                    >
                        <AddIcon id="insert-node-on-edge-button-icon" fill={"white"} />
                    </ColorIconButton>
                </Tooltip>
                {/* Connect unlinked node TODO */}
                <Tooltip title='Connect unlinked node'>
                    <ColorIconButton
                        id="connect-node-on-edge-button"
                        background='#d27787'
                        size="small"
                        onClick={() => { }}
                        aria-label='Connect unlinked node'
                    >
                        <LinkIcon id="connect-node-on-edge-button-icon" fill={"white"} />
                    </ColorIconButton>
                </Tooltip>
                {/* Insert Branch */}
                <Tooltip title='Insert branch'>
                    <ColorIconButton
                        id="insert-branch-on-edge-button"
                        background="#248791"
                        size="small"
                        onClick={() => { handleBranch(link); }}
                        aria-label='Insert branch on edge'
                    >
                        <BranchIcon id="insert-branch-on-edge-button-icon" fill='white' />
                    </ColorIconButton>
                </Tooltip>
                {/* Edit Link TODO */}
                <Tooltip title='Edit link'>
                    <ColorIconButton
                        id="edit-edge-button"
                        background='#c5ab17'
                        size="small"
                        onClick={handleEditClick}
                        aria-label='Edit link'
                    >
                        <EditIcon id="insert-node-on-edge-button-icon" fill={"white"} />
                    </ColorIconButton>
                </Tooltip>
                {/* Delete link */}
                <Tooltip title='Delete link'>
                    <ColorIconButton
                        id="delete-link-on-edge-button"
                        background={palette.error.main}
                        size="small"
                        onClick={() => { handleDelete(link); }}
                        aria-label='Delete link button'
                    >
                        <DeleteIcon id="delete-link-on-edge-button-icon" fill='white' />
                    </ColorIconButton>
                </Tooltip>
            </Stack>
        );
    }, [isEditing, palette.secondary.main, palette.error.main, handleEditClick, handleAdd, link, handleBranch, handleDelete]);

    return <BaseEdge
        containerId='graph-root'
        fromId={`node-${link.from.id}`}
        isEditing={isEditing}
        popoverComponent={popoverComponent}
        popoverT={popoverT}
        thiccness={thiccness}
        timeBetweenDraws={fastUpdate ? 15 : 1000}
        toId={`node-${link.to.id}`}
    />;
};
