import { DUMMY_ID, parseSearchParams, Schedule, ScheduleCreateInput, ScheduleUpdateInput } from "@local/shared";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from "api/utils";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { PageTab } from "components/types";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ScheduleForm } from "forms/ScheduleForm/ScheduleForm";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { PubSub } from "utils/pubsub";
import { CalendarPageTabOption } from "utils/search/objectToSearch";
import { ScheduleShape, shapeSchedule } from "utils/shape/models/schedule";
import { calendarTabParams } from "views/CalendarView/CalendarView";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { ScheduleDialogProps } from "../types";

const titleId = "schedule-dialog-title";

export const ScheduleDialog = ({
    defaultTab,
    existing,
    isMutate,
    isOpen,
    onClose,
    onCreated,
    onUpdated,
    zIndex,
}: ScheduleDialogProps) => {
    const { t } = useTranslation();

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => scheduleInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<Schedule>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useCustomMutation<Schedule, ScheduleCreateInput>(noteVersionCreate);
    const [update, { loading: isUpdateLoading }] = useCustomMutation<Schedule, ScheduleUpdateInput>(noteVersionUpdate);
    const mutation = isCreate ? create : update;

    const handleClose = useCallback((_?: unknown, reason?: "backdropClick" | "escapeKeyDown") => {
        // Confirm dialog is dirty and closed by clicking outside
        formRef.current?.handleClose(onClose, reason !== "backdropClick");
    }, [onClose]);

    // Handle tabs
    const tabs = useMemo<PageTab<CalendarPageTabOption>[]>(() => {
        return calendarTabParams.map((tab, i) => ({
            index: i,
            Icon: tab.Icon,
            label: t(tab.titleKey, { count: 2, defaultValue: tab.titleKey }),
            value: tab.tabType,
        }));
    }, [t]);
    const [currTab, setCurrTab] = useState<PageTab<CalendarPageTabOption>>(() => {
        if (defaultTab !== undefined) {
            const index = calendarTabParams.findIndex(tab => tab.tabType === defaultTab);
            if (index !== -1) return tabs[index];
        }
        const searchParams = parseSearchParams();
        const index = calendarTabParams.findIndex(tab => tab.tabType === searchParams.type);
        // Default to bookmarked tab
        if (index === -1) return tabs[0];
        // Return tab
        return tabs[index];
    });
    useEffect(() => {
        if (defaultTab !== undefined) {
            const index = calendarTabParams.findIndex(tab => tab.tabType === defaultTab);
            if (index !== -1) setCurrTab(tabs[index]);
        }
    }, [defaultTab, tabs]);
    const handleTabChange = useCallback((e: any, tab: PageTab<CalendarPageTabOption>) => {
        e.preventDefault();
        // Update curr tab
        setCurrTab(tab);
    }, []);

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
                <TopBar
                    display="dialog"
                    titleData={{
                        titleKey: isCreate ? "ScheduleCreate" : "ScheduleUpdate",
                    }}
                    onClose={handleClose}
                    below={isCreate && <PageTabs
                        ariaLabel="search-tabs"
                        currTab={currTab}
                        onChange={handleTabChange}
                        tabs={tabs}
                    />}
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
                                successMessage: () => ({ messageKey: isCreate ? "ScheduleCreated" : "ScheduleUpdated" }),
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
