import { DUMMY_ID, LINKS, Session, Team, TeamCreateInput, TeamUpdateInput, endpointGetTeam, endpointPostTeam, endpointPutTeam, noopSubmit, orDefault, teamTranslationValidation, teamValidation } from "@local/shared";
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
import { TeamShape, shapeTeam } from "utils/shape/models/team";
import { validateFormValues } from "utils/validateFormValues";
import { TeamFormProps, TeamUpsertProps } from "../types";

const teamInitialValues = (
    session: Session | undefined,
    existing?: Partial<Team> | null | undefined,
): TeamShape => ({
    __typename: "Team" as const,
    id: DUMMY_ID,
    isOpenToNewMembers: false,
    isPrivate: false,
    tags: [],
    ...existing,
    translations: orDefault(existing?.translations, [{
        __typename: "TeamTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        name: "",
        bio: "",
    }]),
});

const transformTeamValues = (values: TeamShape, existing: TeamShape, isCreate: boolean) =>
    isCreate ? shapeTeam.create(values) : shapeTeam.update(existing, values);

const TeamForm = ({
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
}: TeamFormProps) => {
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
        validationSchema: teamTranslationValidation.create({ env: process.env.NODE_ENV }),
    });

    const { handleCancel, handleCompleted } = useUpsertActions<Team>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "Team",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<Team, TeamCreateInput, TeamUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointPostTeam,
        endpointUpdate: endpointPutTeam,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "Team" });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<TeamCreateInput | TeamUpdateInput, Team>({
        disabled,
        existing,
        fetch,
        inputs: transformTeamValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    return (
        <MaybeLargeDialog
            display={display}
            id="team-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateTeam" : "UpdateTeam")}
            />
            <Button
                href={`${LINKS.Search}?type=${SearchPageTabOption.Team}`}
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
                            objectType={"Team"}
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
                            parent={{ __typename: "Team", id: values.id }}
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

export const TeamUpsert = ({
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: TeamUpsertProps) => {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useObjectFromUrl<Team, TeamShape>({
        ...endpointGetTeam,
        isCreate,
        objectType: "Team",
        overrideObject,
        transform: (existing) => teamInitialValues(session, existing),
    });

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateFormValues(values, existing, isCreate, transformTeamValues, teamValidation)}
        >
            {(formik) => <TeamForm
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
