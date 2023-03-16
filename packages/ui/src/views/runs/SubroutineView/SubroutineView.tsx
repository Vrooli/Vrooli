import { Box, Button, Palette, Stack, useTheme } from "@mui/material";
import { CommentFor, ResourceList, RoutineVersion } from "@shared/consts";
import { SuccessIcon } from "@shared/icons";
import { useLocation } from '@shared/route';
import { uuid } from "@shared/uuid";
import { CommentContainer } from "components/containers/CommentContainer/CommentContainer";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { TextCollapse } from "components/containers/TextCollapse/TextCollapse";
import { GeneratedInputComponentWithLabel } from "components/inputs/generated";
import { RelationshipButtons } from "components/inputs/RelationshipButtons/RelationshipButtons";
import { RelationshipsObject } from "components/inputs/types";
import { ObjectActionsRow } from "components/lists/ObjectActionsRow/ObjectActionsRow";
import { ResourceListHorizontal } from "components/lists/resource";
import { smallHorizontalScrollbar } from "components/lists/styles";
import { TagList } from "components/lists/TagList/TagList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { ObjectTitle } from "components/text/ObjectTitle/ObjectTitle";
import { VersionDisplay } from "components/text/VersionDisplay/VersionDisplay";
import { useFormik } from "formik";
import { FieldData } from "forms/types";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ObjectAction } from "utils/actions/objectActions";
import { defaultRelationships } from "utils/defaults/relationships";
import { defaultResourceList } from "utils/defaults/resourceList";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { PubSub } from "utils/pubsub";
import { formikToRunInputs, runInputsToFormik } from "utils/runUtils";
import { standardVersionToFieldData } from "utils/shape/general";
import { TagShape } from "utils/shape/models/tag";
import { SubroutineViewProps } from "../types";

const containerProps = (palette: Palette) => ({
    boxShadow: 1,
    background: palette.background.paper,
    borderRadius: 1,
    overflow: 'overlay',
    marginTop: 4,
    marginBottom: 4,
    padding: 2,
})

