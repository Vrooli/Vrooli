import { Box, IconButton, Stack, Tab, Tabs, Tooltip, Typography } from "@mui/material"
import { useLocation, useRoute } from "wouter";
import { APP_LINKS, StarFor, VoteFor } from "@local/shared";
import { useMutation, useQuery } from "@apollo/client";
import { project } from "graphql/generated/project";
import { routinesQuery, standardsQuery, projectQuery } from "graphql/query";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import {
    CardGiftcard as DonateIcon,
    DeleteForever as DeleteForeverIcon,
    Edit as EditIcon,
    MoreHoriz as EllipsisIcon,
    Person as PersonIcon,
    ReportProblem as ReportIcon,
    Share as ShareIcon,
    Star as StarFilledIcon,
    StarOutline as StarOutlineIcon,
    SvgIconComponent,
    ThumbUp as UpvoteIcon,
    ThumbDown as DownvoteIcon,
} from "@mui/icons-material";
import { ListMenu, routineDefaultSortOption, RoutineListItem, routineOptionLabel, RoutineSortOptions, SearchList, standardDefaultSortOption, StandardListItem, standardOptionLabel, StandardSortOptions, StarButton } from "components";
import { containerShadow } from "styles";
import { ProjectViewProps } from "../types";
import { LabelledSortOption } from "utils";
import { Routine, Standard } from "types";
import { voteMutation } from "graphql/mutation";
import { vote } from "graphql/generated/vote";

const tabLabels = ['Resources', 'Routines', 'Standards'];

// All available actions
enum Actions {
    Delete = "Delete",
    Donate = "Donate",
    Downvote = "Downvote",
    Report = "Report",
    Share = "Share",
    Star = "Star",
    Unstar = "Unstar",
    Upvote = "Upvote",
}
const allOptionsMap: { [x: string]: [string, SvgIconComponent, string] } = ({
    [Actions.Upvote]: ["Upvote", UpvoteIcon, "#34c38b"],
    [Actions.Downvote]: ["Downvote", DownvoteIcon, "#af2929"],
    [Actions.Star]: ['Favorite', StarOutlineIcon, "#cbae30"],
    [Actions.Unstar]: ['Unfavorite', StarFilledIcon, "#cbae30"],
    [Actions.Donate]: ['Donate', DonateIcon, "default"],
    [Actions.Share]: ['Share', ShareIcon, "default"],
    [Actions.Report]: ['Report', ReportIcon, "default"],
    [Actions.Delete]: ['Delete', DeleteForeverIcon, "default"],
})

