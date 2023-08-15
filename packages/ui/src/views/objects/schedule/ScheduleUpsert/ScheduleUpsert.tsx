import { endpointGetSchedule, endpointPostSchedule, endpointPutSchedule, Schedule, ScheduleCreateInput, ScheduleUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ScheduleForm, scheduleInitialValues, transformScheduleValues, validateScheduleValues } from "forms/ScheduleForm/ScheduleForm";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { useTabs } from "utils/hooks/useTabs";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { PubSub } from "utils/pubsub";
import { CalendarPageTabOption } from "utils/search/objectToSearch";
import { SessionContext } from "utils/SessionContext";
import { ScheduleShape } from "utils/shape/models/schedule";
import { calendarTabParams } from "views/CalendarView/CalendarView";
import { ScheduleUpsertProps } from "../types";

const tabParams = calendarTabParams.filter(tp => tp.tabType !== "All");

export const ScheduleUpsert = ({
    canChangeTab = true,
    canSetScheduleFor = true,
    defaultTab,
    handleDelete,
    isCreate,
    isMutate,
    isOpen,
    listId,
    onCancel,
    onCompleted,
    overrideObject,
    zIndex,
}: ScheduleUpsertProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

    const {
        currTab,
        handleTabChange,
        tabs,
    } = useTabs<CalendarPageTabOption>(
        tabParams,
        defaultTab ? (tabParams.findIndex(tp => tp.tabType === defaultTab) ?? 0) : 0,
    );

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<Schedule, ScheduleShape>({
        ...endpointGetSchedule,
        objectType: "Schedule",
        overrideObject,
        transform: (existing) => scheduleInitialValues(session, { //TODO this might cause a fetch every time a tab is changed, and we lose changed data. Need to test
            ...existing,
            // For creating, set values for linking to an object. 
            // NOTE: We can't set these values to null or undefined like you'd expect, 
            // because Formik will treat them as uncontrolled inputs and throw errors. 
            // Instead, we pretend that false is null and an empty string is undefined.
            ...(isCreate && canSetScheduleFor ? {
                focusMode: currTab.value === "FocusMode" ? false : "",
                meeting: currTab.value === "Meeting" ? false : "",
                runProject: currTab.value === "RunProject" ? false : "",
                runRoutine: currTab.value === "RunRoutine" ? false : "",
            } : {}),
        } as Schedule),
    });

    const formRef = useRef<BaseFormRef>();
    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<Schedule, ScheduleCreateInput, ScheduleUpdateInput>({
        display,
        endpointCreate: endpointPostSchedule,
        endpointUpdate: endpointPutSchedule,
        isCreate,
        onCancel,
        onCompleted,
    });

    return (
        <MaybeLargeDialog
            display={display}
            id="schedule-upsert-dialog"
            isOpen={isOpen ?? false}
            onClose={handleCancel}
            zIndex={zIndex}
        >
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t(`${isCreate ? "Create" : "Update"}${currTab.value.substring(0, currTab.value.length - 1)}` as any)}
                // Can only link to an object when creating
                below={isCreate && canChangeTab && <PageTabs
                    ariaLabel="schedule-link-tabs"
                    currTab={currTab}
                    fullWidth
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
                zIndex={zIndex}
            />
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    if (isMutate) {
                        fetchLazyWrapper<ScheduleCreateInput | ScheduleUpdateInput, Schedule>({
                            fetch,
                            inputs: transformScheduleValues(values, existing, isCreate),
                            onSuccess: (data) => { handleCompleted(data); },
                            onCompleted: () => { helpers.setSubmitting(false); },
                        });
                    } else {
                        onCompleted?.({
                            ...values,
                            created_at: (existing as Partial<Schedule>).created_at ?? new Date().toISOString(),
                            updated_at: (existing as Partial<Schedule>).updated_at ?? new Date().toISOString(),
                        } as Schedule);
                    }
                }}
                validate={async (values) => await validateScheduleValues(values, existing, isCreate)}
            >
                {(formik) => <ScheduleForm
                    canSetScheduleFor={canSetScheduleFor}
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
