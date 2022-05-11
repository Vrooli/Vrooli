import { Box, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material"
import { useLocation, useRoute } from "wouter";
import { APP_LINKS, MemberRole } from "@local/shared";
import { useLazyQuery } from "@apollo/client";
import { standard, standardVariables } from "graphql/generated/standard";
import { standardQuery } from "graphql/query";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import {
    CardGiftcard as DonateIcon,
    MoreHoriz as EllipsisIcon,
    Person as PersonIcon,
    Share as ShareIcon,
    StarOutline as StarOutlineIcon,
} from "@mui/icons-material";
import { containerShadow } from "styles";
import { StandardViewProps } from "../types";
import { BaseObjectActionDialog, SelectLanguageDialog } from "components";
import { BaseObjectAction } from "components/dialogs/types";
import { getCreatedByString, getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils";
import { validate as uuidValidate } from 'uuid';
import { Standard } from "types";

export const StandardView = ({
    partialData,
    session,
}: StandardViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Standard}/:id`);
    const [, params2] = useRoute(`${APP_LINKS.SearchStandards}/view/:id`);
    const id: string = params?.id ?? params2?.id ?? '';
    // Fetch data
    // Fetch data
    const [getData, { data, loading }] = useLazyQuery<standard, standardVariables>(standardQuery);
    const [standard, setStandard] = useState<Standard | null | undefined>(null);
    useEffect(() => {
        if (uuidValidate(id)) getData({ variables: { input: { id } } })
    }, [getData, id]);
    useEffect(() => {
        setStandard(data?.standard);
    }, [data]);
    const canEdit = useMemo<boolean>(() => standard?.role ? [MemberRole.Admin, MemberRole.Owner].includes(standard.role) : false, [standard]);

    useEffect(() => {
        const name = standard?.name ?? partialData?.name ?? '';
        document.title = `${name} | Vrooli`;
    }, [standard, partialData]);

    const [language, setLanguage] = useState<string>('');
    const [availableLanguages, setAvailableLanguages] = useState<string[]>([]);
    useEffect(() => {
        const availableLanguages = standard?.translations?.map(t => getLanguageSubtag(t.language)) ?? [];
        const userLanguages = getUserLanguages(session);
        setAvailableLanguages(availableLanguages);
        setLanguage(getPreferredLanguage(availableLanguages, userLanguages));
    }, [session, standard]);

    const { contributedBy, description, name } = useMemo(() => {
        return {
            contributedBy: getCreatedByString(standard, [language]),
            description: getTranslation(standard, 'description', [language]) ?? getTranslation(partialData, 'description', [language]),
            name: standard?.name ?? partialData?.name,
        };
    }, [language, partialData, standard]);

    const onEdit = useCallback(() => {
        // Depends on if we're in a search popup or a normal organization page
        setLocation(Boolean(params?.id) ? `${APP_LINKS.Standard}/edit/${id}` : `${APP_LINKS.SearchStandards}/edit/${id}`);
    }, [id, params?.id, setLocation]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    /**
     * Displays name, avatar, bio, and quick links
     */
    const overviewComponent = useMemo(() => (
        <Box
            width={'min(500px, 100vw)'}
            borderRadius={2}
            ml='auto'
            mr='auto'
            mt={8}
            bgcolor={palette.background.paper}
            sx={{ ...containerShadow }}
        >
            <Box
                width={'min(100px, 25vw)'}
                height={'min(100px, 25vw)'}
                borderRadius='100%'
                border={`4px solid ${palette.primary.dark}`}
                bgcolor='#939eb9'
                position='absolute'
                display='flex'
                justifyContent='center'
                alignItems='center'
                left='50%'
                top='max(-50px, -12vw)'
                sx={{ transform: 'translateX(-50%)' }}
            >
                <PersonIcon sx={{
                    fill: '#cfd0d1',
                    width: '80%',
                    height: '80%',
                }} />
            </Box>
            <Stack direction="row" padding={1}>
                <Tooltip title="Favorite organization">
                    <IconButton aria-label="Favorite" size="small">
                        <StarOutlineIcon onClick={() => { }} />
                    </IconButton>
                </Tooltip>
                <Tooltip title="See all options">
                    <IconButton aria-label="More" size="small" edge="end">
                        <EllipsisIcon onClick={openMoreMenu} />
                    </IconButton>
                </Tooltip>
            </Stack>
            <Stack direction="column" spacing={1} mt={5}>
                <Typography variant="h4" textAlign="center">{name}</Typography>
                <Typography variant="h4" textAlign="center">Submitted by: {contributedBy}</Typography>
                <Typography variant="body1" sx={{ color: description ? 'black' : 'gray' }}>{description ?? 'No description set'}</Typography>
                <Stack direction="row" spacing={4} alignItems="center">
                    <Tooltip title="Donate">
                        <IconButton aria-label="Donate" size="small">
                            <DonateIcon onClick={() => { }} />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title="Share">
                        <IconButton aria-label="Share" size="small">
                            <ShareIcon onClick={() => { }} />
                        </IconButton>
                    </Tooltip>
                </Stack>
            </Stack>
        </Box>
    ), [contributedBy, description, name, openMoreMenu, palette.background.paper, palette.primary.dark])

    // Determine options available to object, in order
    const moreOptions: BaseObjectAction[] = useMemo(() => {
        // Initialize
        let options: BaseObjectAction[] = [];
        if (session && !canEdit) {
            options.push(standard?.isUpvoted ? BaseObjectAction.Downvote : BaseObjectAction.Upvote);
            options.push(standard?.isStarred ? BaseObjectAction.Unstar : BaseObjectAction.Star);
            options.push(BaseObjectAction.Copy);
        }
        options.push(BaseObjectAction.Donate, BaseObjectAction.Share)
        if (session?.id) {
            options.push(BaseObjectAction.Report);
        }
        if (canEdit) {
            options.push(BaseObjectAction.Delete);
        }
        return options;
    }, [standard, session, canEdit]);

    return (
        <>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <BaseObjectActionDialog
                handleActionComplete={() => { }} //TODO
                handleDelete={() => { }} //TODO
                handleEdit={onEdit}
                objectId={id}
                objectType={'Standard'}
                anchorEl={moreMenuAnchor}
                title='Standard Options'
                availableOptions={moreOptions}
                onClose={closeMoreMenu}
                session={session}
            />
            <Box sx={{
                display: 'flex',
                paddingTop: 5,
                paddingBottom: 5,
                background: "#b2b3b3",
                position: "relative",
            }}>
                {/* Language display/select */}
                <Box sx={{
                    position: 'absolute',
                    top: 8,
                    right: 8,
                }}>
                    <SelectLanguageDialog
                        availableLanguages={availableLanguages}
                        canDropdownOpen={availableLanguages.length > 1}
                        handleSelect={setLanguage}
                        language={language}
                        session={session}
                    />
                </Box>
                {overviewComponent}
            </Box>
        </>
    )
}