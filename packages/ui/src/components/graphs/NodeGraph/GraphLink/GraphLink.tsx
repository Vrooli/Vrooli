import { IconButton, Stack, Tooltip, useTheme } from "@mui/material";
import { usePopover } from "hooks/usePopover.js";
import { AddIcon, DeleteIcon, EditIcon } from "icons/common.js";
import { BranchIcon } from "icons/routineGraph.js";
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { ELEMENT_IDS } from "utils/consts.js";
import { Graph, LinkOperation, NodeLink } from "views/objects/routine/RoutineMultiStepCrud.js";

type Point = {
    x: number;
    y: number;
}
type ToData = {
    link: NodeLink;
    point: Point;
}
type ActionData = {
    label: string;
    onClick: () => unknown;
}

interface GraphLinkProps {
    links: readonly NodeLink[];
    isEditing: boolean;
    onAction: (action: LinkOperation, link: NodeLink) => unknown;
}
interface EdgePopoverMenuProps {
    link: NodeLink;
    allowedActions: readonly LinkOperation[];
    onAction: (action: LinkOperation, link: NodeLink) => unknown;
}

const JUNCTION_OFFSET = 50;
const STROKE_COLOR = "#000";
const LINE_PROPS = { stroke: STROKE_COLOR, strokeWidth: 2 };

/**
 * Calculates the point at the center of a node, relative to the container the graph is in.
 * Since bounding boxes are calculated relative to the viewport, we compensate by 
 * accounting for the graph's scroll position.
 * 
 * @param containerId ID of the container which the point should be measured relative to
 * @param elementId ID of the element that we are finding the center point of
 * @returns x and y coordinates of the center point, as well as the start and end x
 */
function getPoint(containerId: string, nodeId: string): Point | null {
    const graph = document.getElementById(containerId);
    const node = document.getElementById(nodeId);
    if (!graph || !node) {
        console.error("Could not find node to connect to edge", nodeId);
        return null;
    }
    const nodeRect = node.getBoundingClientRect();
    const graphRect = graph.getBoundingClientRect();
    return {
        x: graph.scrollLeft + nodeRect.left + (nodeRect.width / 2) - graphRect.left,
        y: graph.scrollTop + nodeRect.top + (nodeRect.height / 2) - graphRect.top,
    };
}

/**
 * Calculates coordinates needed to draw single or multiple links.
 */
export function calculatePositions(
    links: readonly NodeLink[],
): {
    fromPoint: Point,
    tos: readonly ToData[];
    leftmostX: number;
    rightmostX: number;
    junctionY: number;
} | null {
    if (links.length === 0) return null;

    const fromNode = links[0].from;
    const fromPoint = getPoint(ELEMENT_IDS.RoutineMultiStepCrudGraph, Graph.node.getElementId(fromNode));
    if (!fromPoint) return null;

    const toNodes = links.map((link) => {
        const point = getPoint(ELEMENT_IDS.RoutineMultiStepCrudGraph, Graph.node.getElementId(link.to));
        return point ? { link, point } : null;
    }).filter((x): x is ToData => !!x);

    if (toNodes.length === 0) return null;

    if (toNodes.length === 1) {
        const [{ point: toPoint }] = toNodes;
        const leftmostX = Math.min(fromPoint.x, toPoint.x);
        const rightmostX = Math.max(fromPoint.x, toPoint.x);
        const junctionY = (fromPoint.y + toPoint.y) / 2;
        return { fromPoint, tos: toNodes, leftmostX, rightmostX, junctionY };
    }

    // Multiple links scenario
    const xs = toNodes.map(n => n.point.x);
    const leftmostX = Math.min(...xs, fromPoint.x);
    const rightmostX = Math.max(...xs, fromPoint.x);
    const junctionY = fromPoint.y + JUNCTION_OFFSET;
    return { fromPoint, tos: toNodes, leftmostX, rightmostX, junctionY };
}

/**
 * Renders the lines and buttons for a single link scenario.
 */
