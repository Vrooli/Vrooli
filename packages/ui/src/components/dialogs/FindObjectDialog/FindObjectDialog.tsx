import { AddIcon, addSearchParams, ApiIcon, FindByIdInput, HelpIcon, NoteIcon, OrganizationIcon, parseSearchParams, ProjectIcon, removeSearchParams, RoutineIcon, SmartContractIcon, StandardIcon, SvgProps, useLocation, UserIcon, VisibleIcon } from "@local/shared";
import { Box, Button, ListItemIcon, ListItemText, Menu, MenuItem, Stack, Typography, useTheme } from "@mui/material";
import { useCustomLazyQuery } from "api";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TIDCard } from "components/lists/TIDCard/TIDCard";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { PageTab } from "components/types";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { lazily } from "react-lazily";
import { AutocompleteOption } from "types";
import { getDisplay } from "utils/display/listTools";
import { getObjectUrl } from "utils/navigation/openObject";
import { SearchPageTabOption, SearchType, searchTypeToParams } from "utils/search/objectToSearch";
import { SearchParams } from "utils/search/schemas/base";
import { UpsertProps } from "views/objects/types";
import { ShareSiteDialog } from "../ShareSiteDialog/ShareSiteDialog";
import { FindObjectDialogProps, FindObjectDialogType, SelectOrCreateObject, SelectOrCreateObjectType } from "../types";

const { ApiUpsert } = lazily(() => import("../../../views/objects/api/ApiUpsert/ApiUpsert"));
const { FocusModeUpsert } = lazily(() => import("../../../views/objects/focusMode/FocusModeUpsert/FocusModeUpsert"));
const { MeetingUpsert } = lazily(() => import("../../../views/objects/meeting/MeetingUpsert/MeetingUpsert"));
const { NoteUpsert } = lazily(() => import("../../../views/objects/note/NoteUpsert/NoteUpsert"));
const { OrganizationUpsert } = lazily(() => import("../../../views/objects/organization/OrganizationUpsert/OrganizationUpsert"));
const { ProjectUpsert } = lazily(() => import("../../../views/objects/project/ProjectUpsert/ProjectUpsert"));
const { RoutineUpsert } = lazily(() => import("../../../views/objects/routine/RoutineUpsert/RoutineUpsert"));
const { RunProjectUpsert } = lazily(() => import("../../../views/objects/runProject/RunProjectUpsert/RunProjectUpsert"));
const { RunRoutineUpsert } = lazily(() => import("../../../views/objects/runRoutine/RunRoutineUpsert/RunRoutineUpsert"));
const { SmartContractUpsert } = lazily(() => import("../../../views/objects/smartContract/SmartContractUpsert/SmartContractUpsert"));
const { StandardUpsert } = lazily(() => import("../../../views/objects/standard/StandardUpsert/StandardUpsert"));

type RemoveVersion<T extends string> = T extends `${infer U}Version` ? U : T;
type CreateViewTypes = Exclude<RemoveVersion<SelectOrCreateObjectType>, "User">;

type BaseParams = {
    Icon: (props: SvgProps) => JSX.Element,
    searchType: "All" | SearchType;
    tabType: "All" | SearchPageTabOption;
    where: { [x: string]: any };
}

