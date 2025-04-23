
import { Status } from "@local/shared";
import { Box, BoxProps, IconButton, SxProps, Tooltip, Typography, TypographyProps, styled, useTheme } from "@mui/material";
import { ReactNode, useCallback, useEffect, useRef, useState } from "react";
import Draggable, { DraggableData, DraggableEvent } from "react-draggable";
import { multiLineEllipsis, noSelect } from "../../../../styles.js";
import { ELEMENT_IDS } from "../../../../utils/consts.js";
import { NODE_HIGHLIGHT_ERROR, NODE_HIGHLIGHT_SELECTED, NODE_HIGHLIGHT_WARNING, addHighlight, removeHighlights } from "../../../../utils/display/documentTools.js";
import { getDisplay } from "../../../../utils/display/listTools.js";
import { PubSub } from "../../../../utils/pubsub.js";
// import { routineTypes } from "../../../../utils/search/schemas/routine";
import { Graph, NodeLoop, NodeOperation, NodeRedirect, RoutineListItem, SomeNode, SubroutineOperation, type NodeEnd, type NodeRoutineList } from "../../../../views/objects/routine/RoutineMultiStepCrud.js";

type MessageWithStatus = {
    message: string;
    status: Status;
}
interface NodeBaseProps {
    isEditing: boolean;
    isLinked: boolean;
    isSelected: boolean;
    language: string;
    messages: MessageWithStatus[];
    node: SomeNode;
    onSelect: (node: SomeNode) => unknown;
    scale: number;
}

interface EndNodeProps extends NodeBaseProps {
    handleAction: (action: NodeOperation, node: NodeEnd) => unknown;
    node: NodeEnd;
}

interface LoopNodeProps extends NodeBaseProps {
    node: NodeLoop;
}

interface RedirectNodeProps extends NodeBaseProps {
    handleAction: (action: NodeOperation, node: NodeRedirect) => unknown;
    node: NodeRedirect;// & { redirect: NodeRedirectShape }; TODO
}

interface RoutineListNodeProps extends NodeBaseProps {
    handleAction: (action: NodeOperation, node: NodeRoutineList) => unknown;
    handleSubroutineAction: (action: SubroutineOperation, nodeId: string, item: RoutineListItem) => unknown;
    language: string;
    node: NodeRoutineList;
}

interface DragBoxProps extends BoxProps {
    canDrag: boolean;
    isDragging: boolean;
}

const DEFAULT_CURSOR_POSITION = { x: 0, y: 0 };
const DEFAULT_SCROLL_POSITION = { left: 0, top: 0 };
const SHOW_LABELS_ABOVE_PERCENT = 50;
const NODE_SHADOW = 4;
const NODE_DRAG_CLASSNAME = "handle";

enum NodeType {
    Redirect = "Redirect",
    RoutineList = "RoutineList",
    End = "End",
}

const NodeWidth = {
    [NodeType.Redirect]: {
        Min: 24,
        Default: 100,
        Max: 250,
    },
    [NodeType.RoutineList]: {
        Min: 25,
        Default: 350,
        Max: 750,
    },
} as const;
const IconSize = {
    [NodeType.End]: {
        Min: 24,
        Default: 24,
        Max: 48,
    },
};
const FontSize = {
    [NodeType.End]: {
        Min: 0.5,
        Default: 1,
        Max: 3,
    },
    [NodeType.RoutineList]: {
        Min: 8,
        Default: 12,
        Max: 24,
    },
};

function useIsLabelVisible(scale: number): boolean {
    return scale >= SHOW_LABELS_ABOVE_PERCENT;
}

function useNodeHighlight(element: HTMLElement | null, isSelected: boolean, messages: MessageWithStatus[]) {
    useEffect(function applyStatusStyles() {
        if (!element) return;

        if (isSelected) {
            addHighlight(NODE_HIGHLIGHT_SELECTED, element);
            return;
        }
        removeHighlights(NODE_HIGHLIGHT_SELECTED, element);

        const hasError = messages.some((msg) => msg.status === Status.Invalid);
        if (hasError) {
            addHighlight(NODE_HIGHLIGHT_ERROR, element);
            return;
        }
        removeHighlights(NODE_HIGHLIGHT_ERROR, element);

        const hasWarning = messages.some((msg) => msg.status === Status.Incomplete);
        if (hasWarning) {
            addHighlight(NODE_HIGHLIGHT_WARNING, element);
            return;
        }
        removeHighlights(NODE_HIGHLIGHT_WARNING, element);
    }, [element, isSelected, messages]);
}

