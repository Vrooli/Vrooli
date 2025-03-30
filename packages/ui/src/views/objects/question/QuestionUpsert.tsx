import { DUMMY_ID, endpointsQuestion, LlmTask, noopSubmit, orDefault, Question, QuestionCreateInput, QuestionShape, questionTranslationValidation, QuestionUpdateInput, questionValidation, Session, shapeQuestion } from "@local/shared";
import { useTheme } from "@mui/material";
import { Formik } from "formik";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useSubmitHelper } from "../../../api/fetchWrapper.js";
import { AutoFillButton } from "../../../components/buttons/AutoFillButton.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { MaybeLargeDialog } from "../../../components/dialogs/LargeDialog/LargeDialog.js";
import { LanguageInput } from "../../../components/inputs/LanguageInput/LanguageInput.js";
import { TranslatedRichInput } from "../../../components/inputs/RichInput/RichInput.js";
import { TagSelector } from "../../../components/inputs/TagSelector/TagSelector.js";
import { TranslatedTextInput } from "../../../components/inputs/TextInput/TextInput.js";
import { RelationshipList } from "../../../components/lists/RelationshipList/RelationshipList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { SessionContext } from "../../../contexts/session.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "../../../hooks/forms.js";
import { createUpdatedTranslations, getAutoFillTranslationData, useAutoFill, UseAutoFillProps } from "../../../hooks/tasks.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useTranslatedFields } from "../../../hooks/useTranslatedFields.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { FormContainer, FormSection } from "../../../styles.js";
import { combineErrorsWithTranslations, getUserLanguages } from "../../../utils/display/translationTools.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { QuestionFormProps, QuestionUpsertProps } from "./types.js";

export function questionInitialValues(
    session: Session | undefined,
    existing?: Partial<Question> | null | undefined,
): QuestionShape {
    return {
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
    };
}

function transformQuestionValues(values: QuestionShape, existing: QuestionShape, isCreate: boolean) {
    return isCreate ? shapeQuestion.create(values) : shapeQuestion.update(existing, values);
}

const relationshipListStyle = { marginBottom: 4 } as const;

function QuestionForm({
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
}: QuestionFormProps) {
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
        validationSchema: questionTranslationValidation.create({ env: process.env.NODE_ENV }),
    });

    const { handleCancel, handleCompleted } = useUpsertActions<Question>({
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
        endpointCreate: endpointsQuestion.createOne,
        endpointUpdate: endpointsQuestion.updateOne,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "Question" });

    const getAutoFillInput = useCallback(function getAutoFillInput() {
        return {
            ...getAutoFillTranslationData(values, language),
        };
    }, [language, values]);

    const shapeAutoFillResult = useCallback(function shapeAutoFillResultCallback(data: Parameters<UseAutoFillProps["shapeAutoFillResult"]>[0]) {
        const originalValues = { ...values };
        const { updatedTranslations, rest } = createUpdatedTranslations(values, data, language, ["name", "description"]);
        delete rest.id;
        const updatedValues = {
            ...values,
            translations: updatedTranslations,
        };
        return { originalValues, updatedValues };
    }, [language, values]);

    const { autoFill, isAutoFillLoading } = useAutoFill({
        getAutoFillInput,
        shapeAutoFillResult,
        handleUpdate,
        task: isCreate ? LlmTask.QuestionAdd : LlmTask.QuestionUpdate,
    });

    const isLoading = useMemo(() => isAutoFillLoading || isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isAutoFillLoading, isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

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
                        sx={relationshipListStyle}
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
                sideActionButtons={<AutoFillButton
                    handleAutoFill={autoFill}
                    isAutoFillLoading={isAutoFillLoading}
                />}
            />
        </MaybeLargeDialog>
    );
}

export function QuestionUpsert({
    display,
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: QuestionUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<Question, QuestionShape>({
        ...endpointsQuestion.findOne,
        disabled: display === "dialog" && isOpen !== true,
        isCreate,
        objectType: "Question",
        overrideObject,
        transform: (existing) => questionInitialValues(session, existing),
    });

    async function validateValues(values: QuestionShape) {
        return await validateFormValues(values, existing, isCreate, transformQuestionValues, questionValidation);
    }

    return (
        <Formik
            enableReinitialize={true}
            initialValues={{
                ...existing,
                forObject: null,
            } as QuestionShape}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <QuestionForm
                disabled={!(isCreate || permissions.canUpdate)}
                display={display}
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
}
