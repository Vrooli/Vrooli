import { endpointGetResource, endpointPostResource, endpointPutResource, Resource, ResourceCreateInput, ResourceUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { NewResourceShape, ResourceForm, resourceInitialValues, transformResourceValues, validateResourceValues } from "forms/ResourceForm/ResourceForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { ResourceShape } from "utils/shape/models/resource";
import { ResourceUpsertProps } from "../types";

export const ResourceUpsert = ({
    isCreate,
    isMutate,
    isOpen,
    onCancel,
    onCompleted,
    overrideObject,
}: ResourceUpsertProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<Resource, ResourceShape>({
        ...endpointGetResource,
        objectType: "Resource",
        overrideObject: overrideObject as Resource,
        transform: (existing) => resourceInitialValues(session, existing as NewResourceShape),
    });

    const formRef = useRef<BaseFormRef>();
    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<Resource, ResourceCreateInput, ResourceUpdateInput>({
        display,
        endpointCreate: endpointPostResource,
        endpointUpdate: endpointPutResource,
        isCreate,
        onCancel,
        onCompleted,
    });

    // TODO add this back, and add to every other upsert. Can make a hook for it, or pass an "isDirty" to the dialog or something
    // const handleClose = useCallback((_?: unknown, reason?: "backdropClick" | "escapeKeyDown") => {
    //     // Confirm dialog is dirty and closed by clicking outside
    //     formRef.current?.handleClose(onClose, reason !== "backdropClick");
    // }, [onClose]);

    return (
        <MaybeLargeDialog
            display={display}
            id="resource-upsert-dialog"
            isOpen={isOpen ?? false}
            onClose={handleCancel}
        >
            <TopBar
                display={display}
                onClose={handleCancel}
                title={isCreate ? t("CreateResource") : t("UpdateResource")}
                help={t("ResourceHelp")}
            />
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    console.log("creating resource", values, transformResourceValues(values, existing, isCreate));
                    if (isMutate) {
                        fetchLazyWrapper<ResourceCreateInput | ResourceUpdateInput, Resource>({
                            fetch,
                            inputs: transformResourceValues(values, existing, isCreate),
                            onSuccess: (data) => { handleCompleted(data); },
                            onCompleted: () => { helpers.setSubmitting(false); },
                        });
                    } else {
                        handleCompleted({
                            ...values,
                            created_at: (existing as Resource)?.created_at ?? new Date().toISOString(),
                            updated_at: (existing as Resource)?.updated_at ?? new Date().toISOString(),
                        } as Resource);
                    }
                }}
                validate={async (values) => await validateResourceValues(values, existing, isCreate)}
            >
                {(formik) => <ResourceForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