export const routineNodeCheckboxOption: SxProps = {
    padding: "4px",
    height: "36px",
};

export function routineNodeActionStyle(isEditing: boolean): SxProps {
    return {
        display: "inline-flex",
        alignItems: "center",
        cursor: isEditing ? "pointer" : "default",
        height: "36px",
        paddingRight: "8px",
    } as const;
}

export function calculateNodeSize(
    scale: number,
    sizes: {
        Min: number;
        Default: number;
        Max: number;
    },
): number {
    const targetSize = Math.round(sizes.Default * scale / 100);
    return Math.min(Math.max(targetSize, sizes.Min), sizes.Max);
}

const DragBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isDragging" && prop !== "canDrag",
})<DragBoxProps>(({ canDrag, isDragging }) => ({
    display: "flex",
    justifyContent: "center",
    alignItems: "center",
    zIndex: isDragging ? 100 : 1, // Always above links, with dragging on top
    opacity: isDragging ? 0.5 : 1,
    cursor: canDrag ? "grab" : "pointer",
    "&:active": {
        cursor: canDrag ? "grabbing" : "pointer",
    },
}));

interface BaseNodeProps extends Pick<NodeBaseProps, "isEditing" | "node" | "onSelect">, Omit<BoxProps, "onSelect"> {
    children: ReactNode;
}

/**
 * Base node component that all other nodes extend from.
 * Has three main functions: drag and drop, showing error/warning highlights, and being selectable.
 * 
 * NOTE: The draggable part of a node must have the NODE_DRAG_CLASSNAME class.
 */
