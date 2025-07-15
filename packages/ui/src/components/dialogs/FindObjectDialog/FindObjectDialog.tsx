// AI_CHECK: TYPE_SAFETY=fixed-dialog-types | LAST: 2025-06-28
// Fixed extensive type safety issues: eliminated 20+ 'as any' casts, added proper types for Version/RootObject,
// improved function return types, and replaced unsafe type assertions with proper type definitions
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import { IconButton } from "../../buttons/IconButton.js";
import ListItemIcon from "@mui/material/ListItemIcon";
import ListItemText from "@mui/material/ListItemText";
import Menu from "@mui/material/Menu";
import MenuItem from "@mui/material/MenuItem";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material";
import { funcFalse, getObjectUrl, type AutocompleteOption, type FindByIdInput, type FindVersionInput, type FormInputBase, type ListObject, type SearchType } from "@vrooli/shared";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { lazily } from "react-lazily";
import { useLazyFetch } from "../../../hooks/useFetch.js";
import { useFindMany } from "../../../hooks/useFindMany.js";
import { useTabs } from "../../../hooks/useTabs.js";
import { Icon, IconCommon } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { removeSearchParams } from "../../../route/searchParams.js";
import { type CrudProps } from "../../../types.js";
import { ELEMENT_CLASSES, Z_INDEX } from "../../../utils/consts.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { findObjectTabParams, searchTypeToParams } from "../../../utils/search/objectToSearch.js";
import { type SearchParams } from "../../../utils/search/schemas/base.js";
import { PageTabs } from "../../PageTabs/PageTabs.js";
import { SideActionsButtons } from "../../buttons/SideActionsButtons.js";
import { Dialog } from "../../dialogs/Dialog/Dialog.js";
import { SearchList, SearchListScrollContainer } from "../../lists/SearchList/SearchList.js";
import { TIDCard } from "../../lists/TIDCard/TIDCard.js";
import { type FindObjectDialogProps, type FindObjectDialogType, type FindObjectType } from "../types.js";

const { ApiUpsert } = lazily(() => import("../../../views/objects/api/ApiUpsert.js"));
const { BotUpsert } = lazily(() => import("../../../views/objects/bot/BotUpsert.js"));
const { DataConverterUpsert } = lazily(() => import("../../../views/objects/dataConverter/DataConverterUpsert.js"));
const { DataStructureUpsert } = lazily(() => import("../../../views/objects/dataStructure/DataStructureUpsert.js"));
const { MeetingUpsert } = lazily(() => import("../../../views/objects/meeting/MeetingUpsert.js"));
const { NoteCrud } = lazily(() => import("../../../views/objects/note/NoteCrud.js"));
const { ProjectCrud } = lazily(() => import("../../../views/objects/project/ProjectCrud.js"));
const { PromptUpsert } = lazily(() => import("../../../views/objects/prompt/PromptUpsert.js"));
const { ReminderCrud } = lazily(() => import("../../../views/objects/reminder/ReminderCrud.js"));
const { RoutineMultiStepCrud } = lazily(() => import("../../../views/objects/routine/RoutineMultiStepCrud.js"));
const { RoutineSingleStepUpsert } = lazily(() => import("../../../views/objects/routine/RoutineSingleStepUpsert.js"));
const { RunUpsert } = lazily(() => import("../../../views/objects/run/RunUpsert.js"));
const { SmartContractUpsert } = lazily(() => import("../../../views/objects/smartContract/SmartContractUpsert.js"));
const { TeamUpsert } = lazily(() => import("../../../views/objects/team/TeamUpsert.js"));

type UpsertView = (props: CrudProps<ListObject>) => JSX.Element;

type Version = {
    id: string;
    versionIndex: number;
    versionLabel: string;
    __typename: string;
} & Partial<ListObject>;

