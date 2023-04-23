import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { ResourceUsedFor } from "@local/consts";
import { orDefault } from "@local/utils";
import { DUMMY_ID } from "@local/uuid";
import { resourceValidation, userTranslationValidation } from "@local/validation";
import { Stack } from "@mui/material";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { GridSubmitButtons } from "../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "../../components/inputs/LanguageInput/LanguageInput";
import { LinkInput } from "../../components/inputs/LinkInput/LinkInput";
import { Selector } from "../../components/inputs/Selector/Selector";
import { TranslatedTextField } from "../../components/inputs/TranslatedTextField/TranslatedTextField";
import { getResourceIcon } from "../../utils/display/getResourceIcon";
import { combineErrorsWithTranslations, getUserLanguages } from "../../utils/display/translationTools";
import { useTranslatedFields } from "../../utils/hooks/useTranslatedFields";
import { SessionContext } from "../../utils/SessionContext";
import { validateAndGetYupErrors } from "../../utils/shape/general";
import { shapeResource } from "../../utils/shape/models/resource";
import { BaseForm } from "../BaseForm/BaseForm";
export const resourceInitialValues = (session, listId, existing) => ({
    __typename: "Resource",
    id: DUMMY_ID,
    index: 0,
    link: "",
    list: {
        __typename: "ResourceList",
        id: listId ?? DUMMY_ID,
    },
    usedFor: ResourceUsedFor.Context,
    ...existing,
    translations: orDefault(existing?.translations, [{
            __typename: "ResourceTranslation",
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            description: "",
            name: "",
        }]),
});
export function transformResourceValues(values, existing) {
    return existing === undefined
        ? shapeResource.create(values)
        : shapeResource.update(existing, values);
}
export const validateResourceValues = async (values, existing) => {
    const transformedValues = transformResourceValues(values, existing);
    const validationSchema = resourceValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};
export const ResourceForm = forwardRef(({ display, dirty, isCreate, isLoading, isOpen, onCancel, values, zIndex, ...props }, ref) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { handleAddLanguage, handleDeleteLanguage, language, languages, setLanguage, translationErrors, } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["bio"],
        validationSchema: userTranslationValidation[isCreate ? "create" : "update"]({}),
    });
    return (_jsxs(_Fragment, { children: [_jsx(BaseForm, { dirty: dirty, isLoading: isLoading, ref: ref, style: {
                    display: "block",
                    width: "min(500px, 100vw - 16px)",
                    margin: "auto",
                    paddingLeft: "env(safe-area-inset-left)",
                    paddingRight: "env(safe-area-inset-right)",
                    paddingBottom: "calc(64px + env(safe-area-inset-bottom))",
                }, children: _jsxs(Stack, { direction: "column", spacing: 2, padding: 2, children: [_jsx(LanguageInput, { currentLanguage: language, handleAdd: handleAddLanguage, handleDelete: handleDeleteLanguage, handleCurrent: setLanguage, languages: languages, zIndex: zIndex + 1 }), _jsx(LinkInput, { zIndex: zIndex }), _jsx(Selector, { name: "usedFor", options: Object.keys(ResourceUsedFor), getOptionIcon: (i) => getResourceIcon(i), getOptionLabel: (l) => t(l, { count: 2 }), fullWidth: true, label: t("Type") }), _jsx(TranslatedTextField, { fullWidth: true, label: t("NameOptional"), language: language, name: "name" }), _jsx(TranslatedTextField, { fullWidth: true, label: t("DescriptionOptional"), language: language, multiline: true, maxRows: 8, name: "description" })] }) }), _jsx(GridSubmitButtons, { display: display, errors: combineErrorsWithTranslations(props.errors, translationErrors), isCreate: isCreate, loading: props.isSubmitting, onCancel: onCancel, onSetSubmitting: props.setSubmitting, onSubmit: props.handleSubmit })] }));
});
//# sourceMappingURL=ResourceForm.js.map