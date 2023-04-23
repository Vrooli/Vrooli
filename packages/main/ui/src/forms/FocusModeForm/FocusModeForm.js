import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { AddIcon, DeleteIcon, EditIcon, HeartFilledIcon, InvisibleIcon } from "@local/icons";
import { Box, Button, ListItem, Stack, TextField, useTheme } from "@mui/material";
import { Field, useField } from "formik";
import { forwardRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { GridSubmitButtons } from "../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { ListContainer } from "../../components/containers/ListContainer/ListContainer";
import { ScheduleDialog } from "../../components/dialogs/ScheduleDialog/ScheduleDialog";
import { ResourceListHorizontalInput } from "../../components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { TagSelector } from "../../components/inputs/TagSelector/TagSelector";
import { Subheader } from "../../components/text/Subheader/Subheader";
import { BaseForm } from "../BaseForm/BaseForm";
export const FocusModeForm = forwardRef(({ display, dirty, isCreate, isLoading, isOpen, onCancel, values, zIndex, ...props }, ref) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [scheduleField, scheduleMeta, scheduleHelpers] = useField("schedule");
    const [isScheduleDialogOpen, setIsScheduleDialogOpen] = useState(false);
    const [editingSchedule, setEditingSchedule] = useState(null);
    const handleAddSchedule = () => { setIsScheduleDialogOpen(true); };
    const handleUpdateSchedule = () => {
        setEditingSchedule(scheduleField.value);
        setIsScheduleDialogOpen(true);
    };
    const handleCloseScheduleDialog = () => { setIsScheduleDialogOpen(false); };
    const handleScheduleCreated = (created) => {
        scheduleHelpers.setValue(created);
        setIsScheduleDialogOpen(false);
    };
    const handleScheduleUpdated = (updated) => {
        scheduleHelpers.setValue(updated);
        setIsScheduleDialogOpen(false);
    };
    const handleDeleteSchedule = () => { scheduleHelpers.setValue(null); };
    return (_jsxs(_Fragment, { children: [_jsx(ScheduleDialog, { isCreate: editingSchedule === null, isMutate: false, isOpen: isScheduleDialogOpen, onClose: handleCloseScheduleDialog, onCreated: handleScheduleCreated, onUpdated: handleScheduleUpdated, zIndex: zIndex + 1 }), _jsx(BaseForm, { dirty: dirty, isLoading: isLoading, ref: ref, style: {
                    display: "block",
                    width: "min(600px, 100vw - 16px)",
                    margin: "auto",
                    paddingLeft: "env(safe-area-inset-left)",
                    paddingRight: "env(safe-area-inset-right)",
                    paddingBottom: "calc(64px + env(safe-area-inset-bottom))",
                }, children: _jsxs(Stack, { direction: "column", spacing: 4, padding: 2, children: [_jsxs(Stack, { direction: "column", spacing: 2, children: [_jsx(Field, { fullWidth: true, name: "name", label: t("Name"), as: TextField }), _jsx(Field, { fullWidth: true, name: "description", label: t("Description"), as: TextField })] }), !scheduleField.value && (_jsx(Button, { onClick: handleAddSchedule, startIcon: _jsx(AddIcon, {}), sx: {
                                display: "flex",
                                margin: "auto",
                            }, children: "Add schedule" })), scheduleField.value && _jsx(ListContainer, { isEmpty: false, children: scheduleField.value && (_jsxs(ListItem, { children: [_jsx(Stack, { direction: "column", spacing: 1, pl: 2, sx: {
                                            width: "-webkit-fill-available",
                                            display: "grid",
                                            pointerEvents: "none",
                                        } }), _jsxs(Stack, { direction: "column", spacing: 1, sx: {
                                            pointerEvents: "none",
                                            justifyContent: "center",
                                            alignItems: "start",
                                        }, children: [_jsx(Box, { component: "a", onClick: handleUpdateSchedule, sx: {
                                                    display: "flex",
                                                    justifyContent: "center",
                                                    alignItems: "center",
                                                    cursor: "pointer",
                                                    pointerEvents: "all",
                                                    paddingBottom: "4px",
                                                }, children: _jsx(EditIcon, { fill: palette.secondary.main }) }), _jsx(Box, { component: "a", onClick: handleDeleteSchedule, sx: {
                                                    display: "flex",
                                                    justifyContent: "center",
                                                    alignItems: "center",
                                                    cursor: "pointer",
                                                    pointerEvents: "all",
                                                    paddingBottom: "4px",
                                                }, children: _jsx(DeleteIcon, { fill: palette.secondary.main }) })] })] })) }), _jsx(ResourceListHorizontalInput, { isCreate: true, zIndex: zIndex }), _jsx(Subheader, { Icon: HeartFilledIcon, title: t("TopicsFavorite"), help: t("TopicsFavoriteHelp") }), _jsx(TagSelector, { name: "favorites" }), _jsx(Subheader, { Icon: InvisibleIcon, title: t("TopicsHidden"), help: t("TopicsHiddenHelp") }), _jsx(TagSelector, { name: "hidden" })] }) }), _jsx(GridSubmitButtons, { display: display, errors: props.errors, isCreate: isCreate, loading: props.isSubmitting, onCancel: onCancel, onSetSubmitting: props.setSubmitting, onSubmit: props.handleSubmit })] }));
});
//# sourceMappingURL=FocusModeForm.js.map