export function renderSingleLink(
    fromPoint: Point,
    to: ToData,
    isEditing: boolean,
    onAction: (action: LinkOperation, link: NodeLink) => unknown,
    openMenu: (event: React.MouseEvent<Element>, actions: ActionData[]) => unknown,
) {
    const lines: React.ReactNode[] = [];
    const buttons: React.ReactNode[] = [];

    const actions = [
        { label: "Add Node", onClick: () => onAction("AddNode", link) },
        { label: "Delete Link", onClick: () => onAction("DeleteLink", link) },
    ];
    function handleClick(event: React.MouseEvent<Element>) {
        event.stopPropagation();
        openMenu(event, actions);
    }

    const { link, point: toPoint } = to;
    const midY = (fromPoint.y + toPoint.y) / 2;

    lines.push(<line key="line1" x1={fromPoint.x} y1={fromPoint.y} x2={fromPoint.x} y2={midY} {...LINE_PROPS} />);
    lines.push(<line key="line2" x1={fromPoint.x} y1={midY} x2={toPoint.x} y2={midY} {...LINE_PROPS} />);
    lines.push(<line key="line3" x1={toPoint.x} y1={midY} x2={toPoint.x} y2={toPoint.y} {...LINE_PROPS} />);

    const circleX = (fromPoint.x + toPoint.x) / 2;
    const circleY = midY;

    if (isEditing) {
        buttons.push(
            <circle
                key="single-button"
                cx={circleX}
                cy={circleY}
                r={8}
                fill="#fff"
                stroke="#000"
                style={{ pointerEvents: "auto", cursor: "pointer" }}
                onClick={handleClick}
            />,
        );
    }

    return { lines, buttons };
}

/**
 * Renders the lines and buttons for multiple links (branching).
 */
export function renderMultipleLinks(
    fromPoint: Point,
    tos: readonly ToData[],
    leftmostX: number,
    rightmostX: number,
    junctionY: number,
    isEditing: boolean,
    onAction: (action: LinkOperation, link: NodeLink) => unknown,
    openMenu: (event: React.MouseEvent<Element>, actions: ActionData[]) => unknown,
) {
    const lines: React.ReactNode[] = [];
    const buttons: React.ReactNode[] = [];

    function handleClick(event: React.MouseEvent<Element>) {
        event.stopPropagation();
        openMenu(event, []); //TODO
    }

    // Main vertical line
    lines.push(<line key="vertical-main" x1={fromPoint.x} y1={fromPoint.y} x2={fromPoint.x} y2={junctionY} {...LINE_PROPS} />);
    // Horizontal line at junction
    lines.push(<line key="horizontal-junction" x1={leftmostX} y1={junctionY} x2={rightmostX} y2={junctionY} {...LINE_PROPS} />);

    tos.forEach(({ link, point: toPoint }) => {
        lines.push(<line key={`branch-line-${link.id}`} x1={toPoint.x} y1={junctionY} x2={toPoint.x} y2={toPoint.y} {...LINE_PROPS} />);
        if (isEditing) {
            const circleX = toPoint.x;
            const circleY = (junctionY + toPoint.y) / 2;
            const actions = [
                { label: "Add Node", onClick: () => onAction("AddNode", link) },
                { label: "Delete Link", onClick: () => onAction("DeleteLink", link) },
            ];

            buttons.push(
                <circle
                    key={`branch-button-${link.id}`}
                    cx={circleX}
                    cy={circleY}
                    r={8}
                    fill="#fff"
                    stroke="#000"
                    style={{ pointerEvents: "auto", cursor: "pointer" }}
                    onClick={handleClick}
                />,
            );
        }
    });

    if (isEditing) {
        const circleX = fromPoint.x;
        const circleY = junctionY;
        const junctionActions = [
            { label: "Add Branch", onClick: () => onAction("Branch", tos[0].link) }, // Any link will do
            // { label: "Add Condition", onClick: () => onAction("SetConditions", tos[0].link) }, // Any link will do
        ];

        buttons.push(
            <circle
                key="junction-button"
                cx={circleX}
                cy={circleY}
                r={10}
                fill="#fff"
                stroke="#000"
                style={{ pointerEvents: "auto", cursor: "pointer" }}
                onClick={handleClick}
            />,
        );
    }

    return { lines, buttons };
}

/**
 * Displays a series of icon buttons for allowed actions on a link
 */
