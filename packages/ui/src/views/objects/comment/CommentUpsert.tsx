import { useTheme } from "@mui/material";
import { camelCase, commentFormConfig, commentValidation, createTransformFunction, endpointsComment, noopSubmit, validatePK, type Comment, type CommentSearchInput, type CommentSearchResult, type CommentShape } from "@vrooli/shared";
import { Formik } from "formik";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { Dialog } from "../../../components/dialogs/Dialog/Dialog.js";
import { TranslatedAdvancedInput } from "../../../components/inputs/AdvancedInput/AdvancedInput.js";
import { Box } from "../../../components/layout/Box.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { MarkdownDisplay } from "../../../components/text/MarkdownDisplay.js";
import { SessionContext } from "../../../contexts/session.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useLazyFetch } from "../../../hooks/useFetch.js";
import { useIsMobile } from "../../../hooks/useIsMobile.js";
import { useStandardUpsertForm } from "../../../hooks/useStandardUpsertForm.js";
import { useWindowSize } from "../../../hooks/useWindowSize.js";
import { defaultYou, getDisplay, getYou } from "../../../utils/display/listTools.js";
import { type CommentFormProps, type CommentUpsertProps } from "./types.js";

function CommentForm({
    disabled,
    existing,
    handleUpdate,
    isCreate,
    isReadLoading,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    ...props
}: CommentFormProps) {
    const display = "page";
    const { palette } = useTheme();
    const { t } = useTranslation();

    const inputFeatures = useMemo(() => ({ maxChars: 32768, minRowsCollapsed: 3, maxRowsCollapsed: 15 }), []);
    
    const actionButtons = useMemo(() => [{
        iconInfo: { name: "Send", type: "Common" } as const,
        onClick: () => {
            const message = values.translations.find(t => t.language === language)?.text;
            if (!message || message.trim().length === 0) return;
            onSubmit();
        },
    }], [values.translations, language, onSubmit]);
    
    const inputSxs = useMemo(() => ({
        root: {
            width: "100%",
            background: palette.primary.dark,
            color: palette.primary.contrastText,
            borderRadius: 1,
            overflow: "overlay",
            marginTop: 1,
        },
        topBar: { borderRadius: 0 },
        inputRoot: { background: palette.background.paper },
    }), [palette]);

    // Use the standardized form hook
    const {
        isLoading,
        onSubmit,
        language,
    } = useStandardUpsertForm({
        objectType: "Comment",
        validation: commentFormConfig.validation.schema,
        translationValidation: commentFormConfig.validation.translationSchema,
        transformFunction: createTransformFunction(commentFormConfig),
        endpoints: commentFormConfig.endpoints,
    }, {
        values,
        existing,
        isCreate,
        display: "Dialog", // Comments are typically in dialogs
        disabled,
        isReadLoading,
        isSubmitting: props.isSubmitting,
        handleUpdate,
        setSubmitting: props.setSubmitting,
        onCancel,
        onCompleted,
        onDeleted,
        onClose,
    });

    return (
        <>
            <BaseForm
                display={display}
                isLoading={isLoading}
                style={{ paddingBottom: 0 }}
            >
                <TranslatedAdvancedInput
                    disabled={isLoading}
                    language={language}
                    name="text"
                    placeholder={t("PleaseBeNice")}
                    features={inputFeatures}
                    actionButtons={actionButtons}
                    sxs={inputSxs}
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
/**
 * Dialog for creating/updating a comment. 
 * Only used on mobile; desktop displays RichInput at top of 
 * CommentContainer
 */
export function CommentDialog({
    disabled,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    parent,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    ...props
}: CommentFormProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const isMobile = useIsMobile();

    // Use the standardized form hook
    const {
        isLoading,
        handleCancel,
        onSubmit,
        language,
        translationErrors,
    } = useStandardUpsertForm({
        objectType: "Comment",
        validation: commentFormConfig.validation.schema,
        translationValidation: commentFormConfig.validation.translationSchema,
        transformFunction: createTransformFunction(commentFormConfig),
        endpoints: commentFormConfig.endpoints,
    }, {
        values,
        existing,
        isCreate,
        display: "Dialog",
        disabled,
        isReadLoading,
        isSubmitting: props.isSubmitting,
        handleUpdate,
        setSubmitting: props.setSubmitting,
        onCancel,
        onCompleted,
        onDeleted,
        onClose,
    });

    const { subtitle: parentText } = useMemo(() => getDisplay(parent, [language]), [language, parent]);

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

    const baseFormStyle = useMemo(() => ({
        width: "min(700px, 100vw)",
        paddingBottom: 0,
    }), []);

    const markdownSx = useMemo(() => ({
        padding: "16px",
        color: palette.background.textSecondary,
    }), [palette]);

    if (isMobile) {
        return (
            <Dialog
                isOpen={Boolean(isOpen)}
                onClose={onClose || (() => {})}
                size="full"
            >
                <TopBar
                    display="Dialog"
                    onClose={onClose}
                    title={t(isCreate ? "AddComment" : "EditComment")}
                    titleId={titleId}
                />
                <BaseForm
                    display="Dialog"
                    isLoading={isLoading}
                    style={baseFormStyle}
                >
                    <Box>
                        {parent && <MarkdownDisplay
                            content={parentText}
                            sx={markdownSx}
                        />}
                        <TranslatedAdvancedInput
                            language={language}
                            name="text"
                            placeholder={t("PleaseBeNice")}
                            minRows={10}
                            sxs={inputStyle}
                        />
                    </Box>
                    <BottomActionsButtons
                        display={"Dialog"}
                        errors={translationErrors}
                        hideButtons={disabled}
                        isCreate={isCreate}
                        loading={isLoading}
                        onCancel={handleCancel}
                        onSetSubmitting={props.setSubmitting}
                        onSubmit={onSubmit}
                    />
                </BaseForm>
            </Dialog>
        );
    }
    return (
        <Dialog
            isOpen={Boolean(isOpen)}
            onClose={onClose || (() => {})}
            size="md"
        >
            <TopBar
                display="Dialog"
                onClose={onClose}
                title={t(isCreate ? "AddComment" : "EditComment")}
                titleId={titleId}
            />
            <BaseForm
                display="Dialog"
                isLoading={isLoading}
                style={baseFormStyle}
            >
                <Box>
                    {parent && <MarkdownDisplay
                        content={parentText}
                        sx={markdownSx}
                    />}
                    <TranslatedAdvancedInput
                        language={language}
                        name="text"
                        placeholder={t("PleaseBeNice")}
                        minRows={10}
                        sxs={inputStyle}
                    />
                </Box>
                <BottomActionsButtons
                    display={"Dialog"}
                    errors={translationErrors}
                    hideButtons={disabled}
                    isCreate={isCreate}
                    loading={isLoading}
                    onCancel={handleCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={onSubmit}
                />
            </BaseForm>
        </Dialog>
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
    ...props
}: CommentUpsertProps) {
    const session = useContext(SessionContext);
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width < breakpoints.values.sm);

    const [getData, { data: fetchedData, loading: isReadLoading }] = useLazyFetch<CommentSearchInput, CommentSearchResult>(endpointsComment.findMany);
    useEffect(() => {
        if (!validatePK(objectId)) return;
        getData({ [`${camelCase(objectType)}Id`]: objectId });
    }, [getData, objectId, objectType]);

    const [existing, setExisting] = useState<CommentShape>(commentFormConfig.transformations.getInitialValues(session, {
        commentedOn: { __typename: objectType, id: objectId },
        translations: [{ __typename: "CommentTranslation" as const, id: "", language, text: "" }],
    }));
    useEffect(() => {
        const comment = commentFromSearch(fetchedData);
        if (!comment) return;
        setExisting(commentFormConfig.transformations.apiResultToShape(comment));
    }, [fetchedData, language, objectType, objectId, session]);

    const permissions = useMemo(() => commentFromSearch(fetchedData) ? getYou(commentFromSearch(fetchedData)) : defaultYou, [fetchedData]);

    // Simple validation for the main form wrapper
    const validateValues = useCallback(async (values: CommentShape) => {
        try {
            const schema = isCreate ? commentValidation.create({ env: process.env.NODE_ENV }) : commentValidation.update({ env: process.env.NODE_ENV });
            await schema.validate(values, { abortEarly: false });
            return {};
        } catch (error: unknown) {
            const errors: Record<string, string> = {};
            if (error && typeof error === "object" && "inner" in error) {
                (error as { inner: Array<{ path?: string; message: string }> }).inner.forEach((err) => {
                    if (err.path) errors[err.path] = err.message;
                });
            }
            return errors;
        }
    }, [isCreate]);

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