function BaseNode({
    children,
    isEditing,
    node,
    onSelect,
    ...props
}: BaseNodeProps) {
    const [position, setPosition] = useState(DEFAULT_CURSOR_POSITION);
    const [isDragging, setIsDragging] = useState(false);

    const handleSelect = useCallback(() => {
        onSelect(node);
    }, [node, onSelect]);

    // Track the initial cursor offset relative to the node's top-left corner
    const cursorOffset = useRef(DEFAULT_CURSOR_POSITION);
    const initialCursorPos = useRef(DEFAULT_CURSOR_POSITION);
    const initialScrollPos = useRef(DEFAULT_SCROLL_POSITION);

    const nodeScrollCallback = useCallback((x: number, y: number) => {
        const gridElement = document.getElementById(ELEMENT_IDS.RoutineMultiStepCrudGraph);
        if (!gridElement) return;

        // Similar logic to your old code. 
        // Adjust scrolling based on cursor position within container coords (x,y).

        const speedRef = { current: 0 };
        function calculateSpeed(distToEdge: number, sideLength: number) {
            const maxSpeed = 25;
            const minSpeed = 5;
            const percent = 1 - distToEdge / (sideLength * 0.15);
            return (maxSpeed - minSpeed) * percent + minSpeed;
        }

        // Check horizontal edges
        if (x < (window.innerWidth * 0.15)) {
            const distToEdge = (window.innerWidth * 0.15) - x;
            const speed = calculateSpeed(distToEdge, window.innerWidth);
            gridElement.scrollBy(-speed, 0);
        } else if (x > window.innerWidth - (window.innerWidth * 0.15)) {
            const distToEdge = x - (window.innerWidth - (window.innerWidth * 0.15));
            const speed = calculateSpeed(distToEdge, window.innerWidth);
            gridElement.scrollBy(speed, 0);
        }

        // Check vertical edges
        if (y < (window.innerHeight * 0.15)) {
            const distToEdge = (window.innerHeight * 0.15) - y;
            const speed = calculateSpeed(distToEdge, window.innerHeight);
            gridElement.scrollBy(0, -speed);
        } else if (y > window.innerHeight - (window.innerHeight * 0.15)) {
            const distToEdge = y - (window.innerHeight - (window.innerHeight * 0.15));
            const speed = calculateSpeed(distToEdge, window.innerHeight);
            gridElement.scrollBy(0, speed);
        }

        // You can throttle or debounce this call if needed
    }, []);

    // const handleDragStart = useCallback(
    //     (e: DraggableEvent, data: DraggableData) => {
    //         if (!canDrag) return;

    //         setIsDragging(true);

    //         // Determine cursor position based on event type
    //         let cursorX: number;
    //         let cursorY: number;
    //         if (e instanceof TouchEvent && e.touches.length > 0) {
    //             cursorX = e.touches[0].clientX;
    //             cursorY = e.touches[0].clientY;
    //         } else if (Object.prototype.hasOwnProperty.call(e, "clientX") && Object.prototype.hasOwnProperty.call(e, "clientY")) {
    //             cursorX = (e as MouseEvent).clientX;
    //             cursorY = (e as MouseEvent).clientY;
    //         } else {
    //             console.warn("Unhandled event type in handleDragStart", e);
    //             return; // Skip further processing if event type is unsupported
    //         }

    //         // Calculate the cursor's offset from the node's position
    //         const { x, y } = position;
    //         cursorOffset.current = { x: cursorX - x, y: cursorY - y };
    //     },
    //     [canDrag, position],
    // );
    const handleDragStart = useCallback((e: DraggableEvent, data: DraggableData) => {
        if (!isEditing) return false;

        setIsDragging(true);

        const gridElement = document.getElementById(ELEMENT_IDS.RoutineMultiStepCrudGraph);
        if (!gridElement) return false;

        // Current container scroll offset
        const currentScrollLeft = gridElement.scrollLeft;
        const currentScrollTop = gridElement.scrollTop;
        initialScrollPos.current = { left: currentScrollLeft, top: currentScrollTop };

        // Determine cursor position
        let cursorX: number;
        let cursorY: number;
        if (e instanceof TouchEvent && e.touches.length > 0) {
            cursorX = e.touches[0].clientX;
            cursorY = e.touches[0].clientY;
        } else if ("clientX" in e && "clientY" in e) {
            cursorX = e.clientX;
            cursorY = e.clientY;
        } else {
            console.warn("Unhandled event type in handleDragStart", e);
            return false;
        }

        initialCursorPos.current = { x: cursorX, y: cursorY };

        // Calculate offset: how far the cursor is from the node's top-left corner
        // position is relative to the container. We need to convert cursor coords (viewport) 
        // into container coords by adding the scroll offsets.
        cursorOffset.current = {
            x: cursorX + currentScrollLeft - position.x,
            y: cursorY + currentScrollTop - position.y,
        };
    }, [isEditing, position]);

    // const handleDrag = useCallback(
    //     (e: DraggableEvent) => {
    //         if (!canDrag) return;

    //         // Get cursor position
    //         const cursorX = e instanceof MouseEvent ? e.clientX : e.touches[0].clientX;
    //         const cursorY = e instanceof MouseEvent ? e.clientY : e.touches[0].clientY;

    //         // Update the node position based on the cursor and initial offset
    //         const { x: offsetX, y: offsetY } = cursorOffset.current;
    //         setPosition({ x: cursorX - offsetX, y: cursorY - offsetY });
    //     },
    //     [canDrag],
    // );
    const handleDrag = useCallback((e: DraggableEvent, data: DraggableData) => {
        console.log("in handleDrag", { e, data, isDragging, isEditing });
        if (!isDragging) return;

        const gridElement = document.getElementById(ELEMENT_IDS.RoutineMultiStepCrudGraph);
        if (!gridElement) return;

        // Current cursor position
        let cursorX: number;
        let cursorY: number;
        if (e instanceof TouchEvent && e.touches.length > 0) {
            cursorX = e.touches[0].clientX;
            cursorY = e.touches[0].clientY;
        } else if ("clientX" in e && "clientY" in e) {
            cursorX = e.clientX;
            cursorY = e.clientY;
        } else {
            return;
        }

        // Current container scroll offset
        const currentScrollLeft = gridElement.scrollLeft;
        const currentScrollTop = gridElement.scrollTop;

        // Convert cursor to container coords:
        // The new position should be the cursor position offset by the initial cursor offset,
        // adjusted by the difference in scroll since dragging started.
        const deltaScrollX = currentScrollLeft - initialScrollPos.current.left;
        const deltaScrollY = currentScrollTop - initialScrollPos.current.top;

        const newX = cursorX + currentScrollLeft - cursorOffset.current.x;
        const newY = cursorY + currentScrollTop - cursorOffset.current.y;
        console.log("calculated new position", { newX, newY, cursorX, cursorY, currentScrollLeft, currentScrollTop, cursorOffset: cursorOffset.current });

        setPosition({ x: newX, y: newY });

        // Check if we need to scroll the container because we are near the edge
        nodeScrollCallback(newX - currentScrollLeft, newY - currentScrollTop);

    }, [isEditing, isDragging, nodeScrollCallback]);

    const handleDrop = useCallback(() => {
        if (!isEditing) return;

        setIsDragging(false);
        PubSub.get().publish("nodeDrop", { nodeId: node.id, position });
    }, [isEditing, node.id, position]);

    return (
        <Draggable
            handle={`.${NODE_DRAG_CLASSNAME}`}
            position={position}
            disabled={!isEditing}
            onStart={handleDragStart}
            onDrag={handleDrag}
            onStop={handleDrop}
        >
            <DragBox
                id={Graph.node.getElementId(node, "draggable-container")}
                canDrag={isEditing}
                isDragging={isDragging}
                onClick={handleSelect}
                {...props}
            >
                {children}
            </DragBox>
        </Draggable>
    );
}

