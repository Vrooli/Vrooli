import { Schedule, ScheduleCreateInput, ScheduleUpdateInput } from "@shared/consts";
import { DUMMY_ID } from "@shared/uuid";
import { scheduleValidation } from "@shared/validation";
import { scheduleCreate } from "../../../api/generated/endpoints/schedule_create";
import { scheduleUpdate } from "../../../api/generated/endpoints/schedule_update";
import { useCustomMutation } from "../../../api/hooks";
import { mutationWrapper } from "../../../api/utils";
import { Formik } from "formik";
import { BaseFormRef } from "../../../forms/BaseForm/BaseForm";
import { ScheduleForm } from "../../../forms/ScheduleForm/ScheduleForm";
import { useCallback, useRef } from "react";
import { PubSub } from "../../../utils/pubsub";
import { validateAndGetYupErrors } from "../../../utils/shape/general";
import { ScheduleShape, shapeSchedule } from "../../../utils/shape/models/schedule";
import { DialogTitle } from "../DialogTitle/DialogTitle";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { ScheduleDialogProps } from "../types";

const titleId = "schedule-dialog-title";

export const ScheduleDialog = ({
    isCreate,
    isMutate,
    isOpen,
    onClose,
    onCreated,
    onUpdated,
    partialData,
    zIndex,
}: ScheduleDialogProps) => {
    const formRef = useRef<BaseFormRef>();
    const [addMutation, { loading: addLoading }] = useCustomMutation<Schedule, ScheduleCreateInput>(scheduleCreate);
    const [updateMutation, { loading: updateLoading }] = useCustomMutation<Schedule, ScheduleUpdateInput>(scheduleUpdate);

    const handleClose = useCallback((_?: unknown, reason?: "backdropClick" | "escapeKeyDown") => {
        // Confirm dialog is dirty and closed by clicking outside
        formRef.current?.handleClose(onClose, reason !== "backdropClick");
    }, [onClose]);

    const transformValues = useCallback((values: ScheduleShape) => {
        return isCreate
            ? shapeSchedule.create(values)
            : shapeSchedule.update(partialData as any, values);

    }, [isCreate, partialData]);

    const validateFormValues = useCallback(
        async (values: ScheduleShape) => {
            const transformedValues = transformValues(values);
            const validationSchema = isCreate
                ? scheduleValidation.create({})
                : scheduleValidation.update({});
            const result = await validateAndGetYupErrors(validationSchema, transformedValues);
            return result;
        },
        [isCreate, transformValues],
    );

    return (
        <>
            {/*  Main content */}
            <LargeDialog
                id="schedule-dialog"
                onClose={handleClose}
                isOpen={isOpen}
                titleId={titleId}
                zIndex={zIndex}
            >
                <DialogTitle
                    id={titleId}
                    title={(isCreate) ? "Add Schedule" : "Update Schedule"}
                    onClose={handleClose}
                />
                <Formik
                    enableReinitialize={true}
                    initialValues={{
                        __typename: "Schedule" as const,
                        id: DUMMY_ID,
                        startTime: null,
                        endTime: null,
                        // Default to current timezone
                        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
                        exceptions: [],
                        labelsConnect: [],
                        labelsCreate: [],
                        recurrences: [],
                        ...partialData,
                    } as ScheduleShape}
                    onSubmit={(values, helpers) => {
                        if (isMutate) {
                            const onSuccess = (data: Schedule) => {
                                isCreate ? onCreated(data) : onUpdated(data);
                                helpers.resetForm();
                                onClose();
                            };
                            console.log("yeeeet", values, shapeSchedule.create(values));
                            if (!isCreate && (!partialData || !partialData.id)) {
                                PubSub.get().publishSnack({ messageKey: "ScheduleNotFound", severity: "Error" });
                                return;
                            }
                            mutationWrapper<Schedule, ScheduleCreateInput | ScheduleUpdateInput>({
                                mutation: isCreate ? addMutation : updateMutation,
                                input: transformValues(values),
                                successMessage: () => ({ key: isCreate ? "ScheduleCreated" : "ScheduleUpdated" }),
                                successCondition: (data) => data !== null,
                                onSuccess,
                                onError: () => { helpers.setSubmitting(false); },
                            });
                        } else {
                            onCreated({
                                ...values,
                                created_at: partialData?.created_at ?? new Date().toISOString(),
                                updated_at: partialData?.updated_at ?? new Date().toISOString(),
                            } as Schedule);
                            helpers.resetForm();
                            onClose();
                        }
                    }}
                    validate={async (values) => await validateFormValues(values)}
                >
                    {(formik) => <ScheduleForm
                        display="dialog"
                        isCreate={isCreate}
                        isLoading={addLoading || updateLoading}
                        isOpen={isOpen}
                        onCancel={handleClose}
                        ref={formRef}
                        zIndex={zIndex}
                        {...formik}
                    />}
                </Formik>
            </LargeDialog>
        </>
    );
};
