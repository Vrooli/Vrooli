import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { orDefault } from "@local/utils";
import { DUMMY_ID } from "@local/uuid";
import { projectVersionTranslationValidation, projectVersionValidation } from "@local/validation";
import { Stack, useTheme } from "@mui/material";
import { useField } from "formik";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { GridSubmitButtons } from "../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "../../components/inputs/LanguageInput/LanguageInput";
import { TranslatedTextField } from "../../components/inputs/TranslatedTextField/TranslatedTextField";
import { VersionInput } from "../../components/inputs/VersionInput/VersionInput";
import { DirectoryListHorizontal } from "../../components/lists/directory";
import { RelationshipList } from "../../components/lists/RelationshipList/RelationshipList";
import { getCurrentUser } from "../../utils/authentication/session";
import { combineErrorsWithTranslations, getUserLanguages } from "../../utils/display/translationTools";
import { useTranslatedFields } from "../../utils/hooks/useTranslatedFields";
import { SessionContext } from "../../utils/SessionContext";
import { validateAndGetYupErrors } from "../../utils/shape/general";
import { shapeProjectVersion } from "../../utils/shape/models/projectVersion";
import { BaseForm } from "../BaseForm/BaseForm";
export const projectInitialValues = (session, existing) => ({
    __typename: "ProjectVersion",
    id: DUMMY_ID,
    isComplete: false,
    isPrivate: true,
    resourceList: {
        __typename: "ResourceList",
        id: DUMMY_ID,
    },
    root: {
        __typename: "Project",
        id: DUMMY_ID,
        isPrivate: false,
        owner: { __typename: "User", id: getCurrentUser(session).id },
        parent: null,
        tags: [],
    },
    versionLabel: "1.0.0",
    ...existing,
    directories: orDefault(existing?.directories, [{
            __typename: "ProjectVersionDirectory",
            id: DUMMY_ID,
            isRoot: true,
            childApiVersions: [],
            childNoteVersions: [],
            childOrganizations: [],
            childProjectVersions: [],
            childRoutineVersions: [],
            childSmartContractVersions: [],
            childStandardVersions: [],
        }]),
    translations: orDefault(existing?.translations, [{
            __typename: "ProjectVersionTranslation",
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            name: "",
            description: "",
        }]),
});
export const transformProjectValues = (values, existing) => {
    return existing === undefined
        ? shapeProjectVersion.create(values)
        : shapeProjectVersion.update(existing, values);
};
export const validateProjectValues = async (values, existing) => {
    const transformedValues = transformProjectValues(values, existing);
    const validationSchema = projectVersionValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};
export const ProjectForm = forwardRef(({ display, dirty, isCreate, isLoading, isOpen, onCancel, values, versions, zIndex, ...props }, ref) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const { handleAddLanguage, handleDeleteLanguage, language, languages, setLanguage, translationErrors, } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["description", "name"],
        validationSchema: projectVersionTranslationValidation[isCreate ? "create" : "update"]({}),
    });
    const [directoryField, , directoryHelpers] = useField("directories[0]");
    return (_jsx(_Fragment, { children: _jsxs(BaseForm, { dirty: dirty, isLoading: isLoading, ref: ref, style: {
                display: "block",
                width: "min(700px, 100vw - 16px)",
                margin: "auto",
                paddingLeft: "env(safe-area-inset-left)",
                paddingRight: "env(safe-area-inset-right)",
                paddingBottom: "calc(64px + env(safe-area-inset-bottom))",
            }, children: [_jsxs(Stack, { direction: "column", spacing: 4, sx: {
                        margin: 2,
                        marginBottom: 4,
                    }, children: [_jsx(RelationshipList, { isEditing: true, objectType: "Project", zIndex: zIndex, sx: { marginBottom: 4 } }), _jsxs(Stack, { direction: "column", spacing: 2, children: [_jsx(LanguageInput, { currentLanguage: language, handleAdd: handleAddLanguage, handleDelete: handleDeleteLanguage, handleCurrent: setLanguage, languages: languages, zIndex: zIndex + 1 }), _jsx(TranslatedTextField, { fullWidth: true, label: t("Name"), language: language, name: "name" }), _jsx(TranslatedTextField, { fullWidth: true, label: t("Description"), language: language, multiline: true, minRows: 2, maxRows: 4, name: "description" })] }), _jsx(DirectoryListHorizontal, { canUpdate: true, directory: directoryField.value, handleUpdate: directoryHelpers.setValue, loading: isLoading, mutate: false, zIndex: zIndex }), _jsx(VersionInput, { fullWidth: true, versions: versions })] }), _jsx(GridSubmitButtons, { display: display, errors: combineErrorsWithTranslations(props.errors, translationErrors), isCreate: isCreate, loading: props.isSubmitting, onCancel: onCancel, onSetSubmitting: props.setSubmitting, onSubmit: props.handleSubmit })] }) }));
});
//# sourceMappingURL=ProjectForm.js.map