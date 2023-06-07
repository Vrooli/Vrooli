import { Chat, ChatCreateInput, ChatUpdateInput, endpointGetChat, endpointPostChat, endpointPutChat, FindVersionInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ChatForm, chatInitialValues, transformChatValues, validateChatValues } from "forms/ChatForm/ChatForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { MakeLazyRequest, useLazyFetch } from "utils/hooks/useLazyFetch";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { ChatUpsertProps } from "../types";

export const ChatUpsert = ({
    display = "page",
    isCreate,
    onCancel,
    onCompleted,
    zIndex = 200,
}: ChatUpsertProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useLazyFetch<FindVersionInput, Chat>(endpointGetChat);
    useEffect(() => { id && getData({ id }); }, [getData, id]);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => chatInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<Chat>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useLazyFetch<ChatCreateInput, Chat>(endpointPostChat);
    const [update, { loading: isUpdateLoading }] = useLazyFetch<ChatUpdateInput, Chat>(endpointPutChat);
    const fetch = (isCreate ? create : update) as MakeLazyRequest<ChatCreateInput | ChatUpdateInput, Chat>;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: isCreate ? "CreateChat" : "UpdateChat",
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
                    fetchLazyWrapper<ChatCreateInput | ChatUpdateInput, Chat>({
                        fetch,
                        inputs: transformChatValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateChatValues(values, existing)}
            >
                {(formik) => <ChatForm
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
