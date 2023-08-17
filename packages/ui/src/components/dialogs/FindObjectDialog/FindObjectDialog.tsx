import { CommonKey, FindByIdInput, FindVersionInput } from "@local/shared";
import { Box, Button, ListItemIcon, ListItemText, Menu, MenuItem, Stack, Typography, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TIDCard } from "components/lists/TIDCard/TIDCard";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { AddIcon, FocusModeIcon, OrganizationIcon, ProjectIcon, RoutineIcon, SearchIcon } from "icons";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { lazily } from "react-lazily";
import { removeSearchParams, useLocation } from "route";
import { AutocompleteOption } from "types";
import { getDisplay } from "utils/display/listTools";
import { parseData } from "utils/hooks/useFindMany";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { useTabs } from "utils/hooks/useTabs";
import { getObjectUrl } from "utils/navigation/openObject";
import { CalendarPageTabOption, combineSearchResults, SearchPageTabOption, SearchType, searchTypeToParams } from "utils/search/objectToSearch";
import { SearchParams } from "utils/search/schemas/base";
import { UpsertProps } from "views/objects/types";
import { searchViewTabParams } from "../../../views/SearchView/SearchView";
import { FindObjectDialogProps, FindObjectDialogType, SelectOrCreateObject, SelectOrCreateObjectType } from "../types";

const { ApiUpsert } = lazily(() => import("../../../views/objects/api/ApiUpsert/ApiUpsert"));
const { BotUpsert } = lazily(() => import("../../../views/objects/bot/BotUpsert/BotUpsert"));
const { FocusModeUpsert } = lazily(() => import("../../../views/objects/focusMode/FocusModeUpsert/FocusModeUpsert"));
const { MeetingUpsert } = lazily(() => import("../../../views/objects/meeting/MeetingUpsert/MeetingUpsert"));
const { NoteUpsert } = lazily(() => import("../../../views/objects/note/NoteUpsert/NoteUpsert"));
const { OrganizationUpsert } = lazily(() => import("../../../views/objects/organization/OrganizationUpsert/OrganizationUpsert"));
const { ProjectUpsert } = lazily(() => import("../../../views/objects/project/ProjectUpsert/ProjectUpsert"));
const { QuestionUpsert } = lazily(() => import("../../../views/objects/question/QuestionUpsert/QuestionUpsert"));
const { RoutineUpsert } = lazily(() => import("../../../views/objects/routine/RoutineUpsert/RoutineUpsert"));
const { RunProjectUpsert } = lazily(() => import("../../../views/objects/runProject/RunProjectUpsert/RunProjectUpsert"));
const { RunRoutineUpsert } = lazily(() => import("../../../views/objects/runRoutine/RunRoutineUpsert/RunRoutineUpsert"));
const { SmartContractUpsert } = lazily(() => import("../../../views/objects/smartContract/SmartContractUpsert/SmartContractUpsert"));
const { StandardUpsert } = lazily(() => import("../../../views/objects/standard/StandardUpsert/StandardUpsert"));

type RemoveVersion<T extends string> = T extends `${infer U}Version` ? U : T;
type CreateViewTypes = RemoveVersion<SelectOrCreateObjectType>;

/** 
 * All valid search types for the FindObjectDialog.  
 * Note: The 'Version' types are converted to their non-versioned type.
 */
export type FindObjectTabOption = "All" |
    SearchPageTabOption | `${SearchPageTabOption}` |
    CalendarPageTabOption | `${CalendarPageTabOption}` |
    "ApiVersion" | "NoteVersion" | "ProjectVersion" | "RoutineVersion" | "SmartContractVersion" | "StandardVersion";

