import { Comment, CommentCreateInput, CommentUpdateInput, endpointPostComment, endpointPutComment, exists } from "@local/shared";
import { useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { CommentDialog } from "components/dialogs/CommentDialog/CommentDialog";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { CommentForm, commentInitialValues, transformCommentValues, validateCommentValues } from "forms/CommentForm/CommentForm";
import { useContext, useMemo, useRef } from "react";
import { MakeLazyRequest, useLazyFetch } from "utils/hooks/useLazyFetch";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { useWindowSize } from "utils/hooks/useWindowSize";
import { SessionContext } from "utils/SessionContext";
import { CommentUpsertInputProps } from "../types";

/**
 * MarkdownInput/CommentContainer wrapper for creating comments
 */
export const CommentUpsertInput = ({
    comment,
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

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => commentInitialValues(session, objectType, objectId, language, comment), [comment, language, objectId, objectType, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<Comment>("dialog", !exists(comment), onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useLazyFetch<CommentCreateInput, Comment>(endpointPostComment);
    const [update, { loading: isUpdateLoading }] = useLazyFetch<Comment, CommentUpdateInput>(endpointPutComment);
    const fetch = (!exists(comment) ? create : update) as MakeLazyRequest<CommentCreateInput | CommentUpdateInput, Comment>;

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
                    isOpen={true}
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
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />;
            }}
        </Formik>
    );
};
