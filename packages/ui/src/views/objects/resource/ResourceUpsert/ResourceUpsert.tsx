import { endpointGetResource, endpointPostResource, endpointPutResource, Resource, ResourceCreateInput, ResourceUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { NewResourceShape, ResourceForm, resourceInitialValues, transformResourceValues, validateResourceValues } from "forms/ResourceForm/ResourceForm";
import { useFormDialog } from "hooks/useFormDialog";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useContext } from "react";
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
    const { formRef, handleClose } = useFormDialog({ handleCancel });

    return (
        <MaybeLargeDialog
            display={display}
            id="resource-upsert-dialog"
            isOpen={isOpen}
            onClose={handleClose}
        >
            <TopBar
                display={display}
                onClose={handleClose}
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
                    onClose={handleClose}
                    ref={formRef}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
