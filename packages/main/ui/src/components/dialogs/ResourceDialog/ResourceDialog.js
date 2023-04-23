import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { Formik } from "formik";
import { useCallback, useContext, useMemo, useRef } from "react";
import { resourceCreate } from "../../../api/generated/endpoints/resource_create";
import { resourceUpdate } from "../../../api/generated/endpoints/resource_update";
import { useCustomMutation } from "../../../api/hooks";
import { mutationWrapper } from "../../../api/utils";
import { ResourceForm, resourceInitialValues, transformResourceValues, validateResourceValues } from "../../../forms/ResourceForm/ResourceForm";
import { PubSub } from "../../../utils/pubsub";
import { SessionContext } from "../../../utils/SessionContext";
import { shapeResource } from "../../../utils/shape/models/resource";
import { DialogTitle } from "../DialogTitle/DialogTitle";
import { LargeDialog } from "../LargeDialog/LargeDialog";
const helpText = "## What are resources?\n\nResources provide context to the object they are attached to, such as a  user, organization, project, or routine.\n\n## Examples\n**For a user** - Social media links, GitHub profile, Patreon\n\n**For an organization** - Official website, tools used by your team, news article explaining the vision\n\n**For a project** - Project Catalyst proposal, Donation wallet address\n\n**For a routine** - Guide, external service";
const titleId = "resource-dialog-title";
export const ResourceDialog = ({ mutate, isOpen, onClose, onCreated, onUpdated, index, partialData, listId, zIndex, }) => {
    const session = useContext(SessionContext);
    const formRef = useRef();
    const initialValues = useMemo(() => resourceInitialValues(session, listId, partialData), [listId, partialData, session]);
    const [addMutation, { loading: addLoading }] = useCustomMutation(resourceCreate);
    const [updateMutation, { loading: updateLoading }] = useCustomMutation(resourceUpdate);
    const handleClose = useCallback((_, reason) => {
        formRef.current?.handleClose(onClose, reason !== "backdropClick");
    }, [onClose]);
    return (_jsx(_Fragment, { children: _jsxs(LargeDialog, { id: "resource-dialog", onClose: handleClose, isOpen: isOpen, titleId: titleId, zIndex: zIndex, children: [_jsx(DialogTitle, { id: titleId, title: (index < 0) ? "Add Resource" : "Update Resource", helpText: helpText, onClose: handleClose }), _jsx(Formik, { enableReinitialize: true, initialValues: initialValues, onSubmit: (values, helpers) => {
                        if (mutate) {
                            const onSuccess = (data) => {
                                (index < 0) ? onCreated(data) : onUpdated(index ?? 0, data);
                                helpers.resetForm();
                                onClose();
                            };
                            console.log("yeeeet", index, values, shapeResource.create(values));
                            const isCreating = index < 0;
                            if (!isCreating && (!partialData || !partialData.id)) {
                                PubSub.get().publishSnack({ messageKey: "ResourceNotFound", severity: "Error" });
                                return;
                            }
                            mutationWrapper({
                                mutation: isCreating ? addMutation : updateMutation,
                                input: transformResourceValues(values, partialData),
                                successMessage: () => ({ key: isCreating ? "ResourceCreated" : "ResourceUpdated" }),
                                successCondition: (data) => data !== null,
                                onSuccess,
                                onError: () => { helpers.setSubmitting(false); },
                            });
                        }
                        else {
                            onCreated({
                                ...values,
                                created_at: partialData?.created_at ?? new Date().toISOString(),
                                updated_at: partialData?.updated_at ?? new Date().toISOString(),
                            });
                            helpers.resetForm();
                            onClose();
                        }
                    }, validate: async (values) => await validateResourceValues(values, partialData), children: (formik) => _jsx(ResourceForm, { display: "dialog", isCreate: index < 0, isLoading: addLoading || updateLoading, isOpen: isOpen, onCancel: handleClose, ref: formRef, zIndex: zIndex, ...formik }) })] }) }));
};
//# sourceMappingURL=ResourceDialog.js.map