import { DUMMY_ID, endpointGetQuestion, endpointPostQuestion, endpointPutQuestion, noopSubmit, orDefault, Question, QuestionCreateInput, questionTranslationValidation, QuestionUpdateInput, questionValidation, Session } from "@local/shared";
import { useTheme } from "@mui/material";
import { useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedRichInput } from "components/inputs/TranslatedRichInput/TranslatedRichInput";
import { TranslatedTextInput } from "components/inputs/TranslatedTextInput/TranslatedTextInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { getYou } from "utils/display/listTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { QuestionShape, shapeQuestion } from "utils/shape/models/question";
import { validateFormValues } from "utils/validateFormValues";
import { QuestionFormProps, QuestionUpsertProps } from "../types";

export const questionInitialValues = (
    session: Session | undefined,
    existing?: Partial<Question> | null | undefined,
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

const transformQuestionValues = (values: QuestionShape, existing: QuestionShape, isCreate: boolean) =>
    isCreate ? shapeQuestion.create(values) : shapeQuestion.update(existing, values);

const QuestionForm = ({
    disabled,
    dirty,
    display,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    ...props
}: QuestionFormProps) => {
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
        validationSchema: questionTranslationValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" }),
    });

    const { handleCancel, handleCompleted, isCacheOn } = useUpsertActions<Question>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "Question",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<Question, QuestionCreateInput, QuestionUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointPostQuestion,
        endpointUpdate: endpointPutQuestion,
    });
    useSaveToCache({ isCacheOn, isCreate, values, objectId: values.id, objectType: "Question" });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<QuestionCreateInput | QuestionUpdateInput, Question>({
        disabled,
        existing,
        fetch,
        inputs: transformQuestionValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    return (
        <MaybeLargeDialog
            display={display}
            id="question-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateQuestion" : "UpdateQuestion")}
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={700}
            >
                <FormContainer>
                    <RelationshipList
                        isEditing={true}
                        objectType={"Question"}
                        sx={{ marginBottom: 4 }}
                    />
                    <FormSection>
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleDeleteLanguage}
                            handleCurrent={setLanguage}
                            languages={languages}
                        />
                        <TranslatedTextInput
                            autoFocus
                            fullWidth
                            label={t("Name")}
                            language={language}
                            name="name"
                            placeholder={t("NamePlaceholder")}
                        />
                        <TranslatedRichInput
                            language={language}
                            name="description"
                            placeholder={t("DescriptionPlaceholder")}
                            maxChars={16384}
                            minRows={3}
                            sxs={{
                                topBar: {
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
                    </FormSection>
                    <TagSelector name="tags" />
                </FormContainer>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                hideButtons={disabled}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
            />
        </MaybeLargeDialog>
    );
};

export const QuestionUpsert = ({
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: QuestionUpsertProps) => {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, setObject: setExisting } = useObjectFromUrl<Question, QuestionShape>({
        ...endpointGetQuestion,
        isCreate,
        objectType: "Question",
        overrideObject,
        transform: (existing) => questionInitialValues(session, existing),
    });
    const { canUpdate } = useMemo(() => getYou(existing), [existing]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={{
                ...existing,
                forObject: null,
            } as QuestionShape}
            onSubmit={noopSubmit}
            validate={async (values) => await validateFormValues(values, existing, isCreate, transformQuestionValues, questionValidation)}
        >
            {(formik) => <QuestionForm
                disabled={!(isCreate || canUpdate)}
                existing={existing}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                {...props}
                {...formik}
            />}
        </Formik>
    );
};
