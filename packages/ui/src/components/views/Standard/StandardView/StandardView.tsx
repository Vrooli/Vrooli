import { Box, CircularProgress, IconButton, LinearProgress, Stack, Tooltip, Typography, useTheme } from "@mui/material"
import { useLocation } from '@shared/route';
import { APP_LINKS, InputType } from "@shared/consts";
import { useLazyQuery } from "@apollo/client";
import { standard, standardVariables } from "graphql/generated/standard";
import { standardQuery } from "graphql/query";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import { ObjectActionMenu, BaseStandardInput, CommentContainer, ResourceListHorizontal, SelectLanguageMenu, StarButton, TextCollapse, OwnerLabel, VersionDisplay, SnackSeverity } from "components";
import { StandardViewProps } from "../types";
import { base36ToUuid, firstString, getLanguageSubtag, getLastUrlPart, getObjectSlug, getPreferredLanguage, getTranslation, getUserLanguages, ObjectType, openObject, PubSub, standardToFieldData } from "utils";
import { Standard } from "types";
import { CommentFor, StarFor } from "graphql/generated/globalTypes";
import { uuidValidate } from '@shared/uuid';
import { FieldData, FieldDataJSON } from "forms/types";
import { useFormik } from "formik";
import { generateInputComponent } from "forms/generators";
import { PreviewSwitch } from "components/inputs";
import { EditIcon, EllipsisIcon } from "@shared/icons";
import { ObjectAction, ObjectActionComplete } from "components/dialogs/types";

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
    const [getData, { data, loading }] = useLazyQuery<standard, standardVariables>(standardQuery, { errorPolicy: 'all' });
    useEffect(() => {
        console.log('valid id check', id, versionGroupId);
        if (uuidValidate(id) || uuidValidate(versionGroupId)) getData({ variables: { input: { id, versionGroupId } } });
        else PubSub.get().publishSnack({ message: 'Could not parse ID in URL', severity: SnackSeverity.Error });
    }, [getData, id, versionGroupId])

    const [standard, setStandard] = useState<Standard | null>(null);
    useEffect(() => {
        if (!data) return;
        setStandard(data.standard);
    }, [data]);

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

    const { canEdit, canStar, description, name } = useMemo(() => {
        const { canEdit, canStar } = standard?.permissionsStandard ?? {};
        const { description } = getTranslation(standard ?? partialData, [language]);
        const name = firstString(standard?.name, partialData?.name);
        return { canEdit, canStar, description, name };
    }, [standard, language, partialData]);

    useEffect(() => {
        document.title = `${name} | Vrooli`;
    }, [name]);

    const onEdit = useCallback(() => {
        if (!standard) return;
        setLocation(`${APP_LINKS.Standard}/edit/${getObjectSlug(standard)}`);
    }, [setLocation, standard]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    const onMoreActionStart = useCallback((action: ObjectAction) => {
        switch (action) {
            case ObjectAction.Edit:
                onEdit();
                break;
            case ObjectAction.Stats:
                //TODO
                break;
        }
    }, [onEdit]);

    const onMoreActionComplete = useCallback((action: ObjectActionComplete, data: any) => {
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
            case ObjectActionComplete.Copy:
                openObject(data.standard, setLocation);
                window.location.reload();
                break;
        }
    }, [standard, setLocation]);

    const [isPreviewOn, setIsPreviewOn] = useState<boolean>(true);
    const onPreviewChange = useCallback((isOn: boolean) => { setIsPreviewOn(isOn); }, []);

    const resourceList = useMemo(() => {
        if (!standard ||
            !Array.isArray(standard.resourceLists) ||
            standard.resourceLists.length < 1 ||
            standard.resourceLists[0].resources.length < 1) return null;
        return <ResourceListHorizontal
            title={'Resources'}
            list={(standard as any).resourceLists[0]}
            canEdit={false}
            handleUpdate={() => { }} // Intentionally blank
            loading={loading}
            session={session}
            zIndex={zIndex}
        />
    }, [loading, session, standard, zIndex]);

    /**
     * Display body or loading indicator
     */
    const body = useMemo(() => {
        if (loading) return (
            <Box sx={{
                minHeight: 'min(300px, 25vh)',
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
            }}>
                <CircularProgress color="secondary" />
            </Box>
        )
        return (
            <>
                {/* Stack that shows standard info, such as description */}
                <Stack direction="column" spacing={2} padding={1}>
                    {/* Resources */}
                    {resourceList}
                    {/* Description */}
                    <TextCollapse title="Description" text={description} />
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
            </>
        )
    }, [loading, resourceList, description, isPreviewOn, onPreviewChange, schema, previewFormik, session, zIndex]);

    return (
        <Box sx={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            margin: 'auto',
            // xs: 100vh - navbar (64px) - bottom nav (56px) - iOS nav bar
            // md: 100vh - navbar (80px)
            minHeight: { xs: 'calc(100vh - 64px - 56px - env(safe-area-inset-bottom))', md: 'calc(100vh - 80px)' },
        }}>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                anchorEl={moreMenuAnchor}
                object={standard as any}
                onActionStart={onMoreActionStart}
                onActionComplete={onMoreActionComplete}
                onClose={closeMoreMenu}
                session={session}
                title='Standard Options'
                zIndex={zIndex + 1}
            />
            <Stack direction="column" spacing={5} sx={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                margin: 'auto',
                // xs: 100vh - navbar (64px) - bottom nav (56px) - iOS nav bar
                // md: 100vh - navbar (80px)
                minHeight: { xs: 'calc(100vh - 64px - 56px - env(safe-area-inset-bottom))', md: 'calc(100vh - 80px)' },
                marginBottom: 8,
                width: '100%',
            }}>
                {/* Main container */}
                <Box sx={{
                    background: palette.background.paper,
                    overflowY: 'auto',
                    borderRadius: { xs: '8px 8px 0 0', sm: '8px' },
                    overflow: 'overlay',
                    boxShadow: { xs: 'none', sm: 12 },
                    width: 'min(100%, 600px)',
                }}>
                    {/* Heading container */}
                    <Stack direction="column" spacing={1} sx={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        padding: 2,
                        marginBottom: 1,
                        background: palette.primary.main,
                        color: palette.primary.contrastText,
                    }}>
                        {/* Show star button and ellipsis next to title */}
                        <Stack direction="row" spacing={1} alignItems="center">
                            {loading ?
                                <LinearProgress color="inherit" sx={{
                                    borderRadius: 1,
                                    width: '50vw',
                                    height: 8,
                                    marginTop: '12px !important',
                                    marginBottom: '12px !important',
                                    maxWidth: '300px',
                                }} /> :
                                <Typography variant="h5" sx={{ textAlign: 'center' }}>{name}</Typography>}

                            <Tooltip title="More options">
                                <IconButton
                                    aria-label="More"
                                    size="small"
                                    onClick={openMoreMenu}
                                    sx={{
                                        display: 'block',
                                        marginLeft: 'auto',
                                        marginRight: 1,
                                    }}
                                >
                                    <EllipsisIcon fill={palette.primary.contrastText} />
                                </IconButton>
                            </Tooltip>
                        </Stack>
                        <Stack direction="row" spacing={1} sx={{
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                        }}>
                            <StarButton
                                disabled={!canStar}
                                session={session}
                                objectId={standard?.id ?? ''}
                                showStars={false}
                                starFor={StarFor.Standard}
                                isStar={standard?.isStarred ?? false}
                                stars={standard?.stars ?? 0}
                                onChange={(isStar: boolean) => { standard && setStandard({ ...standard, isStarred: isStar }) }}
                                tooltipPlacement="bottom"
                            />
                            <OwnerLabel objectType={ObjectType.Standard} owner={standard?.creator} session={session} />
                            <VersionDisplay
                                currentVersion={standard?.version}
                                prefix={" - "}
                                versions={standard?.versions}
                            />
                            <SelectLanguageMenu
                                currentLanguage={language}
                                handleCurrent={setLanguage}
                                session={session}
                                translations={standard?.translations ?? partialData?.translations ?? []}
                                zIndex={zIndex}
                            />
                            {canEdit && <Tooltip title="Edit standard">
                                <IconButton
                                    aria-label="Edit standard"
                                    size="small"
                                    onClick={onEdit}
                                >
                                    <EditIcon fill={palette.secondary.main} />
                                </IconButton>
                            </Tooltip>}
                        </Stack>
                    </Stack>
                    {/* Body container */}
                    <Box sx={{
                        padding: 2,
                    }}>
                        {body}
                    </Box>
                    {/* Comments Container */}
                    <CommentContainer
                        language={language}
                        objectId={id ?? ''}
                        objectType={CommentFor.Standard}
                        session={session}
                        sxs={{
                            root: { width: 'min(100%, 600px)' }
                        }}
                        zIndex={zIndex}
                    />
                </Box>
            </Stack>
        </Box >
    )
}