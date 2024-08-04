import { ConfigCallData, DUMMY_ID, FormInputBase, FormSchema, LINKS, LlmModel, NodeLinkShape, NodeShape, RoutineShape, RoutineType, RoutineVersion, RoutineVersionCreateInput, RoutineVersionInputShape, RoutineVersionOutputShape, RoutineVersionShape, RoutineVersionUpdateInput, SearchPageTabOption, Session, defaultConfigCallDataMap, defaultConfigFormInputMap, defaultConfigFormOutputMap, endpointGetRoutineVersion, endpointPostRoutineVersion, endpointPutRoutineVersion, initializeRoutineGraph, noop, noopSubmit, orDefault, parseConfigCallData, parseSchemaInput, parseSchemaOutput, routineVersionTranslationValidation, routineVersionValidation, shapeRoutineVersion, uuid, uuidValidate } from "@local/shared";
import { Checkbox, Divider, FormControlLabel, Grid, Tooltip } from "@mui/material";
import { useSubmitHelper } from "api";
import { AutoFillButton } from "components/buttons/AutoFillButton/AutoFillButton";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { SearchExistingButton } from "components/buttons/SearchExistingButton/SearchExistingButton";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TranslatedRichInput } from "components/inputs/RichInput/RichInput";
import { SelectorBase } from "components/inputs/Selector/Selector";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedTextInput } from "components/inputs/TextInput/TextInput";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { ResourceListInput } from "components/lists/resource/ResourceList/ResourceList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { FieldHelperProps, Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { getRoutineTypeDescription, getRoutineTypeIcon, getRoutineTypeLabel, routineTypes } from "utils/search/schemas/routine";
import { validateFormValues } from "utils/validateFormValues";
import { RoutineApiForm, RoutineCodeForm, RoutineDataForm, RoutineGenerateForm, RoutineInformationalForm, RoutineMultiStepForm, RoutineSmartContractForm } from "../RoutineTypeForms/RoutineTypeForms";
import { BuildRoutineVersion, RoutineFormProps, RoutineUpsertProps } from "../types";