type RootObject = {
    __typename: string;
    id: string;
    versions?: Version[];
} & Partial<ListObject>;

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
    Meeting: MeetingUpsert as UpsertView,
    Note: NoteCrud as UpsertView,
    Project: ProjectCrud as UpsertView,
    Prompt: PromptUpsert as UpsertView,
    Reminder: ReminderCrud as UpsertView,
    RoutineMultiStep: RoutineMultiStepCrud as UpsertView,
    RoutineSingleStep: RoutineSingleStepUpsert as UpsertView,
    Run: RunUpsert as UpsertView,
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
export function convertRootObjectToVersion(item: RootObject, versionId: string): ListObject | undefined {
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
const searchBarId = `${scrollContainerId}-search-bar`;
const searchTitleId = "search-vrooli-for-link-title";
const autoFocusDelayMs = 100;
const dialogStyle = {
    paper: {
        maxWidth: "min(100%, 600px)",
        [`& .${ELEMENT_CLASSES.SearchBar} > div > div`]: {
            borderRadius: 0,
        },
    },
} as const;

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
    const { palette, spacing } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    useEffect(function focusSearchBar() {
        if (isOpen) {
            setTimeout(() => {
                const inputElement = document.getElementById(searchBarId);
                if (inputElement) {
                    inputElement.focus();
                }
            }, autoFocusDelayMs);
        }
    }, [isOpen]);

    const filteredTabs = useMemo(() => getFilteredTabs(limitTo, onlyVersioned), [limitTo, onlyVersioned]);
    const {
        currTab,
        handleTabChange,
        searchType,
        tabs,
    } = useTabs({ id: "find-object-tabs", tabParams: filteredTabs, display: "Dialog" });

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
            const objectUrl = getObjectUrl(item as ListObject);
            const base = `${window.location.origin}${objectUrl}`;
            const url = versionId ? `${base}/${versionId}` : base;
            // If item, store in local storage so we can display it in the link component
            if (item) {
                const rootItem = item as RootObject;
                const itemToStore = versionId ? (rootItem.versions?.find(v => v.id === versionId) ?? {}) : item;
                localStorage.setItem(`objectFromUrl:${url}`, JSON.stringify(itemToStore));
            }
            handleComplete(url as Find extends "Url" ? string : object);
        }
        // Otherwise, return object
        else {
            // Reshape item if needed
            const shapedItem = versionId ? convertRootObjectToVersion(item as RootObject, versionId) : item;
            handleComplete(shapedItem as Find extends "Url" ? string : object);
        }
    }, [find, handleCancel, handleComplete, setLocation]);

    const [selectedObject, setSelectedObject] = useState<{
        versions: { id: string; versionIndex: number, versionLabel: string }[];
    } | null>(null);
    // Reset selected object when dialog is opened/closed
    useEffect(() => {
        setSelectedObject(null);
    }, [isOpen]);
    function removeSelectedObject() {
        setSelectedObject(null);
    }

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
    }, [onSelectCreateTypeClose]);

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
        if (itemData && itemData.id === item.id && (!versionId || itemData.id === versionId)) {
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
            fetchFullData(selectedObject as ObjectBase, version.id);
        } else {
            // Select and close dialog
            onClose(selectedObject as RootObject, version.id);
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
        const rootObject = newValue as RootObject;
        if (rootObject.versions && rootObject.versions.length > 0) {
            // If there is only one version, select it
            if (rootObject.versions.length === 1) {
                // If the full data is requested, fetch the full data for the selected version
                if (find === "Full") {
                    fetchFullData(newValue as ObjectBase, rootObject.versions[0].id);
                }
                // Otherwise, select and close dialog
                else {
                    onClose(rootObject, rootObject.versions[0].id);
                }
            }
            // Otherwise, set selected object so we can choose which version to link to
            setSelectedObject(rootObject);
        }
        // Otherwise, if the full data
        else if (find === "Full") {
            fetchFullData(newValue as ObjectBase);
        }
        // Otherwise, select and close dialog
        else {
            onClose(newValue as RootObject);
        }
    }, [fetchFullData, find, onClose]);

    const CreateView = useMemo<((props: CrudProps<ListObject>) => JSX.Element) | null>(function createViewMemo() {
        if (!createObjectType) return null;
        return createMap[createObjectType] as ((props: CrudProps<ListObject>) => JSX.Element);
    }, [createObjectType]);
    useEffect(() => {
        setSelectCreateTypeAnchorEl(null);
    }, [createObjectType]);

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

    const searchListStyles = useMemo(function searchListStylesMemo() {
        return {
            buttons: {
                borderBottom: `1px solid ${palette.divider}`,
                paddingTop: spacing(0.5),
                padding: spacing(0.5),
            },
            searchBarAndButtonsBox: {
                backgroundColor: palette.background.default,
                position: "sticky",
                top: 0,
                zIndex: Z_INDEX.PageElement + 1,
            },
        };
    }, [palette, spacing]);

    return (
        <>
            {CreateView && <CreateView
                display="Dialog"
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
                    .map(({ iconInfo, key, titleKey }) => {
                        function handleClick() {
                            onSelectCreateTypeClose(key as FindObjectType);
                        }

                        return (
                            <MenuItem
                                key={key}
                                onClick={handleClick}
                            >
                                {iconInfo && <ListItemIcon>
                                    <Icon info={iconInfo} fill={palette.background.textPrimary} />
                                </ListItemIcon>}
                                <ListItemText primary={t(titleKey, { count: 1, defaultValue: titleKey })} />
                            </MenuItem>
                        );
                    })}
            </Menu>}
            {/* Main content */}
            <Dialog
                isOpen={isOpen}
                onClose={handleCancel}
                size="full"
                showCloseButton={true}
            >
                {tabs.length > 1 && Boolean(currTab) && (
                    <Box bgcolor="primary.dark" p={2}>
                        <PageTabs<typeof findObjectTabParams>
                            ariaLabel="Search tabs"
                            currTab={currTab}
                            fullWidth
                            ignoreIcons={true}
                            onChange={handleTabChange}
                            tabs={tabs}
                        />
                    </Box>
                )}
                {!selectedObject && (
                    <SearchListScrollContainer id={scrollContainerId}>
                        <SearchList
                            {...findManyData}
                            canNavigate={funcFalse}
                            display="Dialog"
                            onItemClick={onInputSelect}
                            scrollContainerId={scrollContainerId}
                            searchBarVariant="basic"
                            searchPlaceholder={t("Search") + "..."}
                            sxs={searchListStyles}
                        />
                    </SearchListScrollContainer>
                )}
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
                                    description={getDisplay(version as ListObject).subtitle}
                                    iconInfo={findObjectTabParams.find((t) => t.searchType === version.__typename)?.iconInfo}
                                    key={index}
                                    onClick={handleClick}
                                    title={`${version.versionLabel} - ${getDisplay(version as ListObject).title}`}
                                />
                            );
                        })}
                        {/* Remove selection button */}
                        <Button
                            fullWidth
                            color="secondary"
                            onClick={removeSelectedObject}
                            variant="outlined"
                        >
                            Select a different object
                        </Button>
                    </Stack>
                )}
                <SideActionsButtons display="Dialog">
                    <IconButton
                        aria-label="create-new"
                        onClick={onCreateStart}
                        variant="transparent"
                    >
                        <IconCommon name="Add" />
                    </IconButton>
                </SideActionsButtons>
            </Dialog>
        </>
    );
}
