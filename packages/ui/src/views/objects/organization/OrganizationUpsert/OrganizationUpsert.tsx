import { DUMMY_ID, endpointGetOrganization, endpointPostOrganization, endpointPutOrganization, LINKS, noopSubmit, orDefault, Organization, OrganizationCreateInput, organizationTranslationValidation, OrganizationUpdateInput, organizationValidation, Session } from "@local/shared";
import { Button, useTheme } from "@mui/material";
import { useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ProfilePictureInput } from "components/inputs/ProfilePictureInput/ProfilePictureInput";
import { TranslatedRichInput } from "components/inputs/RichInput/RichInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedTextInput } from "components/inputs/TextInput/TextInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { ResourceListInput } from "components/lists/resource/ResourceList/ResourceList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { SearchIcon } from "icons";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { SearchPageTabOption } from "utils/search/objectToSearch";
import { OrganizationShape, shapeOrganization } from "utils/shape/models/organization";
import { validateFormValues } from "utils/validateFormValues";
import { OrganizationFormProps, OrganizationUpsertProps } from "../types";

const organizationInitialValues = (
    session: Session | undefined,
    existing?: Partial<Organization> | null | undefined,
): OrganizationShape => ({
    __typename: "Organization" as const,
    id: DUMMY_ID,
    isOpenToNewMembers: false,
    isPrivate: false,
    tags: [],
    ...existing,
    translations: orDefault(existing?.translations, [{
        __typename: "OrganizationTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        name: "",
        bio: "",
    }]),
});

const transformOrganizationValues = (values: OrganizationShape, existing: OrganizationShape, isCreate: boolean) =>
    isCreate ? shapeOrganization.create(values) : shapeOrganization.update(existing, values);

const OrganizationForm = ({
    disabled,
    dirty,
    display,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    ...props
}: OrganizationFormProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { palette } = useTheme();

    // Handle translations
    const {
        handleAddLanguage,
        handleDeleteLanguage,
        language,
        languages,
        setLanguage,
        translationErrors,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        validationSchema: organizationTranslationValidation.create({ env: process.env.NODE_ENV }),
    });

    const { handleCancel, handleCompleted } = useUpsertActions<Organization>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "Organization",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<Organization, OrganizationCreateInput, OrganizationUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointPostOrganization,
        endpointUpdate: endpointPutOrganization,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "Organization" });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<OrganizationCreateInput | OrganizationUpdateInput, Organization>({
        disabled,
        existing,
        fetch,
        inputs: transformOrganizationValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    return (
        <MaybeLargeDialog
            display={display}
            id="organization-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateOrganization" : "UpdateOrganization")}
            />
            <Button
                href={`${LINKS.Search}?type=${SearchPageTabOption.Organization}`}
                sx={{
                    color: palette.background.textSecondary,
                    display: "flex",
                    marginTop: 2,
                    textAlign: "center",
                    textTransform: "none",
                }}
                variant="text"
                endIcon={<SearchIcon />}
            >
                Search existing teams
            </Button>
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={700}
            >
                <FormContainer>
                    <ContentCollapse title="Basic info" titleVariant="h4" isOpen={true} sxs={{ titleContainer: { marginBottom: 1 } }}>
                        <RelationshipList
                            isEditing={true}
                            objectType={"Organization"}
                            sx={{ marginBottom: 4 }}
                        />
                        <ProfilePictureInput
                            onBannerImageChange={(newPicture) => props.setFieldValue("bannerImage", newPicture)}
                            onProfileImageChange={(newPicture) => props.setFieldValue("profileImage", newPicture)}
                            name="profileImage"
                            profile={{ ...values }}
                        />
                        <FormSection sx={{ overflowX: "hidden", marginBottom: 2 }}>
                            <TranslatedTextInput
                                autoFocus
                                fullWidth
                                label={t("Name")}
                                language={language}
                                name="name"
                                placeholder={t("NamePlaceholder")}
                            />
                            <TranslatedRichInput
                                language={language}
                                maxChars={2048}
                                minRows={4}
                                name="bio"
                                placeholder={t("Bio")}
                            />
                            <LanguageInput
                                currentLanguage={language}
                                handleAdd={handleAddLanguage}
                                handleDelete={handleDeleteLanguage}
                                handleCurrent={setLanguage}
                                languages={languages}
                                sx={{ flexDirection: "row-reverse" }}
                            />
                        </FormSection>
                        <TagSelector name="root.tags" sx={{ marginBottom: 2 }} />
                        <ResourceListInput
                            horizontal
                            isCreate={true}
                            parent={{ __typename: "Organization", id: values.id }}
                            sxs={{ list: { marginBottom: 2 } }}
                        />
                    </ContentCollapse>
                </FormContainer>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                hideButtons={disabled}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
            />
        </MaybeLargeDialog>
    );
};

export const OrganizationUpsert = ({
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: OrganizationUpsertProps) => {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useObjectFromUrl<Organization, OrganizationShape>({
        ...endpointGetOrganization,
        isCreate,
        objectType: "Organization",
        overrideObject,
        transform: (existing) => organizationInitialValues(session, existing),
    });

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateFormValues(values, existing, isCreate, transformOrganizationValues, organizationValidation)}
        >
            {(formik) => <OrganizationForm
                disabled={!(isCreate || permissions.canUpdate)}
                existing={existing}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                {...props}
                {...formik}
            />}
        </Formik>
    );
};
