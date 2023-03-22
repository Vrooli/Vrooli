import { Organization, OrganizationCreateInput } from "@shared/consts";
import { uuid } from '@shared/uuid';
import { organizationValidation } from '@shared/validation';
import { organizationCreate } from "api/generated/endpoints/organization_create";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { OrganizationForm } from "forms/OrganizationForm/OrganizationForm";
import { useContext, useRef } from "react";
import { getUserLanguages } from "utils/display/translationTools";
import { useCreateActions } from "utils/hooks/useCreateActions";
import { SessionContext } from "utils/SessionContext";
import { shapeOrganization } from "utils/shape/models/organization";
import { OrganizationCreateProps } from "../types";

export const OrganizationCreate = ({
    display = 'page',
    zIndex = 200,
}: OrganizationCreateProps) => {
    const session = useContext(SessionContext);

    const formRef = useRef<BaseFormRef>();
    const { onCancel, onCreated } = useCreateActions<Organization>();
    const [mutation, { loading: isLoading }] = useCustomMutation<Organization, OrganizationCreateInput>(organizationCreate);

    return (
        <>
            <TopBar
                display={display}
                onClose={onCancel}
                titleData={{
                    titleKey: 'CreateOrganization',
                }}
            />
            <Formik
                enableReinitialize={true}
                initialValues={{
                    __typename: 'Organization' as const,
                    id: uuid(),
                    isOpenToNewMembers: false,
                    translationsCreate: [{
                        id: uuid(),
                        language: getUserLanguages(session)[0],
                        name: '',
                        bio: '',
                    }]
                }}
                onSubmit={(values, helpers) => {
                    mutationWrapper<Organization, OrganizationCreateInput>({
                        mutation,
                        input: shapeOrganization.create(values),
                        onSuccess: (data) => { onCreated(data) },
                        onError: () => { helpers.setSubmitting(false) },
                    })
                }}
                validationSchema={organizationValidation.create({})}
            >
                {(formik) => <OrganizationForm
                    display={display}
                    isCreate={true}
                    isLoading={isLoading}
                    isOpen={true}
                    onCancel={onCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </>
    )
}