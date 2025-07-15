import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Divider from "@mui/material/Divider";
import Grid from "@mui/material/Grid";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material/styles";
import { CodeLanguage, DUMMY_ID, LINKS, LlmTask, SearchPageTabOption, dataConverterFormConfig, endpointsResource, noopSubmit, orDefault, ResourceSubType, type ResourceShape, type ResourceVersion, type ResourceVersionCreateInput, type ResourceVersionShape, type ResourceVersionUpdateInput, type Session } from "@vrooli/shared";
import { Formik, useField } from "formik";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { PageContainer } from "../../../components/Page/Page.js";
import { AutoFillButton } from "../../../components/buttons/AutoFillButton.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { SearchExistingButton } from "../../../components/buttons/SearchExistingButton.js";
import { Dialog } from "../../../components/dialogs/Dialog/Dialog.js";
import { useIsMobile } from "../../../hooks/useIsMobile.js";
import { TranslatedAdvancedInput } from "../../../components/inputs/AdvancedInput/AdvancedInput.js";
import { detailsInputFeatures, nameInputFeatures } from "../../../components/inputs/AdvancedInput/styles.js";
import { CodeInput } from "../../../components/inputs/CodeInput/CodeInput.js";
import { LanguageInput } from "../../../components/inputs/LanguageInput/LanguageInput.js";
import { TagSelector } from "../../../components/inputs/TagSelector/TagSelector.js";
import { VersionInput } from "../../../components/inputs/VersionInput/VersionInput.js";
import { RelationshipList } from "../../../components/lists/RelationshipList/RelationshipList.js";
import { ResourceListInput } from "../../../components/lists/ResourceList/ResourceList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { SessionContext } from "../../../contexts/session.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { createUpdatedTranslations, getAutoFillTranslationData, useAutoFill, type UseAutoFillProps } from "../../../hooks/tasks.js";
import { useDimensions } from "../../../hooks/useDimensions.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useStandardUpsertForm } from "../../../hooks/useStandardUpsertForm.js";
import { IconCommon } from "../../../icons/Icons.js";
import { FormContainer, ScrollBox } from "../../../styles.js";
import { getCurrentUser } from "../../../utils/authentication/session.js";
import { combineErrorsWithTranslations, getUserLanguages } from "../../../utils/display/translationTools.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { type DataConverterFormProps, type DataConverterUpsertProps } from "./types.js";

