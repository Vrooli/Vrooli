import Button from "@mui/material/Button";
import Divider from "@mui/material/Divider";
import Grid from "@mui/material/Grid";
import { useTheme } from "@mui/material/styles";
import { CodeLanguage, DUMMY_ID, LINKS, LlmTask, SearchPageTabOption, ResourceSubType, endpointsResource, noopSubmit, orDefault, shapeResourceVersion, resourceVersionTranslationValidation, resourceVersionValidation, type FormSchema, type Session, type ResourceVersion, type ResourceVersionCreateInput, type ResourceVersionShape, type ResourceVersionUpdateInput } from "@vrooli/shared";
import { Formik, useField } from "formik";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSubmitHelper } from "../../../api/fetchWrapper.js";
import { AutoFillButton } from "../../../components/buttons/AutoFillButton.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { SearchExistingButton } from "../../../components/buttons/SearchExistingButton.js";
import { ContentCollapse } from "../../../components/containers/ContentCollapse.js";
import { MaybeLargeDialog } from "../../../components/dialogs/LargeDialog/LargeDialog.js";
import { CodeInputBase } from "../../../components/inputs/CodeInput/CodeInput.js";
import { LanguageInput } from "../../../components/inputs/LanguageInput/LanguageInput.js";
import { TranslatedRichInput } from "../../../components/inputs/RichInput/RichInput.js";
import { TagSelector } from "../../../components/inputs/TagSelector/TagSelector.js";
import { TranslatedTextInput } from "../../../components/inputs/TextInput/TextInput.js";
import { VersionInput } from "../../../components/inputs/VersionInput/VersionInput.js";
import { RelationshipList } from "../../../components/lists/RelationshipList/RelationshipList.js";
import { ResourceListInput } from "../../../components/lists/ResourceList/ResourceList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { SessionContext } from "../../../contexts/session.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { FormView } from "../../../forms/FormView/FormView.js";
import { useSaveToCache, useUpsertActions } from "../../../hooks/forms.js";
import { getAutoFillTranslationData, useAutoFill, type UseAutoFillProps } from "../../../hooks/tasks.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useTranslatedFields } from "../../../hooks/useTranslatedFields.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { IconCommon } from "../../../icons/Icons.js";
import { FormContainer, FormSection } from "../../../styles.js";
import { getCurrentUser } from "../../../utils/authentication/session.js";
import { combineErrorsWithTranslations, getUserLanguages } from "../../../utils/display/translationTools.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { useDimensions } from "../../../hooks/useDimensions.js";
import { type PromptFormProps, type PromptUpsertProps } from "./types.js";

// Default schema for prompt inputs
function defaultSchemaInput(): FormSchema {
    return {
        elements: [],
        layout: "column",
    };
}

// Parse schema from props.inputs into FormSchema
function parseSchema(
    inputs: any,
    defaultFn: () => FormSchema,
    logger: Console,
    label: string,
): FormSchema {
    try {
        if (!inputs || !Array.isArray(inputs)) {
            return defaultFn();
        }
        return {
            elements: inputs,
            layout: "column",
        };
    } catch (error) {
        logger.error(`Error parsing ${label}:`, error);
        return defaultFn();
    }
}

