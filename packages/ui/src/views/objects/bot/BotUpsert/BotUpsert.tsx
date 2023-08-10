import { BotCreateInput, BotUpdateInput, endpointGetUser, endpointPostBot, endpointPutBot, User } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { BotForm, botInitialValues, transformBotValues, validateBotValues } from "forms/BotForm/BotForm";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { MakeLazyRequest, useLazyFetch } from "utils/hooks/useLazyFetch";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { BotShape } from "utils/shape/models/bot";
import { BotUpsertProps } from "../types";

export const BotUpsert = ({
    display = "page",
    isCreate,
    onCancel,
    onCompleted,
    zIndex,
}: BotUpsertProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<User, BotShape>({
        ...endpointGetUser,
        objectType: "User",
        upsertTransform: (data) => botInitialValues(session, data),
    });

    const formRef = useRef<BaseFormRef>();
    const { handleCancel, handleCompleted } = useUpsertActions<User>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useLazyFetch<BotCreateInput, User>(endpointPostBot);
    const [update, { loading: isUpdateLoading }] = useLazyFetch<BotUpdateInput, User>(endpointPutBot);
    const fetch = (isCreate ? create : update) as MakeLazyRequest<BotCreateInput | BotUpdateInput, User>;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t(isCreate ? "CreateBot" : "UpdateBot")}
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
                    fetchLazyWrapper<BotCreateInput | BotUpdateInput, User>({
                        fetch,
                        inputs: transformBotValues(session, values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateBotValues(session, values, existing)}
            >
                {(formik) =>
                    <BotForm
                        display={display}
                        isCreate={isCreate}
                        isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                        isOpen={true}
                        onCancel={handleCancel}
                        ref={formRef}
                        zIndex={zIndex}
                        {...formik}
                    />
                }
            </Formik>
        </>
    );
};
