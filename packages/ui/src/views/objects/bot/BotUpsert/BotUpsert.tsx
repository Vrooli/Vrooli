import { BotCreateInput, BotUpdateInput, endpointGetUser, endpointPostBot, endpointPutBot, User } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { BotForm, botInitialValues, transformBotValues, validateBotValues } from "forms/BotForm/BotForm";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { BotShape } from "utils/shape/models/bot";
import { BotUpsertProps } from "../types";

export const BotUpsert = ({
    isCreate,
    onCancel,
    onCompleted,
    overrideObject,
    zIndex,
}: BotUpsertProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    const { display, isLoading: isReadLoading, object: existing } = useObjectFromUrl<User, BotShape>({
        ...endpointGetUser,
        objectType: "User",
        overrideObject,
        transform: (data) => botInitialValues(session, data),
    });

    const formRef = useRef<BaseFormRef>();
    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<User, BotCreateInput, BotUpdateInput>({
        display,
        endpointCreate: endpointPostBot,
        endpointUpdate: endpointPutBot,
        isCreate,
        onCancel,
        onCompleted,
    });

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
