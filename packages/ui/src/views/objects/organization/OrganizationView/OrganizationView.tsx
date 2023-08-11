import { BookmarkFor, endpointGetOrganization, LINKS, Organization, ResourceList, uuidValidate, VisibilityType } from "@local/shared";
import { Box, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { BookmarkButton } from "components/buttons/BookmarkButton/BookmarkButton";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { ReportsLink } from "components/buttons/ReportsLink/ReportsLink";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { ResourceListVertical } from "components/lists/resource";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TextLoading } from "components/lists/TextLoading/TextLoading";
import { SearchListGenerator } from "components/lists/types";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { Title } from "components/text/Title/Title";
import { PageTab } from "components/types";
import { EditIcon, EllipsisIcon, OrganizationIcon, SearchIcon } from "icons";
import { MouseEvent, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { BannerImageContainer, OverviewContainer, OverviewProfileAvatar, OverviewProfileStack } from "styles";
import { extractImageUrl } from "utils/display/imageTools";
import { placeholderColor, toSearchListData } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { PubSub } from "utils/pubsub";
import { SearchType } from "utils/search/objectToSearch";
import { SessionContext } from "utils/SessionContext";
import { OrganizationViewProps } from "../types";

enum TabOptions {
    Resource = "Resource",
    Project = "Project",
    Member = "Member",
}

type TabParams = {
    searchType: SearchType;
    tabType: TabOptions;
    where: { [x: string]: any };
}

// Data for each tab
const tabParams: TabParams[] = [{
    searchType: SearchType.Resource,
    tabType: TabOptions.Resource,
    where: {},
}, {
    searchType: SearchType.Project,
    tabType: TabOptions.Project,
    where: {},
}, {
    searchType: SearchType.Member,
    tabType: TabOptions.Member,
    where: {},
}];

export const OrganizationView = ({
    isOpen,
    onClose,
    zIndex,
}: OrganizationViewProps) => {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const display = toDisplay(isOpen);
    const profileColors = useMemo(() => placeholderColor(), []);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);

    const { isLoading, object: organization, permissions, setObject: setOrganization } = useObjectFromUrl<Organization>({
        ...endpointGetOrganization,
        objectType: "Organization",
    });

    const availableLanguages = useMemo<string[]>(() => (organization?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [organization?.translations]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { bannerImageUrl, bio, handle, name, resourceList } = useMemo(() => {
        const resourceList: ResourceList | null | undefined = organization?.resourceList;
        const { bio, name } = getTranslation(organization, [language]);
        return {
            bannerImageUrl: extractImageUrl(organization?.bannerImage, organization?.updated_at, 1000),
            bio: bio && bio.trim().length > 0 ? bio : undefined,
            handle: organization?.handle,
            name,
            resourceList,
        };
    }, [language, organization]);

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

    const [showSearchFilters, setShowSearchFilters] = useState<boolean>(false);
    const toggleSearchFilters = useCallback(() => setShowSearchFilters(!showSearchFilters), [showSearchFilters]);

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
                tabTitle={handle ? `${name} (@${handle})` : name}
                zIndex={zIndex}
            />
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={moreMenuAnchor}
                object={organization as any}
                onClose={closeMoreMenu}
                zIndex={zIndex + 1}
            />
            <BannerImageContainer sx={{
                backgroundImage: bannerImageUrl ? `url(${bannerImageUrl})` : undefined,
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
            </BannerImageContainer>
            <OverviewContainer>
                <OverviewProfileStack>
                    <OverviewProfileAvatar
                        src={extractImageUrl(organization?.profileImage, organization?.updated_at, 100)}
                        sx={{
                            backgroundColor: profileColors[0],
                            color: profileColors[1],
                        }}
                    >
                        <OrganizationIcon width="75%" height="75%" />
                    </OverviewProfileAvatar>
                    <Tooltip title={t("MoreOptions")}>
                        <IconButton
                            aria-label={t("MoreOptions")}
                            size="small"
                            onClick={openMoreMenu}
                            sx={{
                                display: "block",
                                marginLeft: "auto",
                                marginRight: 1,
                            }}
                        >
                            <EllipsisIcon fill={palette.background.textSecondary} />
                        </IconButton>
                    </Tooltip>
                    <BookmarkButton
                        objectId={organization?.id ?? ""}
                        bookmarkFor={BookmarkFor.Organization}
                        isBookmarked={organization?.you?.isBookmarked ?? false}
                        bookmarks={organization?.bookmarks ?? 0}
                        onChange={(isBookmarked: boolean) => { }}
                        zIndex={zIndex}
                    />
                </OverviewProfileStack>
                <Stack direction="column" spacing={1} p={2} justifyContent="center" sx={{
                    alignItems: "flex-start",
                }}>
                    {/* Title */}
                    {
                        (isLoading && !name) ? (
                            <TextLoading size="header" sx={{ width: "50%" }} />
                        ) : <Title
                            title={name}
                            variant="header"
                            options={permissions.canUpdate ? [{
                                label: t("Edit"),
                                Icon: EditIcon,
                                onClick: () => { actionData.onActionStart("Edit"); },
                            }] : []}
                            zIndex={zIndex}
                            sxs={{ stack: { padding: 0, paddingBottom: handle ? 0 : 2 } }}
                        />
                    }
                    {/* Handle */}
                    {
                        handle && <Typography
                            variant="h6"
                            textAlign="center"
                            fontFamily="monospace"
                            onClick={() => {
                                navigator.clipboard.writeText(`${window.location.origin}${LINKS.Organization}/${handle}`);
                                PubSub.get().publishSnack({ messageKey: "CopiedToClipboard", severity: "Success" });
                            }}
                            sx={{
                                color: palette.secondary.dark,
                                cursor: "pointer",
                                paddingBottom: 2,
                            }}
                        >@{handle}</Typography>
                    }
                    {/* Bio */}
                    {
                        (isLoading && !bio) ? (
                            <TextLoading lines={2} size="body1" sx={{ width: "85%" }} />
                        ) : (
                            <MarkdownDisplay
                                variant="body1"
                                sx={{ color: bio ? palette.background.textPrimary : palette.background.textSecondary }}
                                content={bio ?? "No bio set"}
                                zIndex={zIndex}
                            />
                        )
                    }
                    <Stack direction="row" spacing={2} sx={{
                        alignItems: "center",
                    }}>
                        {/* Created date */}
                        <DateDisplay
                            loading={isLoading}
                            showIcon={true}
                            textBeforeDate="Created"
                            timestamp={organization?.created_at}
                            zIndex={zIndex}
                        />
                        <ReportsLink object={organization} />
                    </Stack>
                </Stack>
            </OverviewContainer>
            {/* View routines, members, standards, and projects associated with this organization */}
            <Box sx={{ margin: "auto", maxWidth: `min(${breakpoints.values.sm}px, 100%)` }}>
                <PageTabs
                    ariaLabel="organization-tabs"
                    fullWidth
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                    sx={{
                        background: palette.background.paper,
                        borderBottom: `1px solid ${palette.divider}`,
                    }}
                />
                <Box>
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
                                sxs={showSearchFilters ? {
                                    search: { marginTop: 2 },
                                } : {
                                    search: { display: "none" },
                                    buttons: { display: "none" },
                                }}
                                take={20}
                                where={where}
                                zIndex={zIndex}
                            />
                        )
                    }
                </Box>
            </Box>
            <SideActionButtons
                display={display}
                zIndex={zIndex + 2}
                sx={{ position: "fixed" }}
            >
                {/* Toggle search filters */}
                {currTab.value !== TabOptions.Resource ? <ColorIconButton aria-label="filter-list" background={palette.secondary.main} onClick={toggleSearchFilters} >
                    <SearchIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </ColorIconButton> : null}
            </SideActionButtons>
        </>
    );
};
