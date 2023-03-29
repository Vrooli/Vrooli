import { Organization, OrganizationCreateInput } from "@shared/consts";
import { organizationCreate } from "api/generated/endpoints/organization_create";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { OrganizationForm } from "forms/OrganizationForm/OrganizationForm";
import { useContext, useMemo, useRef } from "react";
import { useCreateActions } from "utils/hooks/useCreateActions";
import { SessionContext } from "utils/SessionContext";
import { shapeOrganization } from "utils/shape/models/organization";
import { organizationInitialValues, validateOrganizationValues } from "..";
import { OrganizationCreateProps } from "../types";

export const OrganizationCreate = ({
    display = 'page',
    onCancel,
    onCreated,
    zIndex = 200,
}: OrganizationCreateProps) => {
    const session = useContext(SessionContext);

    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => organizationInitialValues(session), [session]);
    const { handleCancel, handleCreated } = useCreateActions<Organization>(display, onCancel, onCreated);
    const [mutation, { loading: isLoading }] = useCustomMutation<Organization, OrganizationCreateInput>(organizationCreate);

    return (
        <>
            <TopBar
                display={display}
                onClose={handleCancel}
                titleData={{
                    titleKey: 'CreateOrganization',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={(values, helpers) => {
                    mutationWrapper<Organization, OrganizationCreateInput>({
                        mutation,
                        input: shapeOrganization.create(values),
                        onSuccess: (data) => { handleCreated(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
                }}
                validate={async (values) => await validateOrganizationValues(values, true)}
            >
                {(formik) => <OrganizationForm
                    display={display}
                    isCreate={true}
                    isLoading={isLoading}
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