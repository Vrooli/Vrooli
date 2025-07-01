import { LINKS, LlmTask, SearchPageTabOption, createTransformFunction, noopSubmit, teamFormConfig, type Team, type TeamShape } from "@vrooli/shared";
import { Formik } from "formik";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { AutoFillButton } from "../../../components/buttons/AutoFillButton.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { SearchExistingButton } from "../../../components/buttons/SearchExistingButton.js";
import { ContentCollapse } from "../../../components/containers/ContentCollapse.js";
import { Dialog } from "../../../components/dialogs/Dialog/Dialog.js";
import { LanguageInput } from "../../../components/inputs/LanguageInput/LanguageInput.js";
import { ProfilePictureInput } from "../../../components/inputs/ProfilePictureInput/ProfilePictureInput.js";
import { TranslatedAdvancedInput } from "../../../components/inputs/AdvancedInput/AdvancedInput.js";
import { TagSelector } from "../../../components/inputs/TagSelector/TagSelector.js";
import { TranslatedTextInput } from "../../../components/inputs/TextInput/TextInput.js";
import { RelationshipList } from "../../../components/lists/RelationshipList/RelationshipList.js";
import { ResourceListInput } from "../../../components/lists/ResourceList/ResourceList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { SessionContext } from "../../../contexts/session.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useStandardUpsertForm } from "../../../hooks/useStandardUpsertForm.js";
import { getAutoFillTranslationData, useAutoFill, type UseAutoFillProps } from "../../../hooks/tasks.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { FormContainer, FormSection } from "../../../styles.js";
import { type TeamFormProps, type TeamUpsertProps } from "./types.js";


const relationshipListStyle = { marginBottom: 4 } as const;
const formSectionStyle = { overflowX: "hidden", marginBottom: 2 } as const;
const resourceListStyle = { list: { marginBottom: 2 } } as const;

