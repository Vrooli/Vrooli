import { NodeRoutineListItem } from "@local/shared";
import { Box, Button, Collapse, Container, IconButton, Tooltip, Typography, useTheme } from "@mui/material";
import { useDebounce } from "hooks/useDebounce";
import usePress from "hooks/usePress";
import { ActionIcon, AddIcon, CloseIcon, EditIcon, ExpandLessIcon, ExpandMoreIcon, ListBulletIcon, ListNumberIcon, NoActionIcon } from "icons";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { multiLineEllipsis, noSelect, textShadow } from "styles";
import { BuildAction } from "utils/consts";
import { firstString } from "utils/display/stringTools";
import { getTranslation } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { NodeWithRoutineListCrud } from "views/objects/node/NodeWithRoutineListCrud/NodeWithRoutineListCrud";
import { NodeWithRoutineList } from "views/objects/node/types";
import { DraggableNode, SubroutineNode, calculateNodeSize } from "..";
import { NodeContextMenu, NodeWidth } from "../..";
import { routineNodeActionStyle, routineNodeCheckboxOption } from "../styles";
import { RoutineListNodeProps } from "../types";

/**
 * Distance before a click is considered a drag
 */
const DRAG_THRESHOLD = 10;

/**
 * Decides if a clicked element should trigger a collapse/expand. 
 * @param id ID of the clicked element
 */
const shouldCollapse = (id: string | null | undefined): boolean => {
    // Only collapse if clicked on shrink/expand icon, title bar, or title
    return Boolean(id && (
        id.startsWith("toggle-expand-icon-") ||
        id.startsWith("node-")
    ));
};

