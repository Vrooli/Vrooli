import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { DeleteType, FocusModeStopCondition, LINKS, MaxObjects } from "@local/consts";
import { AddIcon, DeleteIcon, EditIcon } from "@local/icons";
import { Box, IconButton, ListItem, ListItemText, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { t } from "i18next";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { mutationWrapper } from "../../../api";
import { deleteOneOrManyDeleteOne } from "../../../api/generated/endpoints/deleteOneOrMany_deleteOne";
import { useCustomMutation } from "../../../api/hooks";
import { ListContainer } from "../../../components/containers/ListContainer/ListContainer";
import { FocusModeDialog } from "../../../components/dialogs/FocusModeDialog/FocusModeDialog";
import { SettingsList } from "../../../components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "../../../components/navigation/SettingsTopBar/SettingsTopBar";
import { multiLineEllipsis } from "../../../styles";
import { getCurrentUser } from "../../../utils/authentication/session";
import { PubSub } from "../../../utils/pubsub";
import { useLocation } from "../../../utils/route";
import { SessionContext } from "../../../utils/SessionContext";
export const SettingsFocusModesView = ({ display = "page", }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const [focusModes, setFocusModes] = useState([]);
    useEffect(() => {
        if (session) {
            setFocusModes(getCurrentUser(session).focusModes ?? []);
        }
    }, [session]);
    const { canAdd, hasPremium } = useMemo(() => {
        const { hasPremium } = getCurrentUser(session);
        const max = hasPremium ? MaxObjects.FocusMode.User.premium : MaxObjects.FocusMode.User.noPremium;
        return { canAdd: focusModes.length < max, hasPremium };
    }, [focusModes.length, session]);
    const [deleteMutation] = useCustomMutation(deleteOneOrManyDeleteOne);
    const handleDelete = useCallback((focusMode) => {
        if (focusModes.length === 1) {
            PubSub.get().publishSnack({ messageKey: "MustHaveFocusMode", severity: "Error" });
            return;
        }
        PubSub.get().publishAlertDialog({
            messageKey: "DeleteFocusModeConfirm",
            buttons: [
                {
                    labelKey: "Yes", onClick: () => {
                        mutationWrapper({
                            mutation: deleteMutation,
                            input: { id: focusMode.id, objectType: DeleteType.FocusMode },
                            successCondition: (data) => data.success,
                            successMessage: () => ({ key: "FocusModeDeleted" }),
                            onSuccess: () => {
                                setFocusModes((prevFocusModes) => prevFocusModes.filter((prevFocusMode) => prevFocusMode.id !== focusMode.id));
                            },
                        });
                    },
                },
                { labelKey: "Cancel" },
            ],
        });
    }, [deleteMutation, focusModes.length]);
    const [isDialogOpen, setIsDialogOpen] = useState(false);
    const [editingFocusMode, setEditingFocusMode] = useState(null);
    const handleAdd = () => {
        if (!canAdd) {
            if (!hasPremium) {
                setLocation(LINKS.Premium);
                PubSub.get().publishSnack({ message: "Upgrade to increase limit", severity: "Info" });
            }
            else
                PubSub.get().publishSnack({ message: "Max reached", severity: "Error" });
            return;
        }
        setIsDialogOpen(true);
    };
    const handleUpdate = (focusMode) => {
        setEditingFocusMode(focusMode);
        setIsDialogOpen(true);
    };
    const handleCloseDialog = () => {
        setIsDialogOpen(false);
    };
    const handleCreated = (newFocusMode) => {
        setIsDialogOpen(false);
        PubSub.get().publishFocusMode({
            __typename: "ActiveFocusMode",
            mode: newFocusMode,
            stopCondition: FocusModeStopCondition.NextBegins,
        });
    };
    return (_jsxs(_Fragment, { children: [_jsx(FocusModeDialog, { isCreate: editingFocusMode === null, isOpen: isDialogOpen, onClose: handleCloseDialog, onCreated: handleCreated, onUpdated: () => { }, zIndex: 1000 }), _jsx(SettingsTopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "FocusMode",
                    titleVariables: { count: 2 },
                } }), _jsxs(Stack, { direction: "row", children: [_jsx(SettingsList, {}), _jsxs(Stack, { direction: "column", sx: {
                            margin: "auto",
                            display: "block",
                        }, children: [_jsxs(Stack, { direction: "row", alignItems: "center", justifyContent: "center", sx: { paddingTop: 2 }, children: [_jsx(Typography, { component: "h2", variant: "h4", children: t("FocusMode", { count: 2 }) }), _jsx(Tooltip, { title: canAdd ? "Add new" : "Max focus modes reached. Upgrade to premium to add more, or edit/delete an existing focus mode.", placement: "top", children: _jsx(IconButton, { size: "medium", onClick: handleAdd, sx: {
                                                padding: 1,
                                            }, children: _jsx(AddIcon, { fill: canAdd ? palette.secondary.main : palette.grey[500], width: '1.5em', height: '1.5em' }) }) })] }), _jsx(ListContainer, { emptyText: t("NoFocusModes", { ns: "error" }), isEmpty: focusModes.length === 0, sx: { maxWidth: "500px" }, children: focusModes.map((focusMode) => (_jsxs(ListItem, { children: [_jsxs(Stack, { direction: "column", spacing: 1, pl: 2, sx: {
                                                width: "-webkit-fill-available",
                                                display: "grid",
                                                pointerEvents: "none",
                                            }, children: [_jsx(ListItemText, { primary: focusMode.name, sx: {
                                                        ...multiLineEllipsis(1),
                                                        lineBreak: "anywhere",
                                                        pointerEvents: "none",
                                                    } }), _jsx(ListItemText, { primary: focusMode.description, sx: { ...multiLineEllipsis(2), color: palette.text.secondary, pointerEvents: "none" } })] }), _jsxs(Stack, { direction: "column", spacing: 1, sx: {
                                                pointerEvents: "none",
                                                justifyContent: "center",
                                                alignItems: "start",
                                            }, children: [_jsx(Box, { component: "a", onClick: () => handleUpdate(focusMode), sx: {
                                                        display: "flex",
                                                        justifyContent: "center",
                                                        alignItems: "center",
                                                        cursor: "pointer",
                                                        pointerEvents: "all",
                                                        paddingBottom: "4px",
                                                    }, children: _jsx(EditIcon, { fill: palette.secondary.main }) }), focusModes.length > 1 && _jsx(Box, { component: "a", onClick: () => handleDelete(focusMode), sx: {
                                                        display: "flex",
                                                        justifyContent: "center",
                                                        alignItems: "center",
                                                        cursor: "pointer",
                                                        pointerEvents: "all",
                                                        paddingBottom: "4px",
                                                    }, children: _jsx(DeleteIcon, { fill: palette.secondary.main }) })] })] }, focusMode.id))) })] })] })] }));
};
//# sourceMappingURL=SettingsFocusModesView.js.map