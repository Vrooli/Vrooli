import { Comment, CommentCreateInput, CommentUpdateInput, endpointPostComment, endpointPutComment, exists } from "@local/shared";
import { useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { CommentDialog } from "components/dialogs/CommentDialog/CommentDialog";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { CommentForm, commentInitialValues, transformCommentValues, validateCommentValues } from "forms/CommentForm/CommentForm";
import { useContext, useMemo, useRef } from "react";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { useWindowSize } from "utils/hooks/useWindowSize";
import { SessionContext } from "utils/SessionContext";
import { CommentUpsertInputProps } from "../types";

/**
 * MarkdownInput/CommentContainer wrapper for creating comments
 */
export const CommentUpsertInput = ({
    comment,
    isOpen,
    language,
    objectId,
    objectType,
    onCancel,
    onCompleted,
    parent,
    zIndex,
}: CommentUpsertInputProps) => {
    const session = useContext(SessionContext);
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width < breakpoints.values.sm);

    const initialValues = useMemo(() => commentInitialValues(session, objectType, objectId, language, comment), [comment, language, objectId, objectType, session]);

    const formRef = useRef<BaseFormRef>();
    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<Comment, CommentCreateInput, CommentUpdateInput>({
        display: isMobile ? "dialog" : "page",
        endpointCreate: endpointPostComment,
        endpointUpdate: endpointPutComment,
        isCreate: !exists(comment),
        onCancel,
        onCompleted,
    });

    return (
        <Formik
            enableReinitialize={true}
            initialValues={initialValues}
            onSubmit={(values, helpers) => {
                // If not logged in, open login dialog
                //TODO
                fetchLazyWrapper<CommentCreateInput | CommentUpdateInput, Comment>({
                    fetch,
                    inputs: transformCommentValues(values, comment),
                    successCondition: (data) => data !== null,
                    successMessage: () => ({ messageKey: "CommentUpdated" }),
                    onSuccess: (data) => {
                        helpers.resetForm();
                        handleCompleted(data);
                    },
                    onError: () => { helpers.setSubmitting(false); },
                });
            }}
            validate={async (values) => await validateCommentValues(values, comment)}
        >
            {(formik) => {
                if (isMobile) return <CommentDialog
                    isCreate={!exists(comment)}
                    isLoading={isCreateLoading || isUpdateLoading}
                    isOpen={isOpen}
                    onCancel={handleCancel}
                    parent={parent}
                    ref={formRef}
                    zIndex={zIndex + 1}
                    {...formik}
                />;
                return <CommentForm
                    display="page"
                    isCreate={!exists(comment)}
                    isLoading={isCreateLoading || isUpdateLoading}
                    isOpen={isOpen}
                    onCancel={handleCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />;
            }}
        </Formik>
    );
};
