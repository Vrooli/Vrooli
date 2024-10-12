import { ResourceListShape, ResourceList as ResourceListType, RoutineShape, RoutineType, Tag, TagShape, exists, generateYupSchema, getTranslation, noop, noopSubmit, parseConfigCallData, parseSchemaInput, parseSchemaOutput } from "@local/shared";
import { Box, Stack } from "@mui/material";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { TextCollapse } from "components/containers/TextCollapse/TextCollapse";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { ResourceList } from "components/lists/ResourceList/ResourceList";
import { TagList } from "components/lists/TagList/TagList";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { VersionDisplay } from "components/text/VersionDisplay/VersionDisplay";
import { SessionContext } from "contexts";
import { Formik } from "formik";
import { useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FormSection } from "styles";
import { getLanguageSubtag, getPreferredLanguage, getUserLanguages } from "utils/display/translationTools";
import { routineInitialValues } from "views/objects/routine";
import { RoutineApiForm, RoutineDataConverterForm, RoutineDataForm, RoutineGenerateForm, RoutineInformationalForm, RoutineSmartContractForm } from "views/objects/routine/RoutineTypeForms/RoutineTypeForms";
import { SubroutineViewProps } from "../types";

const EMPTY_OBJECT = {};

const basicInfoStackStyle = {
    marginLeft: "auto",
    marginRight: "auto",
    width: "min(100%, 700px)",
    padding: 2,
} as const;

export function SubroutineView({
    formikRef,
    handleGenerateOutputs,
    isGeneratingOutputs,
    isLoading,
    routineVersion,
}: SubroutineViewProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);

    const [internalRoutineVersion, setInternalRoutineVersion] = useState(routineVersion);
    useEffect(() => {
        setInternalRoutineVersion(routineVersion);
    }, [routineVersion]);

    const availableLanguages = useMemo<string[]>(() => (internalRoutineVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [internalRoutineVersion?.translations]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { description, instructions } = useMemo(() => {
        const languages = getUserLanguages(session);
        const { description, instructions } = getTranslation(internalRoutineVersion, languages, true);
        return {
            description,
            instructions,
        };
    }, [internalRoutineVersion, session]);

    const configCallData = useMemo(function configCallDataMemo() {
        return parseConfigCallData(routineVersion.configCallData, routineVersion.routineType, console);
    }, [routineVersion.configCallData, routineVersion.routineType]);
    const schemaInput = useMemo(function schemaInputMemo() {
        return parseSchemaInput(routineVersion.configFormInput, routineVersion.routineType, console);
    }, [routineVersion.configFormInput, routineVersion.routineType]);
    const schemaOutput = useMemo(function schemaOutputMemo() {
        return parseSchemaOutput(routineVersion.configFormOutput, routineVersion.routineType, console);
    }, [routineVersion.configFormOutput, routineVersion.routineType]);

    const routineTypeBaseProps = useMemo(function routineTypeBasePropsMemo() {
        return {
            configCallData,
            disabled: false,
            display: "run",
            handleGenerateOutputs,
            isGeneratingOutputs,
            onConfigCallDataChange: noop,
            onSchemaInputChange: noop,
            onSchemaOutputChange: noop,
            schemaInput,
            schemaOutput,
        } as const;
    }, [configCallData, handleGenerateOutputs, isGeneratingOutputs, schemaInput, schemaOutput]);

    // Type-specific components
    const routineTypeComponents = useMemo(function routineTypeComponentsMemo() {
        switch (routineVersion.routineType) {
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
                // NOTE: We don't display the multi-step form, as its data should be coverted into smaller steps
                return null;
        }
    }, [routineTypeBaseProps, routineVersion.routineType]);

    const inputValidationSchema = useMemo(function inputValidationSchemaMemo() {
        // TODO need one to combine inputs and outputs together
        return schemaInput ? generateYupSchema(schemaInput) : undefined;
    }, [schemaInput]);

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
                        loading={isLoading}
                        parent={{ __typename: "RoutineVersion", id: routineVersion?.id ?? "" }}
                    />}
                    {(!!description || !!instructions) && <FormSection>
                        <TextCollapse
                            title="Description"
                            text={description}
                            loading={isLoading}
                            loadingLines={2}
                        />
                        <TextCollapse
                            title="Instructions"
                            text={instructions}
                            loading={isLoading}
                            loadingLines={4}
                        />
                    </FormSection>}
                    {Boolean(routineTypeComponents) && (
                        <Formik
                            initialValues={formikRef.current?.initialValues ?? EMPTY_OBJECT}
                            innerRef={formikRef}
                            onSubmit={noopSubmit} // Form submission is handled elsewhere
                            validationSchema={inputValidationSchema}
                        >
                            {() => (
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
                                    loading={isLoading}
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
