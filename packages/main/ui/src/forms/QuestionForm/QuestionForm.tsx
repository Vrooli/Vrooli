import { Question, Session } from "@local/consts";
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
import { QuestionShape, shapeQuestion } from "../../utils/shape/models/question";
import { BaseForm } from "../BaseForm/BaseForm";
import { QuestionFormProps } from "../types";

export const questionInitialValues = (
    session: Session | undefined,
    existing?: Question | null | undefined,
): QuestionShape => ({
    __typename: "Question" as const,
    id: DUMMY_ID,
    isPrivate: false,
    referencing: undefined,
    forObject: null,
    tags: [],
    ...existing,
    translations: orDefault(existing?.translations, [{
        __typename: "QuestionTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        description: "",
        name: "",
    }]),
});

export function transformQuestionValues(values: QuestionShape, existing?: QuestionShape) {
    return existing === undefined
        ? shapeQuestion.create(values)
        : shapeQuestion.update(existing, values);
}

export const validateQuestionValues = async (values: QuestionShape, existing?: QuestionShape) => {
    const transformedValues = transformQuestionValues(values, existing);
    const validationSchema = questionValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const QuestionForm = forwardRef<any, QuestionFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
    zIndex,
    ...props
}, ref) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Handle translations
    const {
        handleAddLanguage,
        handleDeleteLanguage,
        language,
        languages,
        setLanguage,
        translationErrors,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["description", "name"],
        validationSchema: questionTranslationValidation[isCreate ? "create" : "update"]({}),
    });

    return (
        <>
            <BaseForm
                dirty={dirty}
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: "block",
                    width: "min(700px, 100vw - 16px)",
                    margin: "auto",
                    paddingLeft: "env(safe-area-inset-left)",
                    paddingRight: "env(safe-area-inset-right)",
                    paddingBottom: "calc(64px + env(safe-area-inset-bottom))",
                }}
            >
                <Stack direction="column" spacing={4} sx={{
                    margin: 2,
                    marginBottom: 4,
                }}>
                    <Stack direction="column" spacing={2}>
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleDeleteLanguage}
                            handleCurrent={setLanguage}
                            languages={languages}
                            zIndex={zIndex + 1}
                        />
                        <TranslatedTextField
                            fullWidth
                            label={t("Name")}
                            language={language}
                            name="name"
                        />
                        <TranslatedMarkdownInput
                            language={language}
                            name="description"
                            placeholder={t("Description")}
                            minRows={3}
                            sxs={{
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
                            }}
                        />
                    </Stack>
                    <TagSelector name="tags" />
                </Stack>
                <GridSubmitButtons
                    display={display}
                    errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                    isCreate={isCreate}
                    loading={props.isSubmitting}
                    onCancel={onCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={props.handleSubmit}
                />
            </BaseForm>
        </>
    );
});