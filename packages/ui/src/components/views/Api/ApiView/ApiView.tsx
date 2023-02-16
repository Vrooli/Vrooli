import { Box, IconButton, LinearProgress, Stack, Tooltip, Typography, useTheme } from "@mui/material"
import { useLocation } from '@shared/route';
import { ApiVersion, ResourceList, BookmarkFor, FindVersionInput } from "@shared/consts";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import { ObjectActionMenu, DateDisplay, ReportsLink, SelectLanguageMenu, BookmarkButton } from "components";
import { ApiViewProps } from "../types";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages, placeholderColor, useObjectActions, useObjectFromUrl } from "utils";
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

    const { id, isLoading, object: apiVersion, permissions, setObject: setApiVersion } = useObjectFromUrl<ApiVersion, FindVersionInput>({
        query: apiVersionFindOne,
        endpoint: 'apiVersion',
        partialData,
        session,
    });

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

    const resources = useMemo(() => (resourceList || permissions.canUpdate) ? (
        <ResourceListVertical
            list={resourceList as any}
            session={session}
            canUpdate={permissions.canUpdate}
            handleUpdate={(updatedList) => {
                if (!apiVersion) return;
                setApiVersion({
                    ...apiVersion,
                    resourceList: updatedList
                })
            }}
            loading={isLoading}
            mutate={true}
            zIndex={zIndex}
        />
    ) : null, [resourceList, permissions.canUpdate, session, isLoading, zIndex, apiVersion, setApiVersion]);

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
                    isLoading ? (
                        <Stack sx={{ width: '50%', color: 'grey.500', paddingTop: 2, paddingBottom: 2 }} spacing={2}>
                            <LinearProgress color="inherit" />
                        </Stack>
                    ) : permissions.canUpdate ? (
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
                    loading={isLoading}
                    showIcon={true}
                    textBeforeDate="Joined"
                    timestamp={apiVersion?.created_at}
                    width={"33%"}
                />
                {/* Bio */}
                {
                    isLoading ? (
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
    ), [palette.background.paper, palette.background.textSecondary, palette.background.textPrimary, palette.secondary.main, profileColors, openMoreMenu, isLoading, permissions.canUpdate, name, apiVersion, summary, zIndex, canBookmark, session, actionData]);

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