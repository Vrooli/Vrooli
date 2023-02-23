import { Box, CircularProgress, Palette, Stack, useTheme } from "@mui/material"
import { useLocation } from '@shared/route';
import { CommentFor, FindVersionInput, InputType, ResourceList, StandardVersion } from "@shared/consts";
import { useCallback, useEffect, useMemo, useState } from "react";
import { BaseStandardInput, CommentContainer, ResourceListHorizontal, TextCollapse, VersionDisplay, ObjectTitle, TagList, StatsCompact, DateDisplay, ObjectActionsRow, ColorIconButton } from "components";
import { StandardViewProps } from "../types";
import { defaultRelationships, defaultResourceList, getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages, ObjectAction, standardVersionToFieldData, TagShape, useObjectActions, useObjectFromUrl } from "utils";
import { uuid } from '@shared/uuid';
import { FieldData, FieldDataJSON } from "forms/types";
import { useFormik } from "formik";
import { PreviewSwitch, RelationshipButtons } from "components/inputs";
import { RelationshipsObject } from "components/inputs/types";
import { smallHorizontalScrollbar } from "components/lists/styles";
import { EditIcon } from "@shared/icons";
import { standardVersionFindOne } from "api/generated/endpoints/standardVersion";
import { GeneratedInputComponent } from "components/inputs/generated";

const containerProps = (palette: Palette) => ({
    boxShadow: 1,
    background: palette.background.paper,
    borderRadius: 1,
    overflow: 'overlay',
    marginTop: 4,
    marginBottom: 4,
    padding: 2,
})

