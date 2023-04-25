import { Resource, ResourceCreateInput, ResourceUpdateInput } from "@local/shared";
import { resourceCreate } from "api/generated/endpoints/resource_create";
import { resourceUpdate } from "api/generated/endpoints/resource_update";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from "api/utils";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ResourceForm, resourceInitialValues, transformResourceValues, validateResourceValues } from "forms/ResourceForm/ResourceForm";
import { useCallback, useContext, useMemo, useRef } from "react";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { shapeResource } from "utils/shape/models/resource";
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
    index,
    partialData,
    listId,
    zIndex,
}: ResourceDialogProps) => {
    const session = useContext(SessionContext);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => resourceInitialValues(session, listId, partialData as any), [listId, partialData, session]);
    const [addMutation, { loading: addLoading }] = useCustomMutation<Resource, ResourceCreateInput>(resourceCreate);
    const [updateMutation, { loading: updateLoading }] = useCustomMutation<Resource, ResourceUpdateInput>(resourceUpdate);

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
                    title={(index < 0) ? "Add Resource" : "Update Resource"}
                    helpText={helpText}
                    onClose={handleClose}
                />
                <Formik
                    enableReinitialize={true}
                    initialValues={initialValues}
                    onSubmit={(values, helpers) => {
                        if (mutate) {
                            const onSuccess = (data: Resource) => {
                                (index < 0) ? onCreated(data) : onUpdated(index ?? 0, data);
                                helpers.resetForm();
                                onClose();
                            };
                            console.log("yeeeet", index, values, shapeResource.create(values));
                            // If index is negative, create
                            const isCreating = index < 0;
                            if (!isCreating && (!partialData || !partialData.id)) {
                                PubSub.get().publishSnack({ messageKey: "ResourceNotFound", severity: "Error" });
                                return;
                            }
                            mutationWrapper<Resource, ResourceCreateInput | ResourceUpdateInput>({
                                mutation: isCreating ? addMutation : updateMutation,
                                input: transformResourceValues(values, partialData as any),
                                successMessage: () => ({ key: isCreating ? "ResourceCreated" : "ResourceUpdated" }),
                                successCondition: (data) => data !== null,
                                onSuccess,
                                onError: () => { helpers.setSubmitting(false); },
                            });
                        } else {
                            onCreated({
                                ...values,
                                created_at: partialData?.created_at ?? new Date().toISOString(),
                                updated_at: partialData?.updated_at ?? new Date().toISOString(),
                            } as Resource);
                            helpers.resetForm();
                            onClose();
                        }
                    }}
                    validate={async (values) => await validateResourceValues(values, partialData as any)}
                >
                    {(formik) => <ResourceForm
                        display="dialog"
                        isCreate={index < 0}
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
