import { Box, IconButton, LinearProgress, Stack, Tooltip, Typography, useTheme } from "@mui/material"
import { useLocation } from '@shared/route';
import { FindByIdOrHandleInput, ApiVersion, ResourceList, BookmarkFor } from "@shared/consts";
import { useLazyQuery } from "api/hooks";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import { ObjectActionMenu, DateDisplay, ReportsLink, SelectLanguageMenu, BookmarkButton } from "components";
import { ApiViewProps } from "../types";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages, parseSingleItemUrl, placeholderColor, useObjectActions } from "utils";
import { ResourceListVertical } from "components/lists";
import { DonateIcon, EditIcon, EllipsisIcon, ApiIcon } from "@shared/icons";
import { ShareButton } from "components/buttons/ShareButton/ShareButton";
import { apiVersionFindOne } from "api/generated/endpoints/apiVersion";

export const ApiView = ({
    partialData,
    session,
    zIndex,
}: ApiViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const profileColors = useMemo(() => placeholderColor(), []);
    // Fetch data
    const urlData = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data, loading }] = useLazyQuery<ApiVersion, FindByIdOrHandleInput, 'apiVersion'>(apiVersionFindOne, 'apiVersion', { errorPolicy: 'all' });
    const [apiVersion, setApiVersion] = useState<ApiVersion | null | undefined>(null);
    useEffect(() => {
        if (urlData.id || urlData.handle) getData({ variables: urlData })
    }, [getData, urlData]);
    useEffect(() => {
        setApiVersion(data?.apiVersion);
    }, [data]);
    const canUpdate = useMemo<boolean>(() => apiVersion?.you?.canUpdate === true, [apiVersion?.you?.canUpdate]);

    const availableLanguages = useMemo<string[]>(() => (apiVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [apiVersion?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { canBookmark, details, name, resourceList, summary } = useMemo(() => {
        const { canBookmark } = apiVersion?.root?.you ?? {};
        const resourceList: ResourceList | null | undefined = apiVersion?.resourceList;
        const { details, name, summary } = getTranslation(apiVersion ?? partialData, [language]);
        return {
            details,
            canBookmark,
            name,
            resourceList,
            summary,
        };
    }, [language, apiVersion, partialData]);

    useEffect(() => {
        document.title = `${name} | Vrooli`;
    }, [name]);

    const resources = useMemo(() => (resourceList || canUpdate) ? (
        <ResourceListVertical
            list={resourceList as any}
            session={session}
            canUpdate={canUpdate}
            handleUpdate={(updatedList) => {
                if (!apiVersion) return;
                setApiVersion({
                    ...apiVersion,
                    resourceList: updatedList
                })
            }}
            loading={loading}
            mutate={true}
            zIndex={zIndex}
        />
    ) : null, [canUpdate, loading, apiVersion, resourceList, session, zIndex]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    const actionData = useObjectActions({
        object: apiVersion,
        objectType: 'ApiVersion',
        session,
        setLocation,
        setObject: setApiVersion,
    });

    /**
     * Displays name, avatar, summary, and quick links
     */
    const overviewComponent = useMemo(() => (
        <Box
            position="relative"
            ml='auto'
            mr='auto'
            mt={3}
            bgcolor={palette.background.paper}
            sx={{
                borderRadius: { xs: '0', sm: 2 },
                boxShadow: { xs: 'none', sm: 12 },
                width: { xs: '100%', sm: 'min(500px, 100vw)' }
            }}
        >
            <Box
                width={'min(100px, 25vw)'}
                height={'min(100px, 25vw)'}
                borderRadius='100%'
                position='absolute'
                display='flex'
                justifyContent='center'
                alignItems='center'
                left='50%'
                top="-55px"
                sx={{
                    border: `1px solid black`,
                    backgroundColor: profileColors[0],
                    transform: 'translateX(-50%)',
                }}
            >
                <ApiIcon fill={profileColors[1]} width='80%' height='80%' />
            </Box>
            <Tooltip title="See all options">
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
                    <EllipsisIcon fill={palette.background.textSecondary} />
                </IconButton>
            </Tooltip>
            <Stack direction="column" spacing={1} p={1} alignItems="center" justifyContent="center">
                {/* Title */}
                {
                    loading ? (
                        <Stack sx={{ width: '50%', color: 'grey.500', paddingTop: 2, paddingBottom: 2 }} spacing={2}>
                            <LinearProgress color="inherit" />
                        </Stack>
                    ) : canUpdate ? (
                        <Stack direction="row" alignItems="center" justifyContent="center">
                            <Typography variant="h4" textAlign="center">{name}</Typography>
                            <Tooltip title="Edit apiVersion">
                                <IconButton
                                    aria-label="Edit apiVersion"
                                    size="small"
                                    onClick={() => actionData.onActionStart('Edit')}
                                >
                                    <EditIcon fill={palette.secondary.main} />
                                </IconButton>
                            </Tooltip>
                        </Stack>
                    ) : (
                        <Typography variant="h4" textAlign="center">{name}</Typography>
                    )
                }
                {/* Joined date */}
                <DateDisplay
                    loading={loading}
                    showIcon={true}
                    textBeforeDate="Joined"
                    timestamp={apiVersion?.created_at}
                    width={"33%"}
                />
                {/* Bio */}
                {
                    loading ? (
                        <Stack sx={{ width: '85%', color: 'grey.500' }} spacing={2}>
                            <LinearProgress color="inherit" />
                            <LinearProgress color="inherit" />
                        </Stack>
                    ) : (
                        <Typography variant="body1" sx={{ color: Boolean(summary) ? palette.background.textPrimary : palette.background.textSecondary }}>{summary ?? 'No summary set'}</Typography>
                    )
                }
                <Stack direction="row" spacing={2} alignItems="center">
                    <Tooltip title="Donate">
                        <IconButton aria-label="Donate" size="small" onClick={() => { }}>
                            <DonateIcon fill={palette.background.textSecondary} />
                        </IconButton>
                    </Tooltip>
                    <ShareButton object={apiVersion} zIndex={zIndex} />
                    <ReportsLink object={apiVersion} />
                    <BookmarkButton
                        disabled={!canBookmark}
                        session={session}
                        objectId={apiVersion?.id ?? ''}
                        bookmarkFor={BookmarkFor.Api}
                        isBookmarked={apiVersion?.root?.you?.isBookmarked ?? false}
                        bookmarks={apiVersion?.root?.bookmarks ?? 0}
                        onChange={(isBookmarked: boolean) => { }}
                    />
                </Stack>
            </Stack>
        </Box >
    ), [palette.background.paper, palette.background.textSecondary, palette.background.textPrimary, palette.secondary.main, profileColors, openMoreMenu, loading, canUpdate, name, apiVersion, summary, zIndex, canBookmark, session, actionData]);

    return (
        <>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={moreMenuAnchor}
                object={apiVersion as any}
                onClose={closeMoreMenu}
                session={session}
                zIndex={zIndex + 1}
            />
            <Box sx={{
                background: palette.mode === 'light' ? "#b2b3b3" : "#303030",
                display: 'flex',
                paddingTop: 5,
                paddingBottom: { xs: 0, sm: 2, md: 5 },
                position: "relative",
            }}>
                {/* Language display/select */}
                <Box sx={{
                    position: 'absolute',
                    top: 8,
                    right: 8,
                }}>
                    <SelectLanguageMenu
                        currentLanguage={language}
                        handleCurrent={setLanguage}
                        session={session}
                        translations={apiVersion?.translations ?? partialData?.translations ?? []}
                        zIndex={zIndex}
                    />
                </Box>
                {overviewComponent}
            </Box>
        </>
    )
}