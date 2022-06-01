import { Box, CircularProgress, IconButton, LinearProgress, Stack, Tooltip, Typography, useTheme } from "@mui/material"
import { useLocation, useRoute } from "wouter";
import { APP_LINKS, InputType } from "@local/shared";
import { useLazyQuery } from "@apollo/client";
import { standard, standardVariables } from "graphql/generated/standard";
import { standardQuery } from "graphql/query";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import {
    Edit as EditIcon,
    MoreHoriz as EllipsisIcon,
} from "@mui/icons-material";
import { BaseObjectActionDialog, BaseStandardInput, LinkButton, ResourceListHorizontal, SelectLanguageDialog, StarButton } from "components";
import { StandardViewProps } from "../types";
import { getCreatedByString, getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages, ObjectType, standardToFieldData, TERTIARY_COLOR, toCreatedBy } from "utils";
import { Standard } from "types";
import { StarFor } from "graphql/generated/globalTypes";
import { BaseObjectAction } from "components/dialogs/types";
import { containerShadow } from "styles";
import { validate as uuidValidate } from 'uuid';
import { owns } from "utils/authentication";
import { FieldData } from "forms/types";
import { useFormik } from "formik";
import { generateInputComponent } from "forms/generators";
import { PreviewSwitch } from "components/inputs";

