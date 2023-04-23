import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { orDefault } from "@local/utils";
import { DUMMY_ID } from "@local/uuid";
import { organizationTranslationValidation, organizationValidation } from "@local/validation";
import { Checkbox, FormControlLabel, Stack, Tooltip } from "@mui/material";
import { useField } from "formik";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { GridSubmitButtons } from "../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "../../components/inputs/LanguageInput/LanguageInput";
import { ResourceListHorizontalInput } from "../../components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { TagSelector } from "../../components/inputs/TagSelector/TagSelector";
import { TranslatedTextField } from "../../components/inputs/TranslatedTextField/TranslatedTextField";
import { RelationshipList } from "../../components/lists/RelationshipList/RelationshipList";
import { combineErrorsWithTranslations, getUserLanguages } from "../../utils/display/translationTools";
import { useTranslatedFields } from "../../utils/hooks/useTranslatedFields";
import { SessionContext } from "../../utils/SessionContext";
import { validateAndGetYupErrors } from "../../utils/shape/general";
import { shapeOrganization } from "../../utils/shape/models/organization";
import { BaseForm } from "../BaseForm/BaseForm";
export const organizationInitialValues = (session, existing) => ({
    __typename: "Organization",
    id: DUMMY_ID,
    isOpenToNewMembers: false,
    isPrivate: false,
    tags: [],
    ...existing,
    translations: orDefault(existing?.translations, [{
            __typename: "OrganizationTranslation",
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            name: "",
            bio: "",
        }]),
});
export const transformOrganizationValues = (values, existing) => {
    return existing === undefined
        ? shapeOrganization.create(values)
        : shapeOrganization.update(existing, values);
};
export const validateOrganizationValues = async (values, existing) => {
    const transformedValues = transformOrganizationValues(values, existing);
    const validationSchema = organizationValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};
export const OrganizationForm = forwardRef(({ display, dirty, isCreate, isLoading, isOpen, onCancel, values, zIndex, ...props }, ref) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { handleAddLanguage, handleDeleteLanguage, language, languages, setLanguage, translationErrors, } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["bio", "name"],
        validationSchema: organizationTranslationValidation[isCreate ? "create" : "update"]({}),
    });
    const [fieldIsOpen] = useField("isOpenToNewMembers");
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
                    }, children: [_jsx(RelationshipList, { isEditing: true, objectType: "Organization", zIndex: zIndex }), _jsx(ResourceListHorizontalInput, { isCreate: true, zIndex: zIndex }), _jsxs(Stack, { direction: "column", spacing: 2, children: [_jsx(LanguageInput, { currentLanguage: language, handleAdd: handleAddLanguage, handleDelete: handleDeleteLanguage, handleCurrent: setLanguage, languages: languages, zIndex: zIndex + 1 }), _jsx(TranslatedTextField, { fullWidth: true, label: t("Name"), language: language, name: "name" }), _jsx(TranslatedTextField, { fullWidth: true, label: "Bio", language: language, multiline: true, minRows: 2, maxRows: 4, name: "bio" })] }), _jsx(TagSelector, { name: "tags" }), _jsx(Tooltip, { placement: "top", title: 'Indicates if this organization should be displayed when users are looking for an organization to join', children: _jsx(FormControlLabel, { label: 'Open to new members?', control: _jsx(Checkbox, { id: 'organization-is-open-to-new-members', size: "medium", name: 'isOpenToNewMembers', color: 'secondary', checked: fieldIsOpen.value, onChange: fieldIsOpen.onChange }) }) })] }), _jsx(GridSubmitButtons, { display: display, errors: combineErrorsWithTranslations(props.errors, translationErrors), isCreate: isCreate, loading: props.isSubmitting, onCancel: onCancel, onSetSubmitting: props.setSubmitting, onSubmit: props.handleSubmit })] }) }));
});
//# sourceMappingURL=OrganizationForm.js.map