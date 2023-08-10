import { endpointGetMeeting, endpointPostMeeting, endpointPutMeeting, Meeting, MeetingCreateInput, MeetingUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { MeetingForm, meetingInitialValues, transformMeetingValues, validateMeetingValues } from "forms/MeetingForm/MeetingForm";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { MakeLazyRequest, useLazyFetch } from "utils/hooks/useLazyFetch";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { MeetingShape } from "utils/shape/models/meeting";
import { MeetingUpsertProps } from "../types";

export const MeetingUpsert = ({
    display = "page",
    isCreate,
    onCancel,
    onCompleted,
    zIndex,
}: MeetingUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<Meeting, MeetingShape>({
        ...endpointGetMeeting,
        objectType: "Meeting",
        upsertTransform: (data) => meetingInitialValues(session, data),
    });

    const formRef = useRef<BaseFormRef>();
    const { handleCancel, handleCompleted } = useUpsertActions<Meeting>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useLazyFetch<MeetingCreateInput, Meeting>(endpointPostMeeting);
    const [update, { loading: isUpdateLoading }] = useLazyFetch<MeetingUpdateInput, Meeting>(endpointPutMeeting);
    const fetch = (isCreate ? create : update) as MakeLazyRequest<MeetingCreateInput | MeetingUpdateInput, Meeting>;

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
