import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { EditIcon } from "@local/icons";
import { DialogContent, DialogContentText, IconButton, Stack, TextField } from "@mui/material";
import { useFormik } from "formik";
import { useCallback, useEffect, useState } from "react";
import * as yup from "yup";
import { GridSubmitButtons } from "../../buttons/GridSubmitButtons/GridSubmitButtons";
import { LargeDialog } from "../../dialogs/LargeDialog/LargeDialog";
import { TopBar } from "../../navigation/TopBar/TopBar";
const titleId = "editable-label-dialog-title";
const descriptionAria = "editable-label-dialog-description";
export const EditableLabel = ({ canUpdate, handleUpdate, placeholder, onDialogOpen, renderLabel, sxs, text, validationSchema, zIndex, }) => {
    const [id] = useState(Math.random().toString(36).substr(2, 9));
    const formik = useFormik({
        initialValues: {
            text,
        },
        enableReinitialize: true,
        validationSchema: validationSchema ? yup.object().shape({
            text: validationSchema,
        }) : undefined,
        onSubmit: (values) => {
            handleUpdate(values.text);
            setActive(false);
            formik.setSubmitting(false);
        },
    });
    const [active, setActive] = useState(false);
    useEffect(() => {
        if (typeof onDialogOpen === "function") {
            onDialogOpen(active);
        }
    }, [active, onDialogOpen]);
    const toggleActive = useCallback((event) => {
        if (!canUpdate)
            return;
        setActive(!active);
    }, [active, canUpdate]);
    const handleCancel = useCallback((_, reason) => {
        if (formik.dirty && reason === "backdropClick")
            return;
        setActive(false);
        formik.resetForm();
    }, [formik]);
    return (_jsxs(_Fragment, { children: [_jsxs(LargeDialog, { id: "edit-label-dialog", onClose: handleCancel, isOpen: active, titleId: titleId, zIndex: zIndex + 1, children: [_jsx(TopBar, { display: "dialog", onClose: handleCancel, titleData: { titleId, titleKey: "EditLabel" } }), _jsx(DialogContent, { sx: { paddingBottom: "80px" }, children: _jsx(DialogContentText, { id: descriptionAria, children: _jsx(TextField, { autoFocus: true, margin: "dense", id: "text", type: "text", fullWidth: true, value: formik.values.text, onChange: formik.handleChange, error: Boolean(formik.errors.text), helperText: formik.errors.text }) }) }), _jsx(GridSubmitButtons, { display: "dialog", errors: formik.errors, isCreate: false, loading: formik.isSubmitting, onCancel: handleCancel, onSetSubmitting: formik.setSubmitting, onSubmit: formik.handleSubmit })] }), _jsxs(Stack, { direction: "row", spacing: 0, alignItems: "center", sx: { ...(sxs?.stack ?? {}) }, children: [renderLabel(text.trim().length > 0 ? text : (placeholder ?? "")), canUpdate && (_jsx(IconButton, { id: `edit-label-icon-button-${id}`, onClick: toggleActive, onTouchStart: toggleActive, sx: { color: "inherit" }, children: _jsx(EditIcon, { id: `edit-label-icon-${id}` }) }))] })] }));
};
//# sourceMappingURL=EditableLabel.js.map