function TeamForm({
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
    setFieldValue,
    values,
    ...props
}: TeamFormProps) {
    const { t } = useTranslation();

    // Use the standardized form hook with config
    const {
        session,
        isLoading,
        handleCancel,
        handleCompleted,
        onSubmit,
        validateValues,
        language,
        languages,
        handleAddLanguage,
        handleDeleteLanguage,
        setLanguage,
        translationErrors,
    } = useStandardUpsertForm({
        objectType: "Team",
        validation: teamFormConfig.validation.schema,
        translationValidation: teamFormConfig.validation.translationSchema,
        transformFunction: createTransformFunction(teamFormConfig),
        endpoints: teamFormConfig.endpoints,
    }, {
        values,
        existing,
        isCreate,
        display,
        disabled,
        isReadLoading,
        isSubmitting: props.isSubmitting,
        handleUpdate,
        setSubmitting: props.setSubmitting,
        onCancel,
        onCompleted,
        onDeleted,
        onClose,
    });

    const resourceListParent = useMemo(function resourceListParentMemo() {
        return { __typename: "Team", id: values.id } as const;
    }, [values]);

    const getAutoFillInput = useCallback(function getAutoFillInput() {
        return {
            ...getAutoFillTranslationData(values, language),
            //TODO
        };
    }, [language, values]);

    const shapeAutoFillResult = useCallback(function shapeAutoFillResultCallback(data: Parameters<UseAutoFillProps["shapeAutoFillResult"]>[0]) {
        const originalValues = { ...values };
        const updatedValues = {} as any; //TODO
        return { originalValues, updatedValues };
    }, [language, values]);

    const { autoFill, isAutoFillLoading } = useAutoFill({
        getAutoFillInput,
        shapeAutoFillResult,
        handleUpdate,
        task: isCreate ? LlmTask.TeamAdd : LlmTask.TeamUpdate,
    });

    // Combine loading states
    const combinedIsLoading = isLoading || isAutoFillLoading;

    const handleBannerImageChange = useCallback(function handleBannerImageChangeCallback(newPicture: File | null) {
        setFieldValue("bannerImage", newPicture);
    }, [setFieldValue]);
    const handleProfileImageChange = useCallback(function handleProfileImageChangeCallback(newPicture: File | null) {
        setFieldValue("profileImage", newPicture);
    }, [setFieldValue]);

    return (
        <>
            {display === "Dialog" ? (
                <Dialog
                    isOpen={isOpen ?? false}
                    onClose={onClose ?? (() => console.warn("onClose not passed to dialog"))}
                    size="md"
                >
                    <TopBar
                        display={display}
                        onClose={onClose}
                        title={t(isCreate ? "CreateTeam" : "UpdateTeam")}
                    />
                    <SearchExistingButton
                        href={`${LINKS.Search}?type="${SearchPageTabOption.Team}"`}
                        text="Search existing teams"
                    />
                    <BaseForm
                        display={display}
                        isLoading={combinedIsLoading}
                        maxWidth={700}
                    >
                        <FormContainer>
                            <ContentCollapse title="Basic info" titleVariant="h4" isOpen={true} sxs={{ titleContainer: { marginBottom: 1 } }}>
                                <RelationshipList
                                    isEditing={true}
                                    objectType={"Team"}
                                    sx={relationshipListStyle}
                                />
                                <ProfilePictureInput
                                    onBannerImageChange={handleBannerImageChange}
                                    onProfileImageChange={handleProfileImageChange}
                                    name="profileImage"
                                    profile={values}
                                />
                                <FormSection sx={formSectionStyle}>
                                    <TranslatedTextInput
                                        fullWidth
                                        label={t("Name")}
                                        language={language}
                                        name="name"
                                        placeholder={t("NamePlaceholder")}
                                    />
                                    <TranslatedAdvancedInput
                                        language={language}
                                        features={{ maxChars: 2048, minRowsCollapsed: 4 }}
                                        name="bio"
                                        placeholder={t("Bio")}
                                    />
                                    <LanguageInput
                                        currentLanguage={language}
                                        flexDirection="row-reverse"
                                        handleAdd={handleAddLanguage}
                                        handleDelete={handleDeleteLanguage}
                                        handleCurrent={setLanguage}
                                        languages={languages}
                                    />
                                </FormSection>
                                <TagSelector name="root.tags" sx={{ marginBottom: 2 }} />
                                <ResourceListInput
                                    horizontal
                                    isCreate={true}
                                    parent={resourceListParent}
                                    sxs={resourceListStyle}
                                />
                            </ContentCollapse>
                        </FormContainer>
                    </BaseForm>
                    <BottomActionsButtons
                        display={display}
                        errors={translationErrors}
                        hideButtons={disabled}
                        isCreate={isCreate}
                        loading={combinedIsLoading}
                        onCancel={handleCancel}
                        onSetSubmitting={props.setSubmitting}
                        onSubmit={onSubmit}
                        sideActionButtons={<AutoFillButton
                            handleAutoFill={autoFill}
                            isAutoFillLoading={isAutoFillLoading}
                        />}
                    />
                </Dialog>
            ) : (
                <>
                    <TopBar
                        display={display}
                        onClose={onClose}
                        title={t(isCreate ? "CreateTeam" : "UpdateTeam")}
                    />
                    <SearchExistingButton
                        href={`${LINKS.Search}?type="${SearchPageTabOption.Team}"`}
                        text="Search existing teams"
                    />
                    <BaseForm
                        display={display}
                        isLoading={combinedIsLoading}
                        maxWidth={700}
                    >
                        <FormContainer>
                            <ContentCollapse title="Basic info" titleVariant="h4" isOpen={true} sxs={{ titleContainer: { marginBottom: 1 } }}>
                                <RelationshipList
                                    isEditing={true}
                                    objectType={"Team"}
                                    sx={relationshipListStyle}
                                />
                                <ProfilePictureInput
                                    onBannerImageChange={handleBannerImageChange}
                                    onProfileImageChange={handleProfileImageChange}
                                    name="profileImage"
                                    profile={values}
                                />
                                <FormSection sx={formSectionStyle}>
                                    <TranslatedTextInput
                                        fullWidth
                                        label={t("Name")}
                                        language={language}
                                        name="name"
                                        placeholder={t("NamePlaceholder")}
                                    />
                                    <TranslatedAdvancedInput
                                        language={language}
                                        features={{ maxChars: 2048, minRowsCollapsed: 4 }}
                                        name="bio"
                                        placeholder={t("Bio")}
                                    />
                                    <LanguageInput
                                        currentLanguage={language}
                                        flexDirection="row-reverse"
                                        handleAdd={handleAddLanguage}
                                        handleDelete={handleDeleteLanguage}
                                        handleCurrent={setLanguage}
                                        languages={languages}
                                    />
                                </FormSection>
                                <TagSelector name="root.tags" sx={{ marginBottom: 2 }} />
                                <ResourceListInput
                                    horizontal
                                    isCreate={true}
                                    parent={resourceListParent}
                                    sxs={resourceListStyle}
                                />
                            </ContentCollapse>
                        </FormContainer>
                    </BaseForm>
                    <BottomActionsButtons
                        display={display}
                        errors={translationErrors}
                        hideButtons={disabled}
                        isCreate={isCreate}
                        loading={combinedIsLoading}
                        onCancel={handleCancel}
                        onSetSubmitting={props.setSubmitting}
                        onSubmit={onSubmit}
                        sideActionButtons={<AutoFillButton
                            handleAutoFill={autoFill}
                            isAutoFillLoading={isAutoFillLoading}
                        />}
                    />
                </>
            )}
        </>
    );
}

export function TeamUpsert({
    display,
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: TeamUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<Team, TeamShape>({
        ...teamFormConfig.endpoints.findOne,
        disabled: display === "Dialog" && isOpen !== true,
        isCreate,
        objectType: "Team",
        overrideObject,
        transform: (existing) => teamFormConfig.transformations.getInitialValues(session.session, existing),
    });

    // Ensure we always have valid initial values
    const formValues = existing || teamFormConfig.transformations.getInitialValues(session.session);

    // Simple validation for the wrapper Formik (the actual validation happens in TeamForm via useStandardUpsertForm)
    const validateValues = useCallback(async (values: TeamShape) => {
        // This is just a placeholder - the real validation happens in TeamForm via useStandardUpsertForm
        return {};
    }, []);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={formValues}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <TeamForm
                disabled={!(isCreate || permissions.canUpdate)}
                display={display}
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
}
