import { FormBuilder, ResourceListShape, ResourceList as ResourceListType, RoutineShape, RoutineType, RoutineVersion, RoutineVersionConfig, Tag, TagShape, exists, getTranslation, noop, noopSubmit, uuidValidate } from "@local/shared";
import { Box, Stack } from "@mui/material";
import { Formik } from "formik";
import { useContext, useEffect, useMemo, useState } from "react";
import { ContentCollapse } from "../../components/containers/ContentCollapse/ContentCollapse.js";
import { TextCollapse } from "../../components/containers/TextCollapse/TextCollapse.js";
import { RelationshipList } from "../../components/lists/RelationshipList/RelationshipList.js";
import { ResourceList } from "../../components/lists/ResourceList/ResourceList.js";
import { TagList } from "../../components/lists/TagList/TagList.js";
import { DateDisplay } from "../../components/text/DateDisplay.js";
import { VersionDisplay } from "../../components/text/VersionDisplay.js";
import { SessionContext } from "../../contexts.js";
import { FormSection } from "../../styles.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { getDisplay } from "../../utils/display/listTools.js";
import { getLanguageSubtag, getPreferredLanguage, getUserLanguages } from "../../utils/display/translationTools.js";
import { routineSingleStepInitialValues } from "../objects/routine/RoutineSingleStepUpsert.js";
import { RoutineApiForm, RoutineDataConverterForm, RoutineDataForm, RoutineFormPropsBase, RoutineGenerateForm, RoutineInformationalForm, RoutineSmartContractForm } from "../objects/routine/RoutineTypeForms.js";
import { SubroutineViewProps } from "./types.js";

const EMPTY_OBJECT = {};

const basicInfoStackStyle = {
    marginLeft: "auto",
    marginRight: "auto",
    width: "min(100%, 700px)",
    padding: 2,
} as const;
const tagListStyle = { marginTop: 4 } as const;

export function SubroutineView({
    formikRef,
    handleGenerateOutputs,
    isGeneratingOutputs,
    isLoading,
    routineVersion,
}: SubroutineViewProps) {
    const session = useContext(SessionContext);
    const [, setLanguage] = useState<string>(getUserLanguages(session)[0]);

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

    const hasFormErrors = useMemo(() => Object.values(formikRef.current?.errors ?? {}).some((value) => exists(value)), []);

    const config = useMemo(function configMemo() {
        return RoutineVersionConfig.deserialize({ config: routineVersion.config, routineType: routineVersion.routineType }, console);
    }, [routineVersion.config, routineVersion.routineType]);

    // Type-specific components
    const routineTypeComponents = useMemo(function routineTypeComponentsMemo() {
        const loading = isGeneratingOutputs || isLoading;
        const isLoggedIn = uuidValidate(getCurrentUser(session).id);
        const isDeleted = (routineVersion as RoutineVersion).isDeleted ?? routineVersion.root?.isDeleted ?? false;
        const canRun = routineVersion.you?.canRun ?? false;
        const hasErrors = hasFormErrors;
        const routineTypeBaseProps: RoutineFormPropsBase = {
            config,
            disabled: false,
            display: "view",
            handleClearRun: noop, // Only used in single-step routines, which we are not in
            handleCompleteStep: noop, // Only used in single-step routines, which we are not in
            handleRunStep: handleGenerateOutputs,
            hasErrors,
            isCompleteStepDisabled: true, // Not in a single-step routine
            isPartOfMultiStepRoutine: true,
            isRunStepDisabled: loading || !isLoggedIn || isDeleted || !canRun || hasFormErrors,
            isRunningStep: isGeneratingOutputs,
            onCallDataActionChange: noop, // Only used in edit mode
            onCallDataApiChange: noop, // Only used in edit mode
            onCallDataCodeChange: noop, // Only used in edit mode
            onCallDataGenerateChange: noop, // Only used in edit mode
            onCallDataSmartContractChange: noop, // Only used in edit mode
            onFormInputChange: noop, // Only used in edit mode
            onFormOutputChange: noop, // Only used in edit mode
            onGraphChange: noop, // Only used in edit mode
            onRunChange: noop, // Not in a single-step routine
            routineId: internalRoutineVersion?.id ?? "",
            routineName: getDisplay(internalRoutineVersion).title,
            run: null, // Not in a single-step routine
        };

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
    }, [config, handleGenerateOutputs, hasFormErrors, internalRoutineVersion, isGeneratingOutputs, isLoading, routineVersion, session]);

    const inputValidationSchema = useMemo(function inputValidationSchemaMemo() {
        // TODO might need to do output validation as well
        return config.formInput ? FormBuilder.generateYupSchema(config.formInput.schema, "input") : undefined;
    }, [config.formInput]);

    const initialValues = useMemo(() => routineSingleStepInitialValues(session, internalRoutineVersion), [internalRoutineVersion, session]);
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
                                sx={tagListStyle}
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
