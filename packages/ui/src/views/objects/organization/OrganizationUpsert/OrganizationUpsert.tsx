import { endpointGetOrganization, endpointPostOrganization, endpointPutOrganization, Organization, OrganizationCreateInput, OrganizationUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { OrganizationForm, organizationInitialValues, transformOrganizationValues, validateOrganizationValues } from "forms/OrganizationForm/OrganizationForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { OrganizationShape } from "utils/shape/models/organization";
import { OrganizationUpsertProps } from "../types";

export const OrganizationUpsert = ({
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
    overrideObject,
}: OrganizationUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const formRef = useRef<BaseFormRef>();
    const display = toDisplay(isOpen);
    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<Organization, OrganizationShape>({
        ...endpointGetOrganization,
        objectType: "Organization",
        overrideObject,
        transform: (existing) => organizationInitialValues(session, existing),
    });
    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<Organization, OrganizationCreateInput, OrganizationUpdateInput>({
        display,
        endpointCreate: endpointPostOrganization,
        endpointUpdate: endpointPutOrganization,
        isCreate,
        onCancel,
        onCompleted,
    });

    return (
        <MaybeLargeDialog
            display={display}
            id="organization-upsert-dialog"
            isOpen={isOpen ?? false}
            onClose={handleCancel}
        >
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t(isCreate ? "CreateOrganization" : "UpdateOrganization")}
            />
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<OrganizationCreateInput | OrganizationUpdateInput, Organization>({
                        fetch,
                        inputs: transformOrganizationValues(values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateOrganizationValues(values, existing, isCreate)}
            >
                {(formik) => <OrganizationForm
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
