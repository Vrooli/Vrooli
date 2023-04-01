import { useTheme } from "@mui/material";
import { Comment, CommentUpdateInput as CommentUpdateInputType } from "@shared/consts";
import { commentUpdate } from "api/generated/endpoints/comment_update";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from "api/utils";
import { CommentDialog } from "components/dialogs/CommentDialog/CommentDialog";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { CommentForm, commentInitialValues, validateCommentValues } from "forms/CommentForm/CommentForm";
import { useContext, useMemo, useRef } from "react";
import { useUpdateActions } from "utils/hooks/useUpdateActions";
import { useWindowSize } from "utils/hooks/useWindowSize";
import { SessionContext } from "utils/SessionContext";
import { shapeComment } from "utils/shape/models/comment";
import { CommentUpdateInputProps } from "../types";

/**
 * MarkdownInput/CommentContainer wrapper for creating comments
 */
export const CommentUpdateInput = ({
    comment,
    handleClose,
    language,
    objectId,
    objectType,
    onCommentUpdate,
    parent,
    zIndex,
}: CommentUpdateInputProps) => {
    const session = useContext(SessionContext);
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width < breakpoints.values.sm);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => commentInitialValues(session, objectType, objectId, language, comment), [comment, language, objectId, objectType, session]);
    const { handleCancel, handleUpdated } = useUpdateActions<Comment>('dialog', handleClose, onCommentUpdate);
    const [mutation, { loading: isUpdateLoading }] = useCustomMutation<Comment, CommentUpdateInputType>(commentUpdate);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={initialValues}
            onSubmit={(values, helpers) => {
                // If not logged in, open login dialog
                //TODO
                const input = shapeComment.update(comment, values);
                input !== undefined && mutationWrapper<Comment, CommentUpdateInputType>({
                    mutation,
                    input,
                    successCondition: (data) => data !== null,
                    successMessage: () => ({ key: 'CommentUpdated' }),
                    onSuccess: (data) => {
                        helpers.resetForm();
                        handleUpdated(data);
                    },
                    onError: () => { helpers.setSubmitting(false) },
                })
            }}
            validate={async (values) => await validateCommentValues(values, false)}
        >
            {(formik) => {
                if (isMobile) return <CommentDialog
                    isCreate={false}
                    isLoading={isUpdateLoading}
                    isOpen={true}
                    onCancel={handleClose}
                    parent={parent}
                    ref={formRef}
                    zIndex={zIndex + 1}
                    {...formik}
                />
                return <CommentForm
                    display="page"
                    isCreate={false}
                    isLoading={isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />
            }}
        </Formik>
    )
}