export const StandardView = ({
    partialData,
    session,
}: StandardViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Standard}/:id`);
    const [, params2] = useRoute(`${APP_LINKS.SearchStandards}/view/:id`);
    const id = params?.id ?? params2?.id;
    // Fetch data
    const [getData, { data, loading }] = useLazyQuery<standard, standardVariables>(standardQuery);
    useEffect(() => {
        if (id && uuidValidate(id)) getData({ variables: { input: { id } } });
    }, [getData, id])

    const standard = useMemo(() => data?.standard, [data]);
    const [changedStandard, setChangedStandard] = useState<Standard | null>(null); // Standard may change if it is starred/upvoted/etc.
    const canEdit = useMemo<boolean>(() => owns(standard?.role), [standard]);

    const schema = useMemo<FieldData | null>(() => standardToFieldData(standard, 'preview'), [standard]);
    const previewFormik = useFormik({
        initialValues: {
            preview: schema?.props?.defaultValue,
        },
        enableReinitialize: true,
        onSubmit: () => { },
    });

    useEffect(() => {
        if (standard) { setChangedStandard(standard) }
    }, [standard]);

    const [language, setLanguage] = useState<string>('');
    const [availableLanguages, setAvailableLanguages] = useState<string[]>([]);
    useEffect(() => {
        const availableLanguages = standard?.translations?.map(t => getLanguageSubtag(t.language)) ?? [];
        const userLanguages = getUserLanguages(session);
        setAvailableLanguages(availableLanguages);
        setLanguage(getPreferredLanguage(availableLanguages, userLanguages));
    }, [standard, session]);
    const { description, name } = useMemo(() => {
        return {
            description: getTranslation(standard, 'description', [language]) ?? getTranslation(partialData, 'description', [language]),
            name: standard?.name ?? partialData?.name,
        };
    }, [standard, language, partialData]);

    useEffect(() => {
        document.title = `${name} | Vrooli`;
    }, [name]);

    const createdBy = useMemo<string | null>(() => getCreatedByString(standard, [language]), [standard, language]);
    const toCreator = useCallback(() => { toCreatedBy(standard, setLocation) }, [standard, setLocation]);

    const onEdit = useCallback(() => {
        // Depends on if we're in a search popup or a normal organization page
        setLocation(Boolean(params?.id) ? `${APP_LINKS.Standard}/edit/${id}` : `${APP_LINKS.SearchStandards}/edit/${id}`);
    }, [setLocation, params?.id, id]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    // Determine options available to object, in order
    const moreOptions: BaseObjectAction[] = useMemo(() => {
        // Initialize
        let options: BaseObjectAction[] = [];
        if (canEdit) {
            options.push(BaseObjectAction.Edit);
        }
        options.push(BaseObjectAction.Stats);
        if (session && !canEdit) {
            options.push(standard?.isUpvoted ? BaseObjectAction.Downvote : BaseObjectAction.Upvote);
            options.push(standard?.isStarred ? BaseObjectAction.Unstar : BaseObjectAction.Star);
            options.push(BaseObjectAction.Fork);
        }
        options.push(BaseObjectAction.Donate, BaseObjectAction.Share)
        if (session?.id) {
            options.push(BaseObjectAction.Report);
        }
        if (canEdit) {
            options.push(BaseObjectAction.Delete);
        }
        return options;
    }, [standard, canEdit, session]);

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
        />
    }, [loading, session, standard]);

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
                    <Box sx={{
                        padding: 1,
                        borderRadius: 1,
                        color: Boolean(description) ? palette.background.textPrimary : palette.background.textSecondary,
                    }}>
                        <Typography variant="h6" sx={{ color: palette.background.textPrimary }}>Description</Typography>
                        <Typography variant="body1">{description ?? 'No description set'}</Typography>
                    </Box>
                    {/* Build/Preview switch */}
                    <PreviewSwitch
                        isPreviewOn={isPreviewOn}
                        onChange={onPreviewChange}
                    />
                    {
                        isPreviewOn ?
                            schema ? generateInputComponent({
                                data: schema,
                                disabled: true,
                                formik: previewFormik,
                                session,
                                onUpload: () => { }
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
    }, [loading, resourceList, description, palette.background.textPrimary, palette.background.textSecondary, isPreviewOn, onPreviewChange, schema, previewFormik, session]);

    return (
        <Box sx={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            margin: 'auto',
            // xs: 100vh - navbar (64px) - bottom nav (56px)
            // md: 100vh - navbar (80px)
            minHeight: { xs: 'calc(100vh - 64px - 56px)', md: 'calc(100vh - 80px)' },
        }}>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <BaseObjectActionDialog
                handleActionComplete={() => { }} //TODO
                handleDelete={() => { }} //TODO
                handleEdit={onEdit}
                objectId={id ?? ''}
                objectType={ObjectType.Standard}
                anchorEl={moreMenuAnchor}
                title='Standard Options'
                availableOptions={moreOptions}
                onClose={closeMoreMenu}
                session={session}
            />
            {/* Main container */}
            <Box sx={{
                background: palette.background.paper,
                overflowY: 'auto',
                width: 'min(100%, 600px)',
                borderRadius: { xs: '8px 8px 0 0', sm: '8px' },
                overflow: 'overlay',
                boxShadow: { xs: 'none', sm: (containerShadow as any).boxShadow },
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
                                <EllipsisIcon sx={{ fill: palette.primary.contrastText }} />
                            </IconButton>
                        </Tooltip>
                    </Stack>
                    <Stack direction="row" spacing={1} sx={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                    }}>
                        <StarButton
                            session={session}
                            objectId={standard?.id ?? ''}
                            showStars={false}
                            starFor={StarFor.Standard}
                            isStar={changedStandard?.isStarred ?? false}
                            stars={changedStandard?.stars ?? 0}
                            onChange={(isStar: boolean) => { changedStandard && setChangedStandard({ ...changedStandard, isStarred: isStar }) }}
                            tooltipPlacement="bottom"
                        />
                        {createdBy && (
                            <LinkButton
                                onClick={toCreator}
                                text={createdBy}
                            />
                        )}
                        <Typography variant="body1"> - {standard?.version}</Typography>
                        <SelectLanguageDialog
                            availableLanguages={availableLanguages}
                            canDropdownOpen={availableLanguages.length > 1}
                            currentLanguage={language}
                            handleCurrent={setLanguage}
                            session={session}
                        />
                        {canEdit && <Tooltip title="Edit standard">
                            <IconButton
                                aria-label="Edit standard"
                                size="small"
                                onClick={onEdit}
                            >
                                <EditIcon sx={{ fill: TERTIARY_COLOR }} />
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
            </Box>
        </Box >
    )
}