export const SubroutineView = ({
    display = 'page',
    loading,
    handleUserInputsUpdate,
    handleSaveProgress,
    owner,
    routineVersion,
    run,
    session,
    zIndex,
}: SubroutineViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);

    const [internalRoutineVersion, setInternalRoutineVersion] = useState(routineVersion);
    useEffect(() => {
        setInternalRoutineVersion(routineVersion);
    }, [routineVersion]);
    const updateRoutine = useCallback((routineVersion: RoutineVersion) => { setInternalRoutineVersion(routineVersion); }, [setInternalRoutineVersion]);

    const { description, instructions, name } = useMemo(() => {
        const languages = getUserLanguages(session);
        const { description, instructions, name } = getTranslation(internalRoutineVersion, languages, true);
        return {
            description,
            instructions,
            name,
        }
    }, [internalRoutineVersion, session]);

    const confirmLeave = useCallback((callback: () => any) => {
        // Confirmation dialog for leaving routine
        PubSub.get().publishAlertDialog({
            messageKey: 'RunStopConfirm',
            buttons: [
                {
                    labelKey: 'Yes',
                    onClick: () => {
                        // Save progress
                        handleSaveProgress();
                        // Trigger callback
                        callback();
                    }
                },
                { labelKey: 'Cancel' },
            ]
        });
    }, [handleSaveProgress]);

    // The schema and formik keys for the form
    const formValueMap = useMemo<{ [fieldName: string]: FieldData }>(() => {
        if (!internalRoutineVersion) return {};
        const schemas: { [fieldName: string]: FieldData } = {};
        for (let i = 0; i < internalRoutineVersion.inputs?.length; i++) {
            const currInput = internalRoutineVersion.inputs[i];
            if (!currInput.standardVersion) continue;
            const currSchema = standardVersionToFieldData({
                description: getTranslation(currInput, getUserLanguages(session), false).description ?? getTranslation(currInput.standardVersion, getUserLanguages(session), false).description,
                fieldName: `inputs-${currInput.id}`,
                helpText: getTranslation(currInput, getUserLanguages(session), false).helpText,
                props: currInput.standardVersion.props,
                name: currInput.name ?? currInput.standardVersion.root.name,
                standardType: currInput.standardVersion.standardType,
                yup: currInput.standardVersion.yup,
            });
            if (currSchema) {
                schemas[currSchema.fieldName] = currSchema;
            }
        }
        return schemas;
    }, [internalRoutineVersion, session]);
    const formik = useFormik({
        initialValues: Object.entries(formValueMap).reduce((acc, [key, value]) => {
            acc[key] = value.props.defaultValue ?? '';
            return acc;
        }, {}),
        enableReinitialize: true,
        onSubmit: () => { },
    });

    /**
     * Update formik values with the current user inputs, if any
     */
    useEffect(() => {
        console.log('useeffect1 calculating preview formik values', run)
        if (!run?.inputs || !Array.isArray(run?.inputs) || run.inputs.length === 0) return;
        console.log('useeffect 1calling runInputsToFormik', run.inputs)
        const updatedValues = runInputsToFormik(run.inputs);
        console.log('useeffect1 updating formik, values', updatedValues)
        formik.setValues(updatedValues);
    },
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [formik.setValues, run?.inputs]);

    /**
     * Update run with updated user inputs
     */
    useEffect(() => {
        if (!formik.values) return;
        const updatedValues = formikToRunInputs(formik.values);
        handleUserInputsUpdate(updatedValues);
    }, [handleUserInputsUpdate, formik.values, run?.inputs]);

    /**
     * Copy current value of input to clipboard
     * @param fieldName Name of input
     */
    const copyInput = useCallback((fieldName: string) => {
        const input = formik.values[fieldName];
        if (input) {
            navigator.clipboard.writeText(input);
            PubSub.get().publishSnack({ messageKey: 'CopiedToClipboard', severity: 'Success' });
        } else {
            PubSub.get().publishSnack({ messageKey: 'InputEmpty', severity: 'Error' });
        }
    }, [formik.values]);

    const inputComponents = useMemo(() => {
        if (!internalRoutineVersion?.inputs || !Array.isArray(internalRoutineVersion?.inputs) || internalRoutineVersion.inputs.length === 0) return null;
        return (
            <Box>
                {Object.values(formValueMap).map((fieldData: FieldData, index: number) => (
                    <GeneratedInputComponentWithLabel
                        copyInput={copyInput}
                        disabled={false}
                        fieldData={fieldData}
                        formik={formik}
                        index={index}
                        session={session}
                        textPrimary={palette.background.textPrimary}
                        onUpload={() => { }}
                        zIndex={zIndex}
                    />
                ))}
            </Box>
        )
    }, [copyInput, formValueMap, palette.background.textPrimary, formik, internalRoutineVersion?.inputs, session, zIndex]);

    const [isAddCommentOpen, setIsAddCommentOpen] = useState(false);
    const openAddCommentDialog = useCallback(() => { setIsAddCommentOpen(true); }, []);
    const closeAddCommentDialog = useCallback(() => { setIsAddCommentOpen(false); }, []);

    const actionData = useObjectActions({
        object: internalRoutineVersion,
        objectType: 'RoutineVersion',
        openAddCommentDialog,
        session,
        setLocation,
        setObject: setInternalRoutineVersion,
    });

    // Handle relationships
    const [relationships, setRelationships] = useState<RelationshipsObject>(defaultRelationships(false, null));
    const onRelationshipsChange = useCallback((change: Partial<RelationshipsObject>) => setRelationships({ ...relationships, ...change }), [relationships]);

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>(defaultResourceList);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);

    useEffect(() => {
        setRelationships({
            isComplete: internalRoutineVersion?.isComplete ?? false,
            isPrivate: internalRoutineVersion?.isPrivate ?? false,
            owner: internalRoutineVersion?.root?.owner ?? null,
            parent: null,
            // parent: internalRoutine?.parent ?? null, TODO
            project: null, //TODO
        });
        setResourceList(internalRoutineVersion?.resourceList ?? { id: uuid() } as any);
        setTags(internalRoutineVersion?.root?.tags ?? []);
    }, [internalRoutineVersion]);
    
    return (
        <>
            <TopBar
                display={display}
                onClose={() => {}}
                session={session}
            />
            <Box sx={{
                marginLeft: 'auto',
                marginRight: 'auto',
                width: 'min(100%, 700px)',
                padding: 2,
            }}>
                <ObjectTitle
                    language={language}
                    loading={loading}
                    title={name}
                    session={session}
                    setLanguage={setLanguage}
                    translations={internalRoutineVersion?.translations ?? []}
                    zIndex={zIndex}
                />
                {/* Resources */}
                {Array.isArray(resourceList.resources) && resourceList.resources.length > 0 && <ResourceListHorizontal
                    title={'Resources'}
                    list={resourceList}
                    canUpdate={false}
                    handleUpdate={() => { }} // Intentionally blank
                    loading={loading}
                    session={session}
                    zIndex={zIndex}
                />}
                {/* Box with description and instructions */}
                <Stack direction="column" spacing={4} sx={containerProps(palette)}>
                    {/* Description */}
                    <TextCollapse session={session} title="Description" text={description} loading={loading} loadingLines={2} />
                    {/* Instructions */}
                    <TextCollapse session={session} title="Instructions" text={instructions} loading={loading} loadingLines={4} />
                </Stack>
                <Box sx={containerProps(palette)}>
                    <ContentCollapse title="Inputs">
                        {inputComponents}
                        <Button startIcon={<SuccessIcon />} fullWidth onClick={() => { }} color="secondary" sx={{ marginTop: 2 }}>Submit</Button>
                    </ContentCollapse>
                </Box>
                {/* Action buttons */}
                <ObjectActionsRow
                    actionData={actionData}
                    exclude={[ObjectAction.Edit, ObjectAction.VoteDown, ObjectAction.VoteUp]} // Handled elsewhere
                    object={internalRoutineVersion}
                    session={session}
                    zIndex={zIndex}
                />
                <Box sx={containerProps(palette)}>
                    <ContentCollapse isOpen={false} title="Additional Information">
                        {/* Relationships */}
                        <RelationshipButtons
                            isEditing={false}
                            objectType={'Routine'}
                            onRelationshipsChange={onRelationshipsChange}
                            relationships={relationships}
                            session={session}
                            zIndex={zIndex}
                        />
                        {/* Tags */}
                        {tags.length > 0 && <TagList
                            maxCharacters={30}
                            parentId={routineVersion?.root?.id ?? ''}
                            session={session}
                            tags={tags as any[]}
                            sx={{ ...smallHorizontalScrollbar(palette), marginTop: 4 }}
                        />}
                        {/* Date and version labels */}
                        <Stack direction="row" spacing={1} mt={2} mb={1}>
                            {/* Date created */}
                            <DateDisplay
                                loading={loading}
                                showIcon={true}
                                timestamp={internalRoutineVersion?.created_at}
                            />
                            <VersionDisplay
                                confirmVersionChange={confirmLeave}
                                currentVersion={internalRoutineVersion}
                                prefix={" - "}
                                versions={internalRoutineVersion?.root?.versions}
                            />
                        </Stack>
                        {/* Votes, reports, and other basic stats */}
                        {/* <StatsCompact
                        handleObjectUpdate={updateRoutine}
                        loading={loading}
                        object={internalRoutineVersion ?? null}
                        session={session}
                    /> */}
                    </ContentCollapse>
                </Box>
                {/* Comments */}
                <Box sx={containerProps(palette)}>
                    <CommentContainer
                        forceAddCommentOpen={isAddCommentOpen}
                        isOpen={false}
                        language={language}
                        objectId={internalRoutineVersion?.id ?? ''}
                        objectType={CommentFor.RoutineVersion}
                        onAddCommentClose={closeAddCommentDialog}
                        session={session}
                        zIndex={zIndex}
                    />
                </Box>
            </Box>
        </>
    )
}