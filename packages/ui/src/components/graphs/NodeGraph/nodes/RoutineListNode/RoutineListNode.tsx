import { getTranslation, NodeRoutineListItem } from "@local/shared";
import { Box, Button, Collapse, Container, ContainerProps, IconButton, styled, Tooltip, Typography, TypographyProps, useTheme } from "@mui/material";
import { usePress } from "hooks/gestures";
import { useDebounce } from "hooks/useDebounce";
import { ActionIcon, AddIcon, CloseIcon, EditIcon, ExpandLessIcon, ExpandMoreIcon, ListBulletIcon, ListNumberIcon, NoActionIcon } from "icons";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { multiLineEllipsis, noSelect } from "styles";
import { randomString } from "utils/codes";
import { BuildAction } from "utils/consts";
import { firstString } from "utils/display/stringTools";
import { PubSub } from "utils/pubsub";
import { NodeWithRoutineListCrud } from "views/objects/node/NodeWithRoutineListCrud/NodeWithRoutineListCrud";
import { NodeWithRoutineList } from "views/objects/node/types";
import { calculateNodeSize, DRAG_THRESHOLD, DraggableNode, SHOW_TITLE_ABOVE_SCALE, SubroutineNode } from "..";
import { NodeContextMenu, NodeWidth } from "../..";
import { routineNodeActionStyle, routineNodeCheckboxOption } from "../styles";
import { DraggableNodeProps, RoutineListNodeProps } from "../types";

const COLLAPSE_DEBOUNCE_MS = 100;
const SHOW_BUTTON_LABEL_ABOVE_SCALE = -0.2;

/**
 * Decides if a clicked element should trigger a collapse/expand. 
 * @param id ID of the clicked element
 */
function shouldCollapse(id: string | null | undefined): boolean {
    // Only collapse if clicked on shrink/expand icon, title bar, or title
    return Boolean(id && (
        id.startsWith("toggle-expand-icon-") ||
        id.startsWith("node-")
    ));
}

interface StyledDraggableNodeProps extends Omit<DraggableNodeProps, "borderColor"> {
    borderColor: string | null;
    nodeSize: string;
}

const StyledDraggableNode = styled(DraggableNode, {
    shouldForwardProp: (prop) => prop !== "borderColor" && prop !== "nodeSize",
})<StyledDraggableNodeProps>(({ borderColor, nodeSize, theme }) => ({
    zIndex: 5,
    width: nodeSize,
    position: "relative",
    display: "block",
    borderRadius: "12px",
    overflow: "hidden",
    backgroundColor: theme.palette.background.paper,
    color: theme.palette.background.textPrimary,
    boxShadow: borderColor ? `0px 0px 12px ${borderColor}` : theme.shadows[12],
    "@media print": {
        border: `1px solid ${theme.palette.mode === "light" ? theme.palette.primary.dark : theme.palette.secondary.dark}`,
        boxShadow: "none",
    },
}));

interface DraggableTitleContainerProps extends ContainerProps {
    isEditing: boolean;
}

const DraggableTitleContainer = styled(Container, {
    shouldForwardProp: (prop) => prop !== "isEditing",
})<DraggableTitleContainerProps>(({ isEditing, theme }) => ({
    display: "flex",
    height: "48px", // Lighthouse SEO requirement
    alignItems: "center",
    backgroundColor: theme.palette.mode === "light" ? theme.palette.primary.dark : theme.palette.secondary.dark,
    color: theme.palette.mode === "light" ? theme.palette.primary.contrastText : theme.palette.secondary.contrastText,
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
}));

interface NodeTitleProps extends TypographyProps {
    collapseOpen: boolean;
    fontSize: string;
    scale: number;
}

const NodeTitle = styled(Typography, {
    shouldForwardProp: (prop) => prop !== "collapseOpen" && prop !== "fontSize" && prop !== "scale",
})<NodeTitleProps>(({ collapseOpen, fontSize, scale }) => ({
    ...noSelect,
    // eslint-disable-next-line no-magic-numbers
    ...(!collapseOpen ? multiLineEllipsis(1) : multiLineEllipsis(4)),
    textAlign: "center",
    width: "100%",
    lineBreak: "anywhere" as const,
    whiteSpace: "pre" as const,
    fontSize,
    display: scale < SHOW_TITLE_ABOVE_SCALE ? "none" : "block",
    "@media print": {
        color: "black",
    },
}));

