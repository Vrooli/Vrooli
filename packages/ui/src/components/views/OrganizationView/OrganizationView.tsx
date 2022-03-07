import { Box, IconButton, Stack, Tab, Tabs, Tooltip, Typography } from "@mui/material"
import { useLocation, useRoute } from "wouter";
import { APP_LINKS, MemberRole, StarFor } from "@local/shared";
import { useQuery } from "@apollo/client";
import { organization } from "graphql/generated/organization";
import { usersQuery, projectsQuery, routinesQuery, standardsQuery, organizationQuery } from "graphql/query";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import {
    CardGiftcard as DonateIcon,
    Edit as EditIcon,
    MoreHoriz as EllipsisIcon,
    Person as PersonIcon,
    Share as ShareIcon,
} from "@mui/icons-material";
import { actorDefaultSortOption, ActorListItem, actorOptionLabel, ActorSortOptions, BaseObjectActionDialog, projectDefaultSortOption, ProjectListItem, projectOptionLabel, ProjectSortOptions, routineDefaultSortOption, RoutineListItem, routineOptionLabel, RoutineSortOptions, SearchList, standardDefaultSortOption, StandardListItem, standardOptionLabel, StandardSortOptions, StarButton } from "components";
import { containerShadow } from "styles";
import { OrganizationViewProps } from "../types";
import { User, Project, Routine, Standard } from "types";
import { BaseObjectAction } from "components/dialogs/types";
import { SearchListGenerator } from "components/lists/types";
import { getTranslation } from "utils";

const tabLabels = ['Resources', 'Members', 'Projects', 'Routines', 'Standards'];