export function dataConverterInitialValues(
    session: Session | undefined,
    existing?: Partial<ResourceVersion> | undefined,
): ResourceVersionShape {
    return {
        __typename: "ResourceVersion" as const,
        id: DUMMY_ID,
        codeLanguage: existing?.codeLanguage || CodeLanguage.Haskell,
        resourceSubType: ResourceSubType.CodeDataConverter,
        config: {
            __version: "1.0",
            resources: [],
            content: existing?.config?.content || "",
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


/** Code to display when an example is requested */
const exampleCode = `
/**
 * Converts a comma-separated string of numbers into an array of integers.
 * 
 * This function attempts to parse each element of the input string, 
 * which should be numbers separated by commas, into integers. If any 
 * part of the string cannot be converted to an integer, an error object 
 * is returned instead.
 *
 * @param {string} input - A string of numbers separated by commas.
 * @returns {number[]|Object} - Returns an array of integers if successful, 
 *                              or an object with an error message if the input is invalid.
 * 
 * @example
 * parseList("1, 2, 3, 4"); // returns [1, 2, 3, 4]
 * parseList("1, two, 3");  // returns { error: "Invalid input format" }
 */
function parseList(input) {
    try {
        // Parse the input string into an array of strings split by commas
        const stringArray = input.split(',');

        // Map each string element to an integer
        const numberArray = stringArray.map(element => parseInt(element.trim(), 10));

        // Return the array of numbers
        return numberArray;
    } catch (error) {
        // Return an error message if the input is not as expected
        return { error: "Invalid input format" };
    }
}
`.trim();

const codeLimitTo = [CodeLanguage.Javascript] as const;
const resourceListStyle = { list: { marginBottom: 2 } } as const;
const exampleButtonStyle = { marginLeft: "auto" } as const;
const formSectionTitleStyle = { marginBottom: 1 } as const;
const tagSelectorStyle = { marginBottom: 2 } as const;
const dividerStyle = { display: { xs: "flex", lg: "none" } } as const;

function DataConverterForm({
    disabled,
    display,
    existing,
    handleUpdate,
    isCreate,
    isMutate,
    isOpen,
    isReadLoading,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    versions,
    ...props
}: DataConverterFormProps) {
    const { t } = useTranslation();
    const theme = useTheme();
    const { dimensions, ref } = useDimensions<HTMLDivElement>();
    const isStacked = dimensions.width < theme.breakpoints.values.lg;
    const isMobile = useIsMobile();

    const resourceListParent = useMemo(function resourceListParentMemo() {
        return { __typename: "ResourceVersion", id: values.id } as const;
    }, [values]);

    // Use the standardized form hook
    const {
        session,
        isLoading,
        handleCancel,
        handleCompleted,
        onSubmit,
        language,
        languages,
        handleAddLanguage,
        handleDeleteLanguage,
        setLanguage,
        translationErrors,
    } = useStandardUpsertForm({
        ...dataConverterFormConfig,
        rootObjectType: "Resource",
    }, {
        values,
        existing,
        isCreate,
        display,
        disabled,
        isMutate,
        isReadLoading,
        isSubmitting: props.isSubmitting,
        handleUpdate,
        setSubmitting: props.setSubmitting,
        onCancel,
        onCompleted,
        onDeleted,
        onClose,
    });

    const getAutoFillInput = useCallback(function getAutoFillInput() {
        return {
            ...getAutoFillTranslationData(values, language),
            content: values.config?.content,
            isPrivate: values.isPrivate,
            version: values.versionLabel,
        };
    }, [language, values]);

    const shapeAutoFillResult = useCallback(function shapeAutoFillResultCallback(data: Parameters<UseAutoFillProps["shapeAutoFillResult"]>[0]) {
        const originalValues = { ...values };
        const { updatedTranslations, rest } = createUpdatedTranslations(values, data, language, ["name", "description"]);
        delete rest.id;
        const content = typeof rest.content === "string" ? rest.content : values.config?.content;
        const isPrivate = typeof rest.isPrivate === "boolean" ? rest.isPrivate : values.isPrivate;
        const versionLabel = typeof rest.version === "string" ? rest.version : values.versionLabel;
        const updatedValues = {
            ...values,
            config: {
                ...values.config,
                content,
            },
            isPrivate,
            translations: updatedTranslations,
            versionLabel,
        };
        return { originalValues, updatedValues };
    }, [language, values]);

    const { autoFill, isAutoFillLoading } = useAutoFill({
        getAutoFillInput,
        shapeAutoFillResult,
        handleUpdate,
        task: isCreate ? LlmTask.DataConverterAdd : LlmTask.DataConverterUpdate,
    });

    const finalIsLoading = useMemo(() => isAutoFillLoading || isLoading, [isAutoFillLoading, isLoading]);

    const [, , codeLanguageHelpers] = useField<CodeLanguage>("codeLanguage");
    const [, , contentHelpers] = useField<string>("config.content");
    function showExample() {
        // We only have an example for JavaScript, so switch to that
        codeLanguageHelpers.setValue(CodeLanguage.Javascript);
        // Set value to hard-coded example
        contentHelpers.setValue(exampleCode);
    }

    if (display === "Page" || isMobile) {
        return (
            <Dialog
                isOpen={isOpen}
                onClose={onClose}
                size="full"
            >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateDataConverter" : "UpdateDataConverter")}
            />
            <SearchExistingButton
                href={`${LINKS.Search}?type="${SearchPageTabOption.DataConverter}"`}
                text="Search existing codes"
            />
            <BaseForm
                display={display}
                isLoading={finalIsLoading}
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
                                        placeholder={"CSV to JSON Converter..."}
                                    />
                                    <TranslatedAdvancedInput
                                        features={detailsInputFeatures}
                                        isRequired={false}
                                        language={language}
                                        name="description"
                                        title={t("Description")}
                                        placeholder={"Converts comma-separated values into a structured JSON format..."}
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
                                    <Typography variant="h4">Code</Typography>
                                    <Button
                                        variant="outlined"
                                        onClick={showExample}
                                        startIcon={<IconCommon name="Help" />}
                                        sx={exampleButtonStyle}
                                    >
                                        Show example
                                    </Button>
                                </Box>
                                <CodeInput
                                    disabled={false}
                                    limitTo={codeLimitTo}
                                    name="config.content"
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
                loading={finalIsLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
                sideActionButtons={<AutoFillButton
                    handleAutoFill={autoFill}
                    isAutoFillLoading={isAutoFillLoading}
                />}
            />
            </Dialog>
        );
    }
    return (
        <Dialog
            isOpen={isOpen}
            onClose={onClose}
            size="lg"
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateDataConverter" : "UpdateDataConverter")}
            />
            <SearchExistingButton
                href={`${LINKS.Search}?type="${SearchPageTabOption.DataConverter}"`}
                text="Search existing codes"
            />
            <BaseForm
                display={display}
                isLoading={finalIsLoading}
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
                                        placeholder={"CSV to JSON Converter..."}
                                    />
                                    <TranslatedAdvancedInput
                                        features={detailsInputFeatures}
                                        isRequired={false}
                                        language={language}
                                        name="description"
                                        title={t("Description")}
                                        placeholder={"Converts comma-separated values into a structured JSON format..."}
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
                                    <Typography variant="h4">Code</Typography>
                                    <Button
                                        variant="outlined"
                                        onClick={showExample}
                                        startIcon={<IconCommon name="Help" />}
                                        sx={exampleButtonStyle}
                                    >
                                        Show example
                                    </Button>
                                </Box>
                                <CodeInput
                                    disabled={false}
                                    limitTo={codeLimitTo}
                                    name="config.content"
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
                loading={finalIsLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
                sideActionButtons={<AutoFillButton
                    handleAutoFill={autoFill}
                    isAutoFillLoading={isAutoFillLoading}
                />}
            />
        </Dialog>
    );
}

export function DataConverterUpsert({
    display,
    isCreate,
    isMutate,
    isOpen,
    overrideObject,
    ...props
}: DataConverterUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<ResourceVersion, ResourceVersionShape>({
        ...endpointsResource.findOne,
        disabled: display === "Dialog" && isOpen !== true,
        isCreate,
        objectType: "ResourceVersion",
        overrideObject,
        transform: (existing) => dataConverterInitialValues(session, existing),
    });

    async function validateValues(values: ResourceVersionShape) {
        return await validateFormValues(values, existing, isCreate, dataConverterFormConfig.transformFunction, dataConverterFormConfig.validation);
    }

    const versions = useMemo(function versionsMemo() {
        return (existing?.root as ResourceShape)?.versions?.map(v => v.versionLabel) ?? [];
    }, [existing]);

    return (
        <PageContainer size="fullSize">
            <ScrollBox>
                <Formik
                    enableReinitialize={true}
                    initialValues={existing}
                    onSubmit={noopSubmit}
                    validate={validateValues}
                >
                    {(formik) => <DataConverterForm
                        disabled={!(isCreate || permissions.canUpdate)}
                        display={display}
                        existing={existing}
                        handleUpdate={setExisting}
                        isCreate={isCreate}
                        isMutate={isMutate}
                        isReadLoading={isReadLoading}
                        isOpen={isOpen}
                        versions={versions}
                        {...props}
                        {...formik}
                    />}
                </Formik>
            </ScrollBox>
        </PageContainer>
    );
}
