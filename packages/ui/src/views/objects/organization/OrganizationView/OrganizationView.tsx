import { BookmarkFor, EditIcon, EllipsisIcon, endpointGetOrganization, HelpIcon, LINKS, Organization, OrganizationIcon, ProjectIcon, ResourceList, SvgComponent, useLocation, UserIcon, uuidValidate, VisibilityType } from "@local/shared";
import { Avatar, Box, IconButton, LinearProgress, Link, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { BookmarkButton } from "components/buttons/BookmarkButton/BookmarkButton";
import { ReportsLink } from "components/buttons/ReportsLink/ReportsLink";
import { ShareButton } from "components/buttons/ShareButton/ShareButton";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { ResourceListVertical } from "components/lists/resource";
import { SearchList } from "components/lists/SearchList/SearchList";
import { SearchListGenerator } from "components/lists/types";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { PageTab } from "components/types";
import { MouseEvent, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { placeholderColor, toSearchListData } from "utils/display/listTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { SearchType } from "utils/search/objectToSearch";
import { SessionContext } from "utils/SessionContext";
import { OrganizationViewProps } from "../types";

enum TabOptions {
    Resource = "Resource",
    Project = "Project",
    Member = "Member",
}

type TabParams = {
    Icon: SvgComponent;
    searchType: SearchType;
    tabType: TabOptions;
    where: { [x: string]: any };
}

// Data for each tab
const tabParams: TabParams[] = [{
    Icon: HelpIcon,
    searchType: SearchType.Resource,
    tabType: TabOptions.Resource,
    where: {},
}, {
    Icon: ProjectIcon,
    searchType: SearchType.Project,
    tabType: TabOptions.Project,
    where: {},
}, {
    Icon: UserIcon,
    searchType: SearchType.Member,
    tabType: TabOptions.Member,
    where: {},
}];

export const OrganizationView = ({
    display = "page",
    onClose,
    partialData,
    zIndex = 200,
}: OrganizationViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const profileColors = useMemo(() => placeholderColor(), []);

    const { isLoading, object: organization, permissions, setObject: setOrganization } = useObjectFromUrl<Organization>({
        ...endpointGetOrganization,
        partialData,
    });

    const availableLanguages = useMemo<string[]>(() => (organization?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [organization?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { bio, handle, name, resourceList } = useMemo(() => {
        const resourceList: ResourceList | null | undefined = organization?.resourceList;
        const { bio, name } = getTranslation(organization ?? partialData, [language]);
        return {
            bio: bio && bio.trim().length > 0 ? bio : undefined,
            handle: organization?.handle ?? partialData?.handle,
            name,
            resourceList,
        };
    }, [language, organization, partialData]);

    useEffect(() => {
        if (handle) document.title = `${name} ($${handle}) | Vrooli`;
        else document.title = `${name} | Vrooli`;
    }, [handle, name]);

    const resources = useMemo(() => (resourceList || permissions.canUpdate) ? (
        <ResourceListVertical
            list={resourceList as any}
            canUpdate={permissions.canUpdate}
            handleUpdate={(updatedList) => {
                if (!organization) return;
                setOrganization({
                    ...organization,
                    resourceList: updatedList,
                });
            }}
            loading={isLoading}
            mutate={true}
            zIndex={zIndex}
        />
    ) : null, [isLoading, organization, permissions.canUpdate, resourceList, setOrganization, zIndex]);

    // Handle tabs
    const tabs = useMemo<PageTab<TabOptions>[]>(() => {
        let tabs = tabParams;
        // Remove resources if there are none, and you cannot add them
        if (!resources && !permissions.canUpdate) tabs = tabs.filter(t => t.tabType !== TabOptions.Resource);
        // Return tabs shaped for the tab component
        return tabs.map((tab, i) => ({
            color: tab.tabType === TabOptions.Resource ? "#8e6b00" : palette.secondary.dark, // Custom color for resources
            index: i,
            Icon: tab.Icon,
            label: t(tab.searchType, { count: 2, defaultValue: tab.searchType }),
            value: tab.tabType,
        }));
    }, [palette.secondary.dark, permissions.canUpdate, resources, t]);
    const [currTab, setCurrTab] = useState<PageTab<TabOptions>>(tabs[0]);
    const handleTabChange = useCallback((_: unknown, value: PageTab<TabOptions>) => setCurrTab(value), []);

    // Create search data
    const { searchType, placeholder, where } = useMemo<SearchListGenerator>(() => {
        // NOTE: The first tab doesn't have search results, as it is the user's set resources
        if (currTab.value === TabOptions.Member)
            return toSearchListData("Member", "SearchMember", { organizationId: organization?.id });
        return toSearchListData("Project", "SearchProject", { ownedByOrganizationId: organization?.id, hasCompleteVersion: !permissions.canUpdate ? true : undefined, visibility: VisibilityType.All });
    }, [currTab, organization?.id, permissions.canUpdate]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    const actionData = useObjectActions({
        object: organization,
        objectType: "Organization",
        setLocation,
        setObject: setOrganization,
    });

    /**
     * Displays name, avatar, bio, and quick links
     */
    const overviewComponent = useMemo(() => (
        <Box
            position="relative"
            ml='auto'
            mr='auto'
            mt={3}
            bgcolor={palette.background.paper}
            sx={{
                borderRadius: { xs: "0", sm: 2 },
                boxShadow: { xs: "none", sm: 2 },
                width: { xs: "100%", sm: "min(500px, 100vw)" },
            }}
        >
            <Avatar
                src="/broken-image.jpg" //TODO
                sx={{
                    backgroundColor: profileColors[0],
                    color: profileColors[1],
                    boxShadow: 2,
                    transform: "translateX(-50%)",
                    width: "min(100px, 25vw)",
                    height: "min(100px, 25vw)",
                    left: "50%",
                    top: "-55px",
                    position: "absolute",
                    fontSize: "min(50px, 10vw)",
                }}
            >
                <OrganizationIcon fill="white" width='75%' height='75%' />
            </Avatar>
            <Tooltip title="See all options">
                <IconButton
                    aria-label="More"
                    size="small"
                    onClick={openMoreMenu}
                    sx={{
                        display: "block",
                        marginLeft: "auto",
                        marginRight: 1,
                        paddingRight: "1em",
                    }}
                >
                    <EllipsisIcon fill={palette.background.textSecondary} />
                </IconButton>
            </Tooltip>
            <Stack direction="column" spacing={1} p={1} alignItems="center" justifyContent="center">
                {/* Title */}
                {
                    isLoading ? (
                        <Stack sx={{ width: "50%", color: "grey.500", paddingTop: 2, paddingBottom: 2 }} spacing={2}>
                            <LinearProgress color="inherit" />
                        </Stack>
                    ) : permissions.canUpdate ? (
                        <Stack direction="row" alignItems="center" justifyContent="center">
                            <Typography variant="h4" textAlign="center">{name}</Typography>
                            <Tooltip title="Edit organization">
                                <IconButton
                                    aria-label="Edit organization"
                                    size="small"
                                    onClick={() => actionData.onActionStart("Edit")}
                                >
                                    <EditIcon fill={palette.secondary.main} />
                                </IconButton>
                            </Tooltip>
                        </Stack>
                    ) : (
                        <Typography variant="h4" textAlign="center">{name}</Typography>
                    )
                }
                {/* Handle */}
                {
                    handle && <Link href={`https://handle.me/${handle}`} underline="hover">
                        <Typography
                            variant="h6"
                            textAlign="center"
                            sx={{
                                color: palette.secondary.dark,
                                cursor: "pointer",
                            }}
                        >${handle}</Typography>
                    </Link>
                }
                {/* Joined date */}
                <DateDisplay
                    loading={isLoading}
                    showIcon={true}
                    textBeforeDate="Joined"
                    timestamp={organization?.created_at}
                    width={"33%"}
                />
                {/* Bio */}
                {
                    isLoading ? (
                        <Stack sx={{ width: "85%", color: "grey.500" }} spacing={2}>
                            <LinearProgress color="inherit" />
                            <LinearProgress color="inherit" />
                        </Stack>
                    ) : (
                        <MarkdownDisplay variant="body1" sx={{ color: bio ? palette.background.textPrimary : palette.background.textSecondary }} content={bio ?? "No bio set"} />
                    )
                }
                <Stack direction="row" spacing={2} alignItems="center">
                    {/* <Tooltip title="Donate">
                        <IconButton aria-label="Donate" size="small" onClick={() => { }}>
                            <DonateIcon fill={palette.background.textSecondary} />
                        </IconButton>
                    </Tooltip> */}
                    <ShareButton object={organization} zIndex={zIndex} />
                    <ReportsLink object={organization} />
                    <BookmarkButton
                        disabled={!permissions.canBookmark}
                        objectId={organization?.id ?? ""}
                        bookmarkFor={BookmarkFor.Organization}
                        isBookmarked={organization?.you?.isBookmarked ?? false}
                        bookmarks={organization?.bookmarks ?? 0}
                        onChange={(isBookmarked: boolean) => { }}
                        zIndex={zIndex}
                    />
                </Stack>
            </Stack>
        </Box >
    ), [palette.background.paper, palette.background.textSecondary, palette.background.textPrimary, palette.secondary.main, palette.secondary.dark, profileColors, openMoreMenu, isLoading, permissions.canUpdate, permissions.canBookmark, name, handle, organization, bio, zIndex, actionData]);

    /**
     * Opens add new page
     */
    const toAddNew = useCallback((event: any) => {
        // TODO need member page
        if (currTab.value === TabOptions.Member) return;
        setLocation(`${LINKS[currTab.value]}/add`);
    }, [currTab, setLocation]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                titleData={{
                    titleKey: "Organization",
                }}
            />
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={moreMenuAnchor}
                object={organization as any}
                onClose={closeMoreMenu}
                zIndex={zIndex + 1}
            />
            <Box sx={{
                background: palette.mode === "light" ? "#b2b3b3" : "#303030",
                display: "flex",
                paddingTop: 5,
                paddingBottom: { xs: 0, sm: 2, md: 5 },
                position: "relative",
            }}>
                {/* Language display/select */}
                <Box sx={{
                    position: "absolute",
                    top: 8,
                    right: 8,
                    paddingRight: "1em",
                }}>
                    {availableLanguages.length > 1 && <SelectLanguageMenu
                        currentLanguage={language}
                        handleCurrent={setLanguage}
                        languages={availableLanguages}
                        zIndex={zIndex}
                    />}
                </Box>
                {overviewComponent}
            </Box>
            {/* View routines, members, standards, and projects associated with this organization */}
            <Box>
                <PageTabs
                    ariaLabel="organization-tabs"
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                />
                <Box p={2}>
                    {
                        currTab.value === TabOptions.Resource ? resources : (
                            <SearchList
                                canSearch={() => uuidValidate(organization?.id)}
                                dummyLength={display === "page" ? 5 : 3}
                                handleAdd={permissions.canUpdate ? toAddNew : undefined}
                                hideUpdateButton={true}
                                id="organization-view-list"
                                searchType={searchType}
                                searchPlaceholder={placeholder}
                                take={20}
                                where={where}
                                zIndex={zIndex}
                            />
                        )
                    }
                </Box>
            </Box>
        </>
    );
};
