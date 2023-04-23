import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { orDefault } from "@local/utils";
import { DUMMY_ID } from "@local/uuid";
import { apiVersionTranslationValidation, apiVersionValidation } from "@local/validation";
import { Stack, TextField } from "@mui/material";
import { Field } from "formik";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { GridSubmitButtons } from "../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "../../components/inputs/LanguageInput/LanguageInput";
import { ResourceListHorizontalInput } from "../../components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { TagSelector } from "../../components/inputs/TagSelector/TagSelector";
import { TranslatedTextField } from "../../components/inputs/TranslatedTextField/TranslatedTextField";
import { VersionInput } from "../../components/inputs/VersionInput/VersionInput";
import { RelationshipList } from "../../components/lists/RelationshipList/RelationshipList";
import { getCurrentUser } from "../../utils/authentication/session";
import { combineErrorsWithTranslations, getUserLanguages } from "../../utils/display/translationTools";
import { useTranslatedFields } from "../../utils/hooks/useTranslatedFields";
import { SessionContext } from "../../utils/SessionContext";
import { validateAndGetYupErrors } from "../../utils/shape/general";
import { shapeApiVersion } from "../../utils/shape/models/apiVersion";
import { BaseForm } from "../BaseForm/BaseForm";
export const apiInitialValues = (session, existing) => ({
    __typename: "ApiVersion",
    id: DUMMY_ID,
    callLink: "",
    directoryListings: [],
    isComplete: false,
    isPrivate: false,
    resourceList: {
        __typename: "ResourceList",
        id: DUMMY_ID,
    },
    root: {
        __typename: "Api",
        id: DUMMY_ID,
        isPrivate: false,
        owner: { __typename: "User", id: getCurrentUser(session).id },
        tags: [],
    },
    versionLabel: "1.0.0",
    ...existing,
    translations: orDefault(existing?.translations, [{
            __typename: "ApiVersionTranslation",
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            details: "",
            name: "",
            summary: "",
        }]),
});
export const transformApiValues = (values, existing) => {
    return existing === undefined
        ? shapeApiVersion.create(values)
        : shapeApiVersion.update(existing, values);
};
export const validateApiValues = async (values, existing) => {
    const transformedValues = transformApiValues(values, existing);
    const validationSchema = apiVersionValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};
export const ApiForm = forwardRef(({ display, dirty, isCreate, isLoading, isOpen, onCancel, values, versions, zIndex, ...props }, ref) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { handleAddLanguage, handleDeleteLanguage, language, languages, setLanguage, translationErrors, } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["details", "name", "summary"],
        validationSchema: apiVersionTranslationValidation[isCreate ? "create" : "update"]({}),
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
                    }, children: [_jsx(RelationshipList, { isEditing: true, objectType: "Api", zIndex: zIndex }), _jsx(Field, { fullWidth: true, name: "callLink", label: "Endpoint URL", helperText: "The full URL to the endpoint.", as: TextField }), _jsxs(Stack, { direction: "column", spacing: 2, children: [_jsx(LanguageInput, { currentLanguage: language, handleAdd: handleAddLanguage, handleDelete: handleDeleteLanguage, handleCurrent: setLanguage, languages: languages, zIndex: zIndex + 1 }), _jsx(TranslatedTextField, { fullWidth: true, label: t("Name"), language: language, name: "name" }), _jsx(TranslatedTextField, { fullWidth: true, label: "Summary", language: language, multiline: true, minRows: 2, maxRows: 2, name: "summary" }), _jsx(TranslatedTextField, { fullWidth: true, label: "Details", language: language, multiline: true, minRows: 4, maxRows: 8, name: "details" })] }), _jsx(ResourceListHorizontalInput, { isCreate: true, zIndex: zIndex }), _jsx(TagSelector, { name: "root.tags" }), _jsx(VersionInput, { fullWidth: true, versions: versions })] }), _jsx(GridSubmitButtons, { display: display, errors: combineErrorsWithTranslations(props.errors, translationErrors), isCreate: isCreate, loading: props.isSubmitting, onCancel: onCancel, onSetSubmitting: props.setSubmitting, onSubmit: props.handleSubmit })] }) }));
});
//# sourceMappingURL=ApiForm.js.map