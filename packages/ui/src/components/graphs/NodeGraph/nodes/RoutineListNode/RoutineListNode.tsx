import { AddIcon, CloseIcon, ExpandLessIcon, ExpandMoreIcon, name as nameValidation, NodeRoutineListItem, reqErr } from "@local/shared";
import {
    Checkbox,
    Collapse,
    Container,
    FormControlLabel,
    IconButton,
    Tooltip,
    Typography,
    useTheme
} from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { EditableLabel } from "components/inputs/EditableLabel/EditableLabel";
import { CSSProperties, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { multiLineEllipsis, noSelect, textShadow } from "styles";
import { BuildAction } from "utils/consts";
import { firstString } from "utils/display/stringTools";
import { getTranslation, updateTranslationFields } from "utils/display/translationTools";
import { useDebounce } from "utils/hooks/useDebounce";
import usePress from "utils/hooks/usePress";
import { PubSub } from "utils/pubsub";
import { calculateNodeSize, DraggableNode, SubroutineNode } from "..";
import { NodeContextMenu, NodeWidth } from "../..";
import {
    routineNodeCheckboxLabel, routineNodeCheckboxOption
} from "../styles";
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
    handleUpdate,
    isLinked,
    labelVisible,
    language,
    linksIn,
    linksOut,
    isEditing,
    node,
    scale = 1,
    zIndex,
}: RoutineListNodeProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Default to open if editing and empty
    const [collapseOpen, setCollapseOpen] = useState<boolean>(isEditing && node.routineList.items?.length === 0);
    const collapseDebounce = useDebounce(setCollapseOpen, 100);
    const toggleCollapse = useCallback((target: EventTarget) => {
        if (isLinked && shouldCollapse(target.id)) {
            PubSub.get().publishFastUpdate({ duration: 1000 });
            collapseDebounce(!collapseOpen);
        }
    }, [collapseDebounce, collapseOpen, isLinked]);

    // When fastUpdate is triggered, context menu should never open
    const fastUpdateRef = useRef<boolean>(false);
    const fastUpdateTimeout = useRef<NodeJS.Timeout | null>(null);
    useEffect(() => {
        const fastSub = PubSub.get().subscribeFastUpdate(({ on, duration }) => {
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

    const handleLabelUpdate = useCallback((newLabel: string) => {
        handleUpdate({
            ...node,
            translations: updateTranslationFields(node, language, { title: newLabel }),
        });
    }, [handleUpdate, language, node]);

    const onOrderedChange = useCallback((checked: boolean) => {
        handleUpdate({
            ...node,
            routineList: { ...node.routineList, isOrdered: checked } as any,
        });
    }, [handleUpdate, node]);

    const onOptionalChange = useCallback((checked: boolean) => {
        handleUpdate({
            ...node,
            routineList: { ...node.routineList, isOptional: checked } as any,
        });
    }, [handleUpdate, node]);

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

    const minNodeSize = useMemo(() => calculateNodeSize(NodeWidth.RoutineList, scale), [scale]);
    const maxNodeSize = useMemo(() => calculateNodeSize(NodeWidth.RoutineList, scale) * 2, [scale]);
    const fontSize = useMemo(() => `min(${calculateNodeSize(NodeWidth.RoutineList, scale) / 5}px, 2em)`, [scale]);
    const addSize = useMemo(() => `max(${calculateNodeSize(NodeWidth.RoutineList, scale) / 8}px, 48px)`, [scale]);

    const confirmDelete = useCallback((event: any) => {
        PubSub.get().publishAlertDialog({
            messageKey: "WhatWouldYouLikeToDo",
            buttons: [
                { labelKey: "Unlink", onClick: handleNodeUnlink },
                { labelKey: "Remove", onClick: handleNodeDelete },
            ],
        });
    }, [handleNodeDelete, handleNodeUnlink]);

    const isLabelDialogOpen = useRef<boolean>(false);
    const onLabelDialogOpen = useCallback((isOpen: boolean) => {
        isLabelDialogOpen.current = isOpen;
    }, []);
    const labelObject = useMemo(() => {
        if (!labelVisible) return null;
        return (
            <EditableLabel
                canUpdate={isEditing && collapseOpen}
                handleUpdate={handleLabelUpdate}
                onDialogOpen={onLabelDialogOpen}
                renderLabel={(l) => (
                    <Typography
                        id={`node-routinelist-title-${node.id}`}
                        variant="h6"
                        sx={{
                            ...noSelect,
                            ...textShadow,
                            ...(!collapseOpen ? multiLineEllipsis(1) : multiLineEllipsis(4)),
                            textAlign: "center",
                            width: "100%",
                            lineBreak: "anywhere" as any,
                            whiteSpace: "pre" as any,
                        } as CSSProperties}
                    >{firstString(l, t("Unlinked"))}</Typography>
                )}
                sxs={{
                    stack: {
                        marginLeft: "auto",
                        marginRight: "auto",
                    },
                }}
                text={label}
                validationSchema={nameValidation.required(reqErr)}
                zIndex={zIndex}
            />
        );
    }, [labelVisible, isEditing, collapseOpen, handleLabelUpdate, onLabelDialogOpen, label, zIndex, node.id, t]);

    const optionsCollapse = useMemo(() => (
        <Collapse in={collapseOpen} sx={{
            ...noSelect,
            background: palette.mode === "light" ? "#b0bbe7" : "#384164",
        }}>
            <Tooltip placement={"top"} title={t("MustCompleteRoutinesInOrder")}>
                <FormControlLabel
                    disabled={!isEditing}
                    label='Ordered'
                    control={
                        <Checkbox
                            id={`${label ?? ""}-ordered-option`}
                            size="small"
                            name='isOrderedCheckbox'
                            color='secondary'
                            checked={node.routineList.isOrdered}
                            onChange={(_e, checked) => { onOrderedChange(checked); }}
                            onTouchStart={() => { onOrderedChange(!node.routineList.isOrdered); }}
                            sx={{ ...routineNodeCheckboxOption }}
                        />
                    }
                    sx={{ ...routineNodeCheckboxLabel }}
                />
            </Tooltip>
            <Tooltip placement={"top"} title={t("RoutineCanSkip")}>
                <FormControlLabel
                    disabled={!isEditing}
                    label={t("Optional")}
                    control={
                        <Checkbox
                            id={`${label ?? ""}-optional-option`}
                            size="small"
                            name='isOptionalCheckbox'
                            color='secondary'
                            checked={node.routineList.isOptional}
                            onChange={(_e, checked) => { onOptionalChange(checked); }}
                            onTouchStart={() => { onOptionalChange(!node.routineList.isOptional); }}
                            sx={{ ...routineNodeCheckboxOption }}
                        />
                    }
                    sx={{ ...routineNodeCheckboxLabel }}
                />
            </Tooltip>
        </Collapse>
    ), [collapseOpen, palette.mode, t, isEditing, label, node.routineList.isOrdered, node.routineList.isOptional, onOrderedChange, onOptionalChange]);

    /** 
     * Subroutines, sorted from lowest to highest index
     * */
    const listItems = useMemo(() => [...(node.routineList.items ?? [])].sort((a, b) => a.index - b.index).map(item => (
        <SubroutineNode
            key={`${item.id}`}
            data={item}
            handleAction={handleSubroutineAction}
            handleUpdate={handleSubroutineUpdate}
            isEditing={isEditing}
            isOpen={collapseOpen}
            labelVisible={labelVisible}
            language={language}
            scale={scale}
            zIndex={zIndex}
        />
    )), [node.routineList.items, handleSubroutineAction, handleSubroutineUpdate, isEditing, collapseOpen, labelVisible, language, scale, zIndex]);

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


    const addButton = useMemo(() => isEditing ? (
        <ColorIconButton
            onClick={handleSubroutineAdd}
            onTouchStart={handleSubroutineAdd}
            background='#6daf72'
            sx={{
                boxShadow: 12,
                width: addSize,
                height: addSize,
                position: "relative",
                padding: "0",
                margin: "5px auto",
                display: "flex",
                alignItems: "center",
                color: "white",
                borderRadius: "100%",
            }}
        >
            <AddIcon />
        </ColorIconButton>
    ) : null, [addSize, handleSubroutineAdd, isEditing]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const contextId = useMemo(() => `node-context-menu-${node.id}`, [node]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((target: EventTarget) => {
        // Ignore if not linked, not editing, or in the middle of an event (drag, collapse, move, etc.)
        if (!canDrag || !isLinked || !isEditing || isLabelDialogOpen.current || fastUpdateRef.current) return;
        setContextAnchor(target);
    }, [canDrag, isEditing, isLinked, isLabelDialogOpen]);
    const closeContext = useCallback(() => setContextAnchor(null), []);
    const pressEvents = usePress({
        onLongPress: openContext,
        onClick: toggleCollapse,
        onRightClick: openContext,
    });

    return (
        <>
            <NodeContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                availableActions={[BuildAction.AddListBeforeNode, BuildAction.AddListAfterNode, BuildAction.AddEndAfterNode, BuildAction.MoveNode, BuildAction.UnlinkNode, BuildAction.AddIncomingLink, BuildAction.AddOutgoingLink, BuildAction.DeleteNode, BuildAction.AddSubroutine]}
                handleClose={closeContext}
                handleSelect={(option) => { handleAction(option, node.id); }}
                zIndex={zIndex + 1}
            />
            <DraggableNode
                className="handle"
                canDrag={canDrag}
                nodeId={node.id}
                dragThreshold={DRAG_THRESHOLD}
                sx={{
                    zIndex: 5,
                    minWidth: `${minNodeSize}px`,
                    maxWidth: collapseOpen ? `${maxNodeSize}px` : `${minNodeSize}px`,
                    fontSize,
                    position: "relative",
                    display: "block",
                    borderRadius: "12px",
                    overflow: "hidden",
                    backgroundColor: palette.background.paper,
                    color: palette.background.textPrimary,
                    boxShadow: borderColor ? `0px 0px 12px ${borderColor}` : 12,
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
                                canExpand && minNodeSize > 100 && (
                                    <IconButton
                                        id={`toggle-expand-icon-button-${node.id}`}
                                        aria-label={collapseOpen ? "Collapse" : "Expand"}
                                        color="inherit"
                                    >
                                        {collapseOpen ? <ExpandLessIcon id={`toggle-expand-icon-${node.id}`} /> : <ExpandMoreIcon id={`toggle-expand-icon-${node.id}`} />}
                                    </IconButton>
                                )
                            }
                            {labelObject}
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
                        {addButton}
                    </Collapse>
                </>
            </DraggableNode>
        </>
    );
};
