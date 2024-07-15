import { DUMMY_ID, endpointGetStandardVersion, endpointPostStandardVersion, endpointPutStandardVersion, InputType, LINKS, noopSubmit, orDefault, Session, StandardVersion, StandardVersionCreateInput, standardVersionTranslationValidation, StandardVersionUpdateInput, standardVersionValidation } from "@local/shared";
import { Button, Divider, useTheme } from "@mui/material";
import { useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { SearchExistingButton } from "components/buttons/SearchExistingButton/SearchExistingButton";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { CodeInput, CodeLanguage } from "components/inputs/CodeInput/CodeInput";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TranslatedRichInput } from "components/inputs/RichInput/RichInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedTextInput } from "components/inputs/TextInput/TextInput";
import { ToggleSwitch } from "components/inputs/ToggleSwitch/ToggleSwitch";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
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
import { BuildIcon, HelpIcon, VisibleIcon } from "icons";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { SearchPageTabOption } from "utils/search/objectToSearch";
import { shapeStandardVersion, StandardVersionShape } from "utils/shape/models/standardVersion";
import { validateFormValues } from "utils/validateFormValues";
import { StandardFormProps, StandardUpsertProps } from "../types";

export function standardInitialValues(
    session: Session | undefined,
    existing?: Partial<StandardVersion> | null | undefined,
): StandardVersionShape {
    return {
        __typename: "StandardVersion" as const,
        id: DUMMY_ID,
        directoryListings: [],
        isComplete: false,
        isPrivate: false,
        isFile: false,
        standardType: InputType.JSON,
        props: JSON.stringify({}),
        default: JSON.stringify({}),
        yup: JSON.stringify({}),
        resourceList: {
            __typename: "ResourceList" as const,
            id: DUMMY_ID,
            listFor: {
                __typename: "StandardVersion" as const,
                id: DUMMY_ID,
            },
        },
        versionLabel: "1.0.0",
        ...existing,
        root: {
            __typename: "Standard" as const,
            id: DUMMY_ID,
            isInternal: false,
            isPrivate: false,
            owner: { __typename: "User", id: getCurrentUser(session)?.id ?? "" },
            parent: null,
            permissions: JSON.stringify({}),
            tags: [],
            ...existing?.root,
        },
        translations: orDefault(existing?.translations, [{
            __typename: "StandardVersionTranslation" as const,
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            description: "",
            jsonVariable: null, //TODO
            name: "",
        }]),
    };
}

function transformStandardVersionValues(values: StandardVersionShape, existing: StandardVersionShape, isCreate: boolean) {
    return isCreate ? shapeStandardVersion.create(values) : shapeStandardVersion.update(existing, values);
}

function StandardForm({
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
    versions,
    ...props
}: StandardFormProps) {
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
        validationSchema: standardVersionTranslationValidation.create({ env: process.env.NODE_ENV }),
    });

    const { handleCancel, handleCompleted } = useUpsertActions<StandardVersion>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "StandardVersion",
        rootObjectId: values.root?.id,
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<StandardVersion, StandardVersionCreateInput, StandardVersionUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointPostStandardVersion,
        endpointUpdate: endpointPutStandardVersion,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "StandardVersion" });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<StandardVersionCreateInput | StandardVersionUpdateInput, StandardVersion>({
        disabled,
        existing,
        fetch,
        inputs: transformStandardVersionValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    // Toggle preview/edit mode
    const [isPreviewOn, setIsPreviewOn] = useState<boolean>(false);
    const onPreviewChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => { setIsPreviewOn(event.target.checked); }, []);

    function showExample() {
        console.log("TODO");
    }

    return (
        <MaybeLargeDialog
            display={display}
            id="standard-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateStandard" : "UpdateStandard")}
            />
            <SearchExistingButton
                href={`${LINKS.Search}?type="${SearchPageTabOption.Standard}"`}
                text="Search existing standards"
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
                            objectType={"Standard"}
                            sx={{ marginBottom: 2 }}
                        />
                        <ResourceListInput
                            horizontal
                            isCreate={true}
                            parent={{ __typename: "StandardVersion", id: values.id }}
                            sxs={{ list: { marginBottom: 2 } }}
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
                                name="description"
                                maxChars={2048}
                                minRows={4}
                                maxRows={8}
                                placeholder={t("DescriptionPlaceholder")}
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
                        <VersionInput
                            fullWidth
                            versions={versions}
                            sx={{ marginBottom: 2 }}
                        />
                    </ContentCollapse>
                    <Divider />
                    <ContentCollapse
                        title="Standard"
                        titleVariant="h4"
                        isOpen={true}
                        toTheRight={
                            <>
                                <ToggleSwitch
                                    checked={isPreviewOn}
                                    onChange={onPreviewChange}
                                    OffIcon={BuildIcon}
                                    OnIcon={VisibleIcon}
                                    label={isPreviewOn ? "Preview" : "Edit"}
                                    tooltip={isPreviewOn ? "Switch to edit" : "Switch to preview"}
                                    sx={{ marginBottom: 2 }}
                                />
                                <Button
                                    variant="outlined"
                                    onClick={showExample}
                                    startIcon={<HelpIcon />}
                                    sx={{ marginLeft: 2 }}
                                >
                                    Show example
                                </Button>
                                {/* <IconButton
                                    onClick={toggleFullscreen}
                                    sx={{ marginLeft: 2 }}
                                >
                                    {isFullscreen ? <FullscreenExitIcon /> : <FullscreenIcon />}
                                </IconButton> */}
                            </>
                        }
                        sxs={{ titleContainer: { marginBottom: 1 } }}
                    >
                        {/* TODO replace with FormInputStandard */}
                        <CodeInput
                            disabled={false}
                            // format={isPreviewOn ? "TODO" : undefined}
                            limitTo={isPreviewOn ? [CodeLanguage.JsonStandard] : [CodeLanguage.Json]}
                            name="content"
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
        </MaybeLargeDialog >
    );
}

export function StandardUpsert({
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: StandardUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useObjectFromUrl<StandardVersion, StandardVersionShape>({
        ...endpointGetStandardVersion,
        isCreate,
        objectType: "StandardVersion",
        overrideObject,
        transform: (existing) => standardInitialValues(session, existing),
    });

    async function validateValues(values: StandardVersionShape) {
        return await validateFormValues(values, existing, isCreate, transformStandardVersionValues, standardVersionValidation);
    }

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <StandardForm
                disabled={!(isCreate || permissions.canUpdate)}
                existing={existing}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                versions={existing?.root?.versions?.map(v => v.versionLabel) ?? []}
                {...props}
                {...formik}
            />}
        </Formik>
    );
}
