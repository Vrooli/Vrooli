import { AutocompleteOption, FindByIdInput, FindVersionInput, FormInputBase, ListObject, SearchType, funcFalse, getObjectUrl } from "@local/shared";
import { Button, ListItemIcon, ListItemText, Menu, MenuItem, Stack, Typography, useTheme } from "@mui/material";
import { PageTabs } from "components/PageTabs/PageTabs.js";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons.js";
import { LargeDialog } from "components/dialogs/LargeDialog/LargeDialog.js";
import { SearchList, SearchListScrollContainer } from "components/lists/SearchList/SearchList.js";
import { TIDCard } from "components/lists/TIDCard/TIDCard.js";
import { TopBar } from "components/navigation/TopBar/TopBar.js";
import { useFindMany } from "hooks/useFindMany.js";
import { useLazyFetch } from "hooks/useLazyFetch.js";
import { useTabs } from "hooks/useTabs.js";
import { AddIcon, SearchIcon } from "icons/common.js";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { lazily } from "react-lazily";
import { useLocation } from "route/router.js";
import { removeSearchParams } from "route/searchParams.js";
import { SideActionsButton } from "../../../styles.js";
import { CrudProps } from "../../../types.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { scrollIntoFocusedView } from "../../../utils/display/scroll.js";
import { findObjectTabParams, searchTypeToParams } from "../../../utils/search/objectToSearch.js";
import { SearchParams } from "../../../utils/search/schemas/base.js";
import { FindObjectDialogProps, FindObjectDialogType, FindObjectType } from "../types.js";

const { ApiUpsert } = lazily(() => import("../../../views/objects/api/ApiUpsert.js"));
const { BotUpsert } = lazily(() => import("../../../views/objects/bot/BotUpsert.js"));
const { DataConverterUpsert } = lazily(() => import("../../../views/objects/dataConverter/DataConverterUpsert.js"));
const { DataStructureUpsert } = lazily(() => import("../../../views/objects/dataStructure/DataStructureUpsert.js"));
const { FocusModeUpsert } = lazily(() => import("../../../views/objects/focusMode/FocusModeUpsert.js"));
const { MeetingUpsert } = lazily(() => import("../../../views/objects/meeting/MeetingUpsert.js"));
const { NoteCrud } = lazily(() => import("../../../views/objects/note/NoteCrud.js"));
const { ProjectCrud } = lazily(() => import("../../../views/objects/project/ProjectCrud.js"));
const { PromptUpsert } = lazily(() => import("../../../views/objects/prompt/PromptUpsert.js"));
const { QuestionUpsert } = lazily(() => import("../../../views/objects/question/QuestionUpsert.js"));
const { ReminderCrud } = lazily(() => import("../../../views/objects/reminder/ReminderCrud.js"));
const { RoutineMultiStepCrud } = lazily(() => import("../../../views/objects/routine/RoutineMultiStepCrud.js"));
const { RoutineSingleStepUpsert } = lazily(() => import("../../../views/objects/routine/RoutineSingleStepUpsert.js"));
const { RunProjectUpsert } = lazily(() => import("../../../views/objects/runProject/RunProjectUpsert.js"));
const { RunRoutineUpsert } = lazily(() => import("../../../views/objects/runRoutine/RunRoutineUpsert.js"));
const { SmartContractUpsert } = lazily(() => import("../../../views/objects/smartContract/SmartContractUpsert.js"));
const { TeamUpsert } = lazily(() => import("../../../views/objects/team/TeamUpsert.js"));

type UpsertView = (props: CrudProps<ListObject>) => JSX.Element;

type Version = {
    id: string;
    [key: string]: any;
}
type RootObject = {
    versions?: Version[];
    [key: string]: any;
}

type ObjectBase = {
    __typename: string;
    id: string;
}

/**
 * Maps object types to create components (excluding "User" and types that end with 'Version')
 */
const createMap: { [K in FindObjectType]: UpsertView } = {
    Api: ApiUpsert as UpsertView,
    Bot: BotUpsert as UpsertView,
    DataConverter: DataConverterUpsert as UpsertView,
    DataStructure: DataStructureUpsert as UpsertView,
    FocusMode: FocusModeUpsert as UpsertView,
    Meeting: MeetingUpsert as UpsertView,
    Note: NoteCrud as UpsertView,
    Project: ProjectCrud as UpsertView,
    Prompt: PromptUpsert as UpsertView,
    Question: QuestionUpsert as UpsertView,
    Reminder: ReminderCrud as UpsertView,
    RoutineMultiStep: RoutineMultiStepCrud as UpsertView,
    RoutineSingleStep: RoutineSingleStepUpsert as UpsertView,
    RunProject: RunProjectUpsert as UpsertView,
    RunRoutine: RunRoutineUpsert as UpsertView,
    SmartContract: SmartContractUpsert as UpsertView,
    Team: TeamUpsert as UpsertView,
    User: BotUpsert as UpsertView,
};