const stateToIconColor = {
    Success: "#2e7d32",
    Fail: "#ed6c02",
} as const;
const stateToBackgroundColor = {
    Success: "#edf7ed",
    Fail: "#fff4e5",
} as const;
const stateToLabelColor = {
    Success: "#1e4620",
    Fail: "#663c00",
} as const;
const stateToLabel = {
    Success: "Success",
    Fail: "Fail",
} as const;
// const stateToIcon = {
//     Success: SaveIcon,
//     Fail: WarningIcon,
// } as const;

interface EndNodeOuterProps extends BoxProps {
    isLabelVisible: boolean;
    state: "Success" | "Fail";
}

const EndNodeOuter = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isLabelVisible" && prop !== "state",
})<EndNodeOuterProps>(({ isLabelVisible, state, theme }) => ({
    display: "flex",
    alignItems: "center",
    justifyContent: "space-around",
    background: stateToBackgroundColor[state],
    // eslint-disable-next-line no-magic-numbers
    borderRadius: theme.spacing(4),
    // eslint-disable-next-line no-magic-numbers
    padding: isLabelVisible ? theme.spacing(0.75) : `${theme.spacing(0.5)} ${theme.spacing(1)}`,
    "& .node-end-icon": {
        paddingTop: 0,
        paddingBottom: 0,
    },
    "& .node-end-label": {
        display: isLabelVisible ? "block" : "none",
        padding: 0,
        marginLeft: theme.spacing(1),
    },
    width: "auto",
}));

export function EndNode({
    handleAction,
    isLinked,
    isSelected,
    language,
    messages,
    node,
    scale,
    ...props
}: EndNodeProps) {
    const isLabelVisible = useIsLabelVisible(scale);

    const nodeRef = useRef<HTMLElement>(null);
    useNodeHighlight(nodeRef.current, isSelected, messages);

    const wasSuccessful = node.end?.wasSuccessful === true;
    const state = wasSuccessful ? "Success" : "Fail";
    const iconColor = stateToIconColor[state];
    // const Icon = stateToIcon[state];

    const iconSize = calculateNodeSize(scale, IconSize.End);

    const labelStyle = {
        color: stateToLabelColor[state],
        fontSize: `${calculateNodeSize(scale, FontSize.End)}rem`,
    } as const;

    return (
        <BaseNode
            className={NODE_DRAG_CLASSNAME}
            node={node}
            {...props}
        >
            <EndNodeOuter
                id={Graph.node.getElementId(node)}
                isLabelVisible={isLabelVisible}
                ref={nodeRef}
                state={state}
            >
                {/* <Icon
                    className="node-end-icon"
                    width={iconSize}
                    height={iconSize}
                    fill={iconColor}
                /> */}
                <Typography
                    className="node-end-label"
                    variant="body2"
                    sx={labelStyle}
                >
                    {stateToLabel[state]}
                </Typography>
            </EndNodeOuter>
        </BaseNode>
    );
}