export function EdgePopoverMenu({
    link,
    allowedActions,
    onAction,
}: EdgePopoverMenuProps) {
    const { palette } = useTheme();

    const handleAddNode = useCallback(function handleAddNodeCallback() {
        onAction("AddNode", link);
    }, [onAction, link]);
    const handleBranch = useCallback(function handleBranchCallback() {
        onAction("Branch", link);
    }, [onAction, link]);
    const handleDeleteLink = useCallback(function handleDeleteLinkCallback() {
        onAction("DeleteLink", link);
    }, [onAction, link]);
    const handleEditLink = useCallback(function handleEditLinkCallback() {
        onAction("EditLink", link);
    }, [onAction, link]);

    const buttons = useMemo(() => {
        const btns: JSX.Element[] = [];

        if (allowedActions.includes("AddNode")) {
            btns.push(
                <Tooltip key="addNode" title='Insert node'>
                    <IconButton
                        id="insert-node-on-edge-button"
                        size="small"
                        onClick={handleAddNode}
                        aria-label='Insert node on edge'
                        sx={{ background: palette.secondary.main }}
                    >
                        <AddIcon id="insert-node-on-edge-button-icon" fill="white" />
                    </IconButton>
                </Tooltip>,
            );
        }

        if (allowedActions.includes("Branch")) {
            btns.push(
                <Tooltip key="branch" title='Insert branch'>
                    <IconButton
                        id="insert-branch-on-edge-button"
                        size="small"
                        onClick={handleBranch}
                        aria-label='Insert branch on edge'
                        sx={{ background: "#248791" }}
                    >
                        <BranchIcon id="insert-branch-on-edge-button-icon" fill='white' />
                    </IconButton>
                </Tooltip>,
            );
        }

        if (allowedActions.includes("EditLink")) {
            btns.push(
                <Tooltip key="editLink" title='Edit link'>
                    <IconButton
                        id="edit-edge-button"
                        size="small"
                        onClick={handleEditLink}
                        aria-label='Edit link'
                        sx={{ background: "#c5ab17" }}
                    >
                        <EditIcon id="edit-edge-button-icon" fill="white" />
                    </IconButton>
                </Tooltip>,
            );
        }

        if (allowedActions.includes("DeleteLink")) {
            btns.push(
                <Tooltip key="deleteLink" title='Delete link'>
                    <IconButton
                        id="delete-link-on-edge-button"
                        size="small"
                        onClick={handleDeleteLink}
                        aria-label='Delete link button'
                        sx={{ background: palette.error.main }}
                    >
                        <DeleteIcon id="delete-link-on-edge-button-icon" fill='white' />
                    </IconButton>
                </Tooltip>,
            );
        }

        return btns;
    }, [allowedActions, handleAddNode, palette.secondary.main, palette.error.main, handleBranch, handleEditLink, handleDeleteLink]);

    if (buttons.length === 0) return null;

    return (
        <Stack direction="row" spacing={1}>
            {buttons}
        </Stack>
    );
}

const svgStyle = {
    position: "absolute",
    top: 0,
    left: 0,
    width: "100%",
    height: "100%",
    pointerEvents: "none",
    zIndex: 0, // Should be behind nodes
} as const;

/**
 * Main GraphLink component that uses the helper functions.
 */
export function GraphLink({
    links,
    isEditing,
    onAction,
}: GraphLinkProps) {
    const [coords, setCoords] = useState<ReturnType<typeof calculatePositions> | null>(null);
    const [menuActions, setMenuActions] = useState<ActionData[]>([]);

    const [anchorEl, handleOpen, handleClose, isOpen] = usePopover();

    const openMenu = useCallback((event: React.MouseEvent<Element>, actions: ActionData[]) => {
        setMenuActions(actions);
        handleOpen(event);
    }, [handleOpen]);

    const closeMenu = useCallback(() => {
        handleClose();
        setMenuActions([]);
    }, [handleClose]);

    const recalc = useCallback(() => {
        const c = calculatePositions(links);
        setCoords(c);
    }, [links]);

    useEffect(() => {
        recalc();
    }, [recalc]);

    if (!coords) return null;

    const { fromPoint, tos, leftmostX, rightmostX, junctionY } = coords;
    const isSingleLink = tos.length === 1;

    let lines: React.ReactNode[] = [];
    let buttons: React.ReactNode[] = [];

    if (isSingleLink) {
        const result = renderSingleLink(fromPoint, tos[0], isEditing, onAction, openMenu);
        lines = result.lines;
        buttons = result.buttons;
    } else {
        const result = renderMultipleLinks(fromPoint, tos, leftmostX, rightmostX, junctionY, isEditing, onAction, openMenu);
        lines = result.lines;
        buttons = result.buttons;
    }

    // const popoverComponent = useMemo(() => {
    //     if (!isEditing) return <></>;
    //     return (
    //         <EdgePopoverMenu
    //             link={link}
    //             allowedActions={allowedActions}
    //             onAddNode={handleAdd}
    //             onConnectNode={() => {/* Not implemented yet */}}
    //             onBranch={handleBranch}
    //             onEditLink={handleEditClick}
    //             onDeleteLink={handleDelete}
    //         />
    //     );
    // }, [isEditing, link, allowedActions, handleAdd, handleBranch, handleDelete, handleEditClick]);


    return (
        <>
            <svg
                style={svgStyle}
            >
                {lines}
                {buttons}
            </svg>

            {/* {anchorEl && (
                <PopoverMenu
                    anchorPosition={anchorEl}
                    onClose={closeMenu}
                    actions={menuActions}
                />
            )} */}
        </>
    );
}