export const ProjectView = ({
    session,
    partialData,
}: ProjectViewProps) => {
    const [vote] = useMutation<vote>(voteMutation);
    const [, setLocation] = useLocation();
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Project}/:id`);
    const [, params2] = useRoute(`${APP_LINKS.SearchProjects}/view/:id`);
    const id: string = useMemo(() => params?.id ?? params2?.id ?? '', [params, params2]);
    // Fetch data
    const { data, loading } = useQuery<project>(projectQuery, { variables: { input: { id } } });
    const project = useMemo(() => data?.project, [data]);
    const isOwn: boolean = useMemo(() => true, [project]); // TODO project.isOwn

    // Vote object
    const handleVote = useCallback((e: any, isUpvote: boolean | null) => {
        // Prevent propagation of normal click event
        e.stopPropagation();
        // Send vote mutation
        vote({
            variables: {
                input: {
                    isUpvote,
                    voteFor: VoteFor.Project,
                    forId: project?.id ?? ''
                }
            }
        });
    }, [project?.id, vote]);

    // Determine options available to object, in order
    const moreOptions = useMemo(() => {
        // Initialize
        let options: [string, SvgIconComponent, string][] = [];
        if (project && session && !isOwn) {
            options.push(project.isUpvoted ? allOptionsMap[Actions.Downvote] : allOptionsMap[Actions.Upvote]);
            options.push(project.isStarred ? allOptionsMap[Actions.Unstar] : allOptionsMap[Actions.Star]);
        }
        options.push(allOptionsMap[Actions.Donate], allOptionsMap[Actions.Share])
        if (session?.id) {
            options.push(allOptionsMap[Actions.Report]);
        }
        if (isOwn) {
            options.push(allOptionsMap[Actions.Delete]);
        }
        // Convert options to ListMenuItemData
        return options.map(o => ({
            label: o[0],
            value: o[0],
            Icon: o[1],
            iconColor: o[2],
        }));
    }, [session, project])

    // Handle tabs
    const [tabIndex, setTabIndex] = useState<number>(0);
    const handleTabChange = (event, newValue) => { setTabIndex(newValue) };

    // Create search data
    const [
        placeholder,
        sortOptions,
        defaultSortOption,
        sortOptionLabel,
        searchQuery,
        onSearchSelect,
        searchItemFactory,
    ] = useMemo<[string, LabelledSortOption<any>[], { label: string, value: any }, (o: any) => string, any, (objectData: any) => void, (node: any, index: number) => JSX.Element]>(() => {
        const openLink = (baseLink: string, id: string) => setLocation(`${baseLink}/${id}`);
        // The first tab doesn't have search results, as it is the project's set resources
        switch (tabIndex) {
            case 1:
                return [
                    "Search project's routines...",
                    RoutineSortOptions,
                    routineDefaultSortOption,
                    routineOptionLabel,
                    routinesQuery,
                    (newValue) => openLink(APP_LINKS.Routine, newValue.id),
                    (node: Routine, index: number) => (
                        <RoutineListItem
                            key={`routine-list-item-${index}`}
                            index={index}
                            session={session}
                            data={node}
                            isOwn={false}
                            onClick={(selected: Routine) => openLink(APP_LINKS.Routine, selected.id)}
                        />)
                ];
            case 2:
                return [
                    "Search project's standards...",
                    StandardSortOptions,
                    standardDefaultSortOption,
                    standardOptionLabel,
                    standardsQuery,
                    (newValue) => openLink(APP_LINKS.Standard, newValue.id),
                    (node: Standard, index: number) => (
                        <StandardListItem
                            key={`standard-list-item-${index}`}
                            index={index}
                            session={session}
                            data={node}
                            isOwn={false}
                            onClick={(selected: Standard) => openLink(APP_LINKS.Standard, selected.id)}
                        />)
                ];
            default:
                return [
                    '',
                    [],
                    { label: '', value: null },
                    () => '',
                    null,
                    () => { },
                    () => (<></>)
                ];
        }
    }, [tabIndex]);

    // Handle url search
    const [searchString, setSearchString] = useState<string>('');
    const [sortBy, setSortBy] = useState<string | undefined>(undefined);
    const [timeFrame, setTimeFrame] = useState<string | undefined>(undefined);
    useEffect(() => {
        setSortBy(defaultSortOption.value ?? sortOptions.length > 0 ? sortOptions[0].value : undefined);
    }, [defaultSortOption, sortOptions]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const moreMenuId = useMemo(() => `project-options-menu-${project?.id}`, [project]);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);
    const onMoreMenuSelect = useCallback((value: string) => {
        console.log('onMoreMenuSelect', value);
        switch (value) {
            case Actions.Star:
                break;
            case Actions.Share:
                break;
            case Actions.Donate:
                break;
            case Actions.Report:
                break;
        }
        closeMoreMenu();
    }, [closeMoreMenu]);

    /**
     * Displays name, avatar, bio, and quick links
     */
    const overviewComponent = useMemo(() => (
        <Box
            width={'min(500px, 100vw)'}
            position="relative"
            borderRadius={2}
            ml='auto'
            mr='auto'
            mt={3}
            bgcolor={(t) => t.palette.background.paper}
            sx={{ ...containerShadow }}
        >
            <Box
                width={'min(100px, 25vw)'}
                height={'min(100px, 25vw)'}
                borderRadius='100%'
                border={(t) => `4px solid ${t.palette.primary.dark}`}
                bgcolor='#939eb9'
                position='absolute'
                display='flex'
                justifyContent='center'
                alignItems='center'
                left='50%'
                top="-55px"
                sx={{ transform: 'translateX(-50%)' }}
            >
                <PersonIcon sx={{
                    fill: '#cfd0d1',
                    width: '80%',
                    height: '80%',
                }} />
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
                    <EllipsisIcon />
                </IconButton>
            </Tooltip>
            <Stack direction="column" spacing={1} p={1} alignItems="center" justifyContent="center">
                {
                    isOwn ? (
                        <Stack direction="row" alignItems="center" justifyContent="center">
                            <Typography variant="h4" textAlign="center">{project?.name ?? partialData?.name}</Typography>
                            <Tooltip title="Edit project">
                                <IconButton
                                    aria-label="Edit project"
                                    size="small"
                                    onClick={() => setLocation(`${APP_LINKS.Project}/${id}/edit`)}
                                >
                                    <EditIcon color="primary" />
                                </IconButton>
                            </Tooltip>
                        </Stack>
                    ) : (
                        <Typography variant="h4" textAlign="center">{project?.name ?? partialData?.name}</Typography>

                    )
                }
                <Typography variant="body1" sx={{ color: "#00831e" }}>{project?.created_at ? `🕔 Created ${new Date(project.created_at).toDateString()}` : ''}</Typography>
                <Typography variant="body1" sx={{ color: project?.description ? 'black' : 'gray' }}>{project?.description ?? partialData?.description ?? 'No description set'}</Typography>
                <Stack direction="row" spacing={2} alignItems="center">
                    <Tooltip title="Donate">
                        <IconButton aria-label="Donate" size="small" onClick={() => { }}>
                            <DonateIcon />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title="Share">
                        <IconButton aria-label="Share" size="small" onClick={() => { }}>
                            <ShareIcon />
                        </IconButton>
                    </Tooltip>
                    {
                        !isOwn ? <StarButton
                            session={session}
                            objectId={project?.id ?? ''}
                            starFor={StarFor.Project}
                            isStar={project?.isStarred ?? false}
                            stars={project?.stars ?? 0}
                            onChange={(isStar: boolean) => { }}
                            tooltipPlacement="bottom"
                        /> : null
                    }
                </Stack>
            </Stack>
        </Box>
    ), [project, partialData, isOwn, session]);

    return (
        <>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ListMenu
                id={moreMenuId}
                anchorEl={moreMenuAnchor}
                title='Project Options'
                data={moreOptions}
                onSelect={onMoreMenuSelect}
                onClose={closeMoreMenu}
            />
            <Box sx={{ display: 'flex', paddingTop: 5, paddingBottom: 5, background: "#b2b3b3" }}>
                {overviewComponent}
            </Box>
            {/* View routines and standards associated with this project */}
            <Box>
                <Tabs
                    value={tabIndex}
                    onChange={handleTabChange}
                    indicatorColor="secondary"
                    textColor="inherit"
                    variant="scrollable"
                    scrollButtons="auto"
                    allowScrollButtonsMobile
                    aria-label="site-statistics-tabs"
                    sx={{
                        marginBottom: 1,
                        '& .MuiTabs-flexContainer': {
                            justifyContent: 'space-between',
                        },
                    }}
                >
                    {tabLabels.map((label, index) => (
                        <Tab
                            key={index}
                            id={`profile-tab-${index}`}
                            {...{ 'aria-controls': `profile-tabpanel-${index}` }}
                            label={<span style={{ color: index === 0 ? '#8e6b00' : 'default' }}>{label}</span>}
                        />
                    ))}
                </Tabs>
                <Box p={2}>
                    {
                        tabIndex === 0 ? (
                            <></>
                        ) : (
                            <SearchList
                                searchPlaceholder={placeholder}
                                sortOptions={sortOptions}
                                defaultSortOption={defaultSortOption}
                                query={searchQuery}
                                take={20}
                                searchString={searchString}
                                sortBy={sortBy}
                                timeFrame={timeFrame}
                                setSearchString={setSearchString}
                                setSortBy={setSortBy}
                                setTimeFrame={setTimeFrame}
                                listItemFactory={searchItemFactory}
                                getOptionLabel={sortOptionLabel}
                                onObjectSelect={onSearchSelect}
                            />
                        )
                    }
                </Box>
            </Box>
        </>
    )
}