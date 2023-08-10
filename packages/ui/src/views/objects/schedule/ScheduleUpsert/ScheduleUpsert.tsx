import { endpointGetSchedule, endpointPostSchedule, endpointPutSchedule, Schedule, ScheduleCreateInput, ScheduleUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { PageTab } from "components/types";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ScheduleForm, scheduleInitialValues, transformScheduleValues, validateScheduleValues } from "forms/ScheduleForm/ScheduleForm";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { parseSearchParams } from "route";
import { MakeLazyRequest, useLazyFetch } from "utils/hooks/useLazyFetch";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { PubSub } from "utils/pubsub";
import { CalendarPageTabOption } from "utils/search/objectToSearch";
import { SessionContext } from "utils/SessionContext";
import { ScheduleShape } from "utils/shape/models/schedule";
import { calendarTabParams } from "views/CalendarView/CalendarView";
import { ScheduleUpsertProps } from "../types";

const tabParams = (calendarTabParams) => calendarTabParams.filter(tp => tp.tabType !== "All");

export const ScheduleUpsert = ({
    canChangeTab = true,
    canSetScheduleFor = true,
    defaultTab,
    display = "page",
    handleDelete,
    isCreate,
    isMutate,
    listId,
    onCancel,
    onCompleted,
    partialData,
    zIndex,
}: ScheduleUpsertProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    // Handle tabs
    const tabs = useMemo<PageTab<CalendarPageTabOption>[]>(() => {
        return tabParams(calendarTabParams).map((tab, i) => ({
            index: i,
            Icon: tab.Icon,
            label: t(tab.titleKey, { count: 2, defaultValue: tab.titleKey }),
            value: tab.tabType,
        }));
    }, [t]);
    const [currTab, setCurrTab] = useState<PageTab<CalendarPageTabOption>>(() => {
        if (!isCreate) return tabs[0];
        if (defaultTab !== undefined) {
            const index = tabParams(calendarTabParams).findIndex(tab => tab.tabType === defaultTab);
            if (index !== -1) return tabs[index];
        }
        const searchParams = parseSearchParams();
        const index = tabParams(calendarTabParams).findIndex(tab => tab.tabType === searchParams.type);
        // Default to bookmarked tab
        if (index === -1) return tabs[0];
        // Return tab
        return tabs[index];
    });
    useEffect(() => {
        if (!isCreate && defaultTab !== undefined) {
            const index = tabParams(calendarTabParams).findIndex(tab => tab.tabType === defaultTab);
            if (index !== -1) setCurrTab(tabs[index]);
        }
    }, [defaultTab, isCreate, tabs]);
    const handleTabChange = useCallback((e: any, tab: PageTab<CalendarPageTabOption>) => {
        e.preventDefault();
        // Update curr tab
        setCurrTab(tab);
    }, []);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<Schedule, ScheduleShape>({
        ...endpointGetSchedule,
        objectType: "Schedule",
        upsertTransform: (existing) => scheduleInitialValues(session, { //TODO this might cause a fetch every time a tab is changed, and we lose changed data. Need to test
            ...existing,
            // For creating, set values for linking to an object. 
            // NOTE: We can't set these values to null or undefined like you'd expect, 
            // because Formik will treat them as uncontrolled inputs and throw errors. 
            // Instead, we pretend that false is null and an empty string is undefined.
            ...(isCreate && canSetScheduleFor ? {
                focusMode: currTab.value === "FocusModes" ? false : "",
                meeting: currTab.value === "Meetings" ? false : "",
                runProject: currTab.value === "RunProjects" ? false : "",
                runRoutine: currTab.value === "RunRoutines" ? false : "",
            } : {}),
        } as Schedule),
    });

    const formRef = useRef<BaseFormRef>();
    const { handleCancel, handleCompleted } = useUpsertActions<Schedule>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useLazyFetch<ScheduleCreateInput, Schedule>(endpointPostSchedule);
    const [update, { loading: isUpdateLoading }] = useLazyFetch<ScheduleUpdateInput, Schedule>(endpointPutSchedule);
    const fetch = (isCreate ? create : update) as MakeLazyRequest<ScheduleCreateInput | ScheduleUpdateInput, Schedule>;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t(`${isCreate ? "Create" : "Update"}${currTab.value.substring(0, currTab.value.length - 1)}` as any)}
                // Can only link to an object when creating
                below={isCreate && canChangeTab && <PageTabs
                    ariaLabel="schedule-link-tabs"
                    currTab={currTab}
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
                            inputs: transformScheduleValues(values, existing),
                            onSuccess: (data) => { handleCompleted(data); },
                            onError: () => { helpers.setSubmitting(false); },
                        });
                    } else {
                        onCompleted?.({
                            ...values,
                            created_at: partialData?.created_at ?? new Date().toISOString(),
                            updated_at: partialData?.updated_at ?? new Date().toISOString(),
                        } as Schedule);
                    }
                }}
                validate={async (values) => await validateScheduleValues(values, existing)}
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
        </>
    );
};