// Data for each tab. Ordered by tab index
const tabParams = [
    ...searchViewTabParams,
    {
        Icon: FocusModeIcon,
        titleKey: "FocusMode" as CommonKey,
        searchType: SearchType.FocusMode,
        tabType: CalendarPageTabOption.FocusMode,
        where: () => ({}),
    }, {
        Icon: OrganizationIcon,
        titleKey: "Meeting" as CommonKey,
        searchType: SearchType.Meeting,
        tabType: CalendarPageTabOption.Meeting,
        where: () => ({}),
    }, {
        Icon: RoutineIcon,
        titleKey: "RunRoutine" as CommonKey,
        searchType: SearchType.RunRoutine,
        tabType: CalendarPageTabOption.RunRoutine,
        where: () => ({}),
    }, {
        Icon: ProjectIcon,
        titleKey: "RunProject" as CommonKey,
        searchType: SearchType.RunProject,
        tabType: CalendarPageTabOption.RunProject,
        where: () => ({}),
    },
];

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
    Question: QuestionUpsert,
    Routine: RoutineUpsert,
    RunProject: RunProjectUpsert,
    RunRoutine: RunRoutineUpsert,
    SmartContract: SmartContractUpsert,
    Standard: StandardUpsert,
    User: BotUpsert,
};

const searchTitleId = "search-vrooli-for-link-title";

/**
 * Dialog for selecting or creating an object
 */
