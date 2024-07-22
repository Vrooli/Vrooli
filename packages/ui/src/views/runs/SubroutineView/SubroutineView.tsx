import { exists, noop, noopSubmit, ResourceList as ResourceListType, RoutineType, RoutineVersion, Tag } from "@local/shared";
import { Box, Stack } from "@mui/material";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { TextCollapse } from "components/containers/TextCollapse/TextCollapse";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { ResourceList } from "components/lists/resource";
import { TagList } from "components/lists/TagList/TagList";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { VersionDisplay } from "components/text/VersionDisplay/VersionDisplay";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { generateInitialValues, generateYupSchema } from "forms/generators";
import { useObjectActions } from "hooks/useObjectActions";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { FormSection } from "styles";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { defaultSchemaInput, defaultSchemaOutput, parseConfigCallData, parseSchemaInputOutput } from "utils/runUtils";
import { ResourceListShape } from "utils/shape/models/resourceList";
import { RoutineShape } from "utils/shape/models/routine";
import { TagShape } from "utils/shape/models/tag";
import { routineInitialValues } from "views/objects/routine";
import { RoutineApiForm, RoutineCodeForm, RoutineDataForm, RoutineGenerateForm, RoutineInformationalForm, RoutineSmartContractForm } from "views/objects/routine/RoutineTypeForms/RoutineTypeForms";
import { SubroutineViewProps } from "../types";

const basicInfoStackStyle = {
    marginLeft: "auto",
    marginRight: "auto",
    width: "min(100%, 700px)",
    padding: 2,
} as const;