export function routineInitialValues(
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
        nodeLinks: [],
        nodes: [],
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
const languageInputStyle = { flexDirection: "row-reverse" } as const;
const tagSelectorStyle = { marginBottom: 2 } as const;
const versionInputStyle = { marginBottom: 2 } as const;

function RoutineForm({
    disabled,
    display,
    existing,
    isCreate,
    isOpen,
    isReadLoading,
    isSubroutine,
    onClose,
    values,
    versions,
    ...props
}: RoutineFormProps) {
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
    const translationData = useMemo(() => ({
        language,
        setLanguage,
        handleAddLanguage,
        handleDeleteLanguage,
        languages,
    }), [language, languages, setLanguage, handleAddLanguage, handleDeleteLanguage]);

    // Formik fields we need to access and/or set values for
    const [idField] = useField<string>("id");
    const [translationsField, , translationsHelpers] = useField<RoutineVersion["translations"]>("translations");
    const [routineTypeField, , routineTypeHelpers] = useField<RoutineVersion["routineType"]>("routineType");
    const [nodesField, , nodesHelpers] = useField<NodeShape[]>("nodes");
    const [nodeLinksField, , nodeLinksHelpers] = useField<NodeLinkShape[]>("nodeLinks");
    const [inputsField, , inputsHelpers] = useField<RoutineVersionInputShape[]>("inputs");
    const [outputsField, , outputsHelpers] = useField<RoutineVersionOutputShape[]>("outputs");
    const [configCallDataField, , configCallDataHelpers] = useField<RoutineVersion["configCallData"]>("configCallData");
    const [configFormInputField, , configFormInputHelpers] = useField<RoutineVersion["configFormInput"]>("configFormInput");
    const [configFormOutputField, , configFormOutputHelpers] = useField<RoutineVersion["configFormOutput"]>("configFormOutput");
    const [apiVersionField, , apiVersionHelpers] = useField<RoutineVersion["apiVersion"]>("apiVersion");
    const [codeVersionField, , codeVersionHelpers] = useField<RoutineVersion["codeVersion"]>("codeVersion");

    const configCallData = useMemo(function configCallDataMemo() {
        return parseConfigCallData(configCallDataField.value, routineTypeField.value, console);
    }, [configCallDataField.value, routineTypeField.value]);
    const onConfigCallDataChange = useCallback(function onConfigCallDataChange(config: ConfigCallData) {
        configCallDataHelpers.setValue(JSON.stringify(config));
    }, [configCallDataHelpers]);

    const schemaInput = useMemo(function schemeInputMemo() {
        return parseSchemaInput(configFormInputField.value, routineTypeField.value, console);
    }, [configFormInputField.value, routineTypeField.value]);
    const schemaOutput = useMemo(function schemaOutputMemo() {
        return parseSchemaOutput(configFormOutputField.value, routineTypeField.value, console);
    }, [configFormOutputField.value, routineTypeField.value]);
    const onSchemaInputChange = useCallback(function onSchemaInputChange(schema: FormSchema) {
        configFormInputHelpers.setValue(JSON.stringify(schema));
        updateSchemaElements({
            currentElements: inputsField.value,
            elementsHelpers: inputsHelpers,
            language,
            routineVersionId: idField.value,
            schema,
            type: "inputs",
        });
    }, [configFormInputHelpers, idField.value, inputsField.value, inputsHelpers, language]);
    const onSchemaOutputChange = useCallback(function onSchemaOutputChange(schema: FormSchema) {
        configFormOutputHelpers.setValue(JSON.stringify(schema));
        updateSchemaElements({
            currentElements: outputsField.value,
            elementsHelpers: outputsHelpers,
            language,
            routineVersionId: idField.value,
            schema,
            type: "outputs",
        });
    }, [configFormOutputHelpers, idField.value, language, outputsField.value, outputsHelpers]);

    // Multi-step routine data
    const [isGraphOpen, setIsGraphOpen] = useState(false);
    const handleGraphOpen = useCallback(() => {
        // Create initial nodes/links, if not already created
        if (
            (!Array.isArray(nodesField.value) || nodesField.value.length === 0) &&
            (!Array.isArray(nodeLinksField.value) || nodeLinksField.value.length === 0)
        ) {
            const { nodes, nodeLinks } = initializeRoutineGraph(language, idField.value);
            nodesHelpers.setValue(nodes);
            nodeLinksHelpers.setValue(nodeLinks);
        }
        setIsGraphOpen(true);
    }, [idField.value, language, nodeLinksField.value, nodeLinksHelpers, nodesField.value, nodesHelpers]);
    const handleGraphClose = useCallback(() => { setIsGraphOpen(false); }, [setIsGraphOpen]);
    const handleGraphSubmit = useCallback(({ nodes, nodeLinks, translations }: BuildRoutineVersion) => {
        nodesHelpers.setValue(nodes);
        nodeLinksHelpers.setValue(nodeLinks);
        translationsHelpers.setValue(translations);
        setIsGraphOpen(false);
    }, [nodeLinksHelpers, nodesHelpers, translationsHelpers]);

    // Generate AI routine data
    const [model, setModel] = useState<LlmModel | null>(null);

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
            [RoutineType.Action]: hasData(configCallDataField.value, configFormInputField.value, configFormOutputField.value),
            // Has API information
            [RoutineType.Api]: hasData(configCallDataField.value, configFormInputField.value, configFormOutputField.value, apiVersionField.value),
            // Has code information
            [RoutineType.Code]: hasData(configCallDataField.value, configFormInputField.value, configFormOutputField.value, codeVersionField.value),
            // Has an output
            [RoutineType.Data]: hasData(configFormOutputField.value),
            // Has an input or call data
            [RoutineType.Generate]: hasData(configCallDataField.value, configFormInputField.value, configFormOutputField.value),
            // Has input data
            [RoutineType.Informational]: hasData(configFormInputField.value),
            // Has graph information
            [RoutineType.MultiStep]: hasData(nodesField.value, nodeLinksField.value),
            // Also uses code information
            [RoutineType.SmartContract]: hasData(configCallDataField.value, configFormInputField.value, configFormOutputField.value, codeVersionField.value),
        };
        // Helper function to remove all type-specific data on switch
        function performSwitch() {
            apiVersionHelpers.setValue(null);
            codeVersionHelpers.setValue(null);
            inputsHelpers.setValue([]);
            outputsHelpers.setValue([]);
            nodesHelpers.setValue([]);
            nodeLinksHelpers.setValue([]);
            routineTypeHelpers.setValue(type);
            onConfigCallDataChange(defaultConfigCallDataMap[type]());
            onSchemaInputChange(defaultConfigFormInputMap[type]());
            onSchemaOutputChange(defaultConfigFormOutputMap[type]());
            // If we switch to a multi-step routine, open the graph
            if (type === RoutineType.MultiStep) handleGraphOpen();
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
    }, [routineTypeField.value, configCallDataField.value, configFormInputField.value, configFormOutputField.value, apiVersionField.value, codeVersionField.value, nodesField.value, nodeLinksField.value, apiVersionHelpers, codeVersionHelpers, inputsHelpers, outputsHelpers, nodesHelpers, nodeLinksHelpers, routineTypeHelpers, onConfigCallDataChange, onSchemaInputChange, onSchemaOutputChange, handleGraphOpen]);

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
        endpointCreate: endpointPostRoutineVersion,
        endpointUpdate: endpointPutRoutineVersion,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "RoutineVersion" });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<RoutineVersionCreateInput | RoutineVersionUpdateInput, RoutineVersion>({
        disabled,
        existing,
        fetch,
        inputs: transformRoutineVersionValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    const routineTypeBaseProps = useMemo(function routineTypeBasePropsMemo() {
        return {
            configCallData,
            disabled,
            display: "edit",
            handleGenerateOutputs: noop,
            isGeneratingOutputs: false,
            onConfigCallDataChange,
            onSchemaInputChange,
            onSchemaOutputChange,
            schemaInput,
            schemaOutput,
        } as const;
    }, [configCallData, disabled, onConfigCallDataChange, onSchemaInputChange, onSchemaOutputChange, schemaInput, schemaOutput]);

    // Type-specific components
    const routineTypeComponents = useMemo(function routineTypeComponentsMemo() {
        switch (routineTypeField.value) {
            case RoutineType.Api:
                return <RoutineApiForm {...routineTypeBaseProps} />;
            case RoutineType.Code:
                return <RoutineCodeForm {...routineTypeBaseProps} />;
            case RoutineType.Data:
                return <RoutineDataForm {...routineTypeBaseProps} />;
            case RoutineType.Generate:
                return <RoutineGenerateForm {...routineTypeBaseProps} />;
            case RoutineType.Informational:
                return <RoutineInformationalForm {...routineTypeBaseProps} />;
            case RoutineType.MultiStep:
                return <RoutineMultiStepForm
                    {...routineTypeBaseProps}
                    isGraphOpen={isGraphOpen}
                    handleGraphClose={handleGraphClose}
                    handleGraphOpen={handleGraphOpen}
                    handleGraphSubmit={handleGraphSubmit}
                    nodeLinks={nodeLinksField.value}
                    nodes={nodesField.value}
                    routineId={idField.value}
                    translations={translationsField.value}
                    translationData={translationData}
                />;
            case RoutineType.SmartContract:
                return <RoutineSmartContractForm {...routineTypeBaseProps} />;
        }
    }, [handleGraphClose, handleGraphOpen, handleGraphSubmit, idField.value, isGraphOpen, nodeLinksField.value, nodesField.value, routineTypeField.value, routineTypeBaseProps, translationData, translationsField.value]);

    const resourceListParent = useMemo(function resourceListParent() {
        return { __typename: "RoutineVersion", id: values.id } as const;
    }, [values.id]);

    return (
        <MaybeLargeDialog
            display={display}
            id="routine-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateRoutine" : "UpdateRoutine")}
            />
            <SearchExistingButton
                href={`${LINKS.Search}?type="${SearchPageTabOption.Routine}"`}
                text="Search existing routines"
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={700}
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
                            {/* TODO: work on fix for autoFocus accessibility issue. Probably need to use ref and useEffect, which also requires making RichInput a forwardRef. If doing this, then we can autoFocus in the helpbutton edit mode as well */}
                            <TranslatedTextInput
                                autoFocus
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
                    handleAutoFill={() => { }} //TODO
                    isLoadingAutoFill={false} //TODO
                />}
            />
        </MaybeLargeDialog >
    );
}

export function RoutineUpsert({
    isCreate,
    isOpen,
    isSubroutine = false,
    overrideObject,
    ...props
}: RoutineUpsertProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useObjectFromUrl<RoutineVersion, RoutineVersionShape>({
        ...endpointGetRoutineVersion,
        isCreate,
        objectType: "RoutineVersion",
        overrideObject,
        transform: (existing) => routineInitialValues(session, existing),
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
            {(formik) => <RoutineForm
                disabled={!(isCreate || permissions.canUpdate)}
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
