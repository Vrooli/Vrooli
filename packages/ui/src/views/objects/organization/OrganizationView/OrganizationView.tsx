import { BookmarkFor, CommonKey, endpointGetOrganization, LINKS, Organization, ResourceList, uuidValidate, VisibilityType } from "@local/shared";
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
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { Title } from "components/text/Title/Title";
import { EditIcon, EllipsisIcon, OrganizationIcon, SearchIcon } from "icons";
import { MouseEvent, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { BannerImageContainer, OverviewContainer, OverviewProfileAvatar, OverviewProfileStack } from "styles";
import { extractImageUrl } from "utils/display/imageTools";
import { placeholderColor, YouInflated } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { useTabs } from "utils/hooks/useTabs";
import { PubSub } from "utils/pubsub";
import { OrganizationPageTabOption, SearchType } from "utils/search/objectToSearch";
import { SessionContext } from "utils/SessionContext";
import { OrganizationViewProps } from "../types";

type TabWhereParams = {
    organizationId: string;
    permissions: YouInflated;
}

const tabParams = [{
    titleKey: "Resource" as CommonKey,
    searchType: SearchType.Resource,
    tabType: OrganizationPageTabOption.Resource,
    where: () => ({}),
}, {
    titleKey: "Project" as CommonKey,
    searchType: SearchType.Project,
    tabType: OrganizationPageTabOption.Project,
    where: ({ organizationId, permissions }: TabWhereParams) => ({ ownedByOrganizationId: organizationId, hasCompleteVersion: !permissions.canUpdate ? true : undefined, visibility: VisibilityType.All }),
}, {
    titleKey: "Member" as CommonKey,
    searchType: SearchType.Member,
    tabType: OrganizationPageTabOption.Member,
    where: ({ organizationId }: TabWhereParams) => ({ organizationId }),
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
            list={resourceList}
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
            parent={{ __typename: "Organization", id: organization?.id ?? "" }}
            zIndex={zIndex}
        />
    ) : null, [isLoading, organization, permissions.canUpdate, resourceList, setOrganization, zIndex]);

    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
        where,
    } = useTabs<OrganizationPageTabOption>({ tabParams, display });

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
        if (currTab.tabType === OrganizationPageTabOption.Member) return;
        setLocation(`${LINKS[currTab.tabType]}/add`);
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
                        currTab.tabType === OrganizationPageTabOption.Resource ? resources : (
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
                                    listContainer: { borderRadius: 0 },
                                } : {
                                    search: { display: "none" },
                                    buttons: { display: "none" },
                                    listContainer: { borderRadius: 0 },
                                }}
                                take={20}
                                where={where({ organizationId: organization?.id ?? "", permissions })}
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
                {currTab.tabType !== OrganizationPageTabOption.Resource ? <ColorIconButton aria-label="filter-list" background={palette.secondary.main} onClick={toggleSearchFilters} >
                    <SearchIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                </ColorIconButton> : null}
            </SideActionButtons>
        </>
    );
};