export const RoutineListNode = ({
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
}: RoutineListNodeProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Default to open if editing and empty
    const [collapseOpen, setCollapseOpen] = useState<boolean>(isEditing && node.routineList.items?.length === 0);
    const [collapseDebounce] = useDebounce(setCollapseOpen, 100);
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
        const fastSub = PubSub.get().subscribe("fastUpdate", ({ on, duration }) => {
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
        return () => { PubSub.get().unsubscribe(fastSub); };
    }, []);



    const handleNodeUnlink = useCallback(() => { handleAction(BuildAction.UnlinkNode, node.id); }, [handleAction, node.id]);
    const handleNodeDelete = useCallback(() => { handleAction(BuildAction.DeleteNode, node.id); }, [handleAction, node.id]);

    const onOrderedChange = useCallback((checked: boolean) => {
        if (!isEditing) return;
        handleUpdate({
            ...node,
            routineList: { ...node.routineList, isOrdered: checked } as any,
        });
    }, [handleUpdate, isEditing, node]);

    const onOptionalChange = useCallback((checked: boolean) => {
        if (!isEditing) return;
        handleUpdate({
            ...node,
            routineList: { ...node.routineList, isOptional: checked } as any,
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

    const { label } = useMemo(() => {
        return {
            label: getTranslation(node, [language], true).name ?? "",
        };
    }, [language, node]);

    const nodeSize = useMemo(() => `min(max(${calculateNodeSize(NodeWidth.RoutineList, scale) * 2}px, 20vw), 500px)`, [scale]);
    const fontSize = useMemo(() => `min(${calculateNodeSize(NodeWidth.RoutineList, scale) / 8}px, 1.5em)`, [scale]);
    const addSize = useMemo(() => `max(min(${calculateNodeSize(NodeWidth.RoutineList, scale) / 4}px, 48px), 24px)`, [scale]);

    const confirmDelete = useCallback(() => {
        PubSub.get().publish("alertDialog", {
            messageKey: "WhatWouldYouLikeToDo",
            buttons: [
                { labelKey: "Unlink", onClick: handleNodeUnlink },
                { labelKey: "Remove", onClick: handleNodeDelete },
            ],
        });
    }, [handleNodeDelete, handleNodeUnlink]);

    const optionsCollapse = useMemo(() => (
        <Collapse in={collapseOpen} sx={{
            ...noSelect,
            background: palette.mode === "light" ? "#b0bbe7" : "#384164",
        }}>
            <Tooltip placement={"top"} title={t("MustCompleteRoutinesInOrder")}>
                <Box
                    onClick={() => { onOrderedChange(!node.routineList.isOrdered); }}
                    onTouchStart={() => { onOrderedChange(!node.routineList.isOrdered); }}
                    sx={routineNodeActionStyle(isEditing)}
                >
                    <IconButton
                        size="small"
                        sx={{ ...routineNodeCheckboxOption }}
                        disabled={!isEditing}
                    >
                        {node.routineList.isOrdered ? <ListNumberIcon fill={palette.background.textPrimary} /> : <ListBulletIcon fill={palette.background.textPrimary} />}
                    </IconButton>
                    {scale > -0.2 && <Typography sx={{ marginLeft: "4px" }}>{node.routineList.isOrdered ? "Ordered" : "Unordered"}</Typography>}
                </Box>
            </Tooltip>
            <Tooltip placement={"top"} title={t("RoutineCanSkip")}>
                <Box
                    onClick={() => { onOptionalChange(!node.routineList.isOptional); }}
                    onTouchStart={() => { onOptionalChange(!node.routineList.isOptional); }}
                    sx={routineNodeActionStyle(isEditing)}
                >
                    <IconButton
                        size="small"
                        sx={{ ...routineNodeCheckboxOption }}
                        disabled={!isEditing}
                    >
                        {node.routineList.isOptional ? <NoActionIcon fill={palette.background.textPrimary} /> : <ActionIcon fill={palette.background.textPrimary} />}
                    </IconButton>
                    {scale > -0.2 && <Typography sx={{ marginLeft: "4px" }}>{node.routineList.isOptional ? "Optional" : "Required"}</Typography>}
                </Box>
            </Tooltip>
            {isEditing && <Tooltip placement={"top"} title={t("NodeRoutineListInfo")}>
                <Box
                    onClick={() => { openEditDialog(); }}
                    onTouchStart={() => { openEditDialog(); }}
                    sx={routineNodeActionStyle(isEditing)}
                >
                    <IconButton
                        id={`${label ?? ""}-edit-option`}
                        size="small"
                        sx={{
                            ...routineNodeCheckboxOption,
                            "@media print": {
                                display: "none",
                            },
                        }}
                        disabled={!isEditing}
                    >
                        <EditIcon fill={palette.background.textPrimary} />
                    </IconButton>
                    {scale > -0.2 && <Typography sx={{ marginLeft: "4px" }}>{t("Edit")}</Typography>}
                </Box>
            </Tooltip>}
        </Collapse>
    ), [collapseOpen, palette.mode, palette.background.textPrimary, t, isEditing, node.routineList.isOrdered, node.routineList.isOptional, scale, label, onOrderedChange, onOptionalChange, openEditDialog]);

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
            isOpen={collapseOpen}
            labelVisible={labelVisible}
            language={language}
            scale={scale}
        />
    )), [node.routineList.items, handleSubroutineAction, handleSubroutineUpdate, isEditing, collapseOpen, labelVisible, language, scale]);

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

    return (
        <>
            {/* Right-click context menu */}
            <NodeContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                availableActions={[BuildAction.AddListBeforeNode, BuildAction.AddListAfterNode, BuildAction.AddEndAfterNode, BuildAction.MoveNode, BuildAction.UnlinkNode, BuildAction.AddIncomingLink, BuildAction.AddOutgoingLink, BuildAction.DeleteNode, BuildAction.AddSubroutine]}
                handleClose={closeContext}
                handleSelect={(option) => { handleAction(option, node.id); }}
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
            <DraggableNode
                className="handle"
                canDrag={canDrag}
                nodeId={node.id}
                dragThreshold={DRAG_THRESHOLD}
                sx={{
                    zIndex: 5,
                    width: nodeSize,
                    position: "relative",
                    display: "block",
                    borderRadius: "12px",
                    overflow: "hidden",
                    backgroundColor: palette.background.paper,
                    color: palette.background.textPrimary,
                    boxShadow: borderColor ? `0px 0px 12px ${borderColor}` : 12,
                    "@media print": {
                        border: `1px solid ${palette.mode === "light" ? palette.primary.dark : palette.secondary.dark}`,
                        boxShadow: "none",
                    },
                }}
            >
                <>
                    <Tooltip placement={"top"} title={firstString(label, t("RoutineList"))}>
                        <Container
                            id={`${isLinked ? "" : "unlinked-"}node-${node.id}`}
                            aria-owns={contextOpen ? contextId : undefined}
                            {...pressEvents}
                            sx={{
                                display: "flex",
                                height: "48px", // Lighthouse SEO requirement
                                alignItems: "center",
                                backgroundColor: palette.mode === "light" ? palette.primary.dark : palette.secondary.dark,
                                color: palette.mode === "light" ? palette.primary.contrastText : palette.secondary.contrastText,
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
                            }}
                        >
                            {
                                canExpand && scale > -2 && (
                                    <IconButton
                                        id={`toggle-expand-icon-button-${node.id}`}
                                        aria-label={collapseOpen ? "Collapse" : "Expand"}
                                        color="inherit"
                                    >
                                        {collapseOpen ? <ExpandLessIcon id={`toggle-expand-icon-${node.id}`} /> : <ExpandMoreIcon id={`toggle-expand-icon-${node.id}`} />}
                                    </IconButton>
                                )
                            }
                            {labelVisible && <Typography
                                id={`node-routinelist-title-${node.id}`}
                                variant="h6"
                                sx={{
                                    ...noSelect,
                                    ...textShadow,
                                    ...(!collapseOpen ? multiLineEllipsis(1) : multiLineEllipsis(4)),
                                    textAlign: "center",
                                    width: "100%",
                                    lineBreak: "anywhere" as const,
                                    whiteSpace: "pre" as const,
                                    fontSize,
                                    display: scale < -2 ? "none" : "block",
                                    "@media print": {
                                        textShadow: "none",
                                        color: "black",
                                    },
                                }}
                            >{firstString(label, t("Unlinked"))}</Typography>}
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
                        </Container>
                    </Tooltip>
                    {optionsCollapse}
                    <Collapse
                        in={collapseOpen}
                        sx={{
                            padding: collapseOpen ? "0.5em" : "0",
                        }}
                    >
                        {listItems}
                        {isEditing && <Button
                            fullWidth
                            onClick={handleSubroutineAdd}
                            startIcon={<AddIcon />}
                            variant="outlined"
                            sx={{
                                "@media print": {
                                    display: "none",
                                },
                            }}
                        >{t("Add")}</Button>}
                    </Collapse>
                </>
            </DraggableNode>
        </>
    );
};
