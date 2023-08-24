import { Comment, CommentCreateInput, CommentUpdateInput, endpointPostComment, endpointPutComment, exists } from "@local/shared";
import { useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { CommentDialog } from "components/dialogs/CommentDialog/CommentDialog";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { CommentForm, commentInitialValues, transformCommentValues, validateCommentValues } from "forms/CommentForm/CommentForm";
import { useFormDialog } from "hooks/useFormDialog";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useWindowSize } from "hooks/useWindowSize";
import { useContext, useMemo } from "react";
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
}: CommentUpsertInputProps) => {
    const session = useContext(SessionContext);
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width < breakpoints.values.sm);

    const initialValues = useMemo(() => commentInitialValues(session, objectType, objectId, language, comment), [comment, language, objectId, objectType, session]);

    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<Comment, CommentCreateInput, CommentUpdateInput>({
        display: "dialog", // Set this to dialog, since it's more correct that page (there is no option for in-page yet)
        endpointCreate: endpointPostComment,
        endpointUpdate: endpointPutComment,
        isCreate: !exists(comment),
        onCancel,
        onCompleted,
    });
    const { formRef, handleClose } = useFormDialog({ handleCancel });

    return (
        <Formik
            enableReinitialize={true}
            initialValues={initialValues}
            onSubmit={(values, helpers) => {
                // If not logged in, open login dialog
                //TODO
                fetchLazyWrapper<CommentCreateInput | CommentUpdateInput, Comment>({
                    fetch,
                    inputs: transformCommentValues(values, initialValues, !exists(comment)),
                    successCondition: (data) => data !== null,
                    successMessage: () => ({ messageKey: "CommentUpdated" }),
                    onSuccess: (data) => {
                        helpers.resetForm();
                        handleCompleted(data);
                    },
                    onCompleted: () => { helpers.setSubmitting(false); },
                });
            }}
            validate={async (values) => await validateCommentValues(values, initialValues, !exists(comment))}
        >
            {(formik) => {
                if (isMobile) return <CommentDialog
                    isCreate={!exists(comment)}
                    isLoading={isCreateLoading || isUpdateLoading}
                    isOpen={isOpen}
                    onCancel={handleCancel}
                    onClose={handleClose}
                    parent={parent}
                    ref={formRef}
                    {...formik}
                />;
                return <CommentForm
                    display="page"
                    isCreate={!exists(comment)}
                    isLoading={isCreateLoading || isUpdateLoading}
                    isOpen={isOpen}
                    onCancel={handleCancel}
                    onClose={handleClose}
                    ref={formRef}
                    {...formik}
                />;
            }}
        </Formik>
    );
};
