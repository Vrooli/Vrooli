import { DUMMY_ID, endpointGetFocusMode, endpointPostFocusMode, endpointPutFocusMode, FocusMode, FocusModeCreateInput, FocusModeUpdateInput, focusModeValidation, noopSubmit, Schedule, Session } from "@local/shared";
import { Box, Button, ListItem, Stack, useTheme } from "@mui/material";
import { useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Title } from "components/text/Title/Title";
import { SessionContext } from "contexts/SessionContext";
import { Field, Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { AddIcon, DeleteIcon, EditIcon, HeartFilledIcon, InvisibleIcon } from "icons";
import { useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { CalendarPageTabOption } from "utils/search/objectToSearch";
import { FocusModeShape, shapeFocusMode } from "utils/shape/models/focusMode";
import { validateFormValues } from "utils/validateFormValues";
import { ScheduleUpsert } from "views/objects/schedule";
import { FocusModeFormProps, FocusModeUpsertProps } from "../types";

export const focusModeInitialValues = (
    session: Session | undefined,
    existing?: Partial<FocusMode> | null | undefined,
): FocusModeShape => ({
    __typename: "FocusMode" as const,
    id: DUMMY_ID,
    description: "",
    name: "",
    reminderList: {
        __typename: "ReminderList" as const,
        id: DUMMY_ID,
        reminders: [],
    },
    resourceList: {
        __typename: "ResourceList" as const,
        id: DUMMY_ID,
        resources: [],
        listFor: {
            __typename: "FocusMode" as const,
            id: DUMMY_ID,
        },
    },
    filters: [],
    schedule: null,
    ...existing,
});

export const transformFocusModeValues = (values: FocusModeShape, existing: FocusModeShape, isCreate: boolean) =>
    isCreate ? shapeFocusMode.create(values) : shapeFocusMode.update(existing, values);

const FocusModeForm = ({
    disabled,
    dirty,
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
}: FocusModeFormProps) => {
    const { palette } = useTheme();
    const display = toDisplay(isOpen);
    const { t } = useTranslation();

    // Handle scheduling
    const [scheduleField, , scheduleHelpers] = useField<Schedule | null>("schedule");
    const [isScheduleDialogOpen, setIsScheduleDialogOpen] = useState(false);
    const [editingSchedule, setEditingSchedule] = useState<Schedule | null>(null);
    const handleAddSchedule = () => { setIsScheduleDialogOpen(true); };
    const handleUpdateSchedule = () => {
        setEditingSchedule(scheduleField.value);
        setIsScheduleDialogOpen(true);
    };
    const handleCloseScheduleDialog = () => { setIsScheduleDialogOpen(false); };
    const handleScheduleCompleted = (created: Schedule) => {
        scheduleHelpers.setValue(created);
        setIsScheduleDialogOpen(false);
    };
    const handleDeleteSchedule = () => { scheduleHelpers.setValue(null); };

    const { handleCancel, handleCompleted, isCacheOn } = useUpsertActions<FocusMode>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "FocusMode",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<FocusMode, FocusModeCreateInput, FocusModeUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointPostFocusMode,
        endpointUpdate: endpointPutFocusMode,
    });
    useSaveToCache({ isCacheOn, isCreate, values, objectId: values.id, objectType: "FocusMode" });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<FocusModeCreateInput | FocusModeUpdateInput, FocusMode>({
        disabled,
        existing,
        fetch,
        inputs: transformFocusModeValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    return (
        <MaybeLargeDialog
            display={display}
            id="focus-mode-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateFocusMode" : "UpdateFocusMode")}
            />
            {/* Dialog to create/update schedule */}
            <ScheduleUpsert
                canChangeTab={false}
                canSetScheduleFor={false}
                defaultTab={CalendarPageTabOption.FocusMode}
                handleDelete={handleDeleteSchedule}
                isCreate={editingSchedule === null}
                isMutate={false}
                isOpen={isScheduleDialogOpen}
                onCancel={handleCloseScheduleDialog}
                onCompleted={handleScheduleCompleted}
                overrideObject={editingSchedule ?? { __typename: "Schedule" }}
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={600}
            >
                <Stack direction="column" spacing={4} padding={2}>
                    <Stack direction="column" spacing={2}>
                        <Field
                            fullWidth
                            name="name"
                            label={t("Name")}
                            as={TextInput}
                        />
                        <Field
                            fullWidth
                            name="description"
                            label={t("Description")}
                            as={TextInput}
                        />
                    </Stack>
                    {/* Handle adding, updating, and removing schedule */}
                    {!scheduleField.value && (
                        <Button
                            onClick={handleAddSchedule}
                            startIcon={<AddIcon />}
                            sx={{
                                display: "flex",
                                margin: "auto",
                            }}
                            variant="outlined"
                        >{"Add schedule"}</Button>
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
                                        <EditIcon fill={palette.secondary.main} />
                                    </Box>
                                    {/* Delete */}
                                    <Box
                                        component="a"
                                        onClick={handleDeleteSchedule}
                                        sx={{
                                            display: "flex",
                                            justifyContent: "center",
                                            alignItems: "center",
                                            cursor: "pointer",
                                            pointerEvents: "all",
                                            paddingBottom: "4px",
                                        }}>
                                        <DeleteIcon fill={palette.secondary.main} />
                                    </Box>
                                </Stack>
                            </ListItem>
                        )}
                    </ListContainer>}
                    <ResourceListHorizontalInput
                        isCreate={true}
                        parent={{ __typename: "FocusMode", id: values.id }}
                    />
                    <Title
                        Icon={HeartFilledIcon}
                        title={t("TopicsFavorite")}
                        help={t("TopicsFavoriteHelp")}
                        variant="subheader"
                    />
                    <TagSelector name="favorites" />
                    <Title
                        Icon={InvisibleIcon}
                        title={t("TopicsHidden")}
                        help={t("TopicsHiddenHelp")}
                        variant="subheader"
                    />
                    <TagSelector name="hidden" />
                </Stack>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={props.errors}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
            />
        </MaybeLargeDialog>
    );
};

export const FocusModeUpsert = ({
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: FocusModeUpsertProps) => {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, setObject: setExisting } = useObjectFromUrl<FocusMode, FocusModeShape>({
        ...endpointGetFocusMode,
        isCreate,
        objectType: "FocusMode",
        overrideObject,
        transform: (data) => focusModeInitialValues(session, data),
    });

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateFormValues(values, existing, isCreate, transformFocusModeValues, focusModeValidation)}
        >
            {(formik) => <FocusModeForm
                disabled={false} // Can always update focus mode
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
};