// Data for each tab
const tabParams: BaseParams[] = [{
    Icon: VisibleIcon,
    searchType: "All",
    tabType: "All",
    where: {},
}, {
    Icon: RoutineIcon,
    searchType: SearchType.Routine,
    tabType: SearchPageTabOption.Routines,
    where: {},
}, {
    Icon: ProjectIcon,
    searchType: SearchType.Project,
    tabType: SearchPageTabOption.Projects,
    where: {},
}, {
    Icon: HelpIcon,
    searchType: SearchType.Question,
    tabType: SearchPageTabOption.Questions,
    where: {},
}, {
    Icon: NoteIcon,
    searchType: SearchType.Note,
    tabType: SearchPageTabOption.Notes,
    where: {},
}, {
    Icon: OrganizationIcon,
    searchType: SearchType.Organization,
    tabType: SearchPageTabOption.Organizations,
    where: {},
}, {
    Icon: UserIcon,
    searchType: SearchType.User,
    tabType: SearchPageTabOption.Users,
    where: {},
}, {
    Icon: StandardIcon,
    searchType: SearchType.Standard,
    tabType: SearchPageTabOption.Standards,
    where: {},
}, {
    Icon: ApiIcon,
    searchType: SearchType.Api,
    tabType: SearchPageTabOption.Apis,
    where: {},
}, {
    Icon: SmartContractIcon,
    searchType: SearchType.SmartContract,
    tabType: SearchPageTabOption.SmartContracts,
    where: {},
}];

/**
 * Maps SelectOrCreateObject types to create components (excluding "User" and types that end with 'Version')
 */
const createMap: { [K in CreateViewTypes]: (props: UpsertProps<any>) => JSX.Element } = {
    Api: ApiUpsert,
    FocusMode: FocusModeUpsert,
    Meeting: MeetingUpsert,
    Note: NoteUpsert,
    Organization: OrganizationUpsert,
    Project: ProjectUpsert,
    Routine: RoutineUpsert,
    RunProject: RunProjectUpsert,
    RunRoutine: RunRoutineUpsert,
    SmartContract: SmartContractUpsert,
    Standard: StandardUpsert,
};

const searchTitleId = "search-vrooli-for-link-title";

