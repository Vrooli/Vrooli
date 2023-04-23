import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { CloseIcon, OpenInNewIcon } from "@local/icons";
import { exists, orDefault } from "@local/utils";
import { DUMMY_ID, uuid } from "@local/uuid";
import { nodeRoutineListItemValidation, routineVersionTranslationValidation } from "@local/validation";
import { Box, Grid, IconButton, Tooltip, Typography, useTheme } from "@mui/material";
import { useField } from "formik";
import { forwardRef, useCallback, useContext } from "react";
import { useTranslation } from "react-i18next";
import { GridSubmitButtons } from "../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { EditableText } from "../../components/containers/EditableText/EditableText";
import { EditableTextCollapse } from "../../components/containers/EditableTextCollapse/EditableTextCollapse";
import { SelectLanguageMenu } from "../../components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { IntegerInput } from "../../components/inputs/IntegerInput/IntegerInput";
import { LanguageInput } from "../../components/inputs/LanguageInput/LanguageInput";
import { TagSelector } from "../../components/inputs/TagSelector/TagSelector";
import { TranslatedTextField } from "../../components/inputs/TranslatedTextField/TranslatedTextField";
import { VersionInput } from "../../components/inputs/VersionInput/VersionInput";
import { InputOutputContainer } from "../../components/lists/inputOutput";
import { RelationshipList } from "../../components/lists/RelationshipList/RelationshipList";
import { ResourceListHorizontal } from "../../components/lists/resource";
import { TagList } from "../../components/lists/TagList/TagList";
import { VersionDisplay } from "../../components/text/VersionDisplay/VersionDisplay";
import { combineErrorsWithTranslations, getUserLanguages } from "../../utils/display/translationTools";
import { useTranslatedFields } from "../../utils/hooks/useTranslatedFields";
import { SessionContext } from "../../utils/SessionContext";
import { validateAndGetYupErrors } from "../../utils/shape/general";
import { shapeNodeRoutineListItem } from "../../utils/shape/models/nodeRoutineListItem";
import { BaseForm } from "../BaseForm/BaseForm";
import { routineInitialValues } from "../RoutineForm/RoutineForm";
export const subroutineInitialValues = (session, existing) => ({
    __typename: "NodeRoutineListItem",
    id: uuid(),
    index: 0,
    isOptional: false,
    list: {},
    routineVersion: routineInitialValues(session, existing?.routineVersion),
    ...existing,
    translations: orDefault(existing?.translations, [{
            __typename: "NodeRoutineListItemTranslation",
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            description: "",
            name: "",
        }]),
});
export const transformSubroutineValues = (values, existing) => {
    return existing === undefined
        ? shapeNodeRoutineListItem.create(values)
        : shapeNodeRoutineListItem.update(existing, values);
};
export const validateSubroutineValues = async (values, existing) => {
    const transformedValues = transformSubroutineValues(values, existing);
    const validationSchema = nodeRoutineListItemValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};
