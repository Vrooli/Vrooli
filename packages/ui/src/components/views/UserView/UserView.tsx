import { Box, IconButton, Stack, Tab, Tabs, Tooltip, Typography } from "@mui/material"
import { useLocation, useRoute } from "wouter";
import { APP_LINKS, StarFor } from "@local/shared";
import { useMutation, useQuery } from "@apollo/client";
import { user } from "graphql/generated/user";
import { organizationsQuery, projectsQuery, routinesQuery, standardsQuery, userQuery } from "graphql/query";
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
    SvgIconComponent
} from "@mui/icons-material";
import { BaseObjectActionDialog, ListMenu, organizationDefaultSortOption, OrganizationListItem, organizationOptionLabel, OrganizationSortOptions, projectDefaultSortOption, ProjectListItem, projectOptionLabel, ProjectSortOptions, routineDefaultSortOption, RoutineListItem, routineOptionLabel, RoutineSortOptions, SearchList, standardDefaultSortOption, StandardListItem, standardOptionLabel, StandardSortOptions, StarButton } from "components";
import { containerShadow } from "styles";
import { UserViewProps } from "../types";
import { LabelledSortOption } from "utils";
import { Organization, Project, Routine, Standard } from "types";
import { starMutation } from "graphql/mutation";
import { star } from 'graphql/generated/star';
import { BaseObjectAction } from "components/dialogs/types";

const tabLabels = ['Resources', 'Organizations', 'Projects', 'Routines', 'Standards'];

export const UserView = ({
    session,
    partialData,
}: UserViewProps) => {
    const [star] = useMutation<star>(starMutation);
    const [, setLocation] = useLocation();
    // Get URL params
    const [isProfile, params] = useRoute(`${APP_LINKS.Profile}`);
    const [, params2] = useRoute(`${APP_LINKS.SearchUsers}/view/:id`);
    const id: string = useMemo(() => {
        if (isProfile) return session?.id ?? '';
        return params2?.id ?? ';'
    }, [params, params2]);
    const isOwn: boolean = useMemo(() => Boolean(session?.id && session.id === id), [id, session]);
    // Fetch data
    const { data, loading } = useQuery<user>(userQuery, { variables: { input: { id } } });
    const user = useMemo(() => data?.user, [data]);

    // Star object
    const handleStar = useCallback((e: any, isStar: boolean) => {
        // Prevent propagation of normal click event
        e.stopPropagation();
        console.log('going to star', user, isStar);
        // Send star mutation
        star({
            variables: {
                input: {
                    isStar,
                    starFor: StarFor.User,
                    forId: user?.id ?? ''
                }
            }
        });
    }, [user?.id, star]);

    // Determine options available to object, in order
    const moreOptions: BaseObjectAction[] = useMemo(() => {
        // Initialize
        let options: BaseObjectAction[] = [];
        if (user && session && !isOwn) {
            options.push(user.isStarred ? BaseObjectAction.Unstar : BaseObjectAction.Star);
        }
        options.push(BaseObjectAction.Donate, BaseObjectAction.Share)
        if (session?.id) {
            options.push(BaseObjectAction.Report);
        }
        if (isOwn) {
            options.push(BaseObjectAction.Delete);
        }
        return options;
    }, [user, isOwn, session]);

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
        // The first tab doesn't have search results, as it is the user's set resources
        switch (tabIndex) {
            case 1:
                return [
                    "Search user's organizations...",
                    OrganizationSortOptions,
                    organizationDefaultSortOption,
                    organizationOptionLabel,
                    organizationsQuery,
                    (newValue) => openLink(APP_LINKS.Organization, newValue.id),
                    (node: Organization, index: number) => (
                        <OrganizationListItem
                            key={`organization-list-item-${index}`}
                            index={index}
                            session={session}
                            data={node}
                            isOwn={false}
                            onClick={(selected: Organization) => openLink(APP_LINKS.Organization, selected.id)}
                        />)
                ];
            case 2:
                return [
                    "Search user's projects...",
                    ProjectSortOptions,
                    projectDefaultSortOption,
                    projectOptionLabel,
                    projectsQuery,
                    (newValue) => openLink(APP_LINKS.Project, newValue.id),
                    (node: Project, index: number) => (
                        <ProjectListItem
                            key={`project-list-item-${index}`}
                            index={index}
                            session={session}
                            data={node}
                            isOwn={false}
                            onClick={(selected: Project) => openLink(APP_LINKS.Project, selected.id)}
                        />)
                ];
            case 3:
                return [
                    "Search user's routines...",
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
            case 4:
                return [
                    "Search user's standards...",
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
    const moreMenuId = useMemo(() => `user-options-menu-${user?.id}`, [user]);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    /**
     * Displays username, avatar, bio, and quick links
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
                            <Typography variant="h4" textAlign="center">{user?.username ?? partialData?.username}</Typography>
                            <Tooltip title="Edit profile">
                                <IconButton
                                    aria-label="Edit profile"
                                    size="small"
                                    onClick={() => setLocation(`${APP_LINKS.Profile}/edit/${id}`)}
                                >
                                    <EditIcon color="primary" />
                                </IconButton>
                            </Tooltip>
                        </Stack>
                    ) : (
                        <Typography variant="h4" textAlign="center">{user?.username ?? partialData?.username}</Typography>

                    )
                }
                <Typography variant="body1" sx={{ color: "#00831e" }}>{user?.created_at ? `🕔 Joined ${new Date(user.created_at).toDateString()}` : ''}</Typography>
                <Typography variant="body1" sx={{ color: user?.bio ? 'black' : 'gray' }}>{user?.bio ?? partialData?.bio ?? 'No bio set'}</Typography>
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
                    {!isOwn ? <StarButton
                        session={session}
                        isStar={user?.isStarred ?? false}
                        stars={user?.stars ?? 0}
                        onStar={handleStar}
                        tooltipPlacement="bottom"
                    /> : null}
                </Stack>
            </Stack>
        </Box>
    ), [user, partialData, isOwn, handleStar]);

    return (
        <>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <BaseObjectActionDialog
                objectId={id}
                objectType={'User'}
                anchorEl={moreMenuAnchor}
                title='User Options'
                availableOptions={moreOptions}
                onClose={closeMoreMenu}
            />
            <Box sx={{ display: 'flex', paddingTop: 5, paddingBottom: 5, background: "#b2b3b3" }}>
                {overviewComponent}
            </Box>
            {/* View routines, organizations, standards, and projects associated with this user */}
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