export const FindObjectDialog = <Find extends FindObjectDialogType, ObjectType extends SelectOrCreateObject>({
    find,
    handleCancel,
    handleComplete,
    isOpen,
    limitTo,
    searchData,
    zIndex,
}: FindObjectDialogProps<Find, ObjectType>) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    // Tabs to filter by object type
    const tabs = useMemo<PageTab<"All" | SearchPageTabOption>[]>(() => {
        // If limitTo is set, only show those tabs
        let filteredTabParams = tabParams;
        if (limitTo && limitTo.length > 0) {
            const unversionedLimitTo = limitTo.map(l => l.replace("Version", "") as any);
            filteredTabParams = tabParams.filter(tab => unversionedLimitTo.includes(tab.searchType as any));
        }
        return filteredTabParams.map((tab, i) => ({
            index: i,
            Icon: tab.Icon,
            label: t(tab.searchType, { count: 2, defaultValue: tab.searchType }),
            value: tab.tabType,
        }));
    }, [limitTo, t]);

    const [currTab, setCurrTab] = useState<PageTab<"All" | SearchPageTabOption> | null>(null);
    useEffect(() => {
        // Get tab from search params
        const searchParams = parseSearchParams();
        const index = tabs.findIndex(tab => tab.value === searchParams.type);
        // If not found, default to the first tab
        if (index === -1) {
            setCurrTab(tabs[0]);
        } else {
            setCurrTab(tabs[index]);
        }
    }, [tabs]);
    console.log("yeeeeet", currTab);

    const handleTabChange = useCallback((e: any, tab: PageTab<SearchPageTabOption>) => {
        e.preventDefault();
        // Update search params
        addSearchParams(setLocation, { type: tab.value });
        // Update curr tab
        setCurrTab(tab);
    }, [setLocation]);

    // Dialog for creating new object
    const [createObjectType, setCreateObjectType] = useState<CreateViewTypes | "User" | null>(null);

    // Dialog for inviting new user
    const [isInviteUserOpen, setIsInviteUserOpen] = useState(false);
    const onInviteUserClose = useCallback(() => setIsInviteUserOpen(false), []);

    // Menu for selection object type to create
    const [selectCreateTypeAnchorEl, setSelectCreateTypeAnchorEl] = useState<null | HTMLElement>(null);

    // Info for querying full object data
    const [{ advancedSearchSchema, query }, setSearchParams] = useState<Partial<SearchParams>>({});
    useEffect(() => {
        const fetchParams = async () => {
            const params = searchTypeToParams[createObjectType!];
            if (!params) return;
            setSearchParams(await params());
        };
        createObjectType !== null && fetchParams();
    }, [createObjectType]);
    /**
     * Before closing, remove all URL search params for advanced search
     */
    const onClose = useCallback((item?: ObjectType, versionId?: string) => {
        console.log("onCloseeeeee", item);
        // Clear search params
        removeSearchParams(setLocation, [
            ...(advancedSearchSchema?.fields.map(f => f.fieldName) ?? []),
            "advanced",
            "sort",
            "time",
        ]);
        // If no item, close dialog
        if (!item) handleCancel();
        // If url requested, return url
        else if (find === "Url") {
            const objectUrl = getObjectUrl(item as any);
            const base = `${window.location.origin}${objectUrl}`;
            const url = versionId ? `${base}/${versionId}` : base;
            // If item, store in local storage so we can display it in the link component
            if (item) {
                const itemToStore = versionId ? ((item as any).versions?.find(v => v.id === versionId) ?? {}) : item;
                localStorage.setItem(`objectFromUrl:${url}`, JSON.stringify(itemToStore));
            }
            handleComplete((versionId ? `${base}/${versionId}` : base) as any);
        }
        // Otherwise, return object
        else {
            // If versionId is set, return the version
            if (versionId) {
                const version = (item as any).versions?.find(v => v.id === versionId);
                const { versions, ...rest } = item as any;
                handleComplete({ ...version, root: rest } as any);
            }
            // Otherwise, return the item
            else handleComplete(item as any);
        }
    }, [advancedSearchSchema?.fields, find, handleCancel, handleComplete, setLocation]);

    const [selectedObject, setSelectedObject] = useState<{
        versions: { id: string; versionIndex: number, versionLabel: string }[];
    } | null>(null);
    // Reset selected object when dialog is opened/closed
    useEffect(() => {
        setSelectedObject(null);
    }, [isOpen]);

    // On tab change, update search params
    const { searchType, where } = useMemo<Pick<BaseParams, "searchType" | "where">>(() => {
        console.log("yeet calculating search type", searchData, currTab, tabs);
        if (searchData) return searchData as any;
        if (currTab) return { searchType: tabParams.find(tab => tab.tabType === currTab.value)?.searchType ?? "All", where: {} };
        return { searchType: "All", where: {} };
    }, [currTab, searchData, tabs]);

    const onCreateStart = useCallback((e: React.MouseEvent<HTMLElement>) => {
        e.preventDefault();
        // If tab is 'All', open menu to select type
        if (searchType === "All" || !currTab) setSelectCreateTypeAnchorEl(e.currentTarget);
        // If tab is 'User', open invite user dialog
        else if (searchType === "User") setIsInviteUserOpen(true);
        // Otherwise, open create dialog for current tab
        setCreateObjectType(tabParams.find(tab => tab.tabType === currTab!.value)?.searchType as any);
    }, [currTab, searchType]);
    const onSelectCreateTypeClose = useCallback((type?: SearchType) => {
        if (type) {
            if (type === "User") {
                setIsInviteUserOpen(true); // Open the Invite User dialog
                setSelectCreateTypeAnchorEl(null); // Close the Popover

            } else {
                setCreateObjectType(type as any); // Open the Create Object dialog
                // Wait for the Create Object dialog to open fully (which is loaded asynchronously) 
                // before closing the Popover. Otherwise, it can get stuck open.
            }
        }
        else setSelectCreateTypeAnchorEl(null);
    }, []);

    const handleCreated = useCallback((item: SelectOrCreateObject) => {
        onClose(item as ObjectType);
        setCreateObjectType(null);
    }, [onClose]);
    const handleCreateClose = useCallback(() => {
        setCreateObjectType(null);
    }, []);

    // If item selected from search AND find is 'Object', query for full data
    const [getItem, { data: itemData }] = useCustomLazyQuery<ObjectType, FindByIdInput>(query);
    const queryingRef = useRef(false);
    const fetchFullData = useCallback((item: ObjectType, versionId?: string) => {
        if (!query || find !== "Full") return false;
        // Query for full item data, if not already known (would be known if the same item was selected last time)
        if (itemData && itemData.id === item.id && (!versionId || (itemData as any).versionId === versionId)) {
            onClose(itemData);
        } else {
            queryingRef.current = true;
            getItem({ variables: { id: item.id, versionId } }); // Pass versionId to the query
        }
        // Return false so the list item does not navigate
        return false;
    }, [query, find, itemData, onClose, getItem]);

    const onVersionSelect = useCallback((version: { id: string }) => {
        if (!selectedObject) return;
        // If the full data is requested, fetch the full data for the selected version
        if (find === "Full") {
            fetchFullData(selectedObject as any, version.id);
        } else {
            // Select and close dialog
            onClose(selectedObject as any, version.id);
        }
    }, [onClose, selectedObject, fetchFullData, find]);

    useEffect(() => {
        if (!query) return;
        if (itemData && find === "Full" && queryingRef.current) {
            onClose(itemData);
        }
        queryingRef.current = false;
    }, [onClose, handleCreateClose, itemData, query, find]);

    /**
     * Handles selecting an object. A few things can happen:
     * 1. If the object is not versioned or only has one version and we 
     * only need a URL, return the URL
     * 2. If the object is not versioned or only has one version and we 
     * need the object, fetch for full object data and return it
     * 3. If the object has multiple versions, display buttons to select
     * which version to link to
     */
    const onInputSelect = useCallback((newValue: AutocompleteOption) => {
        // If value is not an object, return;
        if (!newValue || newValue.__typename === "Shortcut" || newValue.__typename === "Action") return false;
        // If object has versions
        if ((newValue as any).versions && (newValue as any).versions.length > 0) {
            // If there is only one version, select it
            if ((newValue as any).versions.length === 1) {
                // Select and close dialog
                onClose(newValue as any, (newValue as any).versions[0].id);
                return false;
            }
            // Otherwise, set selected object so we can choose which version to link to
            setSelectedObject(newValue as any);
            return false;
        }
        // Select and close dialog
        onClose(newValue as any);
        return false;
    }, [onClose]);

    const CreateView = useMemo<((props: UpsertProps<any>) => JSX.Element) | null>(() =>
        ["User", null].includes(createObjectType) ? null : (createMap as any)[createObjectType!.replace("Version", "")], [createObjectType]);
    useEffect(() => {
        setSelectCreateTypeAnchorEl(null);
    }, [createObjectType]);

    console.log("yeeeet searchType", searchType);
    return (
        <>
            {/* Invite user dialog (when you select 'User' as create type) */}
            <ShareSiteDialog
                onClose={onInviteUserClose}
                open={isInviteUserOpen}
                zIndex={zIndex + 2}
            />
            {/* Dialog for creating new object type */}
            <LargeDialog
                id="create-object-dialog"
                onClose={handleCreateClose}
                isOpen={createObjectType !== null}
                titleId="create-object-dialog-title"
                zIndex={zIndex + 2}
            >
                {CreateView && <CreateView
                    display="dialog"
                    isCreate={true}
                    onCompleted={handleCreated}
                    onCancel={handleCreateClose}
                    zIndex={zIndex + 2}
                />}
            </LargeDialog>
            {/* Menu for selecting create object type */}
            {!CreateView && <Menu
                id="select-create-type-mnu"
                anchorEl={selectCreateTypeAnchorEl}
                disableScrollLock={true}
                open={Boolean(selectCreateTypeAnchorEl)}
                onClose={() => onSelectCreateTypeClose()}
            >
                {/* Never show 'All'=' */}
                {tabParams.filter((t) => !["All"].includes(t.searchType as any)).map(tab => (
                    <MenuItem
                        key={tab.searchType}
                        onClick={() => onSelectCreateTypeClose(tab.searchType as SearchType)}
                    >
                        <ListItemIcon>
                            <tab.Icon fill={palette.background.textPrimary} />
                        </ListItemIcon>
                        <ListItemText primary={t(tab.searchType, { count: 1, defaultValue: tab.searchType })} />
                    </MenuItem>
                ))}
            </Menu>}
            {/* Main content */}
            <LargeDialog
                id="resource-find-object-dialog"
                isOpen={isOpen}
                onClose={() => { handleCancel(); }}
                titleId={searchTitleId}
                zIndex={zIndex}
            >
                <TopBar
                    display="dialog"
                    onClose={() => { handleCancel(); }}
                    titleData={{
                        hideOnDesktop: true,
                        titleKey: "SearchVrooli",
                        helpKey: "FindObjectDialogHelp",
                    }}
                    below={tabs.length > 1 && Boolean(currTab) && <PageTabs
                        ariaLabel="search-tabs"
                        currTab={currTab!}
                        onChange={handleTabChange}
                        tabs={tabs}
                    />}
                />
                <Box sx={{
                    minHeight: "500px",
                    margin: { xs: 0, sm: 2 },
                    paddingTop: 4,
                }}>
                    {/* Create object button. Convenient for when you can't find 
                what you're looking for */}
                    <SideActionButtons display="dialog" zIndex={zIndex + 1}>
                        <ColorIconButton aria-label="create-new" background={palette.secondary.main} onClick={onCreateStart}>
                            <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                        </ColorIconButton>
                    </SideActionButtons>
                    {/* Search list to find object */}
                    {!selectedObject && <SearchList
                        id="find-object-search-list"
                        beforeNavigation={onInputSelect}
                        take={20}
                        // Combine results for each object type
                        resolve={(data: { [x: string]: any }) => {
                            // Find largest array length 
                            const max: number = Object.values(data).reduce((acc: number, val: any[]) => {
                                return Math.max(acc, val.length);
                            }, -Infinity);
                            // Initialize result array
                            const result: any[] = [];
                            // Loop through each index
                            for (let i = 0; i < max; i++) {
                                // Loop through each object type
                                for (const key in data) {
                                    // If index exists, push to result
                                    if (Array.isArray(data[key]) && data[key][i]) result.push(data[key][i]);
                                }
                            }
                            return result;
                        }}
                        searchType={searchData?.searchType ?? "Popular"}
                        zIndex={zIndex}
                        where={searchData?.where ?? { ...where, objectType: searchType === "All" ? undefined : searchType }}
                    />}
                    {/* If object selected (and supports versioning), display buttons to select version */}
                    {selectedObject && (
                        <Stack spacing={2} direction="column" m={2}>
                            <Typography variant="h6" mt={2} textAlign="center">
                                Select a version
                            </Typography>
                            {[...(selectedObject.versions ?? [])].sort((a, b) => b.versionIndex - a.versionIndex).map((version, index) => (
                                <TIDCard
                                    buttonText={t("Select")}
                                    description={getDisplay(version as any).subtitle}
                                    key={index}
                                    Icon={tabParams.find((t) => t.searchType === (version as any).__typename)?.Icon}
                                    onClick={() => onVersionSelect(version)}
                                    title={`${version.versionLabel} - ${getDisplay(version as any).title}`}
                                />
                            ))}
                            {/* Remove selection button */}
                            <Button
                                fullWidth
                                color="secondary"
                                onClick={() => setSelectedObject(null)}
                            >
                                Select a different object
                            </Button>
                        </Stack>
                    )}
                </Box>
            </LargeDialog>
        </>
    );
};