export function promptInitialValues(
    session: Session | undefined,
    existing?: Partial<ResourceVersion> | null | undefined,
): ResourceVersionShape {
    return {
        __typename: "ResourceVersion" as const,
        id: DUMMY_ID,
        codeLanguage: existing?.codeLanguage || CodeLanguage.Javascript,
        resourceSubType: ResourceSubType.StandardPrompt,
        config: {
            __version: "1.0",
            resources: [],
            schema: existing?.config?.schema || JSON.stringify({
                default: {},
                yup: {},
            }),
            schemaLanguage: existing?.config?.schemaLanguage || CodeLanguage.Javascript,
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

function transformPromptVersionValues(values: ResourceVersionShape, existing: ResourceVersionShape, isCreate: boolean) {
    return isCreate ? shapeResourceVersion.create(values) : shapeResourceVersion.update(existing, values);
}

//TODO
const exampleStandardJavaScript = [`
{
  "inputs": [
    {
      "label": "Question",
      "type": "text",
      "placeholder": "Type your question here..."
    },
    {
      "label": "Context",
      "type": "textarea",
      "placeholder": "Provide any background information..."
    },
    {
      "label": "Urgency",
      "type": "select",
      "options": ["Low", "Medium", "High"],
      "default": "Medium"
    }
  ],
  "output": {
    "type": "text",
    "combinePattern": "{Question} Context: {Context}. Urgency: {Urgency}."
  }
}
`.trim(), `
/**
 * Generates a simple concatenated prompt from various inputs.
 * This function takes any number of inputs and combines their values into a single string.
 * Each input is expected to be an object with at least two properties: 'label' and 'value'.
 *
 * Example of inputs:
 * [
 *   { label: "Name", value: "Alice" },
 *   { label: "Age", value: 25 }
 * ]
 *
 * @param {...Object} inputs - An array of objects, where each object represents an input field.
 * @returns {string} A concatenated string of all input values.
 */
function generatePrompt(...inputs) {
  // Initialize an empty string to hold the final concatenated prompt.
  let prompt = "";

  // Loop through each input object.
  inputs.forEach(input => {
    // Check if the input value is not empty.
    if (input.value.trim() !== "") {
      // Append the input value to the prompt string.
      // If the prompt is not empty, add a new line before appending new content.
      prompt += prompt ? "\\n" : "";
    }
  });

  // Return the final concatenated prompt.
  return prompt;
}
`.trim()];

const codeLimitTo = [CodeLanguage.Javascript] as const;
const relationshipListStyle = { marginBottom: 2 } as const;
const formSectionStyle = { overflowX: "hidden", marginBottom: 2 } as const;
const resourceListStyle = { list: { marginBottom: 2 } } as const;
const exampleButtonStyle = { marginLeft: "auto" } as const;
const dividerStyle = { marginTop: 4, marginBottom: 2 } as const;
const codeCollapseStyle = { titleContainer: { marginBottom: 1 } } as const;

function PromptForm({
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
}: PromptFormProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const theme = useTheme();
    const { dimensions, ref } = useDimensions<HTMLDivElement>();
    const isStacked = dimensions.width < theme.breakpoints.values.lg;

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
        validationSchema: resourceVersionTranslationValidation.create({ env: process.env.NODE_ENV }),
    });

    const resourceListParent = useMemo(function resourceListParentMemo() {
        return { __typename: "ResourceVersion", id: values.id } as const;
    }, [values]);

    const { handleCancel, handleCompleted } = useUpsertActions<ResourceVersion>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "ResourceVersion",
        rootObjectId: values.root?.id,
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<ResourceVersion, ResourceVersionCreateInput, ResourceVersionUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointsResource.createOne,
        endpointUpdate: endpointsResource.updateOne,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "ResourceVersion" });

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

    const isLoading = useMemo(() => isAutoFillLoading || isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isAutoFillLoading, isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<ResourceVersionCreateInput | ResourceVersionUpdateInput, ResourceVersion>({
        disabled,
        existing,
        fetch,
        inputs: transformPromptVersionValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });


    const [codeLanguageField, , codeLanguageHelpers] = useField<StandardVersion["codeLanguage"]>("codeLanguage");
    const [propsField, , propsHelpers] = useField<StandardVersion["props"]>("props");

    const [outputFunction, setOutputFunction] = useState<string>();
    useEffect(function setOutputFunctionEffect() {
        try {
            const props = JSON.parse(propsField.value);
            setOutputFunction(typeof props.output === "string" ? props.output : "");
        } catch (error) {
            console.error("Error setting output function", error);
        }
    }, [propsField.value]);

    const schemaInput = useMemo(function schemeInputMemo() {
        try {
            const props = JSON.parse(propsField.value);
            return parseSchema(props.inputs, defaultSchemaInput, console, "Prompt input");
        } catch (error) {
            console.error("Error parsing schema", error);
            return defaultSchemaInput();
        }
    }, [propsField.value]);
    const onSchemaInputChange = useCallback(function onSchemaInputChange(schema: FormSchema) {
        try {
            const props = JSON.parse(propsField.value);
            props.inputs = schema.elements;
            propsHelpers.setValue(JSON.stringify(props));
        } catch (error) {
            console.error("Error setting schema", error);
        }
    }, [propsHelpers, propsField.value]);

    const showExample = useCallback(function showExampleCallback() {
        const exampleProps = {
            input: exampleStandardJavaScript[0],
            output: exampleStandardJavaScript[1],
        };
        propsHelpers.setValue(JSON.stringify(exampleProps));
    }, [propsHelpers]);

    return (
        <MaybeLargeDialog
            display={display}
            id="prompt-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreatePrompt" : "UpdatePrompt")}
            />
            <SearchExistingButton
                href={`${LINKS.Search}?type="${SearchPageTabOption.Prompt}"`}
                text="Search existing prompts"
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={700}
                ref={ref}
            >
                <FormContainer>
                    <Grid container spacing={2}>
                        <Grid item xs={12} lg={isStacked ? 12 : 6}>
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
                        </Grid>
                        <Grid item xs={12} lg={isStacked ? 12 : 6}>
                            <ContentCollapse helpText={"Specify inputs for building the prompt."} title={t("Input", { count: schemaInput.elements.length })} isOpen={!disabled} titleVariant="h4">
                                <FormView
                                    disabled={disabled}
                                    isEditing={true}
                                    onSchemaChange={onSchemaInputChange}
                                    schema={schemaInput}
                                />
                            </ContentCollapse>
                            <Divider sx={dividerStyle} />
                            <ContentCollapse helpText={"Define a function that combines the input values into a single output string.\n\nIf not provided or invalid, the output will be a list of the input values in the order of the form."} title={t("Output", { count: 1 })} isOpen={!disabled} titleVariant="h4" toTheRight={
                                <Button
                                    variant="outlined"
                                    onClick={showExample}
                                    startIcon={<IconCommon name="Help" />}
                                    sx={exampleButtonStyle}
                                >
                                    Show example
                                </Button>
                            } sxs={codeCollapseStyle}>
                                <CodeInputBase
                                    codeLanguage={codeLanguageField.value as CodeLanguage}
                                    content={outputFunction || ""}
                                    defaultValue={undefined}
                                    format={undefined}
                                    handleCodeLanguageChange={codeLanguageHelpers.setValue}
                                    handleContentChange={setOutputFunction}
                                    limitTo={codeLimitTo}
                                    name="output"
                                    variables={undefined}
                                />
                            </ContentCollapse>
                        </Grid>
                    </Grid>
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

export function PromptUpsert({
    display,
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: PromptUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<StandardVersion, StandardVersionShape>({
        ...endpointsStandardVersion.findOne,
        disabled: display === "Dialog" && isOpen !== true,
        isCreate,
        objectType: "StandardVersion",
        overrideObject,
        transform: (existing) => promptInitialValues(session, existing),
    });

    async function validateValues(values: StandardVersionShape) {
        return await validateFormValues(values, existing, isCreate, transformPromptVersionValues, standardVersionValidation);
    }

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <PromptForm
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
