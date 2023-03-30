import { Stack, TextField, useTheme } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { TimezoneSelector } from "components/inputs/TimezoneSelector/TimezoneSelector";
import { Field } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { ScheduleFormProps } from "forms/types";
import { forwardRef } from "react";
import { useTranslation } from "react-i18next";

export const ScheduleForm = forwardRef<any, ScheduleFormProps>(({
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
                    maxWidth: '700px',
                    paddingBottom: '64px',
                }}
            >
                <Stack direction="column" spacing={4} padding={2}>
                    <Stack direction="column" spacing={2}>
                        <Field
                            fullWidth
                            name="startTime"
                            label={"Start time"}
                            type="datetime-local"
                            InputLabelProps={{
                                shrink: true,
                            }}
                            as={TextField}
                        />
                        <Field
                            fullWidth
                            name="endTime"
                            label={"End time"}
                            type="datetime-local"
                            InputLabelProps={{
                                shrink: true,
                            }}
                            as={TextField}
                        />
                        <TimezoneSelector name="timezone" label="Timezone" />
                    </Stack>
                    {/* <List
                        name="exceptionsCreate"
                        label={t('Exceptions')}
                        formComponent={ScheduleExceptionForm}
                        zIndex={zIndex}
                    />
                    <List
                        name="recurrencesCreate"
                        label={t('Recurrences')}
                        formComponent={ScheduleRecurrenceForm}
                        zIndex={zIndex}
                    /> */}
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