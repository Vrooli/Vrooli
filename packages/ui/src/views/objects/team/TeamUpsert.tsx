import { DUMMY_ID, LINKS, LlmTask, SearchPageTabOption, Session, Team, TeamCreateInput, TeamShape, TeamUpdateInput, endpointsTeam, noopSubmit, orDefault, shapeTeam, teamTranslationValidation, teamValidation } from "@local/shared";
import { Formik } from "formik";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useSubmitHelper } from "../../../api/fetchWrapper.js";
import { AutoFillButton } from "../../../components/buttons/AutoFillButton.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { SearchExistingButton } from "../../../components/buttons/SearchExistingButton.js";
import { ContentCollapse } from "../../../components/containers/ContentCollapse.js";
import { MaybeLargeDialog } from "../../../components/dialogs/LargeDialog/LargeDialog.js";
import { LanguageInput } from "../../../components/inputs/LanguageInput/LanguageInput.js";
import { ProfilePictureInput } from "../../../components/inputs/ProfilePictureInput/ProfilePictureInput.js";
import { TranslatedRichInput } from "../../../components/inputs/RichInput/RichInput.js";
import { TagSelector } from "../../../components/inputs/TagSelector/TagSelector.js";
import { TranslatedTextInput } from "../../../components/inputs/TextInput/TextInput.js";
import { RelationshipList } from "../../../components/lists/RelationshipList/RelationshipList.js";
import { ResourceListInput } from "../../../components/lists/ResourceList/ResourceList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { SessionContext } from "../../../contexts/session.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "../../../hooks/forms.js";
import { UseAutoFillProps, getAutoFillTranslationData, useAutoFill } from "../../../hooks/tasks.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useTranslatedFields } from "../../../hooks/useTranslatedFields.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { FormContainer, FormSection } from "../../../styles.js";
import { ELEMENT_IDS } from "../../../utils/consts.js";
import { combineErrorsWithTranslations, getUserLanguages } from "../../../utils/display/translationTools.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { TeamFormProps, TeamUpsertProps } from "./types.js";

function teamInitialValues(
    session: Session | undefined,
    existing?: Partial<Team> | null | undefined,
): TeamShape {
    return {
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
    };
}

function transformTeamValues(values: TeamShape, existing: TeamShape, isCreate: boolean) {
    return isCreate ? shapeTeam.create(values) : shapeTeam.update(existing, values);
}

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
    const session = useContext(SessionContext);
    const { t } = useTranslation();

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

    const resourceListParent = useMemo(function resourceListParentMemo() {
        return { __typename: "Team", id: values.id } as const;
    }, [values]);

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
        endpointCreate: endpointsTeam.createOne,
        endpointUpdate: endpointsTeam.updateOne,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "Team" });

    const getAutoFillInput = useCallback(function getAutoFillInput() {
        return {
            ...getAutoFillTranslationData(values, language),
            //TODO
        };
    }, [language, values]);

    const shapeAutoFillResult = useCallback(function shapeAutoFillResultCallback(data: Parameters<UseAutoFillProps["shapeAutoFillResult"]>[0]) {
        const originalValues = { ...values };
        const updatedValues = {} as any; //TODO
        console.log("in shapeAutoFillResult", language, data, originalValues, updatedValues);
        return { originalValues, updatedValues };
    }, [language, values]);

    const { autoFill, isAutoFillLoading } = useAutoFill({
        getAutoFillInput,
        shapeAutoFillResult,
        handleUpdate,
        task: isCreate ? LlmTask.TeamAdd : LlmTask.TeamUpdate,
    });

    const isLoading = useMemo(() => isAutoFillLoading || isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isAutoFillLoading, isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<TeamCreateInput | TeamUpdateInput, Team>({
        disabled,
        existing,
        fetch,
        inputs: transformTeamValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    const handleBannerImageChange = useCallback(function handleBannerImageChangeCallback(newPicture: File | null) {
        setFieldValue("bannerImage", newPicture);
    }, [setFieldValue]);
    const handleProfileImageChange = useCallback(function handleProfileImageChangeCallback(newPicture: File | null) {
        setFieldValue("profileImage", newPicture);
    }, [setFieldValue]);

    return (
        <MaybeLargeDialog
            display={display}
            id={ELEMENT_IDS.TeamUpsertDialog}
            isOpen={isOpen}
            onClose={onClose}
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
                isLoading={isLoading}
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
                            <TranslatedRichInput
                                language={language}
                                maxChars={2048}
                                minRows={4}
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
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                hideButtons={disabled}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
                sideActionButtons={<AutoFillButton
                    handleAutoFill={autoFill}
                    isAutoFillLoading={isAutoFillLoading}
                />}
            />
        </MaybeLargeDialog>
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
        ...endpointsTeam.findOne,
        disabled: display === "dialog" && isOpen !== true,
        isCreate,
        objectType: "Team",
        overrideObject,
        transform: (existing) => teamInitialValues(session, existing),
    });

    async function validateValues(values: TeamShape) {
        return await validateFormValues(values, existing, isCreate, transformTeamValues, teamValidation);
    }

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
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
