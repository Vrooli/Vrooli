import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { orDefault } from "@local/utils";
import { DUMMY_ID } from "@local/uuid";
import { smartContractVersionTranslationValidation, smartContractVersionValidation } from "@local/validation";
import { Stack, useTheme } from "@mui/material";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { GridSubmitButtons } from "../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "../../components/inputs/LanguageInput/LanguageInput";
import { VersionInput } from "../../components/inputs/VersionInput/VersionInput";
import { getCurrentUser } from "../../utils/authentication/session";
import { combineErrorsWithTranslations, getUserLanguages } from "../../utils/display/translationTools";
import { useTranslatedFields } from "../../utils/hooks/useTranslatedFields";
import { SessionContext } from "../../utils/SessionContext";
import { validateAndGetYupErrors } from "../../utils/shape/general";
import { shapeSmartContractVersion } from "../../utils/shape/models/smartContractVersion";
import { BaseForm } from "../BaseForm/BaseForm";
export const smartContractInitialValues = (session, existing) => ({
    __typename: "SmartContractVersion",
    id: DUMMY_ID,
    directoryListings: [],
    isComplete: false,
    isPrivate: false,
    content: "",
    contractType: "",
    resourceList: {
        __typename: "ResourceList",
        id: DUMMY_ID,
    },
    root: {
        __typename: "SmartContract",
        id: DUMMY_ID,
        isPrivate: false,
        owner: { __typename: "User", id: getCurrentUser(session).id },
        parent: null,
        tags: [],
    },
    versionLabel: "1.0.0",
    ...existing,
    translations: orDefault(existing?.translations, [{
            __typename: "SmartContractVersionTranslation",
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            description: "",
            jsonVariable: "",
            name: "",
        }]),
});
export const transformSmartContractValues = (values, existing) => {
    return existing === undefined
        ? shapeSmartContractVersion.create(values)
        : shapeSmartContractVersion.update(existing, values);
};
export const validateSmartContractValues = async (values, existing) => {
    const transformedValues = transformSmartContractValues(values, existing);
    const validationSchema = smartContractVersionValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};
export const SmartContractForm = forwardRef(({ display, dirty, isCreate, isLoading, isOpen, onCancel, values, versions, zIndex, ...props }, ref) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const { handleAddLanguage, handleDeleteLanguage, language, languages, setLanguage, translationErrors, } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["description", "jsonVariable", "name"],
        validationSchema: smartContractVersionTranslationValidation[isCreate ? "create" : "update"]({}),
    });
    return (_jsx(_Fragment, { children: _jsxs(BaseForm, { dirty: dirty, isLoading: isLoading, ref: ref, style: {
                display: "block",
                width: "min(700px, 100vw - 16px)",
                margin: "auto",
                paddingLeft: "env(safe-area-inset-left)",
                paddingRight: "env(safe-area-inset-right)",
                paddingBottom: "calc(64px + env(safe-area-inset-bottom))",
            }, children: [_jsx(Stack, { direction: "column", spacing: 2, paddingTop: 2, children: _jsx(LanguageInput, { currentLanguage: language, handleAdd: handleAddLanguage, handleDelete: handleDeleteLanguage, handleCurrent: setLanguage, languages: languages, zIndex: zIndex + 1 }) }), _jsx(VersionInput, { fullWidth: true, versions: versions }), _jsx(GridSubmitButtons, { display: display, errors: combineErrorsWithTranslations(props.errors, translationErrors), isCreate: isCreate, loading: props.isSubmitting, onCancel: onCancel, onSetSubmitting: props.setSubmitting, onSubmit: props.handleSubmit })] }) }));
});
//# sourceMappingURL=SmartContractForm.js.map