export const OrganizationView = ({
    session,
    partialData,
}: OrganizationViewProps) => {
    const [, setLocation] = useLocation();
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Organization}/:id`);
    const [, params2] = useRoute(`${APP_LINKS.SearchOrganizations}/view/:id`);
    const id: string = useMemo(() => params?.id ?? params2?.id ?? '', [params, params2]);
    // Fetch data
    const { data, loading } = useQuery<organization>(organizationQuery, { variables: { input: { id } } });
    const organization = useMemo(() => data?.organization, [data]);
    const canEdit: boolean = useMemo(() => [MemberRole.Admin, MemberRole.Owner].includes(organization?.role ?? ''), [organization]);

    const { name, bio } = useMemo(() => {
        const languages = session?.languages ?? navigator.languages;
        return {
            name: getTranslation(organization, 'name', languages) ?? getTranslation(partialData, 'name', languages),
            bio: getTranslation(organization, 'bio', languages) ?? getTranslation(partialData, 'bio', languages),
        };
    }, [organization, partialData, session]);

    const onEdit = useCallback(() => {
        // Depends on if we're in a search popup or a normal organization page
        setLocation(Boolean(params?.id) ? `${APP_LINKS.Organization}/${id}/edit` : `${APP_LINKS.SearchOrganizations}/edit/${id}`);
    }, [setLocation, id]);

    // Handle tabs
    const [tabIndex, setTabIndex] = useState<number>(0);
    const handleTabChange = (event, newValue) => { setTabIndex(newValue) };

    // Create search data
    const { placeholder, sortOptions, defaultSortOption, sortOptionLabel, searchQuery, where, noResultsText, onSearchSelect, searchItemFactory } = useMemo<SearchListGenerator>(() => {
        const openLink = (baseLink: string, id: string) => setLocation(`${baseLink}/${id}`);
        // The first tab doesn't have search results, as it is the organization's set resources
        switch (tabIndex) {
            case 1:
                return {
                    placeholder: "Search orgnization's members...",
                    noResultsText: "No members found",
                    sortOptions: ActorSortOptions,
                    defaultSortOption: actorDefaultSortOption,
                    sortOptionLabel: actorOptionLabel,
                    searchQuery: usersQuery,
                    where: { organizationId: id },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.User, newValue.id),
                    searchItemFactory: (node: User, index: number) => (
                        <ActorListItem
                            key={`member-list-item-${index}`}
                            index={index}
                            session={session}
                            data={node}
                            onClick={(selected: User) => openLink(APP_LINKS.User, selected.id)}
                        />)
                };
            case 2:
                return {
                    placeholder: "Search organization's projects...",
                    noResultsText: "No projects found",
                    sortOptions: ProjectSortOptions,
                    defaultSortOption: projectDefaultSortOption,
                    sortOptionLabel: projectOptionLabel,
                    searchQuery: projectsQuery,
                    where: { organizationId: id },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Project, newValue.id),
                    searchItemFactory: (node: Project, index: number) => (
                        <ProjectListItem
                            key={`project-list-item-${index}`}
                            index={index}
                            session={session}
                            data={node}
                            onClick={(selected: Project) => openLink(APP_LINKS.Project, selected.id)}
                        />)
                };
            case 3:
                return {
                    placeholder: "Search organization's routines...",
                    noResultsText: "No routines found",
                    sortOptions: RoutineSortOptions,
                    defaultSortOption: routineDefaultSortOption,
                    sortOptionLabel: routineOptionLabel,
                    searchQuery: routinesQuery,
                    where: { organizationId: id },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Routine, newValue.id),
                    searchItemFactory: (node: Routine, index: number) => (
                        <RoutineListItem
                            key={`routine-list-item-${index}`}
                            index={index}
                            session={session}
                            data={node}
                            onClick={(selected: Routine) => openLink(APP_LINKS.Routine, selected.id)}
                        />)
                };
            case 4:
                return {
                    placeholder: "Search organization's standards...",
                    noResultsText: "No standards found",
                    sortOptions: StandardSortOptions,
                    defaultSortOption: standardDefaultSortOption,
                    sortOptionLabel: standardOptionLabel,
                    searchQuery: standardsQuery,
                    where: { organizationId: id },
                    onSearchSelect: (newValue) => openLink(APP_LINKS.Standard, newValue.id),
                    searchItemFactory: (node: Standard, index: number) => (
                        <StandardListItem
                            key={`standard-list-item-${index}`}
                            index={index}
                            session={session}
                            data={node}
                            onClick={(selected: Standard) => openLink(APP_LINKS.Standard, selected.id)}
                        />)
                }
            default:
                return {
                    placeholder: '',
                    noResultsText: '',
                    sortOptions: [],
                    defaultSortOption: { label: '', value: null },
                    sortOptionLabel: (o: any) => '',
                    searchQuery: null,
                    where: {},
                    onSearchSelect: (o: any) => { },
                    searchItemFactory: (a: any, b: any) => null
                }
            }
    }, [session, id, tabIndex]);

    // Handle url search
    const [searchString, setSearchString] = useState<string>('');
    const [sortBy, setSortBy] = useState<string | undefined>(undefined);
    const [timeFrame, setTimeFrame] = useState<string | undefined>(undefined);
    useEffect(() => {
        setSortBy(defaultSortOption.value ?? sortOptions.length > 0 ? sortOptions[0].value : undefined);
    }, [defaultSortOption, sortOptions]);

    // Determine options available to object, in order
    const moreOptions: BaseObjectAction[] = useMemo(() => {
        // Initialize
        let options: BaseObjectAction[] = [];
        if (session && !canEdit) {
            options.push(organization?.isStarred ? BaseObjectAction.Unstar : BaseObjectAction.Star);
        }
        options.push(BaseObjectAction.Donate, BaseObjectAction.Share)
        if (session?.id) {
            options.push(BaseObjectAction.Report);
        }
        if (canEdit) {
            options.push(BaseObjectAction.Delete);
        }
        return options;
    }, [session, canEdit, organization?.isStarred]);

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
                    canEdit ? (
                        <Stack direction="row" alignItems="center" justifyContent="center">
                            <Typography variant="h4" textAlign="center">{name}</Typography>
                            <Tooltip title="Edit organization">
                                <IconButton
                                    aria-label="Edit organization"
                                    size="small"
                                    onClick={onEdit}
                                >
                                    <EditIcon color="primary" />
                                </IconButton>
                            </Tooltip>
                        </Stack>
                    ) : (
                        <Typography variant="h4" textAlign="center">{name}</Typography>

                    )
                }
                <Typography variant="body1" sx={{ color: "#00831e" }}>{organization?.created_at ? `🕔 Joined ${new Date(organization.created_at).toDateString()}` : ''}</Typography>
                <Typography variant="body1" sx={{ color: bio ? 'black' : 'gray' }}>{bio ?? 'No bio set'}</Typography>
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
                        !canEdit ? <StarButton
                            session={session}
                            objectId={organization?.id ?? ''}
                            starFor={StarFor.Organization}
                            isStar={organization?.isStarred ?? false}
                            stars={organization?.stars ?? 0}
                            onChange={(isStar: boolean) => { }}
                            tooltipPlacement="bottom"
                        /> : null
                    }
                </Stack>
            </Stack>
        </Box>
    ), [bio, session, name, organization, partialData, canEdit, onEdit]);

    return (
        <>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <BaseObjectActionDialog
                objectId={id}
                objectType={'Organization'}
                anchorEl={moreMenuAnchor}
                title='Organization Options'
                availableOptions={moreOptions}
                onClose={closeMoreMenu}
            />
            <Box sx={{ display: 'flex', paddingTop: 5, paddingBottom: 5, background: "#b2b3b3" }}>
                {overviewComponent}
            </Box>
            {/* View routines, members, standards, and projects associated with this organization */}
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
                                where={where}
                                noResultsText={noResultsText}
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