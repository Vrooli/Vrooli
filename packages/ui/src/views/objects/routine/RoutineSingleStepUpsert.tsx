import { CallDataActionConfigObject, CallDataApiConfigObject, CallDataCodeConfigObject, CallDataGenerateConfigObject, CallDataSmartContractConfigObject, DUMMY_ID, FormInputBase, FormInputConfigObject, FormOutputConfigObject, FormSchema, GraphConfigObject, LINKS, LlmTask, RoutineShape, RoutineType, RoutineVersion, RoutineVersionConfig, RoutineVersionCreateInput, RoutineVersionInputShape, RoutineVersionOutputShape, RoutineVersionShape, RoutineVersionUpdateInput, SearchPageTabOption, Session, endpointsRoutineVersion, noop, noopSubmit, orDefault, routineVersionTranslationValidation, routineVersionValidation, shapeRoutineVersion, stringifyObject, uuid, uuidValidate } from "@local/shared";
import { Checkbox, Divider, FormControlLabel, Grid, Tooltip } from "@mui/material";
import { useSubmitHelper } from "api/fetchWrapper.js";
import { AutoFillButton } from "components/buttons/AutoFillButton.js";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons.js";
import { SearchExistingButton } from "components/buttons/SearchExistingButton/SearchExistingButton.js";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse.js";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog.js";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput.js";
import { TranslatedRichInput } from "components/inputs/RichInput/RichInput.js";
import { SelectorBase } from "components/inputs/Selector/Selector.js";
import { TagSelector } from "components/inputs/TagSelector/TagSelector.js";
import { TranslatedTextInput } from "components/inputs/TextInput/TextInput.js";
import { VersionInput } from "components/inputs/VersionInput/VersionInput.js";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList.js";
import { ResourceListInput } from "components/lists/ResourceList/ResourceList.js";
import { TopBar } from "components/navigation/TopBar.js";
import { FieldHelperProps, Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "hooks/forms.js";
import { UseAutoFillProps, getAutoFillTranslationData, useAutoFill } from "hooks/tasks.js";
import { useManagedObject } from "hooks/useManagedObject.js";
import { useTranslatedFields } from "hooks/useTranslatedFields.js";
import { useUpsertFetch } from "hooks/useUpsertFetch.js";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session.js";
import { ELEMENT_IDS } from "utils/consts.js";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools.js";
import { PubSub } from "utils/pubsub.js";
import { getRoutineTypeDescription, getRoutineTypeIcon, getRoutineTypeLabel, routineTypes } from "utils/search/schemas/routine.js";
import { validateFormValues } from "utils/validateFormValues.js";
import { SessionContext } from "../../../contexts.js";
import { FormContainer, FormSection } from "../../../styles.js";
import { RoutineApiForm, RoutineDataConverterForm, RoutineDataForm, RoutineFormPropsBase, RoutineGenerateForm, RoutineInformationalForm, RoutineSmartContractForm } from "./RoutineTypeForms.js";
import { RoutineSingleStepFormProps, RoutineSingleStepUpsertProps } from "./types.js";

export function routineSingleStepInitialValues(
    session: Session | undefined,
    existing?: Partial<RoutineVersion> | null | undefined,
): RoutineVersionShape {
    return {
        __typename: "RoutineVersion" as const,
        id: uuid(), // Cannot be a dummy ID because nodes, links, etc. reference this ID
        inputs: [],
        isComplete: false,
        isPrivate: false,
        directoryListings: [],
        outputs: [],
        routineType: RoutineType.Informational,
        versionLabel: "1.0.0",
        ...existing,
        root: {
            __typename: "Routine" as const,
            id: DUMMY_ID,
            isPrivate: false,
            owner: { __typename: "User", id: getCurrentUser(session)?.id ?? "" },
            parent: null,
            permissions: JSON.stringify({}),
            tags: [],
            ...existing?.root,
        },
        resourceList: orDefault<RoutineVersionShape["resourceList"]>(existing?.resourceList, {
            __typename: "ResourceList" as const,
            id: DUMMY_ID,
            listFor: {
                __typename: "RoutineVersion" as const,
                id: DUMMY_ID,
            },
        }),
        translations: orDefault(existing?.translations, [{
            __typename: "RoutineVersionTranslation" as const,
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            description: "",
            instructions: "",
            name: "",
        }]),
    };
}

function transformRoutineVersionValues(values: RoutineVersionShape, existing: RoutineVersionShape, isCreate: boolean) {
    return isCreate ? shapeRoutineVersion.create(values) : shapeRoutineVersion.update(existing, values);
}

type UpdateSchemaElementsBaseProps<Shape> = {
    currentElements: Shape[];
    elementsHelpers: FieldHelperProps<Shape[]>;
    language: string;
    routineVersionId: string;
    schema: FormSchema;
}

type UpdateSchemaElementsInputsProps = UpdateSchemaElementsBaseProps<RoutineVersionInputShape> & {
    type: "inputs";
}

type UpdateSchemaElementsOutputsProps = UpdateSchemaElementsBaseProps<RoutineVersionOutputShape> & {
    type: "outputs";
}

export type UpdateSchemaElementsProps = UpdateSchemaElementsInputsProps | UpdateSchemaElementsOutputsProps;

/**
 * Updates inputs or outputs based on the provided schema.
 * Ensures existing inputs/outputs are updated instead of creating duplicates.
 * Utilizes the fieldName as the unique identifier for inputs/outputs.
 * 
 * @param schema - The form schema containing elements to be parsed as inputs/outputs.
 * @param elementsHelpers - Formik helpers to set inputs or outputs.
 * @param currentElements - Current inputs or outputs state.
 * @param language - The user's language for translation.
 */
export function updateSchemaElements({
    currentElements,
    elementsHelpers,
    language,
    routineVersionId,
    schema,
    type,
}: UpdateSchemaElementsProps) {
    // Filter out headers and other non-input elements
    const elementsInSchema = schema.elements.filter(
        element => Object.prototype.hasOwnProperty.call(element, "fieldName"),
    ) as FormInputBase[];

    // Loop through schema elements and update existing elements or create new ones
    const updatedElements = elementsInSchema.map(element => {
        // Check if element already exists
        const existingElement = currentElements.find(e => e.name === element.fieldName);

        // Build translation if helpText and description are defined and not empty
        const newTranslation = (element.helpText || element.description) ? {
            language,
            helpText: element.helpText || "",
            description: element.description || "",
        } : null;

        // `isRequired` only applies to inputs
        const isRequired = type === "inputs" ? (element.isRequired || false) : undefined;

        if (existingElement) {
            let updatedTranslations = [...existingElement.translations || []];

            if (newTranslation) {
                const translationIndex = updatedTranslations.findIndex(t => t.language === language);
                if (translationIndex >= 0) {
                    // Update existing translation
                    updatedTranslations[translationIndex] = {
                        ...updatedTranslations[translationIndex],
                        ...newTranslation,
                    };
                } else {
                    // Add new translation
                    updatedTranslations.push({
                        __typename: type === "inputs" ? "RoutineVersionInputTranslation" : "RoutineVersionOutputTranslation",
                        id: uuidValidate(element.id) ? element.id : DUMMY_ID,
                        ...newTranslation,
                    });
                }
            } else {
                // Remove translation for this language if newTranslation is not provided
                updatedTranslations = updatedTranslations.filter(t => t.language !== language);
            }

            // Update existing element
            return {
                ...existingElement,
                isRequired,
                translations: updatedTranslations,
            };
        } else {
            // Create new element
            return {
                __typename: type === "inputs" ? "RoutineVersionInput" : "RoutineVersionOutput",
                id: uuidValidate(element.id) ? element.id : DUMMY_ID,
                name: element.fieldName,
                isRequired,
                routineVersion: {
                    __typename: "RoutineVersion",
                    id: routineVersionId,
                },
                translations: newTranslation ? [newTranslation] : [],
                // TODO: Handle standard version later. Makes sense for inputs like code, but not for inputs like text.
            } as const;
        }
    });

    (elementsHelpers as FieldHelperProps<never>).setValue(updatedElements as never);
}

const basicInfoCollapseStyle = { titleContainer: { marginBottom: 1 } } as const;
const formSectionStyle = { overflowX: "hidden", marginBottom: 2 } as const;
const relationshipListStyle = { marginBottom: 2 } as const;
const resourceListStyle = { list: { marginBottom: 2 } } as const;
const tagSelectorStyle = { marginBottom: 2 } as const;
const versionInputStyle = { marginBottom: 2 } as const;

function RoutineSingleStepForm({
    disabled,
    display,
    existing,
    isCreate,
    handleUpdate,
    isOpen,
    isReadLoading,
    isSubroutine,
    onClose,
    values,
    versions,
    ...props
}: RoutineSingleStepFormProps) {
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
        validationSchema: routineVersionTranslationValidation.create({ env: process.env.NODE_ENV }),
    });

    // Formik fields we need to access and/or set values for
    const [idField] = useField<string>("id");
    const [routineTypeField, , routineTypeHelpers] = useField<RoutineVersion["routineType"]>("routineType");
    const [inputsField, , inputsHelpers] = useField<RoutineVersionInputShape[]>("inputs");
    const [outputsField, , outputsHelpers] = useField<RoutineVersionOutputShape[]>("outputs");
    const [configField, , configHelpers] = useField<RoutineVersion["config"]>("config");
    const [apiVersionField, , apiVersionHelpers] = useField<RoutineVersion["apiVersion"]>("apiVersion");
    const [codeVersionField, , codeVersionHelpers] = useField<RoutineVersion["codeVersion"]>("codeVersion");

    const config = useMemo(function configMemo() {
        return RoutineVersionConfig.deserialize({ config: configField.value, routineType: routineTypeField.value }, console);
    }, [configField.value, routineTypeField.value]);

    const onCallDataActionChange = useCallback(function onCallDataActionChange(callDataAction: CallDataActionConfigObject) {
        configHelpers.setValue(stringifyObject({
            ...config.export(),
            callDataAction,
        }, "json"));
    }, [config, configHelpers]);
    const onCallDataApiChange = useCallback(function onCallDataApiChange(callDataApi: CallDataApiConfigObject) {
        configHelpers.setValue(stringifyObject({
            ...config.export(),
            callDataApi,
        }, "json"));
    }, [config, configHelpers]);
    const onCallDataCodeChange = useCallback(function onCallDataCodeChange(callDataCode: CallDataCodeConfigObject) {
        configHelpers.setValue(stringifyObject({
            ...config.export(),
            callDataCode,
        }, "json"));
    }, [config, configHelpers]);
    const onCallDataGenerateChange = useCallback(function onCallDataGenerateChange(callDataGenerate: CallDataGenerateConfigObject) {
        configHelpers.setValue(stringifyObject({
            ...config.export(),
            callDataGenerate,
        }, "json"));
    }, [config, configHelpers]);
    const onCallDataSmartContractChange = useCallback(function onCallDataSmartContractChange(callDataSmartContract: CallDataSmartContractConfigObject) {
        configHelpers.setValue(stringifyObject({
            ...config.export(),
            callDataSmartContract,
        }, "json"));
    }, [config, configHelpers]);
    const onFormInputChange = useCallback(function onFormInputChange(formInput: FormInputConfigObject) {
        configHelpers.setValue(stringifyObject({
            ...config.export(),
            formInput,
        }, "json"));
        updateSchemaElements({
            currentElements: inputsField.value,
            elementsHelpers: inputsHelpers,
            language,
            routineVersionId: idField.value,
            schema: formInput.schema,
            type: "inputs",
        });
    }, [config, configHelpers, idField.value, inputsField.value, inputsHelpers, language]);
    const onFormOutputChange = useCallback(function onFormOutputChange(formOutput: FormOutputConfigObject) {
        configHelpers.setValue(stringifyObject({
            ...config.export(),
            formOutput,
        }, "json"));
        updateSchemaElements({
            currentElements: outputsField.value,
            elementsHelpers: outputsHelpers,
            language,
            routineVersionId: idField.value,
            schema: formOutput.schema,
            type: "outputs",
        });
    }, [config, configHelpers, idField.value, language, outputsField.value, outputsHelpers]);
    const onGraphChange = useCallback(function onGraphChange(graph: GraphConfigObject) {
        configHelpers.setValue(stringifyObject({
            ...config.export(),
            graph,
        }, "json"));
    }, [config, configHelpers]);

    // Handle routine type
    const routineTypeValue = useMemo(function routineTypeValueMemo() {
        return routineTypes.find(r => r.type === routineTypeField.value) ?? routineTypes[0];
    }, [routineTypeField.value]);
    const handleRoutineTypeChange = useCallback(({ type }: { type: RoutineType }) => {
        // If type is the same, do nothing
        if (type === routineTypeField.value) return;
        // Returns true if one of the values is not empty
        function hasData(...values: unknown[]): boolean {
            return values.some(value =>
                (typeof value === "string" && value.length > 0) ||
                (Array.isArray(value) && value.length > 0) ||
                (!Array.isArray(value) && typeof value === "object" && value !== null && Object.keys(value).length > 0),
            );
        }
        // Map to check if the type we're switching FROM has data that will be lost
        const loseDataCheck = {
            // Has call data
            [RoutineType.Action]: hasData(config.callDataAction, config.formInput, config.formOutput),
            // Has API information
            [RoutineType.Api]: hasData(config.callDataApi, config.formInput, config.formOutput, apiVersionField.value),
            // Has code information
            [RoutineType.Code]: hasData(config.callDataCode, config.formInput, config.formOutput, codeVersionField.value),
            // Has an output
            [RoutineType.Data]: hasData(config.formOutput),
            // Has an input or call data
            [RoutineType.Generate]: hasData(config.callDataGenerate, config.formInput, config.formOutput),
            // Has input data
            [RoutineType.Informational]: hasData(config.formInput),
            // Also uses code information
            [RoutineType.SmartContract]: hasData(config.callDataSmartContract, config.formInput, config.formOutput, codeVersionField.value),
        };
        // Helper function to remove all type-specific data on switch
        function performSwitch() {
            apiVersionHelpers.setValue(null);
            codeVersionHelpers.setValue(null);
            inputsHelpers.setValue([]);
            outputsHelpers.setValue([]);
            routineTypeHelpers.setValue(type);
            configHelpers.setValue(RoutineVersionConfig.deserialize({
                config: undefined,
                routineType: type,
            }, console, { useFallbacks: true }).serialize("json"));
        }
        // If we're losing data, confirm with user
        const losingData = loseDataCheck[routineTypeField.value];
        if (losingData) {
            PubSub.get().publish("alertDialog", {
                messageKey: "RoutineTypeSwitchLoseData",
                buttons: [{
                    labelKey: "Yes",
                    onClick: performSwitch,
                }, {
                    labelKey: "Cancel",
                }],
            });
        }
        // Otherwise, just switch
        else {
            performSwitch();
        }
    }, [config, configHelpers, routineTypeField.value, apiVersionField.value, codeVersionField.value, apiVersionHelpers, codeVersionHelpers, inputsHelpers, outputsHelpers, routineTypeHelpers]);

    const { handleCancel, handleCompleted } = useUpsertActions<RoutineVersion>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "RoutineVersion",
        rootObjectId: values.root?.id,
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<RoutineVersion, RoutineVersionCreateInput, RoutineVersionUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointsRoutineVersion.createOne,
        endpointUpdate: endpointsRoutineVersion.updateOne,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "RoutineVersion" });

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
        task: isCreate ? LlmTask.RoutineAdd : LlmTask.RoutineUpdate,
    });

    const isLoading = useMemo(() => isAutoFillLoading || isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isAutoFillLoading, isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<RoutineVersionCreateInput | RoutineVersionUpdateInput, RoutineVersion>({
        disabled,
        existing,
        fetch,
        inputs: transformRoutineVersionValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    // Type-specific components
    const routineTypeComponents = useMemo(function routineTypeComponentsMemo() {
        const routineTypeBaseProps: RoutineFormPropsBase = {
            config,
            disabled,
            display: "edit",
            handleClearRun: noop, // Not available in edit mode
            handleCompleteStep: noop, // Not available in edit mode
            handleRunStep: noop, // Not available in edit mode
            hasErrors: false, // Not available in edit mode
            isCompleteStepDisabled: true, // Not available in edit mode
            isPartOfMultiStepRoutine: false,
            isRunStepDisabled: true, // Not available in edit mode
            isRunningStep: false, // Not available in edit mode
            onCallDataActionChange,
            onCallDataApiChange,
            onCallDataCodeChange,
            onCallDataGenerateChange,
            onCallDataSmartContractChange,
            onFormInputChange,
            onFormOutputChange,
            onGraphChange,
            onRunChange: noop, // Not available in edit mode
            routineId: "", // Not needed in edit mode
            routineName: "", // Not needed in edit mode
            run: null, // Not available in edit mode
        };

        switch (routineTypeField.value) {
            case RoutineType.Action:
                return <RoutineActionForm {...routineTypeBaseProps} />;
            case RoutineType.Api:
                return <RoutineApiForm {...routineTypeBaseProps} />;
            case RoutineType.Code:
                return <RoutineDataConverterForm {...routineTypeBaseProps} />;
            case RoutineType.Data:
                return <RoutineDataForm {...routineTypeBaseProps} />;
            case RoutineType.Generate:
                return <RoutineGenerateForm {...routineTypeBaseProps} />;
            case RoutineType.Informational:
                return <RoutineInformationalForm {...routineTypeBaseProps} />;
            case RoutineType.SmartContract:
                return <RoutineSmartContractForm {...routineTypeBaseProps} />;
            default:
                return null;
        }
    }, [config, disabled, onCallDataActionChange, onCallDataApiChange, onCallDataCodeChange, onCallDataGenerateChange, onCallDataSmartContractChange, onFormInputChange, onFormOutputChange, onGraphChange, routineTypeField.value]);

    const resourceListParent = useMemo(function resourceListParent() {
        return { __typename: "RoutineVersion", id: values.id } as const;
    }, [values.id]);

    return (
        <MaybeLargeDialog
            display={display}
            id={ELEMENT_IDS.RoutineSingleStepUpsertDialog}
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateRoutine" : "UpdateRoutine")}
            />
            <SearchExistingButton
                href={`${LINKS.Search}?type="${SearchPageTabOption.RoutineSingleStep}"`}
                text="Search existing routines"
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
            >
                <FormContainer>
                    <ContentCollapse title="Basic info" titleVariant="h4" isOpen={display === "page"} sxs={basicInfoCollapseStyle}>
                        <RelationshipList
                            isEditing={true}
                            objectType={"Routine"}
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
                                isRequired
                                label={t("Name")}
                                language={language}
                                name="name"
                                placeholder={t("NamePlaceholder")}
                            />
                            <TranslatedRichInput
                                language={language}
                                name="description"
                                maxChars={2048}
                                maxRows={4}
                                minRows={2}
                                placeholder={t("DescriptionPlaceholder")}
                            />
                            <TranslatedRichInput
                                language={language}
                                name="instructions"
                                maxChars={8192}
                                minRows={4}
                                placeholder={t("Instructions")}
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
                        <TagSelector name="root.tags" sx={tagSelectorStyle} />
                        <VersionInput
                            fullWidth
                            versions={versions}
                            sx={versionInputStyle}
                        />
                    </ContentCollapse>
                    <Divider />
                    {/* Is internal checkbox */}
                    {isSubroutine && (
                        <Grid item xs={12}>
                            <Tooltip placement={"top"} title='Indicates if this routine is meant to be a subroutine for only one other routine. If so, it will not appear in search resutls.'>
                                <FormControlLabel
                                    label='Internal'
                                    control={
                                        <Checkbox
                                            id='routine-info-dialog-is-internal'
                                            size="small"
                                            name='isInternal'
                                            color='secondary'
                                        />
                                    }
                                />
                            </Tooltip>
                        </Grid>
                    )}
                    <FormSection>
                        {/* Selector for routine type*/}
                        <SelectorBase
                            name="routineType"
                            options={routineTypes}
                            getOptionDescription={getRoutineTypeDescription}
                            getOptionLabel={getRoutineTypeLabel}
                            getOptionIcon={getRoutineTypeIcon}
                            fullWidth={true}
                            inputAriaLabel="Routine Type"
                            label="Routine Type"
                            onChange={handleRoutineTypeChange}
                            value={routineTypeValue}
                        />
                        {routineTypeComponents}
                    </FormSection>
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

export function RoutineSingleStepUpsert({
    display,
    isCreate,
    isOpen,
    isSubroutine = false,
    overrideObject,
    ...props
}: RoutineSingleStepUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<RoutineVersion, RoutineVersionShape>({
        ...endpointsRoutineVersion.findOne,
        disabled: display === "dialog" && isOpen !== true,
        isCreate,
        objectType: "RoutineVersion",
        overrideObject,
        transform: (existing) => routineSingleStepInitialValues(session, existing),
    });

    async function validateValues(values: RoutineVersionShape) {
        return await validateFormValues(values, existing, isCreate, transformRoutineVersionValues, routineVersionValidation);
    }

    const versions = useMemo(() => (existing?.root as RoutineShape)?.versions?.map(v => v.versionLabel) ?? [], [existing]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <RoutineSingleStepForm
                disabled={!(isCreate || permissions.canUpdate)}
                display={display}
                existing={existing}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                isSubroutine={isSubroutine}
                versions={versions}
                {...props}
                {...formik}
            />}
        </Formik>
    );
}
