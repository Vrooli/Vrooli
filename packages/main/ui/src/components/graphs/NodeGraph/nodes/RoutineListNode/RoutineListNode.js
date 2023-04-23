import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { AddIcon, CloseIcon, ExpandLessIcon, ExpandMoreIcon } from "@local/icons";
import { name as nameValidation, reqErr } from "@local/validation";
import { Checkbox, Collapse, Container, FormControlLabel, IconButton, Tooltip, Typography, useTheme } from "@mui/material";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { calculateNodeSize, DraggableNode, SubroutineNode } from "..";
import { NodeContextMenu, NodeWidth } from "../..";
import { multiLineEllipsis, noSelect, textShadow } from "../../../../../styles";
import { BuildAction } from "../../../../../utils/consts";
import { firstString } from "../../../../../utils/display/stringTools";
import { getTranslation, updateTranslationFields } from "../../../../../utils/display/translationTools";
import { useDebounce } from "../../../../../utils/hooks/useDebounce";
import usePress from "../../../../../utils/hooks/usePress";
import { PubSub } from "../../../../../utils/pubsub";
import { ColorIconButton } from "../../../../buttons/ColorIconButton/ColorIconButton";
import { EditableLabel } from "../../../../inputs/EditableLabel/EditableLabel";
import { routineNodeCheckboxLabel, routineNodeCheckboxOption } from "../styles";
const DRAG_THRESHOLD = 10;
const shouldCollapse = (id) => {
    return Boolean(id && (id.startsWith("toggle-expand-icon-") ||
        id.startsWith("node-")));
};
export const RoutineListNode = ({ canDrag, canExpand, handleAction, handleUpdate, isLinked, labelVisible, language, linksIn, linksOut, isEditing, node, scale = 1, zIndex, }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [collapseOpen, setCollapseOpen] = useState(isEditing && node.routineList.items?.length === 0);
    const collapseDebounce = useDebounce(setCollapseOpen, 100);
    const toggleCollapse = useCallback((target) => {
        if (isLinked && shouldCollapse(target.id)) {
            PubSub.get().publishFastUpdate({ duration: 1000 });
            collapseDebounce(!collapseOpen);
        }
    }, [collapseDebounce, collapseOpen, isLinked]);
    const fastUpdateRef = useRef(false);
    const fastUpdateTimeout = useRef(null);
    useEffect(() => {
        const fastSub = PubSub.get().subscribeFastUpdate(({ on, duration }) => {
            if (!on) {
                fastUpdateRef.current = false;
                if (fastUpdateTimeout.current)
                    clearTimeout(fastUpdateTimeout.current);
            }
            else {
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
    const handleLabelUpdate = useCallback((newLabel) => {
        handleUpdate({
            ...node,
            translations: updateTranslationFields(node, language, { title: newLabel }),
        });
    }, [handleUpdate, language, node]);
    const onOrderedChange = useCallback((checked) => {
        handleUpdate({
            ...node,
            routineList: { ...node.routineList, isOrdered: checked },
        });
    }, [handleUpdate, node]);
    const onOptionalChange = useCallback((checked) => {
        handleUpdate({
            ...node,
            routineList: { ...node.routineList, isOptional: checked },
        });
    }, [handleUpdate, node]);
    const handleSubroutineAction = useCallback((action, subroutineId) => {
        handleAction(action, node.id, subroutineId);
    }, [handleAction, node.id]);
    const handleSubroutineAdd = useCallback(() => {
        handleAction(BuildAction.AddSubroutine, node.id);
    }, [handleAction, node.id]);
    const handleSubroutineUpdate = useCallback((subroutineId, updatedItem) => {
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
        });
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
    const confirmDelete = useCallback((event) => {
        PubSub.get().publishAlertDialog({
            messageKey: "WhatWouldYouLikeToDo",
            buttons: [
                { labelKey: "Unlink", onClick: handleNodeUnlink },
                { labelKey: "Remove", onClick: handleNodeDelete },
            ],
        });
    }, [handleNodeDelete, handleNodeUnlink]);
    const isLabelDialogOpen = useRef(false);
    const onLabelDialogOpen = useCallback((isOpen) => {
        isLabelDialogOpen.current = isOpen;
    }, []);
    const labelObject = useMemo(() => {
        if (!labelVisible)
            return null;
        return (_jsx(EditableLabel, { canUpdate: isEditing && collapseOpen, handleUpdate: handleLabelUpdate, onDialogOpen: onLabelDialogOpen, renderLabel: (l) => (_jsx(Typography, { id: `node-routinelist-title-${node.id}`, variant: "h6", sx: {
                    ...noSelect,
                    ...textShadow,
                    ...(!collapseOpen ? multiLineEllipsis(1) : multiLineEllipsis(4)),
                    textAlign: "center",
                    width: "100%",
                    lineBreak: "anywhere",
                    whiteSpace: "pre",
                }, children: firstString(l, t("Unlinked")) })), sxs: {
                stack: {
                    marginLeft: "auto",
                    marginRight: "auto",
                },
            }, text: label, validationSchema: nameValidation.required(reqErr), zIndex: zIndex }));
    }, [labelVisible, isEditing, collapseOpen, handleLabelUpdate, onLabelDialogOpen, label, zIndex, node.id, t]);
    const optionsCollapse = useMemo(() => (_jsxs(Collapse, { in: collapseOpen, sx: {
            ...noSelect,
            background: palette.mode === "light" ? "#b0bbe7" : "#384164",
        }, children: [_jsx(Tooltip, { placement: "top", title: t("MustCompleteRoutinesInOrder"), children: _jsx(FormControlLabel, { disabled: !isEditing, label: 'Ordered', control: _jsx(Checkbox, { id: `${label ?? ""}-ordered-option`, size: "small", name: 'isOrderedCheckbox', color: 'secondary', checked: node.routineList.isOrdered, onChange: (_e, checked) => { onOrderedChange(checked); }, onTouchStart: () => { onOrderedChange(!node.routineList.isOrdered); }, sx: { ...routineNodeCheckboxOption } }), sx: { ...routineNodeCheckboxLabel } }) }), _jsx(Tooltip, { placement: "top", title: t("RoutineCanSkip"), children: _jsx(FormControlLabel, { disabled: !isEditing, label: t("Optional"), control: _jsx(Checkbox, { id: `${label ?? ""}-optional-option`, size: "small", name: 'isOptionalCheckbox', color: 'secondary', checked: node.routineList.isOptional, onChange: (_e, checked) => { onOptionalChange(checked); }, onTouchStart: () => { onOptionalChange(!node.routineList.isOptional); }, sx: { ...routineNodeCheckboxOption } }), sx: { ...routineNodeCheckboxLabel } }) })] })), [collapseOpen, palette.mode, t, isEditing, label, node.routineList.isOrdered, node.routineList.isOptional, onOrderedChange, onOptionalChange]);
    const listItems = useMemo(() => [...(node.routineList.items ?? [])].sort((a, b) => a.index - b.index).map(item => (_jsx(SubroutineNode, { data: item, handleAction: handleSubroutineAction, handleUpdate: handleSubroutineUpdate, isEditing: isEditing, isOpen: collapseOpen, labelVisible: labelVisible, language: language, scale: scale, zIndex: zIndex }, `${item.id}`))), [node.routineList.items, handleSubroutineAction, handleSubroutineUpdate, isEditing, collapseOpen, labelVisible, language, scale, zIndex]);
    const borderColor = useMemo(() => {
        if (!isLinked)
            return null;
        if (linksIn.length === 0 || linksOut.length === 0)
            return "red";
        if (listItems.length === 0)
            return "yellow";
        return null;
    }, [isLinked, linksIn.length, linksOut.length, listItems.length]);
    const addButton = useMemo(() => isEditing ? (_jsx(ColorIconButton, { onClick: handleSubroutineAdd, onTouchStart: handleSubroutineAdd, background: '#6daf72', sx: {
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
        }, children: _jsx(AddIcon, {}) })) : null, [addSize, handleSubroutineAdd, isEditing]);
    const [contextAnchor, setContextAnchor] = useState(null);
    const contextId = useMemo(() => `node-context-menu-${node.id}`, [node]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((target) => {
        if (!canDrag || !isLinked || !isEditing || isLabelDialogOpen.current || fastUpdateRef.current)
            return;
        setContextAnchor(target);
    }, [canDrag, isEditing, isLinked, isLabelDialogOpen]);
    const closeContext = useCallback(() => setContextAnchor(null), []);
    const pressEvents = usePress({
        onLongPress: openContext,
        onClick: toggleCollapse,
        onRightClick: openContext,
    });
    return (_jsxs(_Fragment, { children: [_jsx(NodeContextMenu, { id: contextId, anchorEl: contextAnchor, availableActions: [BuildAction.AddListBeforeNode, BuildAction.AddListAfterNode, BuildAction.AddEndAfterNode, BuildAction.MoveNode, BuildAction.UnlinkNode, BuildAction.AddIncomingLink, BuildAction.AddOutgoingLink, BuildAction.DeleteNode, BuildAction.AddSubroutine], handleClose: closeContext, handleSelect: (option) => { handleAction(option, node.id); }, zIndex: zIndex + 1 }), _jsx(DraggableNode, { className: "handle", canDrag: canDrag, nodeId: node.id, dragThreshold: DRAG_THRESHOLD, sx: {
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
                }, children: _jsxs(_Fragment, { children: [_jsx(Tooltip, { placement: "top", title: firstString(label, t("RoutineList")), children: _jsxs(Container, { id: `${isLinked ? "" : "unlinked-"}node-${node.id}`, "aria-owns": contextOpen ? contextId : undefined, ...pressEvents, sx: {
                                    display: "flex",
                                    height: "48px",
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
                                }, children: [canExpand && minNodeSize > 100 && (_jsx(IconButton, { id: `toggle-expand-icon-button-${node.id}`, "aria-label": collapseOpen ? "Collapse" : "Expand", color: "inherit", children: collapseOpen ? _jsx(ExpandLessIcon, { id: `toggle-expand-icon-${node.id}` }) : _jsx(ExpandMoreIcon, { id: `toggle-expand-icon-${node.id}` }) })), labelObject, isEditing && (_jsx(IconButton, { id: `${isLinked ? "" : "unlinked-"}delete-node-icon-${node.id}`, onClick: confirmDelete, onTouchStart: confirmDelete, color: "inherit", children: _jsx(CloseIcon, { id: `delete-node-icon-button-${node.id}` }) }))] }) }), optionsCollapse, _jsxs(Collapse, { in: collapseOpen, sx: {
                                padding: collapseOpen ? "0.5em" : "0",
                            }, children: [listItems, addButton] })] }) })] }));
};
//# sourceMappingURL=RoutineListNode.js.map