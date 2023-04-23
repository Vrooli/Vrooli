import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { CloseIcon } from "@local/icons";
import { name as nameValidation, reqErr } from "@local/validation";
import { Box, Checkbox, Container, FormControlLabel, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { calculateNodeSize } from "..";
import { multiLineEllipsis, noSelect, textShadow } from "../../../../../styles";
import { BuildAction } from "../../../../../utils/consts";
import { getDisplay } from "../../../../../utils/display/listTools";
import { firstString } from "../../../../../utils/display/stringTools";
import { updateTranslationFields } from "../../../../../utils/display/translationTools";
import usePress from "../../../../../utils/hooks/usePress";
import { EditableLabel } from "../../../../inputs/EditableLabel/EditableLabel";
import { NodeContextMenu } from "../../NodeContextMenu/NodeContextMenu";
import { routineNodeCheckboxLabel, routineNodeCheckboxOption } from "../styles";
const shouldOpen = (id) => {
    return Boolean(id && (id.startsWith("subroutine-name-")));
};
export const SubroutineNode = ({ data, scale = 1, labelVisible = true, isOpen, isEditing = true, handleAction, handleUpdate, language, zIndex, }) => {
    const { palette } = useTheme();
    const nodeSize = useMemo(() => `${calculateNodeSize(220, scale)}px`, [scale]);
    const fontSize = useMemo(() => `min(${calculateNodeSize(220, scale) / 5}px, 2em)`, [scale]);
    const canUpdate = useMemo(() => (data?.routineVersion?.root?.isInternal ?? data?.routineVersion?.root?.you?.canUpdate === true), [data.routineVersion]);
    const { title } = useMemo(() => getDisplay(data, navigator.languages), [data]);
    const onAction = useCallback((event, action) => {
        if (event && [BuildAction.EditSubroutine, BuildAction.DeleteSubroutine].includes(action)) {
            event.stopPropagation();
        }
        handleAction(action, data.id);
    }, [data.id, handleAction]);
    const openSubroutine = useCallback((target) => {
        if (!shouldOpen(target.id))
            return;
        onAction(null, BuildAction.OpenSubroutine);
    }, [onAction]);
    const deleteSubroutine = useCallback((event) => { onAction(event, BuildAction.DeleteSubroutine); }, [onAction]);
    const handleLabelUpdate = useCallback((newLabel) => {
        handleUpdate(data.id, {
            ...data,
            translations: updateTranslationFields(data, language, { name: newLabel }),
        });
    }, [handleUpdate, data, language]);
    const onOptionalChange = useCallback((checked) => {
        handleUpdate(data.id, {
            ...data,
            isOptional: checked,
        });
    }, [handleUpdate, data]);
    const labelObject = useMemo(() => {
        if (!labelVisible)
            return null;
        return (_jsx(EditableLabel, { canUpdate: isEditing, handleUpdate: handleLabelUpdate, renderLabel: (t) => (_jsx(Typography, { id: `subroutine-name-${data.id}`, variant: "h6", sx: {
                    ...noSelect,
                    ...textShadow,
                    ...multiLineEllipsis(1),
                    textAlign: "center",
                    width: "100%",
                    lineBreak: "anywhere",
                    whiteSpace: "pre",
                }, children: firstString(t, "Untitled") })), sxs: {
                stack: {
                    marginLeft: "auto",
                    marginRight: "auto",
                },
            }, text: title, validationSchema: nameValidation.required(reqErr), zIndex: zIndex }));
    }, [labelVisible, isEditing, handleLabelUpdate, title, zIndex, data.id]);
    const [contextAnchor, setContextAnchor] = useState(null);
    const contextId = useMemo(() => `subroutine-context-menu-${data.id}`, [data?.id]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((target) => {
        if (!isEditing)
            return;
        setContextAnchor(target);
    }, [isEditing]);
    const closeContext = useCallback(() => { setContextAnchor(null); }, []);
    const pressEvents = usePress({
        onLongPress: openContext,
        onClick: openSubroutine,
        onRightClick: openContext,
    });
    return (_jsxs(_Fragment, { children: [_jsx(NodeContextMenu, { id: contextId, anchorEl: contextAnchor, availableActions: isEditing ?
                    [BuildAction.EditSubroutine, BuildAction.DeleteSubroutine] :
                    [BuildAction.OpenSubroutine, BuildAction.DeleteSubroutine], handleClose: closeContext, handleSelect: (action) => { onAction(null, action); }, zIndex: zIndex + 1 }), _jsxs(Box, { sx: {
                    boxShadow: 12,
                    minWidth: nodeSize,
                    fontSize,
                    position: "relative",
                    display: "block",
                    marginBottom: "8px",
                    borderRadius: "12px",
                    overflow: "hidden",
                    backgroundColor: palette.background.paper,
                    color: palette.background.textPrimary,
                }, children: [_jsxs(Container, { id: `subroutine-name-bar-${data.id}`, ...pressEvents, "aria-owns": contextOpen ? contextId : undefined, sx: {
                            display: "flex",
                            alignItems: "center",
                            backgroundColor: canUpdate ?
                                (palette.mode === "light" ? palette.primary.dark : palette.secondary.dark) :
                                "#667899",
                            color: palette.mode === "light" ? palette.primary.contrastText : palette.secondary.contrastText,
                            padding: "0.1em",
                            textAlign: "center",
                            cursor: "pointer",
                            "&:hover": {
                                filter: "brightness(120%)",
                                transition: "filter 0.2s",
                            },
                        }, children: [labelObject, isEditing && (_jsx(IconButton, { id: `subroutine-delete-icon-button-${data.id}`, onClick: deleteSubroutine, onTouchStart: deleteSubroutine, color: "inherit", children: _jsx(CloseIcon, { id: `subroutine-delete-icon-${data.id}` }) }))] }), _jsx(Stack, { direction: "row", justifyContent: "space-between", borderRadius: 0, sx: { ...noSelect }, children: _jsx(Tooltip, { placement: "top", title: 'Routine can be skipped', children: _jsx(FormControlLabel, { disabled: !isEditing, label: 'Optional', control: _jsx(Checkbox, { id: `${title}-optional-option`, size: "small", name: 'isOptionalCheckbox', color: 'secondary', checked: data?.isOptional, onChange: (_e, checked) => { onOptionalChange(checked); }, onTouchStart: () => { onOptionalChange(!data?.isOptional); }, sx: { ...routineNodeCheckboxOption } }), sx: { ...routineNodeCheckboxLabel } }) }) })] })] }));
};
//# sourceMappingURL=SubroutineNode.js.map