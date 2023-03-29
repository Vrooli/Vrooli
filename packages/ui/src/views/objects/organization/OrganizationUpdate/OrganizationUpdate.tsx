import { FindByIdInput, Organization, OrganizationUpdateInput } from "@shared/consts";
import { organizationFindOne } from "api/generated/endpoints/organization_findOne";
import { organizationUpdate } from "api/generated/endpoints/organization_update";
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { OrganizationForm } from "forms/OrganizationForm/OrganizationForm";
import { useContext, useEffect, useMemo, useRef } from "react";
import { useUpdateActions } from "utils/hooks/useUpdateActions";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { shapeOrganization } from "utils/shape/models/organization";
import { organizationInitialValues, validateOrganizationValues } from "..";
import { OrganizationUpdateProps } from "../types";

export const OrganizationUpdate = ({
    display = 'page',
    onCancel,
    onUpdated,
    zIndex = 200,
}: OrganizationUpdateProps) => {
    const session = useContext(SessionContext);

    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: existing, loading: isReadLoading }] = useCustomLazyQuery<Organization, FindByIdInput>(organizationFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => organizationInitialValues(session, existing), [existing, session]);
    const { handleCancel, handleUpdated } = useUpdateActions<Organization>(display, onCancel, onUpdated);
    const [mutation, { loading: isUpdateLoading }] = useCustomMutation<Organization, OrganizationUpdateInput>(organizationUpdate);

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: 'UpdateOrganization',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    if (!existing) {
                        PubSub.get().publishSnack({ messageKey: 'CouldNotReadObject', severity: 'Error' });
                        return;
                    }
                    mutationWrapper<Organization, OrganizationUpdateInput>({
                        mutation,
                        input: shapeOrganization.update(existing, values),
                        onSuccess: (data) => { handleUpdated(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
                }}
                validate={async (values) => await validateOrganizationValues(values, false)}
            >
                {(formik) => <OrganizationForm
                    display={display}
                    isCreate={false}
                    isLoading={isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    )
}