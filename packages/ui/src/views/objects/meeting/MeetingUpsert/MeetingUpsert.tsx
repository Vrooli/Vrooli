import { endpointGetMeeting, endpointPostMeeting, endpointPutMeeting, Meeting, MeetingCreateInput, MeetingUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { MeetingForm, meetingInitialValues, transformMeetingValues, validateMeetingValues } from "forms/MeetingForm/MeetingForm";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { MeetingShape } from "utils/shape/models/meeting";
import { MeetingUpsertProps } from "../types";

export const MeetingUpsert = ({
    isCreate,
    onCancel,
    onCompleted,
    overrideObject,
    zIndex,
}: MeetingUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const { display, isLoading: isReadLoading, object: existing } = useObjectFromUrl<Meeting, MeetingShape>({
        ...endpointGetMeeting,
        objectType: "Meeting",
        overrideObject,
        transform: (data) => meetingInitialValues(session, data),
    });

    const formRef = useRef<BaseFormRef>();
    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<Meeting, MeetingCreateInput, MeetingUpdateInput>({
        display,
        endpointCreate: endpointPostMeeting,
        endpointUpdate: endpointPutMeeting,
        isCreate,
        onCancel,
        onCompleted,
    });

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t(isCreate ? "CreateMeeting" : "UpdateMeeting")}
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
                    fetchLazyWrapper<MeetingCreateInput | MeetingUpdateInput, Meeting>({
                        fetch,
                        inputs: transformMeetingValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateMeetingValues(values, existing)}
            >
                {(formik) => <MeetingForm
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