/**
 * Determines which tabs to display
 * @param limitTo Limits tabs to only these types
 * @param useVersioned If true, uses tabs for objects that have versions 
 * @returns The filtered tabs
 */
export function getFilteredTabs(
    limitTo: readonly FindObjectType[] | undefined,
    onlyVersioned: boolean | undefined,
) {
    let filtered = findObjectTabParams;
    // Apply limitTo filter if it's a non-empty array
    if (Array.isArray(limitTo) && limitTo.length > 0) filtered = filtered.filter(tab => limitTo.includes(tab.key) || limitTo.includes(`${tab.key}Version`));
    // If onlyVersioned is true, filter out non-versioned tabs
    const typesWithVersions = ["Api", "DataConverter", "DataStructure", "Note", "Project", "Prompt", "Routine", "SmartContract", "Standard"];
    if (onlyVersioned) filtered = filtered.filter(tab => typesWithVersions.includes(tab.key) || typesWithVersions.includes(tab.key.replace("Version", "")));
    return filtered;
}

/**
 * Retrieves a versioned view of a root object based on the specified version ID.
 * In other wrods, this function "flips" the object structure so that the versioned data is the root object.
 * 
 * @param rootObject The root object which contains potential versions.
 * @param versionId - The identifier for the version to retrieve.
 * @returns The versioned object, or undefined if the version ID is not found.
 */
export function convertRootObjectToVersion(item: RootObject, versionId: string): any {
    if (versionId) {
        // Find the specific version based on versionId
        const version = item.versions?.find(v => v.id === versionId);
        if (version) {
            // Destructure to separate versions array from the rest of the root object
            const { versions, ...rootData } = item;
            // Return from the perspective of the version
            return { ...version, root: rootData };
        }
    }
    // If versionId is not provided or no version matches, return undefined or the original item
    return undefined;
}


const scrollContainerId = "find-object-search-scroll";
const searchTitleId = "search-vrooli-for-link-title";
const dialogStyle = { paper: { maxWidth: "min(100%, 600px)" } } as const;

/**
 * Dialog for selecting or creating an object
 */
