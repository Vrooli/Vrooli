import { camelCase, Comment, CommentCreateInput, CommentFor, CommentSearchInput, CommentSearchResult, CommentShape, commentTranslationValidation, CommentUpdateInput, commentValidation, DUMMY_ID, endpointsComment, noopSubmit, orDefault, Session, shapeComment, uuidValidate } from "@local/shared";
import { Box, useTheme } from "@mui/material";
import { useSubmitHelper } from "api/fetchWrapper.js";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons.js";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog.js";
import { TranslatedRichInput } from "components/inputs/RichInput/RichInput.js";
import { TopBar } from "components/navigation/TopBar.js";
import { MarkdownDisplay } from "components/text/MarkdownDisplay.js";
import { Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "hooks/forms.js";
import { useLazyFetch } from "hooks/useLazyFetch.js";
import { useTranslatedFields } from "hooks/useTranslatedFields.js";
import { useUpsertFetch } from "hooks/useUpsertFetch.js";
import { useWindowSize } from "hooks/useWindowSize.js";
import { SendIcon } from "icons/common.js";
import { useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { defaultYou, getDisplay, getYou } from "utils/display/listTools.js";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools.js";
import { validateFormValues } from "utils/validateFormValues.js";
import { SessionContext } from "../../../contexts.js";
import { CommentFormProps, CommentUpsertProps } from "./types.js";

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
    console.log("in transformCommentValues", values, existing, isCreate);
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
        endpointCreate: endpointsComment.createOne,
        endpointUpdate: endpointsComment.updateOne,
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
        endpointCreate: endpointsComment.createOne,
        endpointUpdate: endpointsComment.updateOne,
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

    const inputStyle = useMemo(function inputStyleMemo() {
        return {
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
        } as const;
    }, [palette]);

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
                        sxs={inputStyle}
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

    const [getData, { data: fetchedData, loading: isReadLoading }] = useLazyFetch<CommentSearchInput, CommentSearchResult>(endpointsComment.findMany);
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
