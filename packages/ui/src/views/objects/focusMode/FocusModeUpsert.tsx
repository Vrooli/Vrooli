import { DUMMY_ID, endpointsFocusMode, FocusMode, FocusModeCreateInput, FocusModeShape, FocusModeUpdateInput, focusModeValidation, noopSubmit, Schedule, Session, shapeFocusMode } from "@local/shared";
import { Box, Button, ListItem, Stack, useTheme } from "@mui/material";
import { Field, Formik, useField } from "formik";
import { useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSubmitHelper } from "../../../api/fetchWrapper.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { ContentCollapse } from "../../../components/containers/ContentCollapse.js";
import { ListContainer } from "../../../components/containers/ListContainer.js";
import { MaybeLargeDialog } from "../../../components/dialogs/LargeDialog/LargeDialog.js";
import { TagSelector } from "../../../components/inputs/TagSelector/TagSelector.js";
import { TextInput } from "../../../components/inputs/TextInput/TextInput.js";
import { ResourceListInput } from "../../../components/lists/ResourceList/ResourceList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { Title } from "../../../components/text/Title.js";
import { SessionContext } from "../../../contexts/session.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "../../../hooks/forms.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { IconCommon } from "../../../icons/Icons.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { ScheduleUpsert } from "../schedule/ScheduleUpsert.js";
import { FocusModeFormProps, FocusModeUpsertProps } from "./types.js";

export function focusModeInitialValues(
    session: Session | undefined,
    existing?: Partial<FocusMode> | null | undefined,
): FocusModeShape {
    return {
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
    };
}

export function transformFocusModeValues(values: FocusModeShape, existing: FocusModeShape, isCreate: boolean) {
    return isCreate ? shapeFocusMode.create(values) : shapeFocusMode.update(existing, values);
}

const favoriteIconInfo = { name: "HeartFilled", type: "Common" } as const;
const hiddenIconInfo = { name: "Invisible", type: "Common" } as const;

function FocusModeForm({
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
}: FocusModeFormProps) {
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

    const resourceListParent = useMemo(function resourceListParentMemo() {
        return { __typename: "FocusMode", id: values.id } as const;
    }, [values]);

    const { handleCancel, handleCompleted } = useUpsertActions<FocusMode>({
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
        endpointCreate: endpointsFocusMode.createOne,
        endpointUpdate: endpointsFocusMode.updateOne,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "FocusMode" });

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
            <ScheduleUpsert
                canSetScheduleFor={false}
                defaultScheduleFor="FocusMode"
                display="dialog"
                isCreate={editingSchedule === null}
                isMutate={false}
                isOpen={isScheduleDialogOpen}
                onCancel={handleCloseScheduleDialog}
                onClose={handleCloseScheduleDialog}
                onCompleted={handleScheduleCompleted}
                onDeleted={handleScheduleDeleted}
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
                            startIcon={<IconCommon name="Add" />}
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
                    <ResourceListInput
                        horizontal
                        isCreate={true}
                        parent={resourceListParent}
                    />
                    <ContentCollapse
                        isOpen={false}
                        title={t("Advanced")}
                    >
                        <Title
                            iconInfo={favoriteIconInfo}
                            title={t("TopicsFavorite")}
                            help={t("TopicsFavoriteHelp")}
                            variant="subsection"
                        />
                        <TagSelector name="favorites" />
                        <Title
                            iconInfo={hiddenIconInfo}
                            title={t("TopicsHidden")}
                            help={t("TopicsHiddenHelp")}
                            variant="subsection"
                        />
                        <TagSelector name="hidden" />
                    </ContentCollapse>
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
}

export function FocusModeUpsert({
    display,
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: FocusModeUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, setObject: setExisting } = useManagedObject<FocusMode, FocusModeShape>({
        ...endpointsFocusMode.findOne,
        disabled: display === "dialog" && isOpen !== true,
        isCreate,
        objectType: "FocusMode",
        overrideObject,
        transform: (data) => focusModeInitialValues(session, data),
    });

    async function validateValues(values: FocusModeShape) {
        return await validateFormValues(values, existing, isCreate, transformFocusModeValues, focusModeValidation);
    }

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <FocusModeForm
                disabled={false} // Can always update focus mode
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
