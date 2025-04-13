import { CodeLanguage, DUMMY_ID, endpointsStandardVersion, FormSchema, LINKS, LlmTask, noopSubmit, orDefault, SearchPageTabOption, Session, shapeStandardVersion, StandardType, StandardVersion, StandardVersionCreateInput, StandardVersionShape, standardVersionTranslationValidation, StandardVersionUpdateInput, standardVersionValidation } from "@local/shared";
import { Button, Divider } from "@mui/material";
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
import { getAutoFillTranslationData, useAutoFill, UseAutoFillProps } from "../../../hooks/tasks.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useTranslatedFields } from "../../../hooks/useTranslatedFields.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { IconCommon } from "../../../icons/Icons.js";
import { FormContainer, FormSection } from "../../../styles.js";
import { getCurrentUser } from "../../../utils/authentication/session.js";
import { combineErrorsWithTranslations, getUserLanguages } from "../../../utils/display/translationTools.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { PromptFormProps, PromptUpsertProps } from "./types.js";

export function promptInitialValues(
    session: Session | undefined,
    existing?: Partial<StandardVersion> | null | undefined,
): StandardVersionShape {
    return {
        __typename: "StandardVersion" as const,
        id: DUMMY_ID,
        codeLanguage: CodeLanguage.Javascript,
        default: JSON.stringify({}),
        directoryListings: [],
        isComplete: false,
        isPrivate: false,
        isFile: false,
        props: JSON.stringify({}),
        variant: StandardType.Prompt,
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

function transformPromptVersionValues(values: StandardVersionShape, existing: StandardVersionShape, isCreate: boolean) {
    return isCreate ? shapeStandardVersion.create(values) : shapeStandardVersion.update(existing, values);
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
            id="standard-upsert-dialog"
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
                        helpText={"Specify inputs for building the prompt."}
                        title={t("Input", { count: schemaInput.elements.length })}
                        isOpen={!disabled}
                        titleVariant="h4"
                    >
                        <FormView
                            disabled={disabled}
                            isEditing={true}
                            onSchemaChange={onSchemaInputChange}
                            schema={schemaInput}
                        />
                    </ContentCollapse>
                    <Divider sx={dividerStyle} />
                    <ContentCollapse
                        helpText={"Define a function that combines the input values into a single output string.\n\nIf not provided or invalid, the output will be a list of the input values in the order of the form."}
                        title={t("Output", { count: 1 })}
                        isOpen={!disabled}
                        titleVariant="h4"
                        toTheRight={
                            <Button
                                variant="outlined"
                                onClick={showExample}
                                startIcon={<IconCommon name="Help" />}
                                sx={exampleButtonStyle}
                            >
                                Show example
                            </Button>
                        }
                        sxs={codeCollapseStyle}
                    >
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
        disabled: display === "dialog" && isOpen !== true,
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
