import { jsx as _jsx } from "react/jsx-runtime";
import { exists } from "@local/utils";
import { useTheme } from "@mui/material";
import { Formik } from "formik";
import { useContext, useMemo, useRef } from "react";
import { commentCreate } from "../../../api/generated/endpoints/comment_create";
import { commentUpdate } from "../../../api/generated/endpoints/comment_update";
import { useCustomMutation } from "../../../api/hooks";
import { mutationWrapper } from "../../../api/utils";
import { CommentForm, commentInitialValues, transformCommentValues, validateCommentValues } from "../../../forms/CommentForm/CommentForm";
import { useUpsertActions } from "../../../utils/hooks/useUpsertActions";
import { useWindowSize } from "../../../utils/hooks/useWindowSize";
import { SessionContext } from "../../../utils/SessionContext";
import { CommentDialog } from "../../dialogs/CommentDialog/CommentDialog";
export const CommentUpsertInput = ({ comment, language, objectId, objectType, onCancel, onCompleted, parent, zIndex, }) => {
    const session = useContext(SessionContext);
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width < breakpoints.values.sm);
    const formRef = useRef();
    const initialValues = useMemo(() => commentInitialValues(session, objectType, objectId, language, comment), [comment, language, objectId, objectType, session]);
    const { handleCancel, handleCompleted } = useUpsertActions("dialog", !exists(comment), onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useCustomMutation(commentCreate);
    const [update, { loading: isUpdateLoading }] = useCustomMutation(commentUpdate);
    const mutation = !exists(comment) ? create : update;
    return (_jsx(Formik, { enableReinitialize: true, initialValues: initialValues, onSubmit: (values, helpers) => {
            mutationWrapper({
                mutation,
                input: transformCommentValues(values, comment),
                successCondition: (data) => data !== null,
                successMessage: () => ({ key: "CommentUpdated" }),
                onSuccess: (data) => {
                    helpers.resetForm();
                    handleCompleted(data);
                },
                onError: () => { helpers.setSubmitting(false); },
            });
        }, validate: async (values) => await validateCommentValues(values, comment), children: (formik) => {
            if (isMobile)
                return _jsx(CommentDialog, { isCreate: !exists(comment), isLoading: isCreateLoading || isUpdateLoading, isOpen: true, onCancel: handleCancel, parent: parent, ref: formRef, zIndex: zIndex + 1, ...formik });
            return _jsx(CommentForm, { display: "page", isCreate: !exists(comment), isLoading: isCreateLoading || isUpdateLoading, isOpen: true, onCancel: handleCancel, ref: formRef, zIndex: zIndex, ...formik });
        } }));
};
//# sourceMappingURL=CommentUpsertInput.js.map