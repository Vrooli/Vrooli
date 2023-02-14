import { Box, IconButton, LinearProgress, Link, Stack, Tooltip, Typography, useTheme } from "@mui/material"
import { useLocation } from '@shared/route';
import { APP_LINKS, FindByIdOrHandleInput, SmartContractVersion, BookmarkFor } from "@shared/consts";
import { useLazyQuery } from "api/hooks";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import { ObjectActionMenu, DateDisplay, ReportsLink, SelectLanguageMenu, StarButton } from "components";
import { SmartContractViewProps } from "../types";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages, ObjectAction, ObjectActionComplete, openObject, parseSingleItemUrl, placeholderColor, uuidToBase36 } from "utils";
import { uuidValidate } from '@shared/uuid';
import { DonateIcon, EditIcon, EllipsisIcon, SmartContractIcon } from "@shared/icons";
import { ShareButton } from "components/buttons/ShareButton/ShareButton";
import { setDotNotationValue } from "@shared/utils";
import { smartContractVersionFindOne } from "api/generated/endpoints/smartContractVersion";

export const SmartContractView = ({
    partialData,
    session,
    zIndex,
}: SmartContractViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const profileColors = useMemo(() => placeholderColor(), []);
    // Fetch data
    const urlData = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data, loading }] = useLazyQuery<SmartContractVersion, FindByIdOrHandleInput, 'smartContractVersion'>(smartContractVersionFindOne, 'smartContractVersion', { errorPolicy: 'all' });
    const [smartContractVersion, setSmartContractVersion] = useState<SmartContractVersion | null | undefined>(null);
    useEffect(() => {
        if (urlData.id || urlData.handle) getData({ variables: urlData })
    }, [getData, urlData]);
    useEffect(() => {
        setSmartContractVersion(data?.smartContractVersion);
    }, [data]);
    const canUpdate = useMemo<boolean>(() => smartContractVersion?.you?.canUpdate === true, [smartContractVersion?.you?.canUpdate]);

    const availableLanguages = useMemo<string[]>(() => (smartContractVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [smartContractVersion?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { description, canStar, name } = useMemo(() => {
        const { canStar } = smartContractVersion?.root?.you ?? {};
        const { description, name } = getTranslation(smartContractVersion ?? partialData, [language]);
        return {
            description: description && description.trim().length > 0 ? description : undefined,
            canStar,
            name,
        };
    }, [language, smartContractVersion, partialData]);

    useEffect(() => {
        document.title = `${name} | Vrooli`;
    }, [name]);

    const onEdit = useCallback(() => {
        setLocation(`${APP_LINKS.SmartContract}/edit/${uuidToBase36(smartContractVersion?.id ?? '')}`);
    }, [smartContractVersion?.id, setLocation]);

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
            case ObjectActionComplete.Star:
            case ObjectActionComplete.StarUndo:
                if (data.success && smartContractVersion) {
                    setSmartContractVersion(setDotNotationValue(smartContractVersion, 'root.you.isStarred', action === ObjectActionComplete.Star))
                }
                break;
            case ObjectActionComplete.Fork:
                openObject(data.smartContractVersion, setLocation);
                window.location.reload();
                break;
        }
    }, [smartContractVersion, setLocation]);

    /**
     * Displays name, avatar, description, and quick links
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
                <SmartContractIcon fill={profileColors[1]} width='80%' height='80%' />
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
                            <Tooltip title="Edit smartContractVersion">
                                <IconButton
                                    aria-label="Edit smartContractVersion"
                                    size="small"
                                    onClick={onEdit}
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
                    timestamp={smartContractVersion?.created_at}
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
                        <Typography variant="body1" sx={{ color: Boolean(description) ? palette.background.textPrimary : palette.background.textSecondary }}>{description ?? 'No description set'}</Typography>
                    )
                }
                <Stack direction="row" spacing={2} alignItems="center">
                    <Tooltip title="Donate">
                        <IconButton aria-label="Donate" size="small" onClick={() => { }}>
                            <DonateIcon fill={palette.background.textSecondary} />
                        </IconButton>
                    </Tooltip>
                    <ShareButton object={smartContractVersion} zIndex={zIndex} />
                    <ReportsLink object={smartContractVersion} />
                    <StarButton
                        disabled={!canStar}
                        session={session}
                        objectId={smartContractVersion?.id ?? ''}
                        starFor={BookmarkFor.SmartContract}
                        isStar={smartContractVersion?.root?.you?.isStarred ?? false}
                        stars={smartContractVersion?.root?.stars ?? 0}
                        onChange={(isStar: boolean) => { }}
                        tooltipPlacement="bottom"
                    />
                </Stack>
            </Stack>
        </Box >
    ), [palette.background.paper, palette.background.textSecondary, palette.background.textPrimary, palette.secondary.main, profileColors, openMoreMenu, loading, canUpdate, name, onEdit, smartContractVersion, description, zIndex, canStar, session]);

    return (
        <>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                anchorEl={moreMenuAnchor}
                object={smartContractVersion as any}
                onActionStart={onMoreActionStart}
                onActionComplete={onMoreActionComplete}
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
                        translations={smartContractVersion?.translations ?? partialData?.translations ?? []}
                        zIndex={zIndex}
                    />
                </Box>
                {overviewComponent}
            </Box>
            {/* TODO */}
        </>
    )
}