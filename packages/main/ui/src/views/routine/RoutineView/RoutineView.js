import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { CommentFor, LINKS } from "@local/consts";
import { EditIcon, RoutineIcon, SuccessIcon } from "@local/icons";
import { exists, setDotNotationValue } from "@local/utils";
import { Box, Button, Dialog, Stack, useTheme } from "@mui/material";
import { Formik, useFormik } from "formik";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { routineVersionFindOne } from "../../../api/generated/endpoints/routineVersion_findOne";
import { runRoutineComplete } from "../../../api/generated/endpoints/runRoutine_complete";
import { useCustomMutation } from "../../../api/hooks";
import { mutationWrapper } from "../../../api/utils";
import { ColorIconButton } from "../../../components/buttons/ColorIconButton/ColorIconButton";
import { RunButton } from "../../../components/buttons/RunButton/RunButton";
import { SideActionButtons } from "../../../components/buttons/SideActionButtons/SideActionButtons";
import { CommentContainer } from "../../../components/containers/CommentContainer/CommentContainer";
import { ContentCollapse } from "../../../components/containers/ContentCollapse/ContentCollapse";
import { TextCollapse } from "../../../components/containers/TextCollapse/TextCollapse";
import { UpTransition } from "../../../components/dialogs/transitions";
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
import { getCurrentUser } from "../../../utils/authentication/session";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "../../../utils/display/translationTools";
import { useObjectActions } from "../../../utils/hooks/useObjectActions";
import { useObjectFromUrl } from "../../../utils/hooks/useObjectFromUrl";
import { PubSub } from "../../../utils/pubsub";
import { parseSearchParams, setSearchParams, useLocation } from "../../../utils/route";
import { formikToRunInputs, runInputsCreate } from "../../../utils/runUtils";
import { SessionContext } from "../../../utils/SessionContext";
import { standardVersionToFieldData } from "../../../utils/shape/general";
import { BuildView } from "../../BuildView/BuildView";
const statsHelpText = "Statistics are calculated to measure various aspects of a routine. \n\n**Complexity** is a rough measure of the maximum amount of effort it takes to complete a routine. This takes into account the number of inputs, the structure of its subroutine graph, and the complexity of every subroutine.\n\n**Simplicity** is calculated similarly to complexity, but takes the shortest path through the subroutine graph.\n\nThere will be many more statistics in the near future.";
const containerProps = (palette) => ({
    boxShadow: 1,
    background: palette.background.paper,
    borderRadius: 1,
    overflow: "overlay",
    marginTop: 4,
    marginBottom: 4,
    padding: 2,
});
export const RoutineView = ({ display = "page", partialData, zIndex = 200, }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const [language, setLanguage] = useState(getUserLanguages(session)[0]);
    const { id, isLoading, object: existing, permissions, setObject: setRoutineVersion } = useObjectFromUrl({
        query: routineVersionFindOne,
        onInvalidUrlParams: ({ build }) => {
            if (!build || build !== true)
                PubSub.get().publishSnack({ messageKey: "InvalidUrlId", severity: "Error" });
        },
        partialData,
    });
    const availableLanguages = useMemo(() => (existing?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [existing?.translations]);
    useEffect(() => {
        if (availableLanguages.length === 0)
            return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);
    const { name, description, instructions } = useMemo(() => {
        const { description, instructions, name } = getTranslation(existing ?? partialData, [language]);
        return { name, description, instructions };
    }, [existing, language, partialData]);
    useEffect(() => {
        document.title = `${name} | Vrooli`;
    }, [name]);
    const [isBuildOpen, setIsBuildOpen] = useState(Boolean(parseSearchParams()?.build));
    const viewGraph = useCallback(() => {
        setSearchParams(setLocation, { build: true });
        setIsBuildOpen(true);
    }, [setLocation]);
    const stopBuild = useCallback(() => {
        setIsBuildOpen(false);
    }, []);
    const handleRunDelete = useCallback((run) => {
        if (!existing)
            return;
        setRoutineVersion(setDotNotationValue(existing, "you.runs", existing.you.runs.filter(r => r.id !== run.id)));
    }, [existing, setRoutineVersion]);
    const handleRunAdd = useCallback((run) => {
        if (!existing)
            return;
        setRoutineVersion(setDotNotationValue(existing, "you.runs", [run, ...existing.you.runs]));
    }, [existing, setRoutineVersion]);
    const [isAddCommentOpen, setIsAddCommentOpen] = useState(false);
    const openAddCommentDialog = useCallback(() => { setIsAddCommentOpen(true); }, []);
    const closeAddCommentDialog = useCallback(() => { setIsAddCommentOpen(false); }, []);
    const actionData = useObjectActions({
        object: existing,
        objectType: "Routine",
        openAddCommentDialog,
        setLocation,
        setObject: setRoutineVersion,
    });
    const formValueMap = useMemo(() => {
        if (!existing)
            return null;
        const schemas = {};
        for (let i = 0; i < existing.inputs?.length; i++) {
            const currInput = existing.inputs[i];
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
    }, [existing, session]);
    const formik = useFormik({
        initialValues: Object.entries(formValueMap ?? {}).reduce((acc, [key, value]) => {
            acc[key] = value.props.defaultValue ?? "";
            return acc;
        }, {}),
        enableReinitialize: true,
        onSubmit: () => { },
    });
    const [runComplete] = useCustomMutation(runRoutineComplete);
    const markAsComplete = useCallback(() => {
        if (!existing)
            return;
        mutationWrapper({
            mutation: runComplete,
            input: {
                id: existing.id,
                exists: false,
                name: name ?? "Unnamed Routine",
                ...runInputsCreate(formikToRunInputs(formik.values), existing.id),
            },
            successMessage: () => ({ key: "RoutineCompleted" }),
            onSuccess: () => {
                PubSub.get().publishCelebration();
                setLocation(LINKS.Home);
            },
        });
    }, [formik.values, existing, runComplete, setLocation, name]);
    const copyInput = useCallback((fieldName) => {
        const input = formik.values[fieldName];
        if (input) {
            navigator.clipboard.writeText(input);
            PubSub.get().publishSnack({ messageKey: "CopiedToClipboard", severity: "Success" });
        }
        else {
            PubSub.get().publishSnack({ messageKey: "InputEmpty", severity: "Error" });
        }
    }, [formik]);
    const initialValues = useMemo(() => routineInitialValues(session, existing), [existing, session]);
    const resourceList = useMemo(() => initialValues.resourceList, [initialValues]);
    const tags = useMemo(() => initialValues.root?.tags, [initialValues]);
    console.log("formik values", formik.values);
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "Routine",
                } }), _jsx(Formik, { enableReinitialize: true, initialValues: initialValues, onSubmit: () => { }, children: (formik) => _jsxs(Stack, { direction: "column", spacing: 4, sx: {
                        marginLeft: "auto",
                        marginRight: "auto",
                        width: "min(100%, 700px)",
                        padding: 2,
                    }, children: [existing && _jsx(Dialog, { id: "run-routine-view-dialog", fullScreen: true, open: isBuildOpen, onClose: stopBuild, TransitionComponent: UpTransition, sx: {
                                zIndex: zIndex + 1,
                            }, children: _jsx(BuildView, { handleCancel: stopBuild, handleClose: stopBuild, handleSubmit: () => { }, isEditing: false, loading: isLoading, routineVersion: existing, translationData: {
                                    language,
                                    languages: availableLanguages,
                                    setLanguage,
                                    handleAddLanguage: () => { },
                                    handleDeleteLanguage: () => { },
                                }, zIndex: zIndex + 1 }) }), _jsx(ObjectTitle, { language: language, languages: availableLanguages, loading: isLoading, title: name, setLanguage: setLanguage, translations: existing?.translations ?? [], zIndex: zIndex }), _jsx(RelationshipList, { isEditing: false, objectType: "Routine", zIndex: zIndex }), exists(resourceList) && Array.isArray(resourceList.resources) && resourceList.resources.length > 0 && _jsx(ResourceListHorizontal, { title: "Resources", list: resourceList, canUpdate: false, handleUpdate: () => { }, loading: isLoading, zIndex: zIndex }), (!!description || !!instructions) && _jsxs(Stack, { direction: "column", spacing: 4, sx: containerProps(palette), children: [_jsx(TextCollapse, { title: "Description", text: description, loading: isLoading, loadingLines: 2 }), _jsx(TextCollapse, { title: "Instructions", text: instructions, loading: isLoading, loadingLines: 4 })] }), Object.keys(formik.values).length > 0 && _jsx(Box, { sx: containerProps(palette), children: _jsxs(ContentCollapse, { isOpen: false, title: "Inputs", children: [Object.values(formValueMap ?? {}).map((fieldData, index) => (_jsx(GeneratedInputComponentWithLabel, { copyInput: copyInput, disabled: false, fieldData: fieldData, index: index, textPrimary: palette.background.textPrimary, onUpload: () => { }, zIndex: zIndex }))), getCurrentUser(session).id && _jsx(Button, { startIcon: _jsx(SuccessIcon, {}), fullWidth: true, onClick: markAsComplete, color: "secondary", sx: { marginTop: 2 }, children: t("MarkAsComplete") })] }) }), existing?.nodes?.length ? _jsx(Button, { startIcon: _jsx(RoutineIcon, {}), fullWidth: true, onClick: viewGraph, color: "secondary", children: "View Graph" }) : null, exists(tags) && tags.length > 0 && _jsx(TagList, { maxCharacters: 30, parentId: existing?.id ?? "", tags: tags, sx: { ...smallHorizontalScrollbar(palette), marginTop: 4 } }), _jsxs(Stack, { direction: "row", spacing: 1, mt: 2, mb: 1, children: [_jsx(DateDisplay, { loading: isLoading, showIcon: true, timestamp: existing?.created_at }), _jsx(VersionDisplay, { currentVersion: existing, prefix: " - ", versions: existing?.root?.versions })] }), _jsx(ObjectActionsRow, { actionData: actionData, exclude: [ObjectAction.Edit, ObjectAction.VoteDown, ObjectAction.VoteUp], object: existing, zIndex: zIndex }), _jsx(Box, { sx: containerProps(palette), children: _jsx(CommentContainer, { forceAddCommentOpen: isAddCommentOpen, language: language, objectId: existing?.id ?? "", objectType: CommentFor.RoutineVersion, onAddCommentClose: closeAddCommentDialog, zIndex: zIndex }) })] }) }), _jsxs(SideActionButtons, { display: isBuildOpen ? "dialog" : display, zIndex: zIndex + 2, sx: { position: "fixed" }, children: [permissions.canUpdate ? (_jsx(ColorIconButton, { "aria-label": "edit-routine", background: palette.secondary.main, onClick: () => { actionData.onActionStart(ObjectAction.Edit); }, children: _jsx(EditIcon, { fill: palette.secondary.contrastText, width: '36px', height: '36px' }) })) : null, existing?.nodes?.length ? _jsx(RunButton, { canUpdate: permissions.canUpdate, handleRunAdd: handleRunAdd, handleRunDelete: handleRunDelete, isBuildGraphOpen: isBuildOpen, isEditing: false, runnableObject: existing, zIndex: zIndex }) : null] })] }));
};
//# sourceMappingURL=RoutineView.js.map