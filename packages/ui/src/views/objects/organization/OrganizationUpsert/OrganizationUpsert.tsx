import { endpointGetOrganization, endpointPostOrganization, endpointPutOrganization, Organization, OrganizationCreateInput, OrganizationUpdateInput } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { OrganizationForm, organizationInitialValues, transformOrganizationValues, validateOrganizationValues } from "forms/OrganizationForm/OrganizationForm";
import { useContext, useRef } from "react";
import { useTranslation } from "react-i18next";
import { MakeLazyRequest, useLazyFetch } from "utils/hooks/useLazyFetch";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { OrganizationShape } from "utils/shape/models/organization";
import { OrganizationUpsertProps } from "../types";

export const OrganizationUpsert = ({
    display = "page",
    isCreate,
    onCancel,
    onCompleted,
    zIndex,
}: OrganizationUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<Organization, OrganizationShape>({
        ...endpointGetOrganization,
        objectType: "Organization",
        upsertTransform: (existing) => organizationInitialValues(session, existing),
    });

    const formRef = useRef<BaseFormRef>();
    const { handleCancel, handleCompleted } = useUpsertActions<Organization>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useLazyFetch<OrganizationCreateInput, Organization>(endpointPostOrganization);
    const [update, { loading: isUpdateLoading }] = useLazyFetch<OrganizationUpdateInput, Organization>(endpointPutOrganization);
    const fetch = (isCreate ? create : update) as MakeLazyRequest<OrganizationCreateInput | OrganizationUpdateInput, Organization>;

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
