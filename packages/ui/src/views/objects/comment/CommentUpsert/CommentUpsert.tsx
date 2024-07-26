import { camelCase, Comment, CommentCreateInput, CommentFor, CommentSearchInput, CommentSearchResult, CommentShape, commentTranslationValidation, CommentUpdateInput, commentValidation, DUMMY_ID, endpointGetComments, endpointPostComment, endpointPutComment, noopSubmit, orDefault, Session, shapeComment, uuidValidate } from "@local/shared";
import { Box, useTheme } from "@mui/material";
import { useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TranslatedRichInput } from "components/inputs/RichInput/RichInput";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { useWindowSize } from "hooks/useWindowSize";
import { SendIcon } from "icons";
import { useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { defaultYou, getDisplay, getYou } from "utils/display/listTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { validateFormValues } from "utils/validateFormValues";
import { CommentFormProps, CommentUpsertProps } from "../types";

export function commentInitialValues(
    session: Session | undefined,
    objectType: CommentFor,
    objectId: string,
    language: string,
    existing?: Partial<CommentShape> | null | undefined,
): CommentShape {
    return {
        __typename: "Comment" as const,
        id: DUMMY_ID,
        commentedOn: { __typename: objectType, id: objectId },
        ...existing,
        translations: orDefault(existing?.translations, [{
            __typename: "CommentTranslation" as const,
            id: DUMMY_ID,
            language,
            text: "",
        }]),
    };
}

export function transformCommentValues(values: CommentShape, existing: CommentShape, isCreate: boolean) {
    return isCreate ? shapeComment.create(values) : shapeComment.update(existing, values);
}

function CommentForm({
    disabled,
    dirty,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    objectId,
    objectType,
    parent,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    ...props
}: CommentFormProps) {
    const session = useContext(SessionContext);
    const display = "page";
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Handle translations
    const {
        language,
        translationErrors,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        validationSchema: commentTranslationValidation.create({ env: process.env.NODE_ENV }),
    });

    const { handleCancel, handleCompleted } = useUpsertActions<Comment>({
        isCreate,
        objectType: "Comment",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<Comment, CommentCreateInput, CommentUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointPostComment,
        endpointUpdate: endpointPutComment,
    });
    useSaveToCache({ isCreate: false, values, objectId, objectType: "Comment" }); // Tied to ID of object being commented on, which is always already created
    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<CommentCreateInput | CommentUpdateInput, Comment>({
        disabled,
        existing,
        fetch,
        inputs: transformCommentValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    return (
        <>
            <BaseForm
                display={display}
                isLoading={isLoading}
                style={{ paddingBottom: 0 }}
            >
                <TranslatedRichInput
                    disabled={isLoading}
                    language={language}
                    name="text"
                    placeholder={t("PleaseBeNice")}
                    maxChars={32768}
                    minRows={3}
                    maxRows={15}
                    actionButtons={[{
                        Icon: SendIcon,
                        onClick: () => {
                            const message = values.translations.find(t => t.language === language)?.text;
                            if (!message || message.trim().length === 0) return;
                            onSubmit();
                        },
                    }]}
                    sxs={{
                        root: {
                            width: "100%",
                            background: palette.primary.dark,
                            color: palette.primary.contrastText,
                            borderRadius: 1,
                            overflow: "overlay",
                            marginTop: 1,
                        }
                        ,
                        topBar: { borderRadius: 0 },
                        inputRoot: { background: palette.background.paper },
                    }}
                />
            </BaseForm>
        </>
    );
}

const titleId = "comment-dialog-title";

/**
 * Dialog for creating/updating a comment. 
 * Only used on mobile; desktop displays RichInput at top of 
 * CommentContainer
 */
export function CommentDialog({
    disabled,
    dirty,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    objectId,
    objectType,
    parent,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    ...props
}: CommentFormProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Handle translations
    const {
        language,
        translationErrors,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        validationSchema: commentTranslationValidation.create({ env: process.env.NODE_ENV }),
    });

    const { subtitle: parentText } = useMemo(() => getDisplay(parent, [language]), [language, parent]);

    const { handleCancel, handleCompleted } = useUpsertActions<Comment>({
        isCreate,
        objectType: "Comment",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<Comment, CommentCreateInput, CommentUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointPostComment,
        endpointUpdate: endpointPutComment,
    });
    useSaveToCache({ isCreate: false, values, objectId, objectType: "Comment" }); // Tied to ID of object being commented on, which is always already created
    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<CommentCreateInput | CommentUpdateInput, Comment>({
        disabled,
        existing,
        fetch,
        inputs: transformCommentValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    return (
        <LargeDialog
            id="comment-dialog"
            isOpen={Boolean(isOpen)}
            onClose={onClose!}
            titleId={titleId}
        >
            <TopBar
                display="dialog"
                onClose={onClose}
                title={t(isCreate ? "AddComment" : "EditComment")}
                titleId={titleId}
            />
            <BaseForm
                display="dialog"
                isLoading={isLoading}
                style={{
                    width: "min(700px, 100vw)",
                    paddingBottom: 0,
                }}
            >
                <Box>
                    {parent && <MarkdownDisplay
                        content={parentText}
                        sx={{
                            padding: "16px",
                            color: palette.background.textSecondary,
                        }}
                    />}
                    <TranslatedRichInput
                        language={language}
                        name="text"
                        placeholder={t("PleaseBeNice")}
                        minRows={10}
                        sxs={{
                            topBar: {
                                borderRadius: 0,
                                background: palette.primary.main,
                                position: "sticky",
                                top: 0,
                            },
                            root: {
                                height: "100%",
                                position: "relative",
                                maxWidth: "800px",
                                marginBottom: "64px",
                            },
                            textArea: {
                                borderRadius: 0,
                                resize: "none",
                                height: "100%",
                                overflow: "hidden", // Container handles scrolling
                                background: palette.background.paper,
                                border: "none",
                            },
                        }}
                    />
                </Box>
                <BottomActionsButtons
                    display={"dialog"}
                    errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                    hideButtons={disabled}
                    isCreate={isCreate}
                    loading={isLoading}
                    onCancel={handleCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={onSubmit}
                />
            </BaseForm>
        </LargeDialog>
    );
}

function commentFromSearch(searchResult: CommentSearchResult | undefined): Comment | null {
    if (!searchResult) return null;
    const comment = Array.isArray(searchResult.threads) && searchResult.threads.length > 0 ? searchResult.threads[0].comment : null;
    return comment;
}

/**
 * RichInput/CommentContainer wrapper for creating comments
 */
export function CommentUpsert({
    isCreate,
    isOpen,
    language,
    objectId,
    objectType,
    overrideObject,
    ...props
}: CommentUpsertProps) {
    const session = useContext(SessionContext);
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width < breakpoints.values.sm);

    const [getData, { data: fetchedData, loading: isReadLoading }] = useLazyFetch<CommentSearchInput, CommentSearchResult>(endpointGetComments);
    useEffect(() => {
        if (!uuidValidate(objectId)) return;
        getData({ [`${camelCase(objectType)}Id`]: objectId });
    }, [getData, objectId, objectType]);

    const [existing, setExisting] = useState<CommentShape>(commentInitialValues(session, objectType, objectId, language, {}));
    useEffect(() => {
        const comment = commentFromSearch(fetchedData);
        if (!comment) return;
        setExisting(commentInitialValues(session, objectType, objectId, language, comment));
    }, [fetchedData, language, objectType, objectId, session]);

    const permissions = useMemo(() => commentFromSearch(fetchedData) ? getYou(commentFromSearch(fetchedData)) : defaultYou, [fetchedData]);

    async function validateValues(values: CommentShape) {
        return await validateFormValues(values, existing, isCreate, transformCommentValues, commentValidation);
    }

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => {
                if (isMobile) return <CommentDialog
                    disabled={!(isCreate || permissions.canUpdate)}
                    existing={existing}
                    handleUpdate={setExisting}
                    isCreate={isCreate}
                    isReadLoading={isReadLoading}
                    isOpen={isOpen}
                    language={language}
                    objectId={objectId}
                    objectType={objectType}
                    {...props}
                    {...formik}
                />;
                return <CommentForm
                    disabled={!(isCreate || permissions.canUpdate)}
                    existing={existing}
                    handleUpdate={setExisting}
                    isCreate={isCreate}
                    isReadLoading={isReadLoading}
                    isOpen={isOpen}
                    language={language}
                    objectId={objectId}
                    objectType={objectType}
                    {...props}
                    {...formik}
                />;
            }}
        </Formik>
    );
}
