import { useTheme } from "@mui/material";
import { Comment, CommentCreateInput as CommentCreateInputType } from "@shared/consts";
import { commentCreate } from "api/generated/endpoints/comment_create";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from "api/utils";
import { CommentDialog } from "components/dialogs/CommentDialog/CommentDialog";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { CommentForm, commentInitialValues, validateCommentValues } from "forms/CommentForm/CommentForm";
import { useContext, useMemo, useRef } from "react";
import { useCreateActions } from "utils/hooks/useCreateActions";
import { useWindowSize } from "utils/hooks/useWindowSize";
import { SessionContext } from "utils/SessionContext";
import { shapeComment } from "utils/shape/models/comment";
import { CommentCreateInputProps } from "../types";

/**
 * MarkdownInput/CommentContainer wrapper for creating comments
 */
export const CommentCreateInput = ({
    handleClose,
    language,
    objectId,
    objectType,
    onCommentAdd,
    parent,
    zIndex,
}: CommentCreateInputProps) => {
    const session = useContext(SessionContext);
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width < breakpoints.values.sm);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => commentInitialValues(session, objectType, objectId, language), [language, objectId, objectType]);
    const { handleCancel, handleCreated } = useCreateActions<Comment>('dialog', handleClose, onCommentAdd);
    const [mutation, { loading: isLoading }] = useCustomMutation<Comment, any>(commentCreate);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={initialValues}
            onSubmit={(values, helpers) => {
                // If not logged in, open login dialog
                //TODO
                mutationWrapper<Comment, CommentCreateInputType>({
                    mutation,
                    input: shapeComment.create(values),
                    successCondition: (data) => data !== null,
                    successMessage: () => ({ key: 'CommentCreated' }),
                    onSuccess: (data) => {
                        helpers.resetForm();
                        handleCreated(data);
                    },
                    onError: () => { helpers.setSubmitting(false) },
                })
            }}
            validate={async (values) => await validateCommentValues(values, true)}
        >
            {(formik) => {
                if (isMobile) return <CommentDialog
                    isCreate={true}
                    isLoading={isLoading}
                    isOpen={true}
                    onCancel={handleClose}
                    parent={parent}
                    ref={formRef}
                    zIndex={zIndex + 1}
                    {...formik}
                />
                return <CommentForm
                    display="page"
                    isCreate={true}
                    isLoading={isLoading}
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