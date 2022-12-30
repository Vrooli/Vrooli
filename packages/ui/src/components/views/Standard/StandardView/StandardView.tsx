import { Box, CircularProgress, Palette, Stack, useTheme } from "@mui/material"
import { useLocation } from '@shared/route';
import { FindByIdInput, InputType } from "@shared/consts";
import { useLazyQuery } from "graphql/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { BaseStandardInput, CommentContainer, ResourceListHorizontal, TextCollapse, VersionDisplay, SnackSeverity, ObjectTitle, TagList, StatsCompact, DateDisplay, ObjectActionsRow, ColorIconButton } from "components";
import { StandardViewProps } from "../types";
import { base36ToUuid, firstString, getLanguageSubtag, getLastUrlPart, getObjectEditUrl, getPreferredLanguage, getTranslation, getUserLanguages, ObjectAction, ObjectActionComplete, openObject, PubSub, standardToFieldData, TagShape } from "utils";
import { ResourceList, Standard } from "types";
import { uuid, uuidValidate } from '@shared/uuid';
import { FieldData, FieldDataJSON } from "forms/types";
import { useFormik } from "formik";
import { generateInputComponent } from "forms/generators";
import { PreviewSwitch, RelationshipButtons } from "components/inputs";
import { RelationshipsObject } from "components/inputs/types";
import { smallHorizontalScrollbar } from "components/lists/styles";
import { EditIcon } from "@shared/icons";
import { standardEndpoint } from "graphql/endpoints";

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
    // Fetch data
    const { id, versionGroupId } = useMemo(() => {
        // URL is /object/:versionGroupId/?:id
        const last = base36ToUuid(getLastUrlPart(0), false);
        const secondLast = base36ToUuid(getLastUrlPart(1), false);
        return {
            id: uuidValidate(secondLast) ? last : undefined,
            versionGroupId: uuidValidate(secondLast) ? secondLast : last,
        }
    }, []);
    const [getData, { data, loading }] = useLazyQuery<Standard, FindByIdInput, 'standard'>(...standardEndpoint.findOne, { errorPolicy: 'all' });
    useEffect(() => {
        if (uuidValidate(id) || uuidValidate(versionGroupId)) getData({ variables: { id, versionGroupId } });
        else PubSub.get().publishSnack({ messageKey: 'InvalidUrlId', severity: SnackSeverity.Error });
    }, [getData, id, versionGroupId])

    const [standard, setStandard] = useState<Standard | null>(null);
    useEffect(() => {
        if (!data) return;
        setStandard(data.standard);
    }, [data]);
    const updateStandard = useCallback((newStandard: Standard) => { setStandard(newStandard); }, [setStandard]);

    const availableLanguages = useMemo<string[]>(() => (standard?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [standard?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const schema = useMemo<FieldData | null>(() => (standard ? standardToFieldData({
        fieldName: 'preview',
        description: getTranslation(standard, [language]).description,
        helpText: null,
        props: standard.props,
        name: standard.name,
        type: standard.type,
        yup: standard.yup,
    }) : null), [language, standard]);
    const previewFormik = useFormik({
        initialValues: {
            preview: JSON.stringify((schema as FieldDataJSON)?.props?.format),
        },
        enableReinitialize: true,
        onSubmit: () => { },
    });

    const { canEdit, description, name } = useMemo(() => {
        const { canEdit } = standard?.permissionsStandard ?? {};
        const { description } = getTranslation(standard ?? partialData, [language]);
        const name = firstString(standard?.name, partialData?.name);
        return { canEdit, description, name };
    }, [standard, language, partialData]);

    useEffect(() => {
        document.title = `${name} | Vrooli`;
    }, [name]);

    const onEdit = useCallback(() => {
        if (!standard) return;
        setLocation(getObjectEditUrl(standard));
    }, [setLocation, standard]);

    const [isAddCommentOpen, setIsAddCommentOpen] = useState(false);
    const openAddCommentDialog = useCallback(() => { setIsAddCommentOpen(true); }, []);
    const closeAddCommentDialog = useCallback(() => { setIsAddCommentOpen(false); }, []);

    const onActionStart = useCallback((action: ObjectAction) => {
        switch (action) {
            case ObjectAction.Comment:
                openAddCommentDialog();
                break;
            case ObjectAction.Edit:
                onEdit();
                break;
            case ObjectAction.Stats:
                //TODO
                break;
        }
    }, [onEdit, openAddCommentDialog]);

    const onActionComplete = useCallback((action: ObjectActionComplete, data: any) => {
        switch (action) {
            case ObjectActionComplete.VoteDown:
            case ObjectActionComplete.VoteUp:
                if (data.success) {
                    setStandard({
                        ...standard,
                        isUpvoted: action === ObjectActionComplete.VoteUp,
                    } as any)
                }
                break;
            case ObjectActionComplete.Star:
            case ObjectActionComplete.StarUndo:
                if (data.success) {
                    setStandard({
                        ...standard,
                        isStarred: action === ObjectActionComplete.Star,
                    } as any)
                }
                break;
            case ObjectActionComplete.Fork:
                openObject(data.standard, setLocation);
                window.location.reload();
                break;
        }
    }, [standard, setLocation]);

    const [isPreviewOn, setIsPreviewOn] = useState<boolean>(true);
    const onPreviewChange = useCallback((isOn: boolean) => { setIsPreviewOn(isOn); }, []);

    // Handle relationships
    const [relationships, setRelationships] = useState<RelationshipsObject>({
        isComplete: true,
        isPrivate: false,
        owner: null,
        parent: null,
        project: null,
    });
    const onRelationshipsChange = useCallback((newRelationshipsObject: Partial<RelationshipsObject>) => {
        setRelationships({
            ...relationships,
            ...newRelationshipsObject,
        });
    }, [relationships]);

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>({ id: uuid() } as any);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>((partialData?.tags as TagShape[] | undefined) ?? []);

    useEffect(() => {
        setRelationships({
            isComplete: false, //TODO
            isPrivate: standard?.isPrivate ?? false,
            owner: standard?.creator ?? null,
            parent: null,
            // parent: standard?.parent ?? null, TODO
            project: null // TODO
        });
        setResourceList(standard?.resourceList ?? { id: uuid() } as any);
        setTags(standard?.tags ?? []);
    }, [standard]);

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
                {canEdit ? (
                    <ColorIconButton aria-label="confirm-title-change" background={palette.secondary.main} onClick={() => { onActionStart(ObjectAction.Edit) }} >
                        <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </ColorIconButton>
                ) : null}
            </Stack>
            <ObjectTitle
                language={language}
                loading={loading}
                title={standard?.name ?? partialData?.name ?? ''}
                session={session}
                setLanguage={setLanguage}
                translations={standard?.translations ?? partialData?.translations ?? []}
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
                canEdit={false}
                handleUpdate={() => { }} // Intentionally blank
                loading={loading}
                session={session}
                zIndex={zIndex}
            />}
            {/* Box with description */}
            <Box sx={containerProps(palette)}>
                <TextCollapse title="Description" text={description} loading={loading} loadingLines={2} />
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
                        schema ? generateInputComponent({
                            disabled: true,
                            fieldData: schema,
                            formik: previewFormik,
                            session,
                            onUpload: () => { },
                            zIndex,
                        }) :
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
                parentId={standard?.id ?? ''}
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
                    timestamp={standard?.created_at}
                />
                <VersionDisplay
                    currentVersion={standard?.version}
                    prefix={" - "}
                    versions={standard?.versions}
                />
            </Stack>
            {/* Votes, reports, and other basic stats */}
            <StatsCompact
                handleObjectUpdate={updateStandard}
                loading={loading}
                object={standard}
                session={session}
            />
            {/* Action buttons */}
            <ObjectActionsRow
                exclude={[ObjectAction.Edit, ObjectAction.VoteDown, ObjectAction.VoteUp]} // Handled elsewhere
                onActionStart={onActionStart}
                onActionComplete={onActionComplete}
                object={standard}
                session={session}
                zIndex={zIndex}
            />
            {/* Comments */}
            <Box sx={containerProps(palette)}>
                <CommentContainer
                    forceAddCommentOpen={isAddCommentOpen}
                    language={language}
                    objectId={id ?? ''}
                    objectType={CommentFor.Routine}
                    onAddCommentClose={closeAddCommentDialog}
                    session={session}
                    zIndex={zIndex}
                />
            </Box>
        </Box >
    )
}