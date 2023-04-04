import { useTheme } from "@mui/material";
import { Reminder, Session } from "@shared/consts";
import { DUMMY_ID } from "@shared/uuid";
import { reminderValidation } from "@shared/validation";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { ReminderFormProps } from "forms/types";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { SessionContext } from "utils/SessionContext";
import { validateAndGetYupErrors } from "utils/shape/general";
import { ReminderShape, shapeReminder } from "utils/shape/models/reminder";

export const reminderInitialValues = (
    session: Session | undefined,
    reminderListId: string | undefined,
    existing?: Reminder | null | undefined
): ReminderShape => ({
    __typename: 'Reminder' as const,
    id: DUMMY_ID,
    description: null,
    dueDate: null,
    index: 0,
    isComplete: false,
    name: '',
    reminderList: {
        __typename: 'ReminderList' as const,
        id: reminderListId ?? DUMMY_ID,
    },
    reminderItems: [],
    ...existing,
});

export function transformReminderValues(values: ReminderShape, existing?: ReminderShape) {
    return existing === undefined
        ? shapeReminder.create(values)
        : shapeReminder.update(existing, values)
}

export const validateReminderValues = async (values: ReminderShape, existing?: ReminderShape) => {
    const transformedValues = transformReminderValues(values, existing);
    const validationSchema = existing === undefined
        ? reminderValidation.create({})
        : reminderValidation.update({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
}

export const ReminderForm = forwardRef<any, ReminderFormProps>(({
    display,
    dirty,
    index,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    reminderListId,
    values,
    zIndex,
    ...props
}, ref) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

    return (
        <>
            <BaseForm
                dirty={dirty}
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: 'block',
                    paddingBottom: '64px',
                }}
            >
                {/* TODO */}
                <GridSubmitButtons
                    display={display}
                    errors={props.errors as any}
                    isCreate={isCreate}
                    loading={props.isSubmitting}
                    onCancel={onCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={props.handleSubmit}
                />
            </BaseForm>
        </>
    )
})