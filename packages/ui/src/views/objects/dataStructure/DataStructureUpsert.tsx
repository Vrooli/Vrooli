import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Divider from "@mui/material/Divider";
import Grid from "@mui/material/Grid";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material/styles";
import { CodeLanguage, DUMMY_ID, LINKS, LlmTask, SearchPageTabOption, ResourceSubType, dataStructureFormConfig, endpointsResource, noopSubmit, orDefault, type Session, type ResourceVersion, type ResourceVersionCreateInput, type ResourceVersionShape, type ResourceVersionUpdateInput } from "@vrooli/shared";
import { Formik, useField } from "formik";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { PageContainer } from "../../../components/Page/Page.js";
import { AutoFillButton } from "../../../components/buttons/AutoFillButton.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { SearchExistingButton } from "../../../components/buttons/SearchExistingButton.js";
import { Dialog } from "../../../components/dialogs/Dialog/Dialog.js";
import { TranslatedAdvancedInput } from "../../../components/inputs/AdvancedInput/AdvancedInput.js";
import { detailsInputFeatures, nameInputFeatures } from "../../../components/inputs/AdvancedInput/styles.js";
import { CodeInput } from "../../../components/inputs/CodeInput/CodeInput.js";
import { LanguageInput } from "../../../components/inputs/LanguageInput/LanguageInput.js";
import { TagSelector } from "../../../components/inputs/TagSelector/TagSelector.js";
import { Switch } from "../../../components/inputs/Switch/Switch.js";
import { VersionInput } from "../../../components/inputs/VersionInput/VersionInput.js";
import { RelationshipList } from "../../../components/lists/RelationshipList/RelationshipList.js";
import { ResourceListInput } from "../../../components/lists/ResourceList/ResourceList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { SessionContext } from "../../../contexts/session.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useStandardUpsertForm } from "../../../hooks/useStandardUpsertForm.js";
import { getAutoFillTranslationData, useAutoFill, type UseAutoFillProps } from "../../../hooks/tasks.js";
import { useDimensions } from "../../../hooks/useDimensions.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { IconCommon } from "../../../icons/Icons.js";
import { FormContainer, ScrollBox } from "../../../styles.js";
import { getCurrentUser } from "../../../utils/authentication/session.js";
import { combineErrorsWithTranslations, getUserLanguages } from "../../../utils/display/translationTools.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { type DataStructureFormProps, type DataStructureUpsertProps } from "./types.js";

export function transformDataStructureVersionValues(
    values: ResourceVersionShape,
    existing?: ResourceVersionShape,
    isCreate?: boolean,
): ResourceVersionCreateInput | ResourceVersionUpdateInput {
    return isCreate 
        ? values as ResourceVersionCreateInput
        : values as ResourceVersionUpdateInput;
}