//TODO update to latest from RoutineView
export function SubroutineView({
    loading,
    handleUserInputsUpdate,
    handleSaveProgress,
    onClose,
    owner,
    routineVersion,
    run,
}: SubroutineViewProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);

    const [internalRoutineVersion, setInternalRoutineVersion] = useState(routineVersion);
    useEffect(() => {
        setInternalRoutineVersion(routineVersion);
    }, [routineVersion]);
    const updateRoutine = useCallback((routineVersion: RoutineVersion) => { setInternalRoutineVersion(routineVersion); }, [setInternalRoutineVersion]);

    const availableLanguages = useMemo<string[]>(() => (internalRoutineVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [internalRoutineVersion?.translations]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { description, instructions, name } = useMemo(() => {
        const languages = getUserLanguages(session);
        const { description, instructions, name } = getTranslation(internalRoutineVersion, languages, true);
        return {
            description,
            instructions,
            name,
        };
    }, [internalRoutineVersion, session]);

    const confirmLeave = useCallback((callback: () => any) => {
        // Confirmation dialog for leaving routine
        PubSub.get().publish("alertDialog", {
            messageKey: "RunStopConfirm",
            buttons: [
                {
                    labelKey: "Yes",
                    onClick: () => {
                        // Save progress
                        handleSaveProgress();
                        // Trigger callback
                        callback();
                    },
                },
                { labelKey: "Cancel" },
            ],
        });
    }, [handleSaveProgress]);

    const configCallData = useMemo(function configCallDataMemo() {
        return parseConfigCallData(routineVersion.configCallData, routineVersion.routineType);
    }, [routineVersion.configCallData, routineVersion.routineType]);
    const schemaInput = useMemo(() => parseSchemaInputOutput(routineVersion.configFormInput, defaultSchemaInput), [routineVersion.configFormInput]);
    const schemaOutput = useMemo(() => parseSchemaInputOutput(routineVersion.configFormOutput, defaultSchemaOutput), [routineVersion.configFormOutput]);

    const routineTypeBaseProps = useMemo(function routineTypeBasePropsMemo() {
        return {
            configCallData,
            disabled: false,
            isEditing: false,
            onConfigCallDataChange: noop,
            onSchemaInputChange: noop,
            onSchemaOutputChange: noop,
            schemaInput,
            schemaOutput,
        };
    }, [configCallData, schemaInput, schemaOutput]);

    // Type-specific components
    const routineTypeComponents = useMemo(function routineTypeComponentsMemo() {
        switch (routineVersion.routineType) {
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
            case RoutineType.SmartContract:
                return <RoutineSmartContractForm {...routineTypeBaseProps} />;
            default:
                // NOTE: We don't display the multi-step form, as its data should be coverted into smaller steps
                return null;
        }
    }, [routineTypeBaseProps, routineVersion.routineType]);

    const inputInitialValues = useMemo(function inputInitialValuesMemo() {
        console.log("calculating inputInitialValues", schemaInput.elements, generateInitialValues(schemaInput.elements));
        return generateInitialValues(schemaInput.elements);
    }, [schemaInput]);
    const inputValidationSchema = useMemo(function inputValidationSchemaMemo() {
        return schemaInput ? generateYupSchema(schemaInput) : undefined;
    }, [schemaInput]);

    //TODO
    // /**
    //  * Update formik values with the current user inputs, if any
    //  */
    // useEffect(() => {
    //     if (!run?.inputs || !Array.isArray(run?.inputs) || run.inputs.length === 0) return;
    //     const updatedValues = runInputsToFormik(run.inputs);
    //     formik.setValues(updatedValues);
    //     // eslint-disable-next-line react-hooks/exhaustive-deps
    // }, [formik.setValues, run?.inputs]);

    //TODO
    // /**
    //  * Update run with updated user inputs
    //  */
    // useEffect(() => {
    //     if (!formik.values) return;
    //     const updatedValues = formikToRunInputs(formik.values);
    //     handleUserInputsUpdate(updatedValues);
    // }, [handleUserInputsUpdate, formik.values, run?.inputs]);

    const markAsComplete = () => { }; //TODO

    const actionData = useObjectActions({
        object: internalRoutineVersion,
        objectType: "RoutineVersion",
        openAddCommentDialog: noop,
        setLocation,
        setObject: setInternalRoutineVersion,
    });

    const initialValues = useMemo(() => routineInitialValues(session, internalRoutineVersion), [internalRoutineVersion, session]);
    const resourceList = useMemo<ResourceListShape | null | undefined>(() => initialValues.resourceList as ResourceListShape | null | undefined, [initialValues]);
    const tags = useMemo<TagShape[] | null | undefined>(() => (initialValues.root as RoutineShape)?.tags as TagShape[] | null | undefined, [initialValues]);

    return (
        <Box mb={8} width="min(100%, 600px)">
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={noopSubmit}
            >
                {() => <Stack direction="column" spacing={4} sx={basicInfoStackStyle}>
                    {exists(resourceList) && Array.isArray(resourceList.resources) && resourceList.resources.length > 0 && <ResourceList
                        horizontal
                        title={"Resources"}
                        list={resourceList as unknown as ResourceListType}
                        canUpdate={false}
                        handleUpdate={noop}
                        loading={loading}
                        parent={{ __typename: "RoutineVersion", id: routineVersion?.id ?? "" }}
                    />}
                    {(!!description || !!instructions) && <FormSection>
                        <TextCollapse
                            title="Description"
                            text={description}
                            loading={loading}
                            loadingLines={2}
                        />
                        <TextCollapse
                            title="Instructions"
                            text={instructions}
                            loading={loading}
                            loadingLines={4}
                        />
                    </FormSection>}
                    {Boolean(routineTypeComponents) && (
                        <Formik
                            enableReinitialize={true}
                            initialValues={inputInitialValues}
                            onSubmit={markAsComplete}
                            validationSchema={inputValidationSchema}
                        >
                            {(formik) => (
                                <FormSection>
                                    {routineTypeComponents}
                                </FormSection>
                            )}
                        </Formik>
                    )}
                    <FormSection>
                        <ContentCollapse
                            isOpen={false}
                            title="Additional Information"
                        >
                            <RelationshipList
                                isEditing={false}
                                objectType={"Routine"}
                            />
                            {/* Tags */}
                            {exists(tags) && tags.length > 0 && <TagList
                                maxCharacters={30}
                                parentId={internalRoutineVersion?.id ?? ""}
                                tags={tags as Tag[]}
                                sx={{ marginTop: 4 }}
                            />}
                            {/* Date and version labels */}
                            <Stack direction="row" spacing={1} mt={2} mb={1}>
                                {/* Date created */}
                                <DateDisplay
                                    loading={loading}
                                    timestamp={internalRoutineVersion?.created_at}
                                />
                                <VersionDisplay
                                    currentVersion={internalRoutineVersion}
                                    prefix={" - v"}
                                    versions={internalRoutineVersion?.root?.versions}
                                />
                            </Stack>
                        </ContentCollapse>
                    </FormSection>
                </Stack>}
            </Formik>
        </Box>
    );
}