export function RedirectNode({
    isSelected,
    messages,
    node,
    scale,
    ...props
}: RedirectNodeProps) {
    // const nodeSize = `${calculateNodeSize(scale, NodeWidth.Redirect)}px`;
    const nodeRef = useRef<HTMLElement>(null);
    useNodeHighlight(nodeRef.current, isSelected, messages);

    return (
        <BaseNode
            className={NODE_DRAG_CLASSNAME}
            node={node}
            {...props}
        >
            <Tooltip placement={"top"} title='Redirect'>
                <IconButton
                    id={Graph.node.getElementId(node)}
                // ref={nodeRef}
                // sx={{
                //     width: nodeSize,
                //     height: nodeSize,
                //     position: "relative",
                //     display: "block",
                //     backgroundColor: "#6daf72",
                //     color: "white",
                //     boxShadow: "0px 0px 12px gray",
                //     "&:hover": {
                //         backgroundColor: "#6daf72",
                //         filter: "brightness(120%)",
                //         transition: "filter 0.2s",
                //     },
                // }}
                >
                    {/* <RedirectIcon
                        id={Graph.node.getElementId(node, "redirect-icon")}
                    // sx={{
                    //     width: '100%',
                    //     height: '100%',
                    //     color: '#00000044',
                    //     '&:hover': {
                    //         transform: 'scale(1.2)',
                    //         transition: 'scale .2s ease-in-out',
                    //     }
                    // }}
                    /> */}
                </IconButton>
            </Tooltip>
        </BaseNode>
    );
}

const SHOW_BUTTON_LABEL_ABOVE_SCALE = 30;

interface RoutineListOuterProps extends BoxProps {
    isEditing: boolean;
    nodeSize: string;
}

const RoutineListOuter = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isEditing" && prop !== "nodeSize",
})<RoutineListOuterProps>(({ isEditing, nodeSize, theme }) => ({
    display: "block",
    alignItems: "center",
    backgroundColor: theme.palette.background.paper,
    color: theme.palette.background.textPrimary,
    paddingLeft: "0.1em!important",
    paddingRight: "0.1em!important",
    textAlign: "center",
    cursor: isEditing ? "grab" : "pointer",
    "&:active": {
        cursor: isEditing ? "grabbing" : "pointer",
    },
    "&:hover": {
        filter: "brightness(120%)",
        transition: "filter 0.2s",
    },
    width: nodeSize,
    position: "relative",
    borderRadius: theme.spacing(1),
    overflow: "hidden",
    boxShadow: theme.shadows[NODE_SHADOW],
    "@media print": {
        border: `1px solid ${theme.palette.mode === "light" ? theme.palette.primary.dark : theme.palette.secondary.dark}`,
        boxShadow: "none",
    },
}));

interface NodeTitle2Props extends TypographyProps {
    fontSize: string;
    isLabelVisible: boolean;
}

const NodeTitle2 = styled(Typography, {
    shouldForwardProp: (prop) => prop !== "fontSize" && prop !== "isLabelVisible",
})<NodeTitle2Props>(({ fontSize, isLabelVisible, theme }) => ({
    ...noSelect,
    ...multiLineEllipsis(1),
    textAlign: "start",
    paddingLeft: theme.spacing(1),
    width: "100%",
    lineBreak: "anywhere" as const,
    whiteSpace: "pre" as const,
    fontSize,
    display: isLabelVisible ? "block" : "none",
    "@media print": {
        color: "black",
    },
}));

const isOptionalButtonStyle = { ...routineNodeCheckboxOption };
const isOrderedButtonStyle = { ...routineNodeCheckboxOption };
const deleteButtonStyle = { marginLeft: "auto" } as const;
const buttonLabelStyle = { marginLeft: "4px" } as const;