export function dataStructureInitialValues(
    session: Session | undefined,
    existing?: Partial<ResourceVersion> | null | undefined,
): ResourceVersionShape {
    return {
        __typename: "ResourceVersion" as const,
        id: DUMMY_ID,
        codeLanguage: existing?.codeLanguage || CodeLanguage.Json,
        resourceSubType: ResourceSubType.StandardDataStructure,
        config: {
            __version: "1.0",
            resources: [],
            schema: existing?.config?.schema || JSON.stringify({
                default: {},
                yup: {},
            }),
            schemaLanguage: existing?.config?.schemaLanguage || CodeLanguage.Json,
            props: existing?.config?.props || {},
            isFile: existing?.config?.isFile || false,
            ...existing?.config,
        },
        isComplete: false,
        isPrivate: false,
        versionLabel: "1.0.0",
        ...existing,
        root: {
            __typename: "Resource" as const,
            id: DUMMY_ID,
            isPrivate: false,
            owner: { __typename: "User", id: getCurrentUser(session)?.id ?? "" },
            tags: [],
            ...existing?.root,
        },
        translations: orDefault(existing?.translations, [{
            __typename: "ResourceVersionTranslation" as const,
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            description: "",
            name: "",
        }]),
    };
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
const resourceListStyle = { list: { marginBottom: 2 } } as const;
const exampleButtonStyle = { marginLeft: "auto" } as const;
const offIconInfo = { name: "Build", type: "Common" } as const;
const onIconInfo = { name: "Visible", type: "Common" } as const;
const formSectionTitleStyle = { marginBottom: 1 } as const;
const tagSelectorStyle = { marginBottom: 2 } as const;
const dividerStyle = { display: { xs: "flex", lg: "none" } } as const;

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
    const { t } = useTranslation();
    const theme = useTheme();
    const { dimensions, ref } = useDimensions<HTMLDivElement>();
    const isStacked = dimensions.width < theme.breakpoints.values.lg;

    // Use the standardized form hook
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
        ...dataStructureFormConfig,
        rootObjectType: "Resource",
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
        onCancel: props.onCancel,
        onCompleted: props.onCompleted,
        onDeleted: props.onDeleted,
        onAction: props.onAction,
        onClose,
    });

    const resourceListParent = useMemo(function resourceListParentMemo() {
        return { __typename: "ResourceVersion", id: values.id } as const;
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
        task: isCreate ? LlmTask.StandardAdd : LlmTask.StandardUpdate,
    });

    // Combine our hook's loading state with autofill loading
    const combinedIsLoading = useMemo(() => isAutoFillLoading || isLoading, [isAutoFillLoading, isLoading]);

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
                        title={t(isCreate ? "CreateDataStructure" : "UpdateDataStructure")}
                    />
                    <SearchExistingButton
                        href={`${LINKS.Search}?type="${SearchPageTabOption.DataStructure}"`}
                        text="Search existing standards"
                    />
                    <BaseForm
                        display={display}
                        isLoading={combinedIsLoading}
                        maxWidth={1200}
                        ref={ref}
                    >
                        <FormContainer>
                            <Grid container spacing={2}>
                                <Grid item xs={12} lg={isStacked ? 12 : 6}>
                                    <Box width="100%" padding={2}>
                                        <Typography variant="h4" sx={formSectionTitleStyle}>Basic info</Typography>
                                        <Box display="flex" flexDirection="column" gap={4}>
                                            <RelationshipList
                                                isEditing={true}
                                                objectType={"Resource"}
                                            />
                                            <ResourceListInput
                                                horizontal
                                                isCreate={true}
                                                parent={resourceListParent}
                                                sxs={resourceListStyle}
                                            />
                                            <TranslatedAdvancedInput
                                                features={nameInputFeatures}
                                                isRequired={true}
                                                language={language}
                                                name="name"
                                                title={t("Name")}
                                                placeholder={"User Profile Schema..."}
                                            />
                                            <TranslatedAdvancedInput
                                                features={detailsInputFeatures}
                                                isRequired={false}
                                                language={language}
                                                name="description"
                                                title={t("Description")}
                                                placeholder={"Defines the structure for user data including name, email, and preferences..."}
                                            />
                                            <LanguageInput
                                                currentLanguage={language}
                                                flexDirection="row-reverse"
                                                handleAdd={handleAddLanguage}
                                                handleDelete={handleDeleteLanguage}
                                                handleCurrent={setLanguage}
                                                languages={languages}
                                            />
                                            <TagSelector name="root.tags" sx={tagSelectorStyle} />
                                            <VersionInput
                                                fullWidth
                                                versions={versions}
                                            />
                                        </Box>
                                    </Box>
                                </Grid>
                                <Grid item xs={12} lg={isStacked ? 12 : 6}>
                                    <Divider sx={dividerStyle} />
                                    <Box width="100%" padding={2}>
                                        <Box display="flex" alignItems="center" sx={formSectionTitleStyle}>
                                            <Typography variant="h4">Standard</Typography>
                                            <Switch
                                                checked={isPreviewOn}
                                                onChange={(checked, event) => onPreviewChange(event)}
                                                offIcon={offIconInfo}
                                                onIcon={onIconInfo}
                                                tooltip={isPreviewOn ? "Switch to edit" : "Switch to preview"}
                                                size="md"
                                                labelPosition="none"
                                            />
                                            <Button
                                                variant="outlined"
                                                onClick={showExample}
                                                startIcon={<IconCommon name="Help" />}
                                                sx={exampleButtonStyle}
                                            >
                                                Show example
                                            </Button>
                                        </Box>
                                        {/* TODO replace with FormInputStandard */}
                                        <CodeInput
                                            disabled={false}
                                            codeLanguageField="config.schemaLanguage"
                                            // format={isPreviewOn ? "TODO" : undefined}
                                            // limitTo={isPreviewOn ? [CodeLanguage.JsonStandard] : [CodeLanguage.Json]}
                                            limitTo={codeLimitTo}
                                            name="config.schema"
                                        />
                                    </Box>
                                </Grid>
                            </Grid>
                        </FormContainer>
                    </BaseForm>
                    <BottomActionsButtons
                        display={display}
                        errors={combineErrorsWithTranslations(props.errors, translationErrors)}
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
                        title={t(isCreate ? "CreateDataStructure" : "UpdateDataStructure")}
                    />
                    <SearchExistingButton
                        href={`${LINKS.Search}?type="${SearchPageTabOption.DataStructure}"`}
                        text="Search existing standards"
                    />
                    <BaseForm
                        display={display}
                        isLoading={combinedIsLoading}
                        maxWidth={1200}
                        ref={ref}
                    >
                        <FormContainer>
                            <Grid container spacing={2}>
                                <Grid item xs={12} lg={isStacked ? 12 : 6}>
                                    <Box width="100%" padding={2}>
                                        <Typography variant="h4" sx={formSectionTitleStyle}>Basic info</Typography>
                                        <Box display="flex" flexDirection="column" gap={4}>
                                            <RelationshipList
                                                isEditing={true}
                                                objectType={"Resource"}
                                            />
                                            <ResourceListInput
                                                horizontal
                                                isCreate={true}
                                                parent={resourceListParent}
                                                sxs={resourceListStyle}
                                            />
                                            <TranslatedAdvancedInput
                                                features={nameInputFeatures}
                                                isRequired={true}
                                                language={language}
                                                name="name"
                                                title={t("Name")}
                                                placeholder={"User Profile Schema..."}
                                            />
                                            <TranslatedAdvancedInput
                                                features={detailsInputFeatures}
                                                isRequired={false}
                                                language={language}
                                                name="description"
                                                title={t("Description")}
                                                placeholder={"Defines the structure for user data including name, email, and preferences..."}
                                            />
                                            <LanguageInput
                                                currentLanguage={language}
                                                flexDirection="row-reverse"
                                                handleAdd={handleAddLanguage}
                                                handleDelete={handleDeleteLanguage}
                                                handleCurrent={setLanguage}
                                                languages={languages}
                                            />
                                            <TagSelector name="root.tags" sx={tagSelectorStyle} />
                                            <VersionInput
                                                fullWidth
                                                versions={versions}
                                            />
                                        </Box>
                                    </Box>
                                </Grid>
                                <Grid item xs={12} lg={isStacked ? 12 : 6}>
                                    <Divider sx={dividerStyle} />
                                    <Box width="100%" padding={2}>
                                        <Box display="flex" alignItems="center" sx={formSectionTitleStyle}>
                                            <Typography variant="h4">Standard</Typography>
                                            <Switch
                                                checked={isPreviewOn}
                                                onChange={(checked, event) => onPreviewChange(event)}
                                                offIcon={offIconInfo}
                                                onIcon={onIconInfo}
                                                tooltip={isPreviewOn ? "Switch to edit" : "Switch to preview"}
                                                size="md"
                                                labelPosition="none"
                                            />
                                            <Button
                                                variant="outlined"
                                                onClick={showExample}
                                                startIcon={<IconCommon name="Help" />}
                                                sx={exampleButtonStyle}
                                            >
                                                Show example
                                            </Button>
                                        </Box>
                                        {/* TODO replace with FormInputStandard */}
                                        <CodeInput
                                            disabled={false}
                                            codeLanguageField="config.schemaLanguage"
                                            // format={isPreviewOn ? "TODO" : undefined}
                                            // limitTo={isPreviewOn ? [CodeLanguage.JsonStandard] : [CodeLanguage.Json]}
                                            limitTo={codeLimitTo}
                                            name="config.schema"
                                        />
                                    </Box>
                                </Grid>
                            </Grid>
                        </FormContainer>
                    </BaseForm>
                    <BottomActionsButtons
                        display={display}
                        errors={combineErrorsWithTranslations(props.errors, translationErrors)}
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

export function DataStructureUpsert({
    display,
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: DataStructureUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<ResourceVersion, ResourceVersionShape>({
        ...endpointsResource.findOne,
        disabled: display === "Dialog" && isOpen !== true,
        isCreate,
        objectType: "ResourceVersion",
        overrideObject,
        transform: (existing) => dataStructureInitialValues(session, existing),
    });

    async function validateValues(values: ResourceVersionShape) {
        return await validateFormValues(values, existing, isCreate, dataStructureFormConfig.transformFunction, dataStructureFormConfig.validation);
    }

    return (
        <PageContainer size="fullSize">
            <ScrollBox>
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
            </ScrollBox>
        </PageContainer>
    );
}
