import { FindByIdInput, parseSearchParams, Schedule, ScheduleCreateInput, ScheduleUpdateInput } from "@local/shared";
import { mutationWrapper } from "api";
import { scheduleCreate } from "api/generated/endpoints/schedule_create";
import { scheduleFindOne } from "api/generated/endpoints/schedule_findOne";
import { scheduleUpdate } from "api/generated/endpoints/schedule_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { PageTab } from "components/types";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ScheduleForm, scheduleInitialValues, transformScheduleValues, validateScheduleValues } from "forms/ScheduleForm/ScheduleForm";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { CalendarPageTabOption } from "utils/search/objectToSearch";
import { SessionContext } from "utils/SessionContext";
import { calendarTabParams } from "views/CalendarView/CalendarView";
import { ScheduleUpsertProps } from "../types";

export const ScheduleUpsert = ({
    defaultTab,
    display = "page",
    handleDelete,
    isCreate,
    isMutate,
    listId,
    onCancel,
    onCompleted,
    partialData,
    zIndex = 200,
}: ScheduleUpsertProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

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
        if (!isCreate) return tabs[0];
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
        if (!isCreate && defaultTab !== undefined) {
            const index = calendarTabParams.findIndex(tab => tab.tabType === defaultTab);
            if (index !== -1) setCurrTab(tabs[index]);
        }
    }, [defaultTab, tabs]);
    const handleTabChange = useCallback((e: any, tab: PageTab<CalendarPageTabOption>) => {
        e.preventDefault();
        // Update curr tab
        setCurrTab(tab);
    }, []);

    // Fetch existing data
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery<Schedule, FindByIdInput>(scheduleFindOne);
    useEffect(() => { id && getData({ variables: { id } }); }, [getData, id]);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => scheduleInitialValues(session, {
        ...existing,
        // For creating, set values for linking to an object
        ...(isCreate ? {
            focusMode: currTab.value === "FocusModes" ? null : undefined,
            meeting: currTab.value === "Meetings" ? null : undefined,
            runProject: currTab.value === "Projects" ? null : undefined,
            runRoutine: currTab.value === "Routines" ? null : undefined,
        } : {}),
    } as Schedule), [existing, listId, partialData, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<Schedule>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useCustomMutation<Schedule, ScheduleCreateInput>(scheduleCreate);
    const [update, { loading: isUpdateLoading }] = useCustomMutation<Schedule, ScheduleUpdateInput>(scheduleUpdate);
    const mutation = isCreate ? create : update;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: isCreate ? "CreateSchedule" : "UpdateSchedule",
                }}
                // Can only link to an object when creating
                below={isCreate && <PageTabs
                    ariaLabel="schedule-link-tabs"
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    if (isMutate) {
                        mutationWrapper<Schedule, ScheduleCreateInput | ScheduleUpdateInput>({
                            mutation,
                            input: transformScheduleValues(values, existing),
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
