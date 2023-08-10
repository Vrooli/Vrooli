import { endpointGetOrganization, endpointPostOrganization, endpointPutOrganization, Organization, OrganizationCreateInput, OrganizationUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { OrganizationForm, organizationInitialValues, transformOrganizationValues, validateOrganizationValues } from "forms/OrganizationForm/OrganizationForm";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { OrganizationShape } from "utils/shape/models/organization";
import { OrganizationUpsertProps } from "../types";

export const OrganizationUpsert = ({
    isCreate,
    onCancel,
    onCompleted,
    overrideObject,
    zIndex,
}: OrganizationUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const { display, isLoading: isReadLoading, object: existing } = useObjectFromUrl<Organization, OrganizationShape>({
        ...endpointGetOrganization,
        objectType: "Organization",
        overrideObject,
        transform: (existing) => organizationInitialValues(session, existing),
    });

    const formRef = useRef<BaseFormRef>();
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
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                title={t(isCreate ? "CreateOrganization" : "UpdateOrganization")}
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
                    fetchLazyWrapper<OrganizationCreateInput | OrganizationUpdateInput, Organization>({
                        fetch,
                        inputs: transformOrganizationValues(values, existing),
                        onSuccess: (data) => { handleCompleted(data); },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateOrganizationValues(values, existing)}
            >
                {(formik) => <OrganizationForm
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
