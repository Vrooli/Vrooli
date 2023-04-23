import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { RoutineIcon } from "@local/icons";
import { orDefault } from "@local/utils";
import { DUMMY_ID, uuid } from "@local/uuid";
import { routineVersionTranslationValidation, routineVersionValidation } from "@local/validation";
import { Button, Checkbox, FormControlLabel, Grid, Stack, Tooltip } from "@mui/material";
import { useField } from "formik";
import { forwardRef, useCallback, useContext, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { GridSubmitButtons } from "../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LargeDialog } from "../../components/dialogs/LargeDialog/LargeDialog";
import { LanguageInput } from "../../components/inputs/LanguageInput/LanguageInput";
import { ResourceListHorizontalInput } from "../../components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { TagSelector } from "../../components/inputs/TagSelector/TagSelector";
import { TranslatedMarkdownInput } from "../../components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { TranslatedTextField } from "../../components/inputs/TranslatedTextField/TranslatedTextField";
import { VersionInput } from "../../components/inputs/VersionInput/VersionInput";
import { InputOutputContainer } from "../../components/lists/inputOutput";
import { RelationshipList } from "../../components/lists/RelationshipList/RelationshipList";
import { Subheader } from "../../components/text/Subheader/Subheader";
import { getCurrentUser } from "../../utils/authentication/session";
import { combineErrorsWithTranslations, getUserLanguages } from "../../utils/display/translationTools";
import { useTranslatedFields } from "../../utils/hooks/useTranslatedFields";
import { PubSub } from "../../utils/pubsub";
import { initializeRoutineGraph } from "../../utils/runUtils";
import { SessionContext } from "../../utils/SessionContext";
import { validateAndGetYupErrors } from "../../utils/shape/general";
import { shapeRoutineVersion } from "../../utils/shape/models/routineVersion";
import { BuildView } from "../../views/BuildView/BuildView";
import { BaseForm } from "../BaseForm/BaseForm";
export const routineInitialValues = (session, existing) => ({
    __typename: "RoutineVersion",
    id: uuid(),
    inputs: [],
    isComplete: false,
    isPrivate: false,
    directoryListings: [],
    nodeLinks: [],
    nodes: [],
    outputs: [],
    resourceList: {
        __typename: "ResourceList",
        id: DUMMY_ID,
    },
    root: {
        __typename: "Routine",
        id: DUMMY_ID,
        isPrivate: false,
        owner: { __typename: "User", id: getCurrentUser(session).id },
        parent: null,
        permissions: JSON.stringify({}),
        tags: [],
    },
    versionLabel: "1.0.0",
    ...existing,
    translations: orDefault(existing?.translations, [{
            __typename: "RoutineVersionTranslation",
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            description: "",
            instructions: "",
            name: "",
        }]),
});
export const transformRoutineValues = (values, existing) => {
    return existing === undefined
        ? shapeRoutineVersion.create(values)
        : shapeRoutineVersion.update(existing, values);
};
export const validateRoutineValues = async (values, existing) => {
    const transformedValues = transformRoutineValues(values, existing);
    const validationSchema = routineVersionValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};
const helpTextSubroutines = "A routine can be made from scratch (single-step), or by combining other routines (multi-step).\n\nA single-step routine defines inputs and outputs, as well as any other data required to display and execute the routine.\n\nA multi-step routine does not do this. Instead, it uses a graph to combine other routines, using nodes and links.";
export const RoutineForm = forwardRef(({ display, dirty, isCreate, isLoading, isOpen, isSubroutine, onCancel, values, versions, zIndex, ...props }, ref) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { handleAddLanguage, handleDeleteLanguage, language, languages, setLanguage, translationErrors, } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["description", "instructions", "name"],
        validationSchema: routineVersionTranslationValidation[isCreate ? "create" : "update"]({}),
    });
    console.log("valuesssssss", values, transformRoutineValues(values), validateAndGetYupErrors(routineVersionValidation.create({}), transformRoutineValues(values)));
    const [idField] = useField("id");
    const [nodesField, , nodesHelpers] = useField("nodes");
    const [nodeLinksField, , nodeLinksHelpers] = useField("nodeLinks");
    const [inputsField, , inputsHelpers] = useField("inputs");
    const [outputsField, , outputsHelpers] = useField("outputs");
    const [isGraphOpen, setIsGraphOpen] = useState(false);
    const handleGraphOpen = useCallback(() => {
        if (nodesField.value.length === 0 && nodeLinksField.value.length === 0) {
            const { nodes, nodeLinks } = initializeRoutineGraph(language, idField.value);
            nodesHelpers.setValue(nodes);
            nodeLinksHelpers.setValue(nodeLinks);
        }
        setIsGraphOpen(true);
    }, [idField.value, language, nodeLinksField.value.length, nodeLinksHelpers, nodesField.value.length, nodesHelpers]);
    const handleGraphClose = useCallback(() => { setIsGraphOpen(false); }, [setIsGraphOpen]);
    const handleGraphSubmit = useCallback(({ nodes, nodeLinks }) => {
        nodesHelpers.setValue(nodes);
        nodeLinksHelpers.setValue(nodeLinks);
        setIsGraphOpen(false);
    }, [nodeLinksHelpers, nodesHelpers]);
    const [isMultiStep, setIsMultiStep] = useState(null);
    useEffect(() => { setIsMultiStep(nodesField.value.length > 0); }, [nodesField.value.length]);
    const handleMultiStepChange = useCallback((isMultiStep) => {
        if (isMultiStep === false && (nodesField.value.length > 0 || nodeLinksField.value.length > 0)) {
            PubSub.get().publishAlertDialog({
                messageKey: "SubroutineGraphDeleteConfirm",
                buttons: [{
                        labelKey: "Yes",
                        onClick: () => { setIsMultiStep(false); handleGraphClose(); },
                    }, {
                        labelKey: "Cancel",
                    }],
            });
        }
        else if (isMultiStep === true && (inputsField.value.length > 0 || outputsField.value.length > 0)) {
            PubSub.get().publishAlertDialog({
                messageKey: "RoutineInputsDeleteConfirm",
                buttons: [{
                        labelKey: "Yes",
                        onClick: () => { setIsMultiStep(true); handleGraphOpen(); },
                    }, {
                        labelKey: "Cancel",
                    }],
            });
        }
        else {
            setIsMultiStep(isMultiStep);
        }
    }, [nodesField.value.length, nodeLinksField.value.length, inputsField.value.length, outputsField.value.length, handleGraphClose, handleGraphOpen]);
    return (_jsxs(_Fragment, { children: [_jsx(BaseForm, { dirty: dirty, isLoading: isLoading, ref: ref, style: {
                    display: "block",
                    width: "min(700px, 100vw - 16px)",
                    margin: "auto",
                    paddingLeft: "env(safe-area-inset-left)",
                    paddingRight: "env(safe-area-inset-right)",
                    paddingBottom: "calc(64px + env(safe-area-inset-bottom))",
                }, children: _jsxs(Stack, { direction: "column", spacing: 4, sx: {
                        margin: 2,
                        marginBottom: 4,
                    }, children: [_jsx(RelationshipList, { isEditing: true, objectType: "Routine", zIndex: zIndex }), _jsx(ResourceListHorizontalInput, { isCreate: true, zIndex: zIndex }), _jsxs(Stack, { direction: "column", spacing: 2, children: [_jsx(LanguageInput, { currentLanguage: language, handleAdd: handleAddLanguage, handleDelete: handleDeleteLanguage, handleCurrent: setLanguage, languages: languages, zIndex: zIndex + 1 }), _jsx(TranslatedTextField, { fullWidth: true, label: t("Name"), language: language, name: "name" }), _jsx(TranslatedTextField, { fullWidth: true, label: t("Description"), language: language, multiline: true, minRows: 2, maxRows: 2, name: "description" }), _jsx(TranslatedMarkdownInput, { language: language, name: "instructions", minRows: 4 })] }), _jsx(TagSelector, { name: "root.tags" }), _jsx(VersionInput, { fullWidth: true, versions: versions }), isSubroutine && (_jsx(Grid, { item: true, xs: 12, children: _jsx(Tooltip, { placement: "top", title: 'Indicates if this routine is meant to be a subroutine for only one other routine. If so, it will not appear in search resutls.', children: _jsx(FormControlLabel, { label: 'Internal', control: _jsx(Checkbox, { id: 'routine-info-dialog-is-internal', size: "small", name: 'isInternal', color: 'secondary' }) }) }) })), _jsxs(Grid, { item: true, xs: 12, mb: isMultiStep === null ? 8 : 2, children: [_jsx(Subheader, { title: "Use subroutines?", help: helpTextSubroutines }), _jsxs(Stack, { direction: "row", display: "flex", alignItems: "center", justifyContent: "center", spacing: 1, children: [_jsx(Button, { fullWidth: true, color: "secondary", onClick: () => handleMultiStepChange(true), variant: isMultiStep === true ? "outlined" : "contained", children: "Yes" }), _jsx(Button, { fullWidth: true, color: "secondary", onClick: () => handleMultiStepChange(false), variant: isMultiStep === false ? "outlined" : "contained", children: "No" })] })] }), isMultiStep === true && (_jsxs(_Fragment, { children: [_jsx(LargeDialog, { id: "build-routine-graph-dialog", onClose: handleGraphClose, isOpen: isGraphOpen, titleId: "", zIndex: zIndex + 1300, sxs: { paper: { display: "contents" } }, children: _jsx(BuildView, { handleCancel: handleGraphClose, handleClose: handleGraphClose, handleSubmit: handleGraphSubmit, isEditing: true, loading: false, routineVersion: {
                                            id: idField.value,
                                            nodeLinks: nodeLinksField.value,
                                            nodes: nodesField.value,
                                        }, translationData: {
                                            language,
                                            setLanguage,
                                            handleAddLanguage,
                                            handleDeleteLanguage,
                                            languages,
                                        }, zIndex: zIndex + 1300 }) }), _jsx(Grid, { item: true, xs: 12, mb: 4, children: _jsx(Button, { startIcon: _jsx(RoutineIcon, {}), fullWidth: true, color: "secondary", onClick: handleGraphOpen, variant: "contained", children: "View Graph" }) })] })), isMultiStep === false && (_jsxs(_Fragment, { children: [_jsx(Grid, { item: true, xs: 12, children: _jsx(InputOutputContainer, { isEditing: true, handleUpdate: inputsHelpers.setValue, isInput: true, language: language, list: inputsField.value, zIndex: zIndex }) }), _jsx(Grid, { item: true, xs: 12, mb: 4, children: _jsx(InputOutputContainer, { isEditing: true, handleUpdate: outputsHelpers.setValue, isInput: false, language: language, list: outputsField.value, zIndex: zIndex }) })] }))] }) }), _jsx(GridSubmitButtons, { display: display, errors: combineErrorsWithTranslations(props.errors, translationErrors), isCreate: isCreate, loading: props.isSubmitting, onCancel: onCancel, onSetSubmitting: props.setSubmitting, onSubmit: props.handleSubmit })] }));
});
//# sourceMappingURL=RoutineForm.js.map