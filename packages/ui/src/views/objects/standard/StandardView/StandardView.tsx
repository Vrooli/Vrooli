import { Box, CircularProgress, Palette, Stack, useTheme } from "@mui/material";
import { CommentFor, FindVersionInput, InputType, ResourceList, StandardVersion } from "@shared/consts";
import { EditIcon } from "@shared/icons";
import { useLocation } from '@shared/route';
import { uuid } from '@shared/uuid';
import { standardVersionFindOne } from "api/generated/endpoints/standardVersion_findOne";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { CommentContainer } from "components/containers/CommentContainer/CommentContainer";
import { TextCollapse } from "components/containers/TextCollapse/TextCollapse";
import { GeneratedInputComponent } from "components/inputs/generated";
import { PreviewSwitch } from "components/inputs/PreviewSwitch/PreviewSwitch";
import { BaseStandardInput } from "components/inputs/standards";
import { ObjectActionsRow } from "components/lists/ObjectActionsRow/ObjectActionsRow";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { ResourceListHorizontal } from "components/lists/resource";
import { smallHorizontalScrollbar } from "components/lists/styles";
import { TagList } from "components/lists/TagList/TagList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { ObjectTitle } from "components/text/ObjectTitle/ObjectTitle";
import { VersionDisplay } from "components/text/VersionDisplay/VersionDisplay";
import { useFormik } from "formik";
import { FieldData, FieldDataJSON } from "forms/types";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { ObjectAction } from "utils/actions/objectActions";
import { defaultResourceList } from "utils/defaults/resourceList";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { SessionContext } from "utils/SessionContext";
import { standardVersionToFieldData } from "utils/shape/general";
import { TagShape } from "utils/shape/models/tag";
import { StandardViewProps } from "../types";

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
    display = 'page',
    partialData,
    zIndex = 200,
}: StandardViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const { id, isLoading, object: standardVersion, permissions, setObject: setStandardVersion } = useObjectFromUrl<StandardVersion, FindVersionInput>({
        query: standardVersionFindOne,
        partialData,
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
        setLocation,
        setObject: setStandardVersion,
    });

    const [isPreviewOn, setIsPreviewOn] = useState<boolean>(true);
    const onPreviewChange = useCallback((isOn: boolean) => { setIsPreviewOn(isOn); }, []);

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
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: 'Standard',
                }}
            />
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
                    languages={availableLanguages}
                    loading={isLoading}
                    title={name}
                    setLanguage={setLanguage}
                    zIndex={zIndex}
                />
                {/* Relationships */}
                <RelationshipList
                    isEditing={false}
                    objectType={'Routine'}
                    zIndex={zIndex}
                />
                {/* Resources */}
                {Array.isArray(resourceList.resources) && resourceList.resources.length > 0 && <ResourceListHorizontal
                    title={'Resources'}
                    list={resourceList}
                    canUpdate={false}
                    handleUpdate={() => { }} // Intentionally blank
                    loading={isLoading}
                    zIndex={zIndex}
                />}
                {/* Box with description */}
                <Box sx={containerProps(palette)}>
                    <TextCollapse title="Description" text={description} loading={isLoading} loadingLines={2} />
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
            /> */}
                {/* Action buttons */}
                <ObjectActionsRow
                    actionData={actionData}
                    exclude={[ObjectAction.Edit, ObjectAction.VoteDown, ObjectAction.VoteUp]} // Handled elsewhere
                    object={standardVersion}
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
                        zIndex={zIndex}
                    />
                </Box>
            </Box>
        </>
    )
}