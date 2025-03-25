import { CodeLanguage, DUMMY_ID, endpointsStandardVersion, LINKS, LlmTask, noopSubmit, orDefault, SearchPageTabOption, Session, shapeStandardVersion, StandardType, StandardVersion, StandardVersionCreateInput, StandardVersionShape, standardVersionTranslationValidation, StandardVersionUpdateInput, standardVersionValidation } from "@local/shared";
import { Button, Divider } from "@mui/material";
import { Formik, useField } from "formik";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSubmitHelper } from "../../../api/fetchWrapper.js";
import { AutoFillButton } from "../../../components/buttons/AutoFillButton.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { SearchExistingButton } from "../../../components/buttons/SearchExistingButton/SearchExistingButton.js";
import { ContentCollapse } from "../../../components/containers/ContentCollapse/ContentCollapse.js";
import { MaybeLargeDialog } from "../../../components/dialogs/LargeDialog/LargeDialog.js";
import { CodeInput } from "../../../components/inputs/CodeInput/CodeInput.js";
import { LanguageInput } from "../../../components/inputs/LanguageInput/LanguageInput.js";
import { TranslatedRichInput } from "../../../components/inputs/RichInput/RichInput.js";
import { TagSelector } from "../../../components/inputs/TagSelector/TagSelector.js";
import { TranslatedTextInput } from "../../../components/inputs/TextInput/TextInput.js";
import { ToggleSwitch } from "../../../components/inputs/ToggleSwitch/ToggleSwitch.js";
import { VersionInput } from "../../../components/inputs/VersionInput/VersionInput.js";
import { RelationshipList } from "../../../components/lists/RelationshipList/RelationshipList.js";
import { ResourceListInput } from "../../../components/lists/ResourceList/ResourceList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { SessionContext } from "../../../contexts.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "../../../hooks/forms.js";
import { getAutoFillTranslationData, useAutoFill, UseAutoFillProps } from "../../../hooks/tasks.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useTranslatedFields } from "../../../hooks/useTranslatedFields.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { BuildIcon, HelpIcon, VisibleIcon } from "../../../icons/common.js";
import { FormContainer, FormSection } from "../../../styles.js";
import { getCurrentUser } from "../../../utils/authentication/session.js";
import { combineErrorsWithTranslations, getUserLanguages } from "../../../utils/display/translationTools.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { DataStructureFormProps, DataStructureUpsertProps } from "./types.js";

export function dataStructureInitialValues(
    session: Session | undefined,
    existing?: Partial<StandardVersion> | null | undefined,
): StandardVersionShape {
    return {
        __typename: "StandardVersion" as const,
        id: DUMMY_ID,
        codeLanguage: CodeLanguage.Json,
        default: JSON.stringify({}),
        directoryListings: [],
        isComplete: false,
        isPrivate: false,
        isFile: false,
        props: JSON.stringify({}),
        variant: StandardType.DataStructure,
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

function transformDataStructureVersionValues(values: StandardVersionShape, existing: StandardVersionShape, isCreate: boolean) {
    return isCreate ? shapeStandardVersion.create(values) : shapeStandardVersion.update(existing, values);
}

const exampleStandardJson = `
{
    TODO: "TODO",
}
`.trim();

const exampleStandardYaml = `
TODO: TODO
`.trim();

const codeLimitTo = [CodeLanguage.Json, CodeLanguage.Yaml] as const;
const relationshipListStyle = { marginBottom: 2 } as const;
const formSectionStyle = { overflowX: "hidden", marginBottom: 2 } as const;
const resourceListStyle = { list: { marginBottom: 2 } } as const;
const exampleButtonStyle = { marginLeft: "auto" } as const;

function DataStructureForm({
    disabled,
    display,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onClose,
    values,
    versions,
    ...props
}: DataStructureFormProps) {
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
        validationSchema: standardVersionTranslationValidation.create({ env: process.env.NODE_ENV }),
    });

    const resourceListParent = useMemo(function resourceListParentMemo() {
        return { __typename: "StandardVersion", id: values.id } as const;
    }, [values]);

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
        endpointCreate: endpointsStandardVersion.createOne,
        endpointUpdate: endpointsStandardVersion.updateOne,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "StandardVersion" });

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
        task: isCreate ? LlmTask.StandardAdd : LlmTask.StandardUpdate,
    });

    const isLoading = useMemo(() => isAutoFillLoading || isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isAutoFillLoading, isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<StandardVersionCreateInput | StandardVersionUpdateInput, StandardVersion>({
        disabled,
        existing,
        fetch,
        inputs: transformDataStructureVersionValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    // Toggle preview/edit mode
    const [isPreviewOn, setIsPreviewOn] = useState<boolean>(false);
    const onPreviewChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => { setIsPreviewOn(event.target.checked); }, []);

    const [standardTypeField, , standardTypeHelpers] = useField<CodeLanguage>("standardType");
    const [propsField, , propsHelpers] = useField<string>("props");
    const showExample = useCallback(function showExampleCallback() {
        // Determine example to show based on current language
        const exampleCode = standardTypeField.value === CodeLanguage.Json ? exampleStandardJson : exampleStandardYaml;
        // Set value to hard-coded example
        propsHelpers.setValue(exampleCode);
    }, [standardTypeField.value, propsHelpers]);

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
                title={t(isCreate ? "CreateDataStructure" : "UpdateDataStructure")}
            />
            <SearchExistingButton
                href={`${LINKS.Search}?type="${SearchPageTabOption.DataStructure}"`}
                text="Search existing standards"
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
            >
                <FormContainer>
                    <ContentCollapse title="Basic info" titleVariant="h4" isOpen={true} sxs={{ titleContainer: { marginBottom: 1 } }}>
                        <RelationshipList
                            isEditing={true}
                            objectType={"Standard"}
                            sx={relationshipListStyle}
                        />
                        <ResourceListInput
                            horizontal
                            isCreate={true}
                            parent={resourceListParent}
                            sxs={resourceListStyle}
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
                                name="description"
                                maxChars={2048}
                                minRows={4}
                                maxRows={8}
                                placeholder={t("DescriptionPlaceholder")}
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
                                    tooltip={isPreviewOn ? "Switch to edit" : "Switch to preview"}
                                />
                                <Button
                                    variant="outlined"
                                    onClick={showExample}
                                    startIcon={<HelpIcon />}
                                    sx={exampleButtonStyle}
                                >
                                    Show example
                                </Button>
                            </>
                        }
                        sxs={{ titleContainer: { marginBottom: 1 } }}
                    >
                        {/* TODO replace with FormInputStandard */}
                        <CodeInput
                            disabled={false}
                            codeLanguageField="standardType"
                            // format={isPreviewOn ? "TODO" : undefined}
                            // limitTo={isPreviewOn ? [CodeLanguage.JsonStandard] : [CodeLanguage.Json]}
                            limitTo={codeLimitTo}
                            name="props"
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
        </MaybeLargeDialog >
    );
}

export function DataStructureUpsert({
    display,
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: DataStructureUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<StandardVersion, StandardVersionShape>({
        ...endpointsStandardVersion.findOne,
        disabled: display === "dialog" && isOpen !== true,
        isCreate,
        objectType: "StandardVersion",
        overrideObject,
        transform: (existing) => dataStructureInitialValues(session, existing),
    });

    async function validateValues(values: StandardVersionShape) {
        return await validateFormValues(values, existing, isCreate, transformDataStructureVersionValues, standardVersionValidation);
    }

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <DataStructureForm
                disabled={!(isCreate || permissions.canUpdate)}
                display={display}
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