export const SubroutineForm = forwardRef(({ canUpdate, dirty, handleViewFull, isCreate, isEditing, isOpen, onCancel, values, versions, zIndex, ...props }, ref) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const { handleAddLanguage, handleDeleteLanguage, language, languages, setLanguage, translationErrors, } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["description", "instructions", "name"],
        validationSchema: routineVersionTranslationValidation[isCreate ? "create" : "update"]({}),
    });
    const [indexField] = useField("index");
    const [isInternalField] = useField("routineVersion.root.isInternal");
    const [listField] = useField("list");
    const [inputsField, , inputsHelpers] = useField("routineVersion.inputs");
    const [outputsField, , outputsHelpers] = useField("routineVersion.outputs");
    const [resourceListField, , resourceListHelpers] = useField("routineVersion.resourceList");
    const [tagsField] = useField("routineVersion.root.tags");
    const [versionlabelField] = useField("routineVersion.versionLabel");
    const [versionsField] = useField("routineVersion.root.versions");
    const toGraph = useCallback(() => {
        handleViewFull();
    }, [handleViewFull]);
    return (_jsxs(_Fragment, { children: [_jsxs(Box, { sx: {
                    display: "flex",
                    flexDirection: "row",
                    alignItems: "center",
                    background: palette.primary.dark,
                    color: palette.primary.contrastText,
                    padding: 1,
                }, children: [_jsx(EditableText, { component: "TranslatedTextField", isEditing: isEditing, name: "name", props: { language }, variant: "h5" }), _jsx(Typography, { variant: "h6", ml: 1, mr: 1, children: `(${(indexField.value ?? 0) + 1} of ${(listField.value?.items?.length ?? 1)})` }), _jsx(VersionDisplay, { currentVersion: { versionLabel: versionlabelField.value }, prefix: " - ", versions: versionsField.value ?? [] }), !isInternalField.value && (_jsx(Tooltip, { title: "Open in full page", children: _jsx(IconButton, { onClick: toGraph, children: _jsx(OpenInNewIcon, { fill: palette.primary.contrastText }) }) })), _jsx(IconButton, { onClick: onCancel, sx: {
                            color: palette.primary.contrastText,
                            borderBottom: `1px solid ${palette.primary.dark}`,
                            justifyContent: "end",
                            marginLeft: "auto",
                        }, children: _jsx(CloseIcon, {}) })] }), _jsxs(BaseForm, { dirty: dirty, isLoading: false, ref: ref, style: {
                    display: "block",
                    width: "min(700px, 100vw - 16px)",
                    margin: "auto",
                    paddingLeft: "env(safe-area-inset-left)",
                    paddingRight: "env(safe-area-inset-right)",
                    paddingBottom: "calc(64px + env(safe-area-inset-bottom))",
                }, children: [_jsxs(Grid, { container: true, spacing: 2, children: [_jsx(Grid, { item: true, xs: 12, children: _jsx(RelationshipList, { isEditing: isEditing, isFormDirty: dirty, objectType: "Routine", zIndex: zIndex }) }), _jsx(Grid, { item: true, xs: 12, children: canUpdate ? _jsx(LanguageInput, { currentLanguage: language, handleAdd: handleAddLanguage, handleDelete: handleDeleteLanguage, handleCurrent: setLanguage, languages: languages, zIndex: zIndex }) : _jsx(SelectLanguageMenu, { currentLanguage: language, handleCurrent: setLanguage, languages: languages, zIndex: zIndex }) }), _jsx(Grid, { item: true, xs: 12, children: canUpdate && _jsxs(Box, { sx: {
                                        marginBottom: 2,
                                    }, children: [_jsx(Typography, { variant: "h6", children: t("Order") }), _jsx(IntegerInput, { disabled: !canUpdate, label: t("Order"), min: 1, max: listField.value?.items?.length ?? 1, name: "index", tooltip: "The order of this subroutine in its parent routine" })] }) }), canUpdate && _jsx(Grid, { item: true, xs: 12, children: _jsx(TranslatedTextField, { fullWidth: true, label: t("Name"), language: language, name: "name" }) }), _jsx(Grid, { item: true, xs: 12, sm: 6, children: _jsx(EditableTextCollapse, { component: 'TranslatedTextField', isEditing: canUpdate, name: "description", props: {
                                        fullWidth: true,
                                        language,
                                        multiline: true,
                                        maxRows: 3,
                                    }, title: t("Description") }) }), _jsx(Grid, { item: true, xs: 12, sm: 6, children: _jsx(EditableTextCollapse, { component: 'TranslatedMarkdown', isEditing: isEditing, name: "instructions", props: {
                                        language,
                                        placeholder: "Instructions",
                                        minRows: 3,
                                    }, title: t("Instructions") }) }), canUpdate && _jsx(Grid, { item: true, xs: 12, children: _jsx(VersionInput, { fullWidth: true, name: "routineVersion.versionLabel", versions: (versionsField.value ?? []).map(v => v.versionLabel) }) }), (canUpdate || (inputsField.value?.length > 0)) && _jsx(Grid, { item: true, xs: 12, sm: 6, children: _jsx(InputOutputContainer, { isEditing: canUpdate, handleUpdate: inputsHelpers.setValue, isInput: true, language: language, list: inputsField.value, zIndex: zIndex }) }), (canUpdate || (outputsField.value?.length > 0)) && _jsx(Grid, { item: true, xs: 12, sm: 6, children: _jsx(InputOutputContainer, { isEditing: canUpdate, handleUpdate: outputsHelpers.setValue, isInput: false, language: language, list: outputsField.value, zIndex: zIndex }) }), (canUpdate || (exists(resourceListField.value) && Array.isArray(resourceListField.value.resources) && resourceListField.value.resources.length > 0)) && _jsx(Grid, { item: true, xs: 12, mb: 2, children: _jsx(ResourceListHorizontal, { title: "Resources", list: resourceListField.value, canUpdate: canUpdate, handleUpdate: (newList) => { resourceListHelpers.setValue(newList); }, mutate: false, zIndex: zIndex }) }), _jsx(Grid, { item: true, xs: 12, marginBottom: 4, children: canUpdate ? _jsx(TagSelector, { name: 'routineVersion.root.tags' }) :
                                    _jsx(TagList, { parentId: "", tags: (tagsField.value ?? []) }) })] }), canUpdate && _jsx(GridSubmitButtons, { display: "dialog", errors: combineErrorsWithTranslations(props.errors, translationErrors), isCreate: isCreate, loading: props.isSubmitting, onCancel: onCancel, onSetSubmitting: props.setSubmitting, onSubmit: props.handleSubmit })] })] }));
});
//# sourceMappingURL=SubroutineForm.js.map