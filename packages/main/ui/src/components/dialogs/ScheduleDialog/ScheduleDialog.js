import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { DUMMY_ID } from "@local/uuid";
import { scheduleValidation } from "@local/validation";
import { Formik } from "formik";
import { useCallback, useRef } from "react";
import { scheduleCreate } from "../../../api/generated/endpoints/schedule_create";
import { scheduleUpdate } from "../../../api/generated/endpoints/schedule_update";
import { useCustomMutation } from "../../../api/hooks";
import { mutationWrapper } from "../../../api/utils";
import { ScheduleForm } from "../../../forms/ScheduleForm/ScheduleForm";
import { PubSub } from "../../../utils/pubsub";
import { validateAndGetYupErrors } from "../../../utils/shape/general";
import { shapeSchedule } from "../../../utils/shape/models/schedule";
import { DialogTitle } from "../DialogTitle/DialogTitle";
import { LargeDialog } from "../LargeDialog/LargeDialog";
const titleId = "schedule-dialog-title";
export const ScheduleDialog = ({ isCreate, isMutate, isOpen, onClose, onCreated, onUpdated, partialData, zIndex, }) => {
    const formRef = useRef();
    const [addMutation, { loading: addLoading }] = useCustomMutation(scheduleCreate);
    const [updateMutation, { loading: updateLoading }] = useCustomMutation(scheduleUpdate);
    const handleClose = useCallback((_, reason) => {
        formRef.current?.handleClose(onClose, reason !== "backdropClick");
    }, [onClose]);
    const transformValues = useCallback((values) => {
        return isCreate
            ? shapeSchedule.create(values)
            : shapeSchedule.update(partialData, values);
    }, [isCreate, partialData]);
    const validateFormValues = useCallback(async (values) => {
        const transformedValues = transformValues(values);
        const validationSchema = isCreate
            ? scheduleValidation.create({})
            : scheduleValidation.update({});
        const result = await validateAndGetYupErrors(validationSchema, transformedValues);
        return result;
    }, [isCreate, transformValues]);
    return (_jsx(_Fragment, { children: _jsxs(LargeDialog, { id: "schedule-dialog", onClose: handleClose, isOpen: isOpen, titleId: titleId, zIndex: zIndex, children: [_jsx(DialogTitle, { id: titleId, title: (isCreate) ? "Add Schedule" : "Update Schedule", onClose: handleClose }), _jsx(Formik, { enableReinitialize: true, initialValues: {
                        __typename: "Schedule",
                        id: DUMMY_ID,
                        startTime: null,
                        endTime: null,
                        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
                        exceptions: [],
                        labelsConnect: [],
                        labelsCreate: [],
                        recurrences: [],
                        ...partialData,
                    }, onSubmit: (values, helpers) => {
                        if (isMutate) {
                            const onSuccess = (data) => {
                                isCreate ? onCreated(data) : onUpdated(data);
                                helpers.resetForm();
                                onClose();
                            };
                            console.log("yeeeet", values, shapeSchedule.create(values));
                            if (!isCreate && (!partialData || !partialData.id)) {
                                PubSub.get().publishSnack({ messageKey: "ScheduleNotFound", severity: "Error" });
                                return;
                            }
                            mutationWrapper({
                                mutation: isCreate ? addMutation : updateMutation,
                                input: transformValues(values),
                                successMessage: () => ({ key: isCreate ? "ScheduleCreated" : "ScheduleUpdated" }),
                                successCondition: (data) => data !== null,
                                onSuccess,
                                onError: () => { helpers.setSubmitting(false); },
                            });
                        }
                        else {
                            onCreated({
                                ...values,
                                created_at: partialData?.created_at ?? new Date().toISOString(),
                                updated_at: partialData?.updated_at ?? new Date().toISOString(),
                            });
                            helpers.resetForm();
                            onClose();
                        }
                    }, validate: async (values) => await validateFormValues(values), children: (formik) => _jsx(ScheduleForm, { display: "dialog", isCreate: isCreate, isLoading: addLoading || updateLoading, isOpen: isOpen, onCancel: handleClose, ref: formRef, zIndex: zIndex, ...formik }) })] }) }));
};
//# sourceMappingURL=ScheduleDialog.js.map