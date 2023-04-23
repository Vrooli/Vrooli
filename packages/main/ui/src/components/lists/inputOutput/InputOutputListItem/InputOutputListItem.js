import { jsxs as _jsxs, jsx as _jsx } from "react/jsx-runtime";
import { DeleteIcon, DragIcon, ExpandLessIcon, ExpandMoreIcon } from "@local/icons";
import { uuid } from "@local/uuid";
import { routineVersionInputValidation, routineVersionOutputValidation } from "@local/validation";
import { Box, Checkbox, Collapse, Container, FormControlLabel, Grid, IconButton, Tooltip, Typography, useTheme } from "@mui/material";
import { useFormik } from "formik";
import { forwardRef, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { standardInitialValues } from "../../../../forms/StandardForm/StandardForm";
import { linkColors } from "../../../../styles";
import { getTranslation, getUserLanguages } from "../../../../utils/display/translationTools";
import { SessionContext } from "../../../../utils/SessionContext";
import { updateArray } from "../../../../utils/shape/general";
import { EditableText } from "../../../containers/EditableText/EditableText";
import { StandardInput } from "../../../inputs/standards/StandardInput/StandardInput";
import { StandardVersionSelectSwitch } from "../../../inputs/StandardVersionSelectSwitch/StandardVersionSelectSwitch";
export const InputOutputListItem = forwardRef(({ dragProps, dragHandleProps, isEditing, index, isInput, isOpen, item, handleOpen, handleClose, handleDelete, handleUpdate, language, zIndex, }, ref) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [standardVersion, setStandardVersion] = useState(item.standardVersion ?? standardInitialValues(session, item.standardVersion));
    useEffect(() => {
        setStandardVersion(item.standardVersion ?? standardInitialValues(session, item.standardVersion));
    }, [item, session]);
    const canUpdateStandardVersion = useMemo(() => isEditing && standardVersion.root.isInternal === true, [isEditing, standardVersion.root.isInternal]);
    const getTranslationsUpdate = useCallback((language, translation) => {
        const index = item.translations?.findIndex(t => language === t.language) ?? -1;
        return index >= 0 ? updateArray(item.translations, index, translation) : [...item.translations, translation];
    }, [item.translations]);
    const formik = useFormik({
        initialValues: {
            id: item.id,
            description: getTranslation(item, [language]).description ?? "",
            helpText: getTranslation(item, [language]).helpText ?? "",
            isRequired: true,
            name: item.name ?? "",
        },
        enableReinitialize: true,
        validationSchema: (isInput ? routineVersionInputValidation : routineVersionOutputValidation).create({}),
        onSubmit: (values) => {
            const allTranslations = getTranslationsUpdate(language, {
                id: uuid(),
                language,
                description: values.description,
                helpText: values.helpText,
            });
            handleUpdate(index, {
                ...item,
                name: values.name,
                isRequired: isInput ? values.isRequired : undefined,
                translations: allTranslations,
                standardVersion: !canUpdateStandardVersion ? standardVersion : standardInitialValues(session, item.standardVersion),
            });
        },
    });
    const toggleOpen = useCallback(() => {
        if (isOpen) {
            formik.handleSubmit();
            handleClose(index);
        }
        else
            handleOpen(index);
    }, [isOpen, handleOpen, index, formik, handleClose]);
    const onSwitchChange = useCallback((s) => {
        if (s && s.root.isInternal === false) {
            setStandardVersion(s);
        }
        else {
            setStandardVersion(standardInitialValues(session, item.standardVersion));
        }
    }, [item, session]);
    return (_jsxs(Box, { id: `${isInput ? "input" : "output"}-item-${index}`, sx: {
            zIndex: 1,
            background: "white",
            overflow: "hidden",
            flexGrow: 1,
        }, ref: ref, ...dragProps, children: [_jsxs(Container, { onClick: toggleOpen, sx: {
                    background: palette.primary.main,
                    color: "white",
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "left",
                    overflow: "hidden",
                    height: "48px",
                    textAlign: "center",
                    cursor: "pointer",
                    paddingLeft: "8px !important",
                    paddingRight: "8px !important",
                    "&:hover": {
                        filter: "brightness(120%)",
                        transition: "filter 0.2s",
                    },
                }, children: [_jsx(Tooltip, { placement: "top", title: "Order", children: _jsxs(Typography, { variant: "h6", sx: {
                                margin: "0",
                                marginRight: 1,
                                padding: "0",
                            }, children: [index + 1, "."] }) }), isEditing && (_jsx(Tooltip, { placement: "top", title: `Delete ${isInput ? "input" : "output"}. This will not delete the standard`, children: _jsx(IconButton, { color: "inherit", onClick: () => handleDelete(index), "aria-label": "delete", sx: {
                                height: "fit-content",
                                marginTop: "auto",
                                marginBottom: "auto",
                            }, children: _jsx(DeleteIcon, { fill: "white" }) }) })), !isOpen && (_jsxs(Box, { sx: { display: "flex", alignItems: "center", overflow: "hidden" }, children: [_jsx(Typography, { variant: "h6", sx: {
                                    fontWeight: "bold",
                                    margin: "0",
                                    padding: "0",
                                    paddingRight: "0.5em",
                                    overflow: "hidden",
                                    textOverflow: "ellipsis",
                                    whiteSpace: "nowrap",
                                }, children: formik.values.name }), _jsx(Typography, { variant: "body2", sx: {
                                    margin: "0",
                                    padding: "0",
                                    overflow: "hidden",
                                    textOverflow: "ellipsis",
                                    whiteSpace: "nowrap",
                                }, children: formik.values.description })] })), isEditing && (_jsx(Box, { ...dragHandleProps, sx: { marginLeft: "auto" }, children: _jsx(DragIcon, {}) })), _jsx(IconButton, { sx: { marginLeft: isEditing ? "unset" : "auto" }, children: isOpen ?
                            _jsx(ExpandMoreIcon, { fill: palette.secondary.contrastText }) :
                            _jsx(ExpandLessIcon, { fill: palette.secondary.contrastText }) })] }), _jsx(Collapse, { in: isOpen, sx: {
                    background: palette.background.paper,
                    color: palette.background.textPrimary,
                }, children: _jsxs(Grid, { container: true, spacing: 2, sx: { padding: 1, ...linkColors(palette) }, children: [_jsx(Grid, { item: true, xs: 12, children: _jsx(EditableText, { component: 'TextField', isEditing: isEditing, name: 'name', props: { label: "identifier" } }) }), _jsx(Grid, { item: true, xs: 12, children: _jsx(EditableText, { component: 'TextField', isEditing: isEditing, name: 'description', props: { placeholder: "Short description (optional)" } }) }), _jsx(Grid, { item: true, xs: 12, children: _jsx(EditableText, { component: 'Markdown', isEditing: isEditing, name: 'helpText', props: { placeholder: "Detailed information (optional)" } }) }), _jsx(Grid, { item: true, xs: 12, children: _jsx(StandardVersionSelectSwitch, { disabled: !isEditing, selected: !canUpdateStandardVersion ? {
                                    translations: standardVersion.translations ?? [{ __typename: "StandardVersionTranslation", language: getUserLanguages(session)[0], name: "" }],
                                } : null, onChange: onSwitchChange, zIndex: zIndex }) }), _jsx(Grid, { item: true, xs: 12, children: _jsx(StandardInput, { disabled: !canUpdateStandardVersion, fieldName: "preview", zIndex: zIndex }) }), isInput && _jsx(Grid, { item: true, xs: 12, children: _jsx(Tooltip, { placement: "right", title: 'Is this input mandatory?', children: _jsx(FormControlLabel, { disabled: !isEditing, label: 'Required', control: _jsx(Checkbox, { id: 'routine-info-dialog-is-internal', size: "small", name: 'isRequired', color: 'secondary', checked: formik.values.isRequired, onChange: formik.handleChange, onBlur: (e) => { formik.handleBlur(e); formik.handleSubmit(); } }) }) }) })] }) })] }));
});
//# sourceMappingURL=InputOutputListItem.js.map