const OptionsBox = styled(Box)(({ theme }) => ({
    background: theme.palette.mode === "light" ? "#b0bbe7" : "#384164",
}));

const isOptionalButtonStyle = { ...routineNodeCheckboxOption };
const isOrderedButtonStyle = { ...routineNodeCheckboxOption };
const editButtonStyle = {
    ...routineNodeCheckboxOption,
    "@media print": {
        display: "none",
    },
} as const;
const addButtonDisplayStyle = {
    "@media print": {
        display: "none",
    },
} as const;
const buttonLabelStyle = { marginLeft: "4px" } as const;
const tooltipStyle = { zIndex: 20000 } as const;
const listContainerStyle = { padding: "0.5em" } as const;

export function RoutineListNode({
    canDrag,
    canExpand,
    handleAction,
    handleDelete,
    handleUpdate,
    isLinked,
    labelVisible,
    language,
    linksIn,
    linksOut,
    isEditing,
    node,
    scale = 1,
}: RoutineListNodeProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Default to open if editing and empty
    const [collapseOpen, setCollapseOpen] = useState<boolean>(isEditing && node.routineList.items?.length === 0);
    const [collapseDebounce] = useDebounce(setCollapseOpen, COLLAPSE_DEBOUNCE_MS);
    const toggleCollapse = useCallback((target: EventTarget) => {
        if (isLinked && shouldCollapse(target.id)) {
            PubSub.get().publish("fastUpdate", { duration: 1000 });
            collapseDebounce(!collapseOpen);
        }
    }, [collapseDebounce, collapseOpen, isLinked]);

    // When fastUpdate is triggered, context menu should never open
    const fastUpdateRef = useRef<boolean>(false);
    const fastUpdateTimeout = useRef<NodeJS.Timeout | null>(null);
    useEffect(() => {
        const unsubscribe = PubSub.get().subscribe("fastUpdate", ({ on, duration }) => {
            if (!on) {
                fastUpdateRef.current = false;
                if (fastUpdateTimeout.current) clearTimeout(fastUpdateTimeout.current);
            } else {
                fastUpdateRef.current = true;
                fastUpdateTimeout.current = setTimeout(() => {
                    fastUpdateRef.current = false;
                }, duration);
            }
        });
        return unsubscribe;
    }, []);

    const handleNodeUnlink = useCallback(() => { handleAction(BuildAction.UnlinkNode, node.id); }, [handleAction, node.id]);
    const handleNodeDelete = useCallback(() => { handleAction(BuildAction.DeleteNode, node.id); }, [handleAction, node.id]);

    const toggleOrdered = useCallback(() => {
        if (!isEditing) return;
        handleUpdate({
            ...node,
            routineList: { ...node.routineList, isOrdered: !node.routineList.isOrdered } as any,
        });
    }, [handleUpdate, isEditing, node]);

    const toggleOptional = useCallback(() => {
        if (!isEditing) return;
        handleUpdate({
            ...node,
            routineList: { ...node.routineList, isOptional: !node.routineList.isOptional } as any,
        });
    }, [handleUpdate, isEditing, node]);

    const [editDialogOpen, setEditDialogOpen] = useState<boolean>(false);
    const openEditDialog = useCallback(() => {
        if (isLinked) {
            setEditDialogOpen(!editDialogOpen);
        }
    }, [isLinked, editDialogOpen]);
    const closeEditDialog = useCallback(() => { setEditDialogOpen(false); }, []);

    const handleSubroutineAction = useCallback((
        action: BuildAction.OpenSubroutine | BuildAction.EditSubroutine | BuildAction.DeleteSubroutine,
        subroutineId: string,
    ) => {
        handleAction(action, node.id, subroutineId);
    }, [handleAction, node.id]);

    // Opens dialog to add a new subroutine, so no suroutineId is passed
    const handleSubroutineAdd = useCallback(() => {
        handleAction(BuildAction.AddSubroutine, node.id);
    }, [handleAction, node.id]);

    const handleSubroutineUpdate = useCallback((subroutineId: string, updatedItem: NodeRoutineListItem) => {
        handleUpdate({
            ...node,
            routineList: {
                ...node.routineList,
                items: node.routineList.items?.map((subroutine) => {
                    if (subroutine.id === subroutineId) {
                        return { ...subroutine, ...updatedItem };
                    }
                    return subroutine;
                }) ?? [],
            },
        } as any);
    }, [handleUpdate, node]);

    const { editButtonId, label } = useMemo(() => {
        const name = getTranslation(node, [language], true).name;
        return {
            editButtonId: name ?? randomString(),
            label: firstString(name, t("RoutineList")),
        };
    }, [language, node, t]);

    const nodeSize = useMemo(() => `min(max(${calculateNodeSize(NodeWidth.RoutineList, scale) * 2}px, 20vw), 500px)`, [scale]);
    const fontSize = useMemo(() => `min(${calculateNodeSize(NodeWidth.RoutineList, scale) / 8}px, 1.5em)`, [scale]);

    const confirmDelete = useCallback(() => {
        PubSub.get().publish("alertDialog", {
            messageKey: "WhatWouldYouLikeToDo",
            buttons: [
                { labelKey: "Unlink", onClick: handleNodeUnlink },
                { labelKey: "Remove", onClick: handleNodeDelete },
            ],
        });
    }, [handleNodeDelete, handleNodeUnlink]);

    /** 
     * Subroutines, sorted from lowest to highest index
     * */
    const listItems = useMemo(() => [...(node.routineList.items ?? [])].sort((a, b) => a.index - b.index).map(item => (
        <SubroutineNode
            key={`${item.id}`}
            data={item as NodeRoutineListItem}
            handleAction={handleSubroutineAction}
            handleUpdate={handleSubroutineUpdate}
            isEditing={isEditing}
            labelVisible={labelVisible}
            language={language}
            scale={scale}
        />
    )), [node.routineList.items, handleSubroutineAction, handleSubroutineUpdate, isEditing, labelVisible, language, scale]);

    /**
     * Border color indicates status of node.
     * Yellow for missing subroutines,
     * Red for not fully connected (missing in or out links)
     */
    const borderColor = useMemo<string | null>(() => {
        if (!isLinked) return null;
        if (linksIn.length === 0 || linksOut.length === 0) return "red";
        if (listItems.length === 0) return "yellow";
        return null;
    }, [isLinked, linksIn.length, linksOut.length, listItems.length]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any | null>(null);
    const contextId = useMemo(() => `node-context-menu-${node.id}`, [node]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((target: EventTarget) => {
        // Ignore if not linked, not editing, or in the middle of an event (drag, collapse, move, etc.)
        if (!canDrag || !isLinked || !isEditing || fastUpdateRef.current) return;
        setContextAnchor(target);
    }, [canDrag, isEditing, isLinked]);
    const closeContext = useCallback(() => setContextAnchor(null), []);
    const pressEvents = usePress({
        onLongPress: openContext,
        onClick: toggleCollapse,
        onRightClick: openContext,
    });

    const handleContextSelect = useCallback(function handleContextSelectCallback(option: BuildAction) {
        handleAction(option, node.id);
    }, [handleAction, node.id]);

    const availableActions = useMemo(() => {
        return isEditing ?
            [BuildAction.AddListBeforeNode, BuildAction.AddListAfterNode, BuildAction.AddEndAfterNode, BuildAction.MoveNode, BuildAction.UnlinkNode, BuildAction.AddIncomingLink, BuildAction.AddOutgoingLink, BuildAction.DeleteNode, BuildAction.AddSubroutine] :
            [];
    }, [isEditing]);

    return (
        <>
            {/* Right-click context menu */}
            <NodeContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                availableActions={availableActions}
                handleClose={closeContext}
                handleSelect={handleContextSelect}
            />
            {/* Normal-click menu */}
            <NodeWithRoutineListCrud
                display="dialog"
                onCancel={closeEditDialog}
                onClose={closeEditDialog}
                onCompleted={handleUpdate}
                onDeleted={handleDelete}
                isCreate={false}
                isEditing={isEditing}
                isOpen={editDialogOpen}
                language={language}
                overrideObject={node as NodeWithRoutineList}
            />
            <StyledDraggableNode
                // @ts-ignore We're overriding borderColor
                borderColor={borderColor}
                className="handle"
                canDrag={canDrag}
                nodeId={node.id}
                nodeSize={nodeSize}
                dragThreshold={DRAG_THRESHOLD}
            >
                <>
                    <DraggableTitleContainer
                        id={`${isLinked ? "" : "unlinked-"}node-${node.id}`}
                        isEditing={isEditing}
                        aria-owns={contextOpen ? contextId : undefined}
                        {...pressEvents}
                    >
                        {
                            canExpand && scale > SHOW_TITLE_ABOVE_SCALE && (
                                <IconButton
                                    id={`toggle-expand-icon-button-${node.id}`}
                                    aria-label={collapseOpen ? "Collapse" : "Expand"}
                                    color="inherit"
                                >
                                    {collapseOpen ? <ExpandLessIcon id={`toggle-expand-icon-${node.id}`} /> : <ExpandMoreIcon id={`toggle-expand-icon-${node.id}`} />}
                                </IconButton>
                            )
                        }
                        {labelVisible && <NodeTitle
                            id={`node-routinelist-title-${node.id}`}
                            collapseOpen={collapseOpen}
                            fontSize={fontSize}
                            scale={scale}
                            variant="h6"
                        >{label}</NodeTitle>}
                        {
                            isEditing && (
                                <IconButton
                                    id={`${isLinked ? "" : "unlinked-"}delete-node-icon-${node.id}`}
                                    onClick={confirmDelete}
                                    onTouchStart={confirmDelete}
                                    color="inherit"
                                >
                                    <CloseIcon id={`delete-node-icon-button-${node.id}`} />
                                </IconButton>
                            )
                        }
                    </DraggableTitleContainer>
                    <Collapse in={collapseOpen}>
                        <OptionsBox>
                            <Tooltip placement={"top"} title={t("MustCompleteRoutinesInOrder")} sx={tooltipStyle}>
                                <Box
                                    onClick={toggleOrdered}
                                    onTouchStart={toggleOrdered}
                                    sx={routineNodeActionStyle(isEditing)}
                                >
                                    <IconButton
                                        size="small"
                                        sx={isOrderedButtonStyle}
                                        disabled={!isEditing}
                                    >
                                        {node.routineList.isOrdered ? <ListNumberIcon fill={palette.background.textPrimary} /> : <ListBulletIcon fill={palette.background.textPrimary} />}
                                    </IconButton>
                                    {scale > SHOW_BUTTON_LABEL_ABOVE_SCALE && <Typography sx={buttonLabelStyle}>{node.routineList.isOrdered ? "Ordered" : "Unordered"}</Typography>}
                                </Box>
                            </Tooltip>
                            <Tooltip placement={"top"} title={t("RoutineCanSkip")} sx={tooltipStyle}>
                                <Box
                                    onClick={toggleOptional}
                                    onTouchStart={toggleOptional}
                                    sx={routineNodeActionStyle(isEditing)}
                                >
                                    <IconButton
                                        size="small"
                                        sx={isOptionalButtonStyle}
                                        disabled={!isEditing}
                                    >
                                        {node.routineList.isOptional ? <NoActionIcon fill={palette.background.textPrimary} /> : <ActionIcon fill={palette.background.textPrimary} />}
                                    </IconButton>
                                    {scale > SHOW_BUTTON_LABEL_ABOVE_SCALE && <Typography sx={buttonLabelStyle}>{node.routineList.isOptional ? "Optional" : "Required"}</Typography>}
                                </Box>
                            </Tooltip>
                            {isEditing && <Tooltip placement={"top"} title={t("NodeRoutineListInfo")} sx={tooltipStyle}>
                                <Box
                                    onClick={openEditDialog}
                                    onTouchStart={openEditDialog}
                                    sx={routineNodeActionStyle(isEditing)}
                                >
                                    <IconButton
                                        id={`${editButtonId}-edit-option`}
                                        size="small"
                                        sx={editButtonStyle}
                                        disabled={!isEditing}
                                    >
                                        <EditIcon fill={palette.background.textPrimary} />
                                    </IconButton>
                                    {scale > SHOW_BUTTON_LABEL_ABOVE_SCALE && <Typography sx={buttonLabelStyle}>{t("Edit")}</Typography>}
                                </Box>
                            </Tooltip>}
                        </OptionsBox>
                        <Box sx={listContainerStyle}>
                            {listItems}
                            {isEditing && <Button
                                fullWidth
                                onClick={handleSubroutineAdd}
                                startIcon={<AddIcon />}
                                variant="outlined"
                                sx={addButtonDisplayStyle}
                            >{t("Add")}</Button>}
                        </Box>
                    </Collapse>
                </>
            </StyledDraggableNode>
        </>
    );
}
