import { FindByIdInput, Meeting, meetingCreate, MeetingCreateInput, meetingFindOne, meetingUpdate, MeetingUpdateInput } from "@local/shared";
import { useCustomLazyQuery, useCustomMutation } from "api";
import { mutationWrapper } from "api/utils";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { MeetingForm, meetingInitialValues, transformMeetingValues, validateMeetingValues } from "forms/MeetingForm/MeetingForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { MeetingUpsertProps } from "../types";

export const MeetingUpsert = ({
    display = "page",
    isCreate,
    onCancel,
    onCompleted,
    zIndex = 200,
}: MeetingUpsertProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery<Meeting, FindByIdInput>(meetingFindOne);
    useEffect(() => { id && getData({ variables: { id } }); }, [getData, id]);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => meetingInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<Meeting>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useCustomMutation<Meeting, MeetingCreateInput>(meetingCreate);
    const [update, { loading: isUpdateLoading }] = useCustomMutation<Meeting, MeetingUpdateInput>(meetingUpdate);
    const mutation = isCreate ? create : update;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: isCreate ? "CreateMeeting" : "UpdateMeeting",
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    mutationWrapper<Meeting, MeetingCreateInput | MeetingUpdateInput>({
                        mutation,
                        input: transformMeetingValues(values, existing),
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
