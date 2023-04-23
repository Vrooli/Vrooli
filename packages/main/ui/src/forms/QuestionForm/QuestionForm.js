import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { orDefault } from "@local/utils";
import { DUMMY_ID } from "@local/uuid";
import { questionTranslationValidation, questionValidation } from "@local/validation";
import { Stack, useTheme } from "@mui/material";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { GridSubmitButtons } from "../../components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "../../components/inputs/LanguageInput/LanguageInput";
import { TagSelector } from "../../components/inputs/TagSelector/TagSelector";
import { TranslatedMarkdownInput } from "../../components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { TranslatedTextField } from "../../components/inputs/TranslatedTextField/TranslatedTextField";
import { combineErrorsWithTranslations, getUserLanguages } from "../../utils/display/translationTools";
import { useTranslatedFields } from "../../utils/hooks/useTranslatedFields";
import { SessionContext } from "../../utils/SessionContext";
import { validateAndGetYupErrors } from "../../utils/shape/general";
import { shapeQuestion } from "../../utils/shape/models/question";
import { BaseForm } from "../BaseForm/BaseForm";
export const questionInitialValues = (session, existing) => ({
    __typename: "Question",
    id: DUMMY_ID,
    isPrivate: false,
    referencing: undefined,
    forObject: null,
    tags: [],
    ...existing,
    translations: orDefault(existing?.translations, [{
            __typename: "QuestionTranslation",
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            description: "",
            name: "",
        }]),
});
export function transformQuestionValues(values, existing) {
    return existing === undefined
        ? shapeQuestion.create(values)
        : shapeQuestion.update(existing, values);
}
export const validateQuestionValues = async (values, existing) => {
    const transformedValues = transformQuestionValues(values, existing);
    const validationSchema = questionValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};
export const QuestionForm = forwardRef(({ display, dirty, isCreate, isLoading, isOpen, onCancel, values, zIndex, ...props }, ref) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const { handleAddLanguage, handleDeleteLanguage, language, languages, setLanguage, translationErrors, } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["description", "name"],
        validationSchema: questionTranslationValidation[isCreate ? "create" : "update"]({}),
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
                    }, children: [_jsxs(Stack, { direction: "column", spacing: 2, children: [_jsx(LanguageInput, { currentLanguage: language, handleAdd: handleAddLanguage, handleDelete: handleDeleteLanguage, handleCurrent: setLanguage, languages: languages, zIndex: zIndex + 1 }), _jsx(TranslatedTextField, { fullWidth: true, label: t("Name"), language: language, name: "name" }), _jsx(TranslatedMarkdownInput, { language: language, name: "description", placeholder: t("Description"), minRows: 3, sxs: {
                                        bar: {
                                            borderRadius: 0,
                                            background: palette.primary.main,
                                        },
                                        textArea: {
                                            borderRadius: 0,
                                            resize: "none",
                                            minHeight: "100vh",
                                            background: palette.background.paper,
                                        },
                                    } })] }), _jsx(TagSelector, { name: "tags" })] }), _jsx(GridSubmitButtons, { display: display, errors: combineErrorsWithTranslations(props.errors, translationErrors), isCreate: isCreate, loading: props.isSubmitting, onCancel: onCancel, onSetSubmitting: props.setSubmitting, onSubmit: props.handleSubmit })] }) }));
});
//# sourceMappingURL=QuestionForm.js.map