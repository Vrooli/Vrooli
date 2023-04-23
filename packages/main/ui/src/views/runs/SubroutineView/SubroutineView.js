import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { CommentFor } from "@local/consts";
import { SuccessIcon } from "@local/icons";
import { exists } from "@local/utils";
import { Box, Button, Stack, useTheme } from "@mui/material";
import { useFormik } from "formik";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { CommentContainer } from "../../../components/containers/CommentContainer/CommentContainer";
import { ContentCollapse } from "../../../components/containers/ContentCollapse/ContentCollapse";
import { TextCollapse } from "../../../components/containers/TextCollapse/TextCollapse";
import { GeneratedInputComponentWithLabel } from "../../../components/inputs/generated";
import { ObjectActionsRow } from "../../../components/lists/ObjectActionsRow/ObjectActionsRow";
import { RelationshipList } from "../../../components/lists/RelationshipList/RelationshipList";
import { ResourceListHorizontal } from "../../../components/lists/resource";
import { smallHorizontalScrollbar } from "../../../components/lists/styles";
import { TagList } from "../../../components/lists/TagList/TagList";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { DateDisplay } from "../../../components/text/DateDisplay/DateDisplay";
import { ObjectTitle } from "../../../components/text/ObjectTitle/ObjectTitle";
import { VersionDisplay } from "../../../components/text/VersionDisplay/VersionDisplay";
import { routineInitialValues } from "../../../forms/RoutineForm/RoutineForm";
import { ObjectAction } from "../../../utils/actions/objectActions";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "../../../utils/display/translationTools";
import { useObjectActions } from "../../../utils/hooks/useObjectActions";
import { PubSub } from "../../../utils/pubsub";
import { useLocation } from "../../../utils/route";
import { formikToRunInputs, runInputsToFormik } from "../../../utils/runUtils";
import { SessionContext } from "../../../utils/SessionContext";
import { standardVersionToFieldData } from "../../../utils/shape/general";
const containerProps = (palette) => ({
    boxShadow: 1,
    background: palette.background.paper,
    borderRadius: 1,
    overflow: "overlay",
    marginTop: 4,
    marginBottom: 4,
    padding: 2,
});
export const SubroutineView = ({ display = "page", loading, handleUserInputsUpdate, handleSaveProgress, owner, routineVersion, run, zIndex, }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const [language, setLanguage] = useState(getUserLanguages(session)[0]);
    const [internalRoutineVersion, setInternalRoutineVersion] = useState(routineVersion);
    useEffect(() => {
        setInternalRoutineVersion(routineVersion);
    }, [routineVersion]);
    const updateRoutine = useCallback((routineVersion) => { setInternalRoutineVersion(routineVersion); }, [setInternalRoutineVersion]);
    const availableLanguages = useMemo(() => (internalRoutineVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [internalRoutineVersion?.translations]);
    useEffect(() => {
        if (availableLanguages.length === 0)
            return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);
    const { description, instructions, name } = useMemo(() => {
        const languages = getUserLanguages(session);
        const { description, instructions, name } = getTranslation(internalRoutineVersion, languages, true);
        return {
            description,
            instructions,
            name,
        };
    }, [internalRoutineVersion, session]);
    const confirmLeave = useCallback((callback) => {
        PubSub.get().publishAlertDialog({
            messageKey: "RunStopConfirm",
            buttons: [
                {
                    labelKey: "Yes",
                    onClick: () => {
                        handleSaveProgress();
                        callback();
                    },
                },
                { labelKey: "Cancel" },
            ],
        });
    }, [handleSaveProgress]);
    const formValueMap = useMemo(() => {
        if (!internalRoutineVersion)
            return {};
        const schemas = {};
        for (let i = 0; i < internalRoutineVersion.inputs?.length; i++) {
            const currInput = internalRoutineVersion.inputs[i];
            if (!currInput.standardVersion)
                continue;
            const currSchema = standardVersionToFieldData({
                description: getTranslation(currInput, getUserLanguages(session), false).description ?? getTranslation(currInput.standardVersion, getUserLanguages(session), false).description,
                fieldName: `inputs-${currInput.id}`,
                helpText: getTranslation(currInput, getUserLanguages(session), false).helpText,
                props: currInput.standardVersion.props,
                name: currInput.name ?? getTranslation(currInput.standardVersion, getUserLanguages(session), false).name ?? "",
                standardType: currInput.standardVersion.standardType,
                yup: currInput.standardVersion.yup,
            });
            if (currSchema) {
                schemas[currSchema.fieldName] = currSchema;
            }
        }
        return schemas;
    }, [internalRoutineVersion, session]);
    const formik = useFormik({
        initialValues: Object.entries(formValueMap).reduce((acc, [key, value]) => {
            acc[key] = value.props.defaultValue ?? "";
            return acc;
        }, {}),
        enableReinitialize: true,
        onSubmit: () => { },
    });
    useEffect(() => {
        console.log("useeffect1 calculating preview formik values", run);
        if (!run?.inputs || !Array.isArray(run?.inputs) || run.inputs.length === 0)
            return;
        console.log("useeffect 1calling runInputsToFormik", run.inputs);
        const updatedValues = runInputsToFormik(run.inputs);
        console.log("useeffect1 updating formik, values", updatedValues);
        formik.setValues(updatedValues);
    }, [formik.setValues, run?.inputs]);
    useEffect(() => {
        if (!formik.values)
            return;
        const updatedValues = formikToRunInputs(formik.values);
        handleUserInputsUpdate(updatedValues);
    }, [handleUserInputsUpdate, formik.values, run?.inputs]);
    const copyInput = useCallback((fieldName) => {
        const input = formik.values[fieldName];
        if (input) {
            navigator.clipboard.writeText(input);
            PubSub.get().publishSnack({ messageKey: "CopiedToClipboard", severity: "Success" });
        }
        else {
            PubSub.get().publishSnack({ messageKey: "InputEmpty", severity: "Error" });
        }
    }, [formik.values]);
    const inputComponents = useMemo(() => {
        if (!internalRoutineVersion?.inputs || !Array.isArray(internalRoutineVersion?.inputs) || internalRoutineVersion.inputs.length === 0)
            return null;
        return (_jsx(Box, { children: Object.values(formValueMap).map((fieldData, index) => (_jsx(GeneratedInputComponentWithLabel, { copyInput: copyInput, disabled: false, fieldData: fieldData, index: index, textPrimary: palette.background.textPrimary, onUpload: () => { }, zIndex: zIndex }))) }));
    }, [copyInput, formValueMap, palette.background.textPrimary, internalRoutineVersion?.inputs, zIndex]);
    const [isAddCommentOpen, setIsAddCommentOpen] = useState(false);
    const openAddCommentDialog = useCallback(() => { setIsAddCommentOpen(true); }, []);
    const closeAddCommentDialog = useCallback(() => { setIsAddCommentOpen(false); }, []);
    const actionData = useObjectActions({
        object: internalRoutineVersion,
        objectType: "RoutineVersion",
        openAddCommentDialog,
        setLocation,
        setObject: setInternalRoutineVersion,
    });
    const initialValues = useMemo(() => routineInitialValues(session, internalRoutineVersion), [internalRoutineVersion, session]);
    const resourceList = useMemo(() => initialValues.resourceList, [initialValues]);
    const tags = useMemo(() => initialValues.root?.tags, [initialValues]);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { } }), _jsxs(Box, { sx: {
                    marginLeft: "auto",
                    marginRight: "auto",
                    width: "min(100%, 700px)",
                    padding: 2,
                }, children: [_jsx(ObjectTitle, { language: language, languages: availableLanguages, loading: loading, title: name, setLanguage: setLanguage, translations: internalRoutineVersion?.translations ?? [], zIndex: zIndex }), exists(resourceList) && Array.isArray(resourceList.resources) && resourceList.resources.length > 0 && _jsx(ResourceListHorizontal, { title: "Resources", list: resourceList, canUpdate: false, handleUpdate: () => { }, loading: loading, zIndex: zIndex }), _jsxs(Stack, { direction: "column", spacing: 4, sx: containerProps(palette), children: [_jsx(TextCollapse, { title: "Description", text: description, loading: loading, loadingLines: 2 }), _jsx(TextCollapse, { title: "Instructions", text: instructions, loading: loading, loadingLines: 4 })] }), _jsx(Box, { sx: containerProps(palette), children: _jsxs(ContentCollapse, { title: "Inputs", children: [inputComponents, _jsx(Button, { startIcon: _jsx(SuccessIcon, {}), fullWidth: true, onClick: () => { }, color: "secondary", sx: { marginTop: 2 }, children: "Submit" })] }) }), _jsx(ObjectActionsRow, { actionData: actionData, exclude: [ObjectAction.Edit, ObjectAction.VoteDown, ObjectAction.VoteUp], object: internalRoutineVersion, zIndex: zIndex }), _jsx(Box, { sx: containerProps(palette), children: _jsxs(ContentCollapse, { isOpen: false, title: "Additional Information", children: [_jsx(RelationshipList, { isEditing: false, objectType: "Routine", zIndex: zIndex }), exists(tags) && tags.length > 0 && _jsx(TagList, { maxCharacters: 30, parentId: internalRoutineVersion?.id ?? "", tags: tags, sx: { ...smallHorizontalScrollbar(palette), marginTop: 4 } }), _jsxs(Stack, { direction: "row", spacing: 1, mt: 2, mb: 1, children: [_jsx(DateDisplay, { loading: loading, showIcon: true, timestamp: internalRoutineVersion?.created_at }), _jsx(VersionDisplay, { currentVersion: internalRoutineVersion, prefix: " - ", versions: internalRoutineVersion?.root?.versions })] })] }) }), _jsx(Box, { sx: containerProps(palette), children: _jsx(CommentContainer, { forceAddCommentOpen: isAddCommentOpen, isOpen: false, language: language, objectId: internalRoutineVersion?.id ?? "", objectType: CommentFor.RoutineVersion, onAddCommentClose: closeAddCommentDialog, zIndex: zIndex }) })] })] }));
};
//# sourceMappingURL=SubroutineView.js.map