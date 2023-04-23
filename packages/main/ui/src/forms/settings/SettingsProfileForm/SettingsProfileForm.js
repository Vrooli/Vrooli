import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { RefreshIcon } from "@local/icons";
import { userTranslationValidation } from "@local/validation";
import { Autocomplete, Grid, Stack, TextField, useTheme } from "@mui/material";
import { Field, useField } from "formik";
import { useCallback, useContext, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useCustomLazyQuery } from "../../../api";
import { walletFindHandles } from "../../../api/generated/endpoints/wallet_findHandles";
import { ColorIconButton } from "../../../components/buttons/ColorIconButton/ColorIconButton";
import { GridSubmitButtons } from "../../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "../../../components/inputs/LanguageInput/LanguageInput";
import { TranslatedTextField } from "../../../components/inputs/TranslatedTextField/TranslatedTextField";
import { combineErrorsWithTranslations, getUserLanguages } from "../../../utils/display/translationTools";
import { useTranslatedFields } from "../../../utils/hooks/useTranslatedFields";
import { PubSub } from "../../../utils/pubsub";
import { SessionContext } from "../../../utils/SessionContext";
import { BaseForm } from "../../BaseForm/BaseForm";
export const SettingsProfileForm = ({ display, dirty, isLoading, numVerifiedWallets, onCancel, values, ...props }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const { handleAddLanguage, handleDeleteLanguage, language, languages, setLanguage, translationErrors, } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["bio"],
        validationSchema: userTranslationValidation.update({}),
    });
    const [handlesField, , handlesHelpers] = useField("handle");
    const [findHandles, { data: handlesData, loading: handlesLoading }] = useCustomLazyQuery(walletFindHandles);
    const [handles, setHandles] = useState([]);
    const fetchHandles = useCallback(() => {
        if (numVerifiedWallets > 0) {
            findHandles({ variables: {} });
        }
        else {
            PubSub.get().publishSnack({ messageKey: "NoVerifiedWallets", severity: "Error" });
        }
    }, [numVerifiedWallets, findHandles]);
    useEffect(() => {
        if (handlesData) {
            setHandles(handlesData);
        }
    }, [handlesData]);
    return (_jsxs(BaseForm, { dirty: dirty, isLoading: isLoading, style: {
            width: { xs: "100%", md: "min(100%, 700px)" },
            margin: "auto",
            display: "block",
        }, children: [_jsxs(Grid, { container: true, spacing: 2, sx: {
                    paddingBottom: 4,
                    paddingLeft: 2,
                    paddingRight: 2,
                    paddingTop: 2,
                }, children: [_jsx(Grid, { item: true, xs: 12, children: _jsx(LanguageInput, { currentLanguage: language, handleAdd: handleAddLanguage, handleDelete: handleDeleteLanguage, handleCurrent: setLanguage, languages: languages, zIndex: 200 }) }), _jsx(Grid, { item: true, xs: 12, children: _jsxs(Stack, { direction: "row", spacing: 0, children: [_jsx(Autocomplete, { disablePortal: true, id: "ada-handle-select", loading: handlesLoading, options: handles, onOpen: fetchHandles, onChange: (_, value) => { handlesHelpers.setValue(value); }, renderInput: (params) => _jsx(TextField, { ...params, label: "(ADA) Handle", sx: {
                                            "& .MuiInputBase-root": {
                                                borderRadius: "5px 0 0 5px",
                                            },
                                        } }), value: handlesField.value, sx: {
                                        width: "100%",
                                    } }), _jsx(ColorIconButton, { "aria-label": 'fetch-handles', background: palette.secondary.main, onClick: fetchHandles, sx: { borderRadius: "0 5px 5px 0" }, children: _jsx(RefreshIcon, {}) })] }) }), _jsx(Grid, { item: true, xs: 12, children: _jsx(Field, { fullWidth: true, name: "name", label: t("Name"), as: TextField }) }), _jsx(Grid, { item: true, xs: 12, children: _jsx(TranslatedTextField, { fullWidth: true, name: "bio", label: t("Bio"), language: language, multiline: true, minRows: 4, maxRows: 10 }) })] }), _jsx(GridSubmitButtons, { display: display, errors: combineErrorsWithTranslations(props.errors, translationErrors), isCreate: false, loading: props.isSubmitting, onCancel: onCancel, onSetSubmitting: props.setSubmitting, onSubmit: props.handleSubmit })] }));
};
//# sourceMappingURL=SettingsProfileForm.js.map