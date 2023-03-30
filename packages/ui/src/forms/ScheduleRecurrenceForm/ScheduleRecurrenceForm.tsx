import { Stack } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { ScheduleRecurrenceFormProps } from "forms/types";
import { forwardRef } from "react";

export const ScheduleRecurrenceForm = forwardRef<any, ScheduleRecurrenceFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
    zIndex,
    ...props
}, ref) => {

    return (
        <>
            <BaseForm
                dirty={dirty}
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: 'block',
                    maxWidth: '700px',
                    paddingBottom: '64px',
                }}
            >
                <Stack direction="column" spacing={4} padding={2}>
                    {/* TODO */}
                </Stack>
            </BaseForm>
            <GridSubmitButtons
                display={display}
                errors={props.errors as any}
                isCreate={isCreate}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
    )
})