export function FindObjectDialog<Find extends FindObjectDialogType>({
    find,
    handleCancel,
    handleComplete,
    isOpen,
    limitTo,
    onlyVersioned,
    where,
}: FindObjectDialogProps<Find>) {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const filteredTabs = useMemo(() => getFilteredTabs(limitTo, onlyVersioned), [limitTo, onlyVersioned]);
    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
    } = useTabs({ id: "find-object-tabs", tabParams: filteredTabs, display: "dialog" });

    // Dialog for creating new object
    const [createObjectType, setCreateObjectType] = useState<FindObjectType | null>(null);

    // Menu for selection object type to create
    const [selectCreateTypeAnchorEl, setSelectCreateTypeAnchorEl] = useState<null | HTMLElement>(null);

    // Info for querying full object data
    const advancedSearchSchemaRef = useRef<SearchParams["advancedSearchSchema"] | undefined>();
    const findOneEndpointRef = useRef<string | undefined>();
    useEffect(() => {
        let searchParams: SearchParams | undefined;
        if (createObjectType !== null && createObjectType in searchTypeToParams) {
            searchParams = searchTypeToParams[createObjectType]();
        } else if (currTab.searchType in searchTypeToParams) {
            searchParams = searchTypeToParams[currTab.searchType]();
        }
        if (searchParams) {
            advancedSearchSchemaRef.current = searchParams.advancedSearchSchema;
            findOneEndpointRef.current = searchParams.findOneEndpoint;
        }
    }, [createObjectType, currTab.searchType]);
    /**
     * Before closing, remove all URL search params for advanced search
     */
    const onClose = useCallback((item?: object, versionId?: string) => {
        // Clear search params
        const advancedSearchFields = advancedSearchSchemaRef.current?.elements?.filter(f => Object.prototype.hasOwnProperty.call(f, "fieldName")).map(f => (f as FormInputBase).fieldName) ?? [];
        const basicSearchFields = ["advanced", "sort", "time"];
        removeSearchParams(setLocation, [...advancedSearchFields, ...basicSearchFields]);
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
            // Reshape item if needed
            const shapedItem = versionId ? convertRootObjectToVersion(item as RootObject, versionId) : item;
            handleComplete(shapedItem as any);
        }
    }, [find, handleCancel, handleComplete, setLocation]);

    const [selectedObject, setSelectedObject] = useState<{
        versions: { id: string; versionIndex: number, versionLabel: string }[];
    } | null>(null);
    // Reset selected object when dialog is opened/closed
    useEffect(() => {
        setSelectedObject(null);
    }, [isOpen]);

    const onCreateStart = useCallback((e: React.MouseEvent<HTMLElement>) => {
        // If tab is 'All', open menu to select type
        if (searchType === "Popular") setSelectCreateTypeAnchorEl(e.currentTarget);
        // Otherwise, open create dialog for current tab
        else setCreateObjectType(currTab.searchType.replace("Version", "") as FindObjectType ?? null); //TODO prob breaks for Code and Standard
    }, [currTab, searchType]);
    const onSelectCreateTypeClose = useCallback((type?: FindObjectType) => {
        if (type) setCreateObjectType(type);
        else setSelectCreateTypeAnchorEl(null);
    }, []);
    const onSelectCreateTypeCloseNoArg = useCallback(() => {
        onSelectCreateTypeClose();
    }, []);

    const handleCreated = useCallback((item: object) => {
        onClose(item);
        setCreateObjectType(null);
    }, [onClose]);
    const handleCreateClose = useCallback(() => {
        setCreateObjectType(null);
    }, []);

    // If item selected from search AND find is 'Object', query for full data
    const [getItem, { data: itemData }] = useLazyFetch<FindByIdInput | FindVersionInput, ObjectBase>({});
    const queryingRef = useRef(false);
    const fetchFullData = useCallback((item: ObjectBase, versionId?: string) => {
        const appendVersion = typeof versionId === "string" && !item.__typename.endsWith("Version");
        const { findOneEndpoint } = searchTypeToParams[`${item.__typename}${appendVersion ? "Version" : ""}` as SearchType]!();
        if (!findOneEndpoint || find !== "Full") return false;
        // Query for full item data, if not already known (would be known if the same item was selected last time)
        if (itemData && itemData.id === item.id && (!versionId || (itemData as any).versionId === versionId)) {
            onClose(itemData);
        } else {
            queryingRef.current = true;
            if (versionId) {
                getItem({ id: versionId }, { endpointOverride: findOneEndpoint });
            } else {
                getItem({ id: item.id }, { endpointOverride: findOneEndpoint });
            }
        }
        // Return false so the list item does not navigate
        return false;
    }, [find, itemData, onClose, getItem]);

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
        if (!findOneEndpointRef.current) return;
        if (itemData && find === "Full" && queryingRef.current) {
            onClose(itemData);
        }
        queryingRef.current = false;
    }, [onClose, handleCreateClose, itemData, find]);

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
        console.log("oninputselect", newValue, find);
        // If value is not an object, return;
        if (!newValue || newValue.__typename === "Shortcut" || newValue.__typename === "Action") return;
        // If object has versions
        if ((newValue as any).versions && (newValue as any).versions.length > 0) {
            // If there is only one version, select it
            if ((newValue as any).versions.length === 1) {
                // If the full data is requested, fetch the full data for the selected version
                if (find === "Full") {
                    fetchFullData(newValue as any, (newValue as any).versions[0].id);
                }
                // Otherwise, select and close dialog
                else {
                    onClose(newValue as any, (newValue as any).versions[0].id);
                }
            }
            // Otherwise, set selected object so we can choose which version to link to
            setSelectedObject(newValue as any);
        }
        // Otherwise, if the full data
        else if (find === "Full") {
            fetchFullData(newValue as any);
        }
        // Otherwise, select and close dialog
        else {
            onClose(newValue as any);
        }
    }, [fetchFullData, find, onClose]);

    const CreateView = useMemo<((props: CrudProps<any>) => JSX.Element) | null>(function createViewMemo() {
        if (!createObjectType) return null;
        return createMap[createObjectType] as ((props: CrudProps<any>) => JSX.Element);
    }, [createObjectType]);
    useEffect(() => {
        setSelectCreateTypeAnchorEl(null);
    }, [createObjectType]);

    function focusSearch() { scrollIntoFocusedView("search-bar-find-object-search-list"); }

    const findManyData = useFindMany<ListObject>({
        controlsUrl: false,
        searchType,
        take: 20,
        where,
    });

    const createViewOverrideObject = useMemo(function createViewOverrideObjectMemo() {
        if (!createObjectType) return undefined;
        let __typename: string = createObjectType;
        if (__typename === "DataConverter" || __typename === "SmartContract") {
            __typename = "Code";
        } else if (__typename === "DataStructure" || __typename === "Prompt") {
            __typename = "Standard";
        }
        return createObjectType ? { __typename } : undefined;
    }, [createObjectType]);

    return (
        <>
            {CreateView && <CreateView
                display="dialog"
                isCreate={true}
                isOpen={createObjectType !== null}
                onCancel={handleCreateClose}
                onClose={handleCreateClose}
                onCompleted={handleCreated}
                onDeleted={handleCreateClose}
                overrideObject={createViewOverrideObject}
            />}
            {/* Menu for selecting create object type */}
            {!CreateView && <Menu
                id="select-create-type-menu"
                anchorEl={selectCreateTypeAnchorEl}
                disableScrollLock={true}
                open={Boolean(selectCreateTypeAnchorEl)}
                onClose={onSelectCreateTypeCloseNoArg}
            >
                {/* Never show 'All' */}
                {findObjectTabParams.filter((t) => !["Popular"]
                    .includes(t.searchType as SearchType))
                    .map(({ Icon, key, titleKey }) => {
                        function handleClick() {
                            onSelectCreateTypeClose(key as FindObjectType);
                        }

                        return (
                            <MenuItem
                                key={key}
                                onClick={handleClick}
                            >
                                {Icon && <ListItemIcon>
                                    <Icon fill={palette.background.textPrimary} />
                                </ListItemIcon>}
                                <ListItemText primary={t(titleKey, { count: 1, defaultValue: titleKey })} />
                            </MenuItem>
                        );
                    })}
            </Menu>}
            {/* Main content */}
            <LargeDialog
                id="resource-find-object-dialog"
                isOpen={isOpen}
                onClose={handleCancel}
                titleId={searchTitleId}
                sxs={dialogStyle}
            >
                <SearchListScrollContainer id={scrollContainerId}>
                    <TopBar
                        display="dialog"
                        onClose={handleCancel}
                        title={t("SearchVrooli")}
                        titleBehaviorDesktop="ShowIn"
                        below={tabs.length > 1 && Boolean(currTab) && <PageTabs<typeof findObjectTabParams>
                            ariaLabel="Search tabs"
                            currTab={currTab}
                            fullWidth
                            ignoreIcons={true}
                            onChange={handleTabChange}
                            tabs={tabs}
                        />}
                    />

                    {/* Search list to find object */}
                    {!selectedObject && <SearchList
                        {...findManyData}
                        canNavigate={funcFalse}
                        display="dialog"
                        onItemClick={onInputSelect}
                        scrollContainerId={scrollContainerId}
                    />}
                    {/* If object selected (and supports versioning), display buttons to select version */}
                    {selectedObject && (
                        <Stack spacing={2} direction="column" m={2}>
                            <Typography variant="h6" mt={2} textAlign="center">
                                Select a version
                            </Typography>
                            {[...(selectedObject.versions ?? [])].sort((a, b) => b.versionIndex - a.versionIndex).map((version, index) => {
                                function handleClick() {
                                    onVersionSelect(version);
                                }

                                return (
                                    <TIDCard
                                        buttonText={t("Select")}
                                        description={getDisplay(version as any).subtitle}
                                        key={index}
                                        Icon={findObjectTabParams.find((t) => t.searchType === (version as any).__typename)?.Icon}
                                        onClick={handleClick}
                                        title={`${version.versionLabel} - ${getDisplay(version as any).title}`}
                                    />
                                );
                            })}
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
                    <SideActionsButtons display="dialog">
                        <SideActionsButton aria-label="filter-list" onClick={focusSearch}>
                            <SearchIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                        </SideActionsButton>
                        <SideActionsButton aria-label="create-new" onClick={onCreateStart}>
                            <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                        </SideActionsButton>
                    </SideActionsButtons>
                </SearchListScrollContainer>
            </LargeDialog>
        </>
    );
}
