import { BotCreateInput, BotUpdateInput, endpointGetUser, endpointPostBot, endpointPutBot, User } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { BotForm, botInitialValues, transformBotValues, validateBotValues } from "forms/BotForm/BotForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { BotShape } from "utils/shape/models/bot";
import { BotUpsertProps } from "../types";

export const BotUpsert = ({
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
    overrideObject,
    zIndex,
}: BotUpsertProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<User, BotShape>({
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
        <MaybeLargeDialog
            display={display}
            id="bot-upsert-dialog"
            isOpen={isOpen ?? false}
            onClose={handleCancel}
            zIndex={zIndex}
        >
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
                        inputs: transformBotValues(session, values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateBotValues(session, values, existing, isCreate)}
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
        </MaybeLargeDialog>
    );
};
