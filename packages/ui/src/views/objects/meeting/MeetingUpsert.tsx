import { DUMMY_ID, endpointsMeeting, Meeting, MeetingCreateInput, MeetingShape, MeetingUpdateInput, meetingValidation, noopSubmit, orDefault, Schedule, Session, shapeMeeting } from "@local/shared";
import { Box, Button, ListItem, Stack, useTheme } from "@mui/material";
import { Formik, useField } from "formik";
import { useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSubmitHelper } from "../../../api/fetchWrapper.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { ListContainer } from "../../../components/containers/ListContainer.js";
import { MaybeLargeDialog } from "../../../components/dialogs/LargeDialog/LargeDialog.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { SessionContext } from "../../../contexts/session.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "../../../hooks/forms.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { IconCommon } from "../../../icons/Icons.js";
import { getUserLanguages } from "../../../utils/display/translationTools.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { ScheduleUpsert } from "../schedule/ScheduleUpsert.js";
import { MeetingFormProps, MeetingUpsertProps } from "./types.js";

export function meetingInitialValues(
    session: Session | undefined,
    existing?: Partial<Meeting> | null | undefined,
): MeetingShape {
    return {
        __typename: "Meeting" as const,
        id: DUMMY_ID,
        openToAnyoneWithInvite: false,
        showOnTeamProfile: true,
        team: {
            __typename: "Team" as const,
            id: DUMMY_ID,
        },
        schedule: null,
        ...existing,
        translations: orDefault(existing?.translations, [{
            __typename: "MeetingTranslation" as const,
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            description: "",
            link: "",
            name: "",
        }]),
    };
}

export function transformMeetingValues(values: MeetingShape, existing: MeetingShape, isCreate: boolean) {
    return isCreate ? shapeMeeting.create(values) : shapeMeeting.update(existing, values);
}

const defaultScheduleOverrideObject = { __typename: "Schedule" } as const;

function MeetingForm({
    disabled,
    dirty,
    display,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    ...props
}: MeetingFormProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Handle scheduling
    const [scheduleField, , scheduleHelpers] = useField<Schedule | null>("schedule");
    const [isScheduleDialogOpen, setIsScheduleDialogOpen] = useState(false);
    const [editingSchedule, setEditingSchedule] = useState<Schedule | null>(null);
    function handleAddSchedule() {
        setIsScheduleDialogOpen(true);
    }
    function handleUpdateSchedule() {
        setEditingSchedule(scheduleField.value);
        setIsScheduleDialogOpen(true);
    }
    function handleCloseScheduleDialog() {
        setIsScheduleDialogOpen(false);
    }
    function handleScheduleCompleted(created: Schedule) {
        scheduleHelpers.setValue(created);
        setIsScheduleDialogOpen(false);
    }
    function handleScheduleDeleted() {
        scheduleHelpers.setValue(null);
        setIsScheduleDialogOpen(false);
    }

    const { handleCancel, handleCompleted } = useUpsertActions<Meeting>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "Meeting",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<Meeting, MeetingCreateInput, MeetingUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointsMeeting.createOne,
        endpointUpdate: endpointsMeeting.updateOne,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "Meeting" });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<MeetingCreateInput | MeetingUpdateInput, Meeting>({
        disabled,
        existing,
        fetch,
        inputs: transformMeetingValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    return (
        <MaybeLargeDialog
            display={display}
            id="meeting-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateMeeting" : "UpdateMeeting")}
            />
            <ScheduleUpsert
                canSetScheduleFor={false}
                defaultScheduleFor="Meeting"
                display="Dialog"
                isCreate={editingSchedule === null}
                isMutate={false}
                isOpen={isScheduleDialogOpen}
                onCancel={handleCloseScheduleDialog}
                onClose={handleCloseScheduleDialog}
                onCompleted={handleScheduleCompleted}
                onDeleted={handleScheduleDeleted}
                overrideObject={editingSchedule ?? defaultScheduleOverrideObject}
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={700}
            >
                <Stack direction="column" spacing={4} padding={2}>
                    {/* TODO */}
                    {/* Handle adding, updating, and removing schedule */}
                    {!scheduleField.value && (
                        <Button
                            onClick={handleAddSchedule}
                            startIcon={<IconCommon name="Add" />}
                            variant="outlined"
                            sx={{
                                display: "flex",
                                margin: "auto",
                            }}
                        >{t("ScheduleCreate")}</Button>
                    )}
                    {scheduleField.value && <ListContainer
                        isEmpty={false}
                    >
                        {scheduleField.value && (
                            <ListItem>
                                <Stack
                                    direction="column"
                                    spacing={1}
                                    pl={2}
                                    sx={{
                                        width: "-webkit-fill-available",
                                        display: "grid",
                                        pointerEvents: "none",
                                    }}
                                >
                                    {/* TODO */}
                                </Stack>
                                <Stack
                                    direction="column"
                                    spacing={1}
                                    sx={{
                                        pointerEvents: "none",
                                        justifyContent: "center",
                                        alignItems: "start",
                                    }}
                                >
                                    {/* Edit */}
                                    <Box
                                        component="a"
                                        onClick={handleUpdateSchedule}
                                        sx={{
                                            display: "flex",
                                            justifyContent: "center",
                                            alignItems: "center",
                                            cursor: "pointer",
                                            pointerEvents: "all",
                                            paddingBottom: "4px",
                                        }}>
                                        <IconCommon name="Edit" fill="secondary.main" />
                                    </Box>
                                    {/* Delete */}
                                    <Box
                                        component="a"
                                        onClick={handleScheduleDeleted}
                                        sx={{
                                            display: "flex",
                                            justifyContent: "center",
                                            alignItems: "center",
                                            cursor: "pointer",
                                            pointerEvents: "all",
                                            paddingBottom: "4px",
                                        }}>
                                        <IconCommon name="Delete" fill="secondary.main" />
                                    </Box>
                                </Stack>
                            </ListItem>
                        )}
                    </ListContainer>}
                    {/* TODO */}
                </Stack>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={props.errors}
                hideButtons={disabled}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
            />
        </MaybeLargeDialog>
    );
}

export function MeetingUpsert({
    display,
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: MeetingUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<Meeting, MeetingShape>({
        ...endpointsMeeting.findOne,
        disabled: display === "Dialog" && isOpen !== true,
        isCreate,
        objectType: "Meeting",
        overrideObject,
        transform: (data) => meetingInitialValues(session, data),
    });

    async function validateValues(values: MeetingShape) {
        return await validateFormValues(values, existing, isCreate, transformMeetingValues, meetingValidation);
    }

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <MeetingForm
                disabled={!(isCreate || permissions.canUpdate)}
                display={display}
                existing={existing}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                {...props}
                {...formik}
            />}
        </Formik>
    );
}
