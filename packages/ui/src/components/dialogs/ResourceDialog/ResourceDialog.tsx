import { endpointPostResource, endpointPutResource, Resource, ResourceCreateInput, ResourceUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ResourceForm, resourceInitialValues, transformResourceValues, validateResourceValues } from "forms/ResourceForm/ResourceForm";
import { useCallback, useContext, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { MakeLazyRequest, useLazyFetch } from "utils/hooks/useLazyFetch";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { ResourceShape } from "utils/shape/models/resource";
import { DialogTitle } from "../DialogTitle/DialogTitle";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { ResourceDialogProps } from "../types";

const helpText =
    "## What are resources?\n\nResources provide context to the object they are attached to, such as a  user, organization, project, or routine.\n\n## Examples\n**For a user** - Social media links, GitHub profile, Patreon\n\n**For an organization** - Official website, tools used by your team, news article explaining the vision\n\n**For a project** - Project Catalyst proposal, Donation wallet address\n\n**For a routine** - Guide, external service";

const titleId = "resource-dialog-title";

export const ResourceDialog = ({
    mutate,
    isOpen,
    onClose,
    onCreated,
    onUpdated,
    resource,
    zIndex,
}: ResourceDialogProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => resourceInitialValues(session, resource), [resource, session]);
    const [create, { loading: addLoading }] = useLazyFetch<ResourceCreateInput, Resource>(endpointPostResource);
    const [update, { loading: updateLoading }] = useLazyFetch<ResourceUpdateInput, Resource>(endpointPutResource);
    const fetch = (resource.index < 0 ? create : update) as MakeLazyRequest<ResourceCreateInput | ResourceUpdateInput, Resource>;

    const handleClose = useCallback((_?: unknown, reason?: "backdropClick" | "escapeKeyDown") => {
        // Confirm dialog is dirty and closed by clicking outside
        formRef.current?.handleClose(onClose, reason !== "backdropClick");
    }, [onClose]);

    return (
        <>
            {/*  Main content */}
            <LargeDialog
                id="resource-dialog"
                onClose={handleClose}
                isOpen={isOpen}
                titleId={titleId}
                zIndex={zIndex}
            >
                <DialogTitle
                    id={titleId}
                    title={(resource.index < 0) ? t("CreateResource") : t("UpdateResource")}
                    help={helpText}
                    onClose={handleClose}
                    zIndex={zIndex}
                />
                <Formik
                    enableReinitialize={true}
                    initialValues={initialValues}
                    onSubmit={(values, helpers) => {
                        console.log("resource dialog onsubmit", mutate, values, resource);
                        const isCreating = resource.index < 0;
                        if (mutate) {
                            const onSuccess = (data: Resource) => {
                                isCreating ? onCreated(data) : onUpdated(resource.index, data);
                                helpers.resetForm();
                                onClose();
                            };
                            if (!isCreating && (!resource || !resource.id)) {
                                PubSub.get().publishSnack({ messageKey: "ResourceNotFound", severity: "Error" });
                                return;
                            }
                            fetchLazyWrapper<ResourceCreateInput | ResourceUpdateInput, Resource>({
                                fetch,
                                inputs: transformResourceValues(values, resource as ResourceShape),
                                successMessage: () => ({ messageKey: isCreating ? "ResourceCreated" : "ResourceUpdated" }),
                                successCondition: (data) => data !== null,
                                onSuccess,
                                onError: () => { helpers.setSubmitting(false); },
                            });
                        } else {
                            if (isCreating) {
                                onCreated({
                                    ...values,
                                    created_at: (resource as Resource)?.created_at ?? new Date().toISOString(),
                                    updated_at: (resource as Resource)?.updated_at ?? new Date().toISOString(),
                                } as Resource);
                            } else {
                                onUpdated(resource.index, values as Resource);
                            }
                            helpers.resetForm();
                            onClose();
                        }
                    }}
                    validate={async (values) => await validateResourceValues(values, resource as ResourceShape)}
                >
                    {(formik) => <ResourceForm
                        display="dialog"
                        isCreate={resource.index < 0}
                        isLoading={addLoading || updateLoading}
                        isOpen={isOpen}
                        onCancel={handleClose}
                        ref={formRef}
                        zIndex={zIndex}
                        {...formik}
                    />}
                </Formik>
            </LargeDialog>
        </>
    );
};
