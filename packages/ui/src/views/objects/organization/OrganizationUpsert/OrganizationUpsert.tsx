import { endpointGetOrganization, endpointPostOrganization, endpointPutOrganization, FindByIdInput, Organization, OrganizationCreateInput, OrganizationUpdateInput } from "@local/shared";
import { useLazyFetch } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { OrganizationForm, organizationInitialValues, transformOrganizationValues, validateOrganizationValues } from "forms/OrganizationForm/OrganizationForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { OrganizationUpsertProps } from "../types";

export const OrganizationUpsert = ({
    display = "page",
    isCreate,
    onCancel,
    onCompleted,
    zIndex = 200,
}: OrganizationUpsertProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => isCreate ? { id: undefined } : parseSingleItemUrl(), [isCreate]);
    const [getData, { data: existing, loading: isReadLoading }] = useLazyFetch<FindByIdInput, Organization>(endpointGetOrganization);
    useEffect(() => { id && getData({ id }); }, [getData, id]);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => organizationInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleCompleted } = useUpsertActions<Organization>(display, isCreate, onCancel, onCompleted);
    const [create, { loading: isCreateLoading }] = useLazyFetch<OrganizationCreateInput, Organization>(endpointPostOrganization);
    const [update, { loading: isUpdateLoading }] = useLazyFetch<OrganizationUpdateInput, Organization>(endpointPutOrganization);
    const mutation = isCreate ? create : update;

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: isCreate ? "CreateOrganization" : "UpdateOrganization",
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    mutationWrapper<Organization, OrganizationCreateInput | OrganizationUpdateInput>({
                        mutation,
                        input: transformOrganizationValues(values, existing),
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
