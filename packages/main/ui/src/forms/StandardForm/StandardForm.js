import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { orDefault } from "@local/utils";
import { DUMMY_ID } from "@local/uuid";
import { standardVersionTranslationValidation, standardVersionValidation } from "@local/validation";
import { Stack } from "@mui/material";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { GridSubmitButtons } from "../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "../../components/inputs/LanguageInput/LanguageInput";
import { ResourceListHorizontalInput } from "../../components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { StandardInput } from "../../components/inputs/standards/StandardInput/StandardInput";
import { TagSelector } from "../../components/inputs/TagSelector/TagSelector";
import { TranslatedTextField } from "../../components/inputs/TranslatedTextField/TranslatedTextField";
import { VersionInput } from "../../components/inputs/VersionInput/VersionInput";
import { RelationshipList } from "../../components/lists/RelationshipList/RelationshipList";
import { getCurrentUser } from "../../utils/authentication/session";
import { InputTypeOptions } from "../../utils/consts";
import { combineErrorsWithTranslations, getUserLanguages } from "../../utils/display/translationTools";
import { useTranslatedFields } from "../../utils/hooks/useTranslatedFields";
import { SessionContext } from "../../utils/SessionContext";
import { validateAndGetYupErrors } from "../../utils/shape/general";
import { shapeStandardVersion } from "../../utils/shape/models/standardVersion";
import { BaseForm } from "../BaseForm/BaseForm";
export const standardInitialValues = (session, existing) => ({
    __typename: "StandardVersion",
    id: DUMMY_ID,
    directoryListings: [],
    isComplete: false,
    isPrivate: false,
    isFile: false,
    standardType: InputTypeOptions[0].value,
    props: JSON.stringify({}),
    default: JSON.stringify({}),
    yup: JSON.stringify({}),
    resourceList: {
        __typename: "ResourceList",
        id: DUMMY_ID,
    },
    root: {
        __typename: "Standard",
        id: DUMMY_ID,
        isInternal: false,
        isPrivate: false,
        owner: { __typename: "User", id: getCurrentUser(session).id },
        parent: null,
        permissions: JSON.stringify({}),
        tags: [],
    },
    versionLabel: "1.0.0",
    ...existing,
    translations: orDefault(existing?.translations, [{
            __typename: "StandardVersionTranslation",
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            description: "",
            jsonVariable: null,
            name: "",
        }]),
});
export const transformStandardValues = (values, existing) => {
    return existing === undefined
        ? shapeStandardVersion.create(values)
        : shapeStandardVersion.update(existing, values);
};
export const validateStandardValues = async (values, existing) => {
    const transformedValues = transformStandardValues(values, existing);
    const validationSchema = standardVersionValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};
export const StandardForm = forwardRef(({ display, dirty, isCreate, isLoading, isOpen, onCancel, values, versions, zIndex, ...props }, ref) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { handleAddLanguage, handleDeleteLanguage, language, languages, setLanguage, translationErrors, } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["description"],
        validationSchema: standardVersionTranslationValidation[isCreate ? "create" : "update"]({}),
    });
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
                    }, children: [_jsx(RelationshipList, { isEditing: true, objectType: "Standard", zIndex: zIndex }), _jsxs(Stack, { direction: "column", spacing: 2, children: [_jsx(LanguageInput, { currentLanguage: language, handleAdd: handleAddLanguage, handleDelete: handleDeleteLanguage, handleCurrent: setLanguage, languages: languages, zIndex: zIndex + 1 }), _jsx(TranslatedTextField, { fullWidth: true, label: t("Name"), language: language, name: "name" }), _jsx(TranslatedTextField, { fullWidth: true, label: t("Description"), language: language, multiline: true, minRows: 4, maxRows: 8, name: "description" })] }), _jsx(StandardInput, { fieldName: "preview", zIndex: zIndex }), _jsx(ResourceListHorizontalInput, { isCreate: true, zIndex: zIndex }), _jsx(TagSelector, { name: "root.tags" }), _jsx(VersionInput, { fullWidth: true, versions: versions })] }), _jsx(GridSubmitButtons, { display: display, errors: combineErrorsWithTranslations(props.errors, translationErrors), isCreate: isCreate, loading: props.isSubmitting, onCancel: onCancel, onSetSubmitting: props.setSubmitting, onSubmit: props.handleSubmit })] }) }));
});
//# sourceMappingURL=StandardForm.js.map