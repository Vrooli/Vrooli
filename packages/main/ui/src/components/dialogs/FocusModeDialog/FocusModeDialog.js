import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { DUMMY_ID } from "@local/uuid";
import { focusModeValidation } from "@local/validation";
import { Formik } from "formik";
import { useCallback, useRef } from "react";
import { focusModeCreate } from "../../../api/generated/endpoints/focusMode_create";
import { focusModeUpdate } from "../../../api/generated/endpoints/focusMode_update";
import { useCustomMutation } from "../../../api/hooks";
import { mutationWrapper } from "../../../api/utils";
import { FocusModeForm } from "../../../forms/FocusModeForm/FocusModeForm";
import { PubSub } from "../../../utils/pubsub";
import { validateAndGetYupErrors } from "../../../utils/shape/general";
import { shapeFocusMode } from "../../../utils/shape/models/focusMode";
import { DialogTitle } from "../DialogTitle/DialogTitle";
import { LargeDialog } from "../LargeDialog/LargeDialog";
const titleId = "focus-mode-dialog-title";
export const FocusModeDialog = ({ isCreate, isOpen, onClose, onCreated, onUpdated, partialData, zIndex, }) => {
    const formRef = useRef();
    const [addMutation, { loading: addLoading }] = useCustomMutation(focusModeCreate);
    const [updateMutation, { loading: updateLoading }] = useCustomMutation(focusModeUpdate);
    const handleClose = useCallback((_, reason) => {
        formRef.current?.handleClose(onClose, reason !== "backdropClick");
    }, [onClose]);
    const transformValues = useCallback((values) => {
        return isCreate
            ? shapeFocusMode.create(values)
            : shapeFocusMode.update(partialData, values);
    }, [isCreate, partialData]);
    const validateFormValues = useCallback(async (values) => {
        console.log("validating a", values, focusModeValidation.create({}));
        const transformedValues = transformValues(values);
        console.log("validating b", transformedValues);
        const validationSchema = isCreate
            ? focusModeValidation.create({})
            : focusModeValidation.update({});
        console.log("validating c", validationSchema);
        const result = await validateAndGetYupErrors(validationSchema, transformedValues);
        console.log("validating d", result);
        return result;
    }, [isCreate, transformValues]);
    return (_jsx(_Fragment, { children: _jsxs(LargeDialog, { id: "focus-mode-dialog", onClose: handleClose, isOpen: isOpen, titleId: titleId, zIndex: zIndex, children: [_jsx(DialogTitle, { id: titleId, title: (isCreate) ? "Add Focus Mode" : "Update Focus Mode", onClose: handleClose }), _jsx(Formik, { enableReinitialize: true, initialValues: {
                        __typename: "FocusMode",
                        id: DUMMY_ID,
                        description: "",
                        name: "",
                        reminderList: {
                            __typename: "ReminderList",
                            id: DUMMY_ID,
                            reminders: [],
                        },
                        resourceList: {
                            __typename: "ResourceList",
                            id: DUMMY_ID,
                            resources: [],
                        },
                        filters: [],
                        schedule: null,
                        ...partialData,
                    }, onSubmit: (values, helpers) => {
                        const onSuccess = (data) => {
                            isCreate ? onCreated(data) : onUpdated(data);
                            helpers.resetForm();
                            onClose();
                        };
                        console.log("yeeeet", values, shapeFocusMode.create(values));
                        const isCreating = isCreate;
                        if (!isCreating && (!partialData || !partialData.id)) {
                            PubSub.get().publishSnack({ messageKey: "NotFound", severity: "Error" });
                            return;
                        }
                        mutationWrapper({
                            mutation: isCreating ? addMutation : updateMutation,
                            input: transformValues(values),
                            successMessage: () => ({ key: isCreating ? "FocusModeCreated" : "FocusModeUpdated" }),
                            successCondition: (data) => data !== null,
                            onSuccess,
                            onError: () => { helpers.setSubmitting(false); },
                        });
                    }, validate: async (values) => await validateFormValues(values), children: (formik) => _jsx(FocusModeForm, { display: "dialog", isCreate: isCreate, isLoading: addLoading || updateLoading, isOpen: isOpen, onCancel: handleClose, ref: formRef, zIndex: zIndex, ...formik }) })] }) }));
};
//# sourceMappingURL=FocusModeDialog.js.map