import { Chat, ChatCreateInput, ChatUpdateInput, endpointGetChat, endpointPostChat, endpointPutChat } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ChatForm, chatInitialValues, transformChatValues, validateChatValues } from "forms/ChatForm/ChatForm";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { MakeLazyRequest, useLazyFetch } from "utils/hooks/useLazyFetch";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { ChatShape } from "utils/shape/models/chat";
import { ChatUpsertProps } from "../types";

export const ChatUpsert = ({
    display = "page",
    isCreate,
    onCancel,
    onCompleted,
    zIndex,
}: ChatUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<Chat, ChatShape>({
        ...endpointGetChat,
        objectType: "Chat",
        upsertTransform: (data) => chatInitialValues(session, data),
    });

    const formRef = useRef<BaseFormRef>();
    const { handleCancel, handleCompleted } = useUpsertActions<Chat>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useLazyFetch<ChatCreateInput, Chat>(endpointPostChat);
    const [update, { loading: isUpdateLoading }] = useLazyFetch<ChatUpdateInput, Chat>(endpointPutChat);
    const fetch = (isCreate ? create : update) as MakeLazyRequest<ChatCreateInput | ChatUpdateInput, Chat>;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t(isCreate ? "CreateChat" : "UpdateChat")}
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