export const FindObjectDialog = <Find extends FindObjectDialogType, ObjectType extends SelectOrCreateObject>({
    find,
    handleCancel,
    handleComplete,
    isOpen,
    limitTo,
    onlyVersioned,
    where,
    zIndex,
}: FindObjectDialogProps<Find, ObjectType>) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const filteredTabs = useMemo(() => {
        let filtered = tabParams;
        // Apply limitTo filter
        if (limitTo) filtered = filtered.filter(tab => limitTo.includes(tab.tabType) || limitTo.includes(`${tab.tabType}Version` as FindObjectTabOption));
        // If onlyVersioned, filter tabs which don't have a corresponding versioned search type
        if (onlyVersioned) filtered = filtered.filter(tab => `${tab.tabType}Version` in SearchType);
        return filtered;
    }, [limitTo, onlyVersioned]);
    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
    } = useTabs<FindObjectTabOption>({ tabParams: filteredTabs, display: "dialog" });

    // Dialog for creating new object
    const [createObjectType, setCreateObjectType] = useState<CreateViewTypes | null>(null);

    // Menu for selection object type to create
    const [selectCreateTypeAnchorEl, setSelectCreateTypeAnchorEl] = useState<null | HTMLElement>(null);

    // Info for querying full object data
    const [{ advancedSearchSchema, endpoint }, setSearchParams] = useState<Partial<SearchParams>>({});
    useEffect(() => {
        if (createObjectType !== null && createObjectType in searchTypeToParams) setSearchParams(searchTypeToParams[createObjectType]());
    }, [createObjectType]);
    /**
     * Before closing, remove all URL search params for advanced search
     */
    const onClose = useCallback((item?: ObjectType, versionId?: string) => {
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

    const onCreateStart = useCallback((e: React.MouseEvent<HTMLElement>) => {
        // If tab is 'All', open menu to select type
        if (searchType === SearchType.Popular) setSelectCreateTypeAnchorEl(e.currentTarget);
        // Otherwise, open create dialog for current tab
        else setCreateObjectType(currTab.searchType.replace("Version", "") as CreateViewTypes ?? null);
    }, [currTab, searchType]);
    const onSelectCreateTypeClose = useCallback((type?: SearchType) => {
        if (type) setCreateObjectType(type.replace("Version", "") as CreateViewTypes);
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
    const [getItem, { data: itemData }] = useLazyFetch<FindByIdInput | FindVersionInput, ObjectType>({ endpoint });
    const queryingRef = useRef(false);
    const fetchFullData = useCallback((item: ObjectType, versionId?: string) => {
        if (!endpoint || find !== "Full") return false;
        // Query for full item data, if not already known (would be known if the same item was selected last time)
        if (itemData && itemData.id === item.id && (!versionId || (itemData as any).versionId === versionId)) {
            onClose(itemData);
        } else {
            queryingRef.current = true;
            if (versionId) {
                getItem({ id: versionId, idRoot: item.id });
            } else {
                getItem({ id: item.id });
            }
        }
        // Return false so the list item does not navigate
        return false;
    }, [endpoint, find, itemData, onClose, getItem]);

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
        if (!endpoint) return;
        if (itemData && find === "Full" && queryingRef.current) {
            onClose(itemData);
        }
        queryingRef.current = false;
    }, [onClose, handleCreateClose, itemData, endpoint, find]);

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
        if (!newValue || newValue.__typename === "Shortcut" || newValue.__typename === "Action") return;
        // If object has versions
        if ((newValue as any).versions && (newValue as any).versions.length > 0) {
            // If there is only one version, select it
            if ((newValue as any).versions.length === 1) {
                // Select and close dialog
                onClose(newValue as any, (newValue as any).versions[0].id);
            }
            // Otherwise, set selected object so we can choose which version to link to
            setSelectedObject(newValue as any);
        }
        // Select and close dialog
        onClose(newValue as any);
    }, [onClose]);

    const CreateView = useMemo<((props: UpsertProps<any>) => JSX.Element) | null>(() => {
        if (!createObjectType) return null;
        return (createMap as any)[createObjectType.replace("Version", "")];
    }, [createObjectType]);
    useEffect(() => {
        setSelectCreateTypeAnchorEl(null);
    }, [createObjectType]);

    const focusSearch = useCallback(() => {
        const searchInput = document.getElementById("search-bar-find-object-search-list");
        searchInput?.focus();
    }, []);

    return (
        <>
            {/* Dialog for creating new object type */}
            {CreateView && <CreateView
                isCreate={true}
                isOpen={createObjectType !== null}
                onCompleted={handleCreated}
                onCancel={handleCreateClose}
                overrideObject={{ __typename: createObjectType }}
                zIndex={zIndex + 2}
            />}
            {/* Menu for selecting create object type */}
            {!CreateView && <Menu
                id="select-create-type-menu"
                anchorEl={selectCreateTypeAnchorEl}
                disableScrollLock={true}
                open={Boolean(selectCreateTypeAnchorEl)}
                onClose={() => onSelectCreateTypeClose()}
            >
                {/* Never show 'All' */}
                {tabParams.filter((t) => ![SearchType.Popular].includes(t.searchType)).map(tab => (
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
                    hideTitleOnDesktop={true}
                    onClose={() => { handleCancel(); }}
                    title={t("SearchVrooli")}
                    below={tabs.length > 1 && Boolean(currTab) && <PageTabs
                        ariaLabel="search-tabs"
                        currTab={currTab}
                        fullWidth
                        ignoreIcons={true}
                        onChange={handleTabChange}
                        tabs={tabs}
                    />}
                    zIndex={zIndex}
                />
                <Box sx={{
                    minHeight: "500px",
                    margin: { xs: 0, sm: 2 },
                    paddingTop: 4,
                }}>
                    {/* Search list to find object */}
                    {!selectedObject && <SearchList
                        id="find-object-search-list"
                        canNavigate={() => false}
                        dummyLength={3}
                        onItemClick={onInputSelect}
                        take={20}
                        resolve={(data, type) => {
                            console.log("findobjectdialog resolve data: ", type, data);
                            if (type === SearchType.Popular) return combineSearchResults(data);
                            return parseData(data, type);
                        }}
                        searchType={searchType}
                        zIndex={zIndex}
                        where={where}
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
                                variant="outlined"
                            >
                                Select a different object
                            </Button>
                        </Stack>
                    )}
                </Box>
                <SideActionButtons display="dialog" zIndex={zIndex + 1001}>
                    <ColorIconButton aria-label="filter-list" background={palette.secondary.main} onClick={focusSearch} >
                        <SearchIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </ColorIconButton>
                    <ColorIconButton aria-label="create-new" background={palette.secondary.main} onClick={onCreateStart}>
                        <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </ColorIconButton>
                </SideActionButtons>
            </LargeDialog>
        </>
    );
};