export function RoutineListNode({
    handleAction,
    handleSubroutineAction,
    isLinked,
    isSelected,
    isEditing,
    language,
    messages,
    node,
    onSelect,
    scale,
}: RoutineListNodeProps) {
    const { palette } = useTheme();

    const label = getDisplay(node, [language]).title;
    const isLabelVisible = useIsLabelVisible(scale);

    const nodeRef = useRef<HTMLDivElement>(null);
    useNodeHighlight(nodeRef.current, isSelected, messages);

    const handleNodeDelete = useCallback(() => { handleAction("delete", node); }, [handleAction, node]);

    const toggleOrdered = useCallback(function toggleOrderedCallback() {
        if (!isEditing) return;
        handleAction("update", {
            ...node,
            routineList: { ...node.routineList, isOrdered: !node.routineList.isOrdered },
        });
    }, [handleAction, isEditing, node]);

    const toggleOptional = useCallback(function toggleOptionalCallback() {
        if (!isEditing) return;
        handleAction("update", {
            ...node,
            routineList: { ...node.routineList, isOptional: !node.routineList.isOptional },
        });
    }, [handleAction, isEditing, node]);

    const [editDialogOpen, setEditDialogOpen] = useState<boolean>(false);
    const openEditDialog = useCallback(() => {
        if (isLinked) {
            setEditDialogOpen(!editDialogOpen);
        }
    }, [isLinked, editDialogOpen]);
    const closeEditDialog = useCallback(() => { setEditDialogOpen(false); }, []);

    // // Opens dialog to add a new subroutine, so no suroutineId is passed
    // const handleSubroutineAdd = useCallback(() => {
    //     handleAction(BuildAction.AddSubroutine, node.id);
    // }, [handleAction, node.id]);

    const nodeSize = `${calculateNodeSize(scale, NodeWidth.RoutineList)}px`;
    const fontSize = `${calculateNodeSize(scale, NodeWidth.RoutineList) / 10}px`;

    const onSubroutineAction = useCallback((action: SubroutineOperation, item: RoutineListItem) => {
        handleSubroutineAction(action, node.id, item);
    }, [handleSubroutineAction, node.id]);

    return (
        // {/* Normal-click menu */ }
        //     {/* <NodeWithRoutineListCrud
        //         display="Dialog"
        //         onCancel={closeEditDialog}
        //         onClose={closeEditDialog}
        //         onCompleted={handleUpdate}
        //         onDeleted={handleDelete}
        //         isCreate={false}
        //         isEditing={isEditing}
        //         isOpen={editDialogOpen}
        //         language={language}
        //         overrideObject={node as NodeRoutineList}
        //     /> */}
        <BaseNode
            className={NODE_DRAG_CLASSNAME}
            isEditing={isEditing}
            node={node}
            onSelect={onSelect}
        >
            <RoutineListOuter
                id={Graph.node.getElementId(node)}
                isEditing={isEditing}
                nodeSize={nodeSize}
                ref={nodeRef}
            >
                <Box display="flex" justifyContent="space-between" alignItems="center" width="100%">
                    <NodeTitle2
                        fontSize={fontSize}
                        id={`node-routinelist-title-${node.id}`}
                        isLabelVisible={isLabelVisible}
                        variant="h6"
                    >{label}</NodeTitle2>
                    {/* {
                        isEditing && (
                            <IconButton
                                id={Graph.node.getElementId(node, "delete-icon-button")}
                                onClick={handleNodeDelete}
                                onTouchStart={handleNodeDelete}
                                color="inherit"
                                sx={deleteButtonStyle}
                            >
                                <CloseIcon id={Graph.node.getElementId(node, "delete-icon")} />
                            </IconButton>
                        )
                    } */}
                </Box>
                <Box display="flex" justifyContent="flex-start" alignItems="center">
                    <Box
                        onClick={toggleOrdered}
                        onTouchStart={toggleOrdered}
                        sx={routineNodeActionStyle(isEditing)}
                    >
                        {/* <IconButton
                            size="small"
                            sx={isOrderedButtonStyle}
                            disabled={!isEditing}
                        >
                            {node.routineList.isOrdered ? <ListNumberIcon fill={palette.background.textPrimary} /> : <ListBulletIcon fill={palette.background.textPrimary} />}
                        </IconButton> */}
                        {scale > SHOW_BUTTON_LABEL_ABOVE_SCALE && <Typography sx={buttonLabelStyle}>{node.routineList.isOrdered ? "Ordered" : "Unordered"}</Typography>}
                    </Box>
                    <Box
                        onClick={toggleOptional}
                        onTouchStart={toggleOptional}
                        sx={routineNodeActionStyle(isEditing)}
                    >
                        {/* <IconButton
                            size="small"
                            sx={isOptionalButtonStyle}
                            disabled={!isEditing}
                        >
                            {node.routineList.isOptional ? <NoActionIcon fill={palette.background.textPrimary} /> : <ActionIcon fill={palette.background.textPrimary} />}
                        </IconButton> */}
                        {scale > SHOW_BUTTON_LABEL_ABOVE_SCALE && <Typography sx={buttonLabelStyle}>{node.routineList.isOptional ? "Optional" : "Required"}</Typography>}
                    </Box>
                </Box>
            </RoutineListOuter>
        </BaseNode>
    );
}