export const StandardView = ({
    partialData,
    session,
    zIndex,
}: StandardViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const { id, isLoading, object: standardVersion, permissions, setObject: setStandardVersion } = useObjectFromUrl<StandardVersion, FindVersionInput>({
        query: standardVersionFindOne,
        endpoint: 'standardVersion',
        partialData,
        session,
    });

    const availableLanguages = useMemo<string[]>(() => (standardVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [standardVersion?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const schema = useMemo<FieldData | null>(() => (standardVersion ? standardVersionToFieldData({
        fieldName: 'preview',
        description: getTranslation(standardVersion, [language]).description,
        helpText: null,
        props: standardVersion.props,
        name: standardVersion.root.name,
        standardType: standardVersion.standardType,
        yup: standardVersion.yup,
    }) : null), [language, standardVersion]);
    const previewFormik = useFormik({
        initialValues: {
            preview: JSON.stringify((schema as FieldDataJSON)?.props?.format),
        },
        enableReinitialize: true,
        onSubmit: () => { },
    });

    const { description, name } = useMemo(() => {
        const { description } = getTranslation(standardVersion ?? partialData, [language]);
        const { name } = standardVersion?.root ?? partialData?.root ?? {};
        return { description, name };
    }, [standardVersion, language, partialData]);

    useEffect(() => {
        document.title = `${name} | Vrooli`;
    }, [name]);

    const [isAddCommentOpen, setIsAddCommentOpen] = useState(false);
    const openAddCommentDialog = useCallback(() => { setIsAddCommentOpen(true); }, []);
    const closeAddCommentDialog = useCallback(() => { setIsAddCommentOpen(false); }, []);

    const actionData = useObjectActions({
        object: standardVersion,
        objectType: 'Standard',
        openAddCommentDialog,
        session,
        setLocation,
        setObject: setStandardVersion,
    });

    const [isPreviewOn, setIsPreviewOn] = useState<boolean>(true);
    const onPreviewChange = useCallback((isOn: boolean) => { setIsPreviewOn(isOn); }, []);

    // Handle relationships
    const [relationships, setRelationships] = useState<RelationshipsObject>(defaultRelationships(false, null));
    const onRelationshipsChange = useCallback((change: Partial<RelationshipsObject>) => setRelationships({ ...relationships, ...change }), [relationships]);

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>(defaultResourceList);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>((partialData?.root?.tags as TagShape[] | undefined) ?? []);

    useEffect(() => {
        setRelationships({
            isComplete: false, //TODO
            isPrivate: standardVersion?.isPrivate ?? false,
            owner: standardVersion?.root?.owner ?? null,
            parent: null,
            // parent: standard?.parent ?? null, TODO
            project: null // TODO
        });
        setResourceList(standardVersion?.resourceList ?? { id: uuid() } as any);
        setTags(standardVersion?.root?.tags ?? []);
    }, [standardVersion]);

    return (
        <Box sx={{
            marginLeft: 'auto',
            marginRight: 'auto',
            width: 'min(100%, 700px)',
            padding: 2,
        }}>
            {/* Edit button, positioned at bottom corner of screen */}
            <Stack direction="row" spacing={2} sx={{
                position: 'fixed',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                zIndex: zIndex + 2,
                bottom: 0,
                right: 0,
                // Accounts for BottomNav
                marginBottom: {
                    xs: 'calc(56px + 16px + env(safe-area-inset-bottom))',
                    md: 'calc(16px + env(safe-area-inset-bottom))'
                },
                marginLeft: 'calc(16px + env(safe-area-inset-left))',
                marginRight: 'calc(16px + env(safe-area-inset-right))',
                height: 'calc(64px + env(safe-area-inset-bottom))',
            }}>
                {/* Edit button */}
                {permissions.canUpdate ? (
                    <ColorIconButton aria-label="confirm-title-change" background={palette.secondary.main} onClick={() => { actionData.onActionStart(ObjectAction.Edit) }} >
                        <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </ColorIconButton>
                ) : null}
            </Stack>
            <ObjectTitle
                language={language}
                loading={isLoading}
                title={name}
                session={session}
                setLanguage={setLanguage}
                translations={standardVersion?.translations ?? partialData?.translations ?? []}
                zIndex={zIndex}
            />
            {/* Relationships */}
            <RelationshipButtons
                isEditing={false}
                objectType={'Routine'}
                onRelationshipsChange={onRelationshipsChange}
                relationships={relationships}
                session={session}
                zIndex={zIndex}
            />
            {/* Resources */}
            {Array.isArray(resourceList.resources) && resourceList.resources.length > 0 && <ResourceListHorizontal
                title={'Resources'}
                list={resourceList}
                canUpdate={false}
                handleUpdate={() => { }} // Intentionally blank
                loading={isLoading}
                session={session}
                zIndex={zIndex}
            />}
            {/* Box with description */}
            <Box sx={containerProps(palette)}>
                <TextCollapse session={session} title="Description" text={description} loading={isLoading} loadingLines={2} />
            </Box>
            {/* Box with standard */}
            <Stack direction="column" spacing={4} sx={containerProps(palette)}>
                {/* Build/Preview switch */}
                <PreviewSwitch
                    isPreviewOn={isPreviewOn}
                    onChange={onPreviewChange}
                />
                {
                    isPreviewOn ?
                        schema ? <GeneratedInputComponent
                            disabled={true}
                            fieldData={schema}
                            formik={previewFormik}
                            session={session}
                            onUpload={() => { }}
                            zIndex={zIndex}
                        /> :
                            <Box sx={{
                                minHeight: 'min(300px, 25vh)',
                                display: 'flex',
                                justifyContent: 'center',
                                alignItems: 'center',
                            }}>
                                <CircularProgress color="secondary" />
                            </Box> :
                        <BaseStandardInput
                            fieldName="preview"
                            inputType={schema?.type ?? InputType.TextField}
                            isEditing={false}
                            schema={schema}
                            onChange={() => { }} // Intentionally blank
                            storageKey={''} // Intentionally blank
                        />
                }
            </Stack>
            {/* Tags */}
            {tags.length > 0 && <TagList
                maxCharacters={30}
                parentId={standardVersion?.id ?? ''}
                session={session}
                tags={tags as any[]}
                sx={{ ...smallHorizontalScrollbar(palette), marginTop: 4 }}
            />}
            {/* Date and version labels */}
            <Stack direction="row" spacing={1} mt={2} mb={1}>
                {/* Date created */}
                <DateDisplay
                    loading={isLoading}
                    showIcon={true}
                    timestamp={standardVersion?.created_at}
                />
                <VersionDisplay
                    currentVersion={standardVersion}
                    prefix={" - "}
                    versions={standardVersion?.root?.versions}
                />
            </Stack>
            {/* Votes, reports, and other basic stats */}
            {/* <StatsCompact
                handleObjectUpdate={updateStandard}
                loading={loading}
                object={standardVersion}
                session={session}
            /> */}
            {/* Action buttons */}
            <ObjectActionsRow
                actionData={actionData}
                exclude={[ObjectAction.Edit, ObjectAction.VoteDown, ObjectAction.VoteUp]} // Handled elsewhere
                object={standardVersion}
                session={session}
                zIndex={zIndex}
            />
            {/* Comments */}
            <Box sx={containerProps(palette)}>
                <CommentContainer
                    forceAddCommentOpen={isAddCommentOpen}
                    language={language}
                    objectId={standardVersion?.id ?? ''}
                    objectType={CommentFor.StandardVersion}
                    onAddCommentClose={closeAddCommentDialog}
                    session={session}
                    zIndex={zIndex}
                />
            </Box>
        </Box >
    )
}