// export function SubroutineNode({
//     data,
//     scale,
//     isEditing,
//     handleAction,
//     label,
// }: SubroutineNodeProps) {
//     const { palette } = useTheme();
//     const isLabelVisible = useIsLabelVisible(scale, label);

//     const nodeSize = useMemo(() => `${calculateNodeSize(220, scale)}px`, [scale]);
//     const fontSize = useMemo(() => `min(${calculateNodeSize(NodeWidth.RoutineList, scale) / 9}px, 1.5em)`, [scale]);
//     // Determines if the subroutine is one you can edit
//     const canUpdate = useMemo<boolean>(() => ((data?.routineVersion?.root as Routine)?.isInternal ?? (data?.routineVersion?.root as Routine)?.you?.canUpdate === true), [data.routineVersion]);

//     const { title } = useMemo(() => getDisplay({ ...data, __typename: "NodeRoutineListItem" }, navigator.languages), [data]);

//     const onAction = useCallback(function onActionCallback(action: SubroutineOperation, subroutine: RoutineListItem, event?: React.MouseEvent<Element>) {
//         if (event) {
//             event.stopPropagation();
//         }
//         handleAction(action, subroutine);
//     }, [handleAction]);

//     // const openSubroutine = useCallback(({ target }: UsePressEvent) => {
//     //     if (!shouldOpen(target.id)) return;
//     //     onAction(null, BuildAction.OpenSubroutine);
//     // }, [onAction]);
//     // const deleteSubroutine = useCallback((event: any) => { onAction(event, BuildAction.DeleteSubroutine); }, [onAction]);

//     const toggleIsOptional = useCallback(function toggleIsOptionalCallback() {
//         if (!isEditing) return;
//         handleAction("update", {
//             ...data,
//             isOptional: !data.isOptional,
//         });
//     }, [isEditing, handleAction, data]);

//     const RoutineTypeIcon = useMemo(() => {
//         const routineType = data?.routineVersion?.routineType;
//         if (!routineType || !routineTypes[routineType]) return null;
//         return routineTypes[routineType].icon;
//     }, [data]);

//     return (
//         <OuterBox nodeSize={nodeSize}>
//             <InnerBox
//                 canUpdate={canUpdate}
//                 id={`subroutine-name-bar-${data.id}`}
//             >
//                 <Tooltip placement={"top"} title='Routine can be skipped'>
//                     <Box
//                         onClick={toggleIsOptional}
//                         onTouchStart={toggleIsOptional}
//                         sx={routineNodeActionStyle(isEditing)}
//                     >
//                         <IconButton
//                             size="small"
//                             sx={isOptionalButtonStyle}
//                             disabled={!isEditing}
//                         >
//                             {data?.isOptional ? <NoActionIcon /> : <ActionIcon />}
//                         </IconButton>
//                     </Box>
//                 </Tooltip>
//                 {RoutineTypeIcon && <RoutineTypeIcon width={16} height={16} fill={palette.background.textPrimary} />}
//                 <NodeTitle3
//                     fontSize={fontSize}
//                     id={`subroutine-name-${data.id}`}
//                     isLabelVisible={isLabelVisible}
//                     variant="h6"
//                 >{firstString(title, "Untitled")}</NodeTitle3>
//                 {/* {isEditing && (
//                         <IconButton
//                             id={`subroutine-delete-icon-button-${data.id}`}
//                             onClick={deleteSubroutine}
//                             onTouchStart={deleteSubroutine}
//                             color="inherit"
//                         >
//                             <CloseIcon id={`subroutine-delete-icon-${data.id}`} />
//                         </IconButton>
//                     )} */}
//             </InnerBox>
//         </OuterBox>
//     );
// }
