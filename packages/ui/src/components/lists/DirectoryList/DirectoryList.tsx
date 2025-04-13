import { ApiVersion, CodeVersion, Count, DeleteManyInput, DeleteType, ListObject, ModelType, NoteVersion, ProjectVersion, ProjectVersionDirectory, RoutineVersion, Session, StandardVersion, Team, TimeFrame, endpointsActions, getObjectUrl, isOfType } from "@local/shared";
import { Box, Button, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { SearchButtonsProps } from "../../../components/buttons/types.js";
import { FindObjectDialog } from "../../../components/dialogs/FindObjectDialog/FindObjectDialog.js";
import { ObjectActionMenu } from "../../../components/dialogs/ObjectActionMenu/ObjectActionMenu.js";
import { ObjectList } from "../../../components/lists/ObjectList/ObjectList.js";
import { TextLoading } from "../../../components/lists/TextLoading/TextLoading.js";
import { ObjectListActions } from "../../../components/lists/types.js";
import { SessionContext } from "../../../contexts/session.js";
import { UsePressEvent, usePress } from "../../../hooks/gestures.js";
import { useBulkObjectActions, useObjectActions } from "../../../hooks/objectActions.js";
import { useLazyFetch } from "../../../hooks/useLazyFetch.js";
import { useObjectContextMenu } from "../../../hooks/useObjectContextMenu.js";
import { useSelectableList } from "../../../hooks/useSelectableList.js";
import { Icon, IconCommon } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { multiLineEllipsis, noSelect } from "../../../styles.js";
import { ArgsType } from "../../../types.js";
import { ObjectAction } from "../../../utils/actions/objectActions.js";
import { DUMMY_LIST_LENGTH, DUMMY_LIST_LENGTH_SHORT } from "../../../utils/consts.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { getUserLanguages } from "../../../utils/display/translationTools.js";
import { PubSub } from "../../../utils/pubsub.js";
import { SortButton, StyledSearchButton, TimeButton } from "../../buttons/SearchButtons.js";
import { ListContainer } from "../../containers/ListContainer.js";
import { DirectoryCardProps, DirectoryItem, DirectoryListHorizontalProps, DirectoryListProps, DirectoryListVerticalProps } from "../types.js";

export enum DirectoryListSortBy {
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    NameAsc = "NameAsc",
    NameDesc = "NameDesc",
}

// Helper function to safely parse dates and return a timestamp
function parseDate(dateString: string) {
    const date = new Date(dateString);
    return !isFinite(date.getTime()) ? 0 : date.getTime(); // Return 0 if date is invalid
}

export function initializeDirectoryList(
    directory: ProjectVersionDirectory | null | undefined,
    sortBy: DirectoryListSortBy | `${DirectoryListSortBy}`,
    timeFrame: TimeFrame | undefined,
    session: Session | null | undefined,
): DirectoryItem[] {
    if (!directory) return [];
    let items = [
        ...directory.childApiVersions,
        ...directory.childCodeVersions,
        ...directory.childNoteVersions,
        ...directory.childProjectVersions,
        ...directory.childRoutineVersions,
        ...directory.childStandardVersions,
        ...directory.childTeams,
    ];
    const userLanguages = getUserLanguages(session);

    // Filter items based on time frame
    if (timeFrame?.after || timeFrame?.before) {
        const afterTime = timeFrame?.after ? parseDate(timeFrame.after) : -Infinity;
        const beforeTime = timeFrame?.before ? parseDate(timeFrame.before) : Infinity;

        items = items.filter(item => {
            const itemTime = parseDate(item.created_at); // Assuming filtering by creation date
            return itemTime > afterTime && itemTime < beforeTime;
        });
    }

    return items.sort((a, b) => {
        // Extracting titles for name-based sorting
        const { title: aTitle } = getDisplay(a, userLanguages);
        const { title: bTitle } = getDisplay(b, userLanguages);

        switch (sortBy) {
            case "DateCreatedAsc":
                return parseDate(a.created_at) - parseDate(b.created_at);
            case "DateCreatedDesc":
                return parseDate(b.created_at) - parseDate(a.created_at);
            case "DateUpdatedAsc":
                return parseDate(a.updated_at) - parseDate(b.updated_at);
            case "DateUpdatedDesc":
                return parseDate(b.updated_at) - parseDate(a.updated_at);
            case "NameAsc":
                return aTitle.localeCompare(bTitle);
            case "NameDesc":
                return bTitle.localeCompare(aTitle);
            default:
                // Default fallback to name ascending if no valid sort by is provided
                return aTitle.localeCompare(bTitle);
        }
    });
}

type ViewMode = "card" | "list";
type DirectorySearchButtonsProps = Omit<SearchButtonsProps, "advancedSearchParams" | "advancedSearchSchema" | "controlsUrl" | "searchType" | "setAdvancedSearchParams" | "setSortBy" | "sortByOptions"> & {
    viewMode: ViewMode;
    setSortBy: (sortBy: DirectoryListSortBy) => unknown;
    setViewMode: (mode: ViewMode) => unknown;
}

function DirectorySearchButtons({
    setSortBy,
    setTimeFrame,
    setViewMode,
    sortBy,
    sx,
    timeFrame,
    viewMode,
}: DirectorySearchButtonsProps) {
    const { t } = useTranslation();

    const toggleViewMode = useCallback(function toggleViewModeCallback() {
        setViewMode(viewMode === "list" ? "card" : "list");
    }, [setViewMode, viewMode]);

    return (
        <Box sx={{ display: "flex", justifyContent: "center", alignItems: "center", marginBottom: 1, ...sx }}>
            <SortButton
                options={DirectoryListSortBy}
                setSortBy={setSortBy as ((sortBy: string) => unknown)}
                sortBy={sortBy}
            />
            <TimeButton
                setTimeFrame={setTimeFrame}
                timeFrame={timeFrame}
            />
            {/* Card/list toggle TODO */}
            <Tooltip title={t(viewMode === "list" ? "CardModeSwitch" : "ListModeSwitch")} placement="top">
                <StyledSearchButton
                    active={false}
                    id="card-list-toggle-button"
                    onClick={toggleViewMode}
                >
                    <IconCommon name={viewMode === "list" ? "List" : "Grid"} fill="secondary.main" />
                </StyledSearchButton>
            </Tooltip>
        </Box>
    );
}

const CardBox = styled(Box)(({ theme }) => ({
    ...noSelect,
    alignItems: "center",
    background: theme.palette.background.paper,
    borderRadius: theme.spacing(3),
    boxShadow: theme.shadows[2],
    color: theme.palette.background.textSecondary,
    display: "flex",
    flexDirection: "row",
    gap: theme.spacing(1),
    height: "40px",
    cursor: "pointer",
    maxWidth: "250px",
    minWidth: "120px",
    padding: theme.spacing(1),
    position: "relative",
    textDecoration: "none",
    width: "fit-content",
    "&:hover": {
        filter: "brightness(120%)",
        transition: "filter 0.2s",
    },
}));

/**
 * Unlike ResourceCard, these aren't draggable. This is because the objects 
 * are not stored in an order - they are stored by object type
 */
export function DirectoryCard({
    canUpdate,
    data,
    onContextMenu,
    onDelete,
}: DirectoryCardProps) {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [showIcons, setShowIcons] = useState(false);

    const { title, subtitle } = useMemo(() => getDisplay(data, getUserLanguages(session)), [data, session]);

    const iconInfo = useMemo(() => {
        if (!data || !data.__typename) return { name: "Help", type: "Common" } as const;
        if (isOfType(data, "ApiVersion")) return { name: "Api", type: "Common" } as const;
        if (isOfType(data, "CodeVersion")) return { name: "Terminal", type: "Common" } as const;
        if (isOfType(data, "NoteVersion")) return { name: "Note", type: "Common" } as const;
        if (isOfType(data, "ProjectVersion")) return { name: "Project", type: "Common" } as const;
        if (isOfType(data, "RoutineVersion")) return { name: "Routine", type: "Routine" } as const;
        if (isOfType(data, "StandardVersion")) return { name: "Standard", type: "Common" } as const;
        if (isOfType(data, "Team")) return { name: "Team", type: "Common" } as const;
        return { name: "Help", type: "Common" } as const;
    }, [data]);

    const href = useMemo(() => data ? getObjectUrl(data) : "#", [data]);
    const handleClick = useCallback(({ target }: UsePressEvent) => {
        // Check if delete button was clicked
        const targetId: string | undefined = target.id;
        if (targetId && targetId.startsWith("delete-")) {
            onDelete?.(data);
        }
        else {
            // Navigate to object
            setLocation(href);
        }
    }, [data, href, onDelete, setLocation]);
    const handleContextMenu = useCallback(({ target }: UsePressEvent) => {
        onContextMenu(target, data);
    }, [onContextMenu, data]);

    const handleHover = useCallback(() => {
        if (canUpdate) {
            setShowIcons(true);
        }
    }, [canUpdate]);

    const handleHoverEnd = useCallback(() => { setShowIcons(false); }, []);

    const pressEvents = usePress({
        onLongPress: handleContextMenu,
        onClick: handleClick,
        onHover: handleHover,
        onHoverEnd: handleHoverEnd,
        onRightClick: handleContextMenu,
        hoverDelay: 100,
    });

    return (
        <Tooltip placement="top" title={`${subtitle ? subtitle + " - " : ""}${href}`}>
            <CardBox
                {...pressEvents}
                component="a"
                href={href}
                onClick={(e) => e.preventDefault()}
            >
                {/* delete icon, only visible on hover */}
                {showIcons && (
                    <>
                        <Tooltip title={t("Delete")}>
                            <IconButton
                                id='delete-icon-button'
                                sx={{ background: palette.error.main, position: "absolute", top: 4, right: 4 }}
                            >
                                <IconCommon name="Delete" id='delete-icon' fill="secondary.contrastText" />
                            </IconButton>
                        </Tooltip>
                    </>
                )}
                {/* Content */}
                <Stack
                    direction="column"
                    justifyContent="center"
                    alignItems="center"
                    sx={{
                        height: "100%",
                        overflow: "hidden",
                        textOverflow: "ellipsis",
                    }}
                >
                    <Icon
                        fill="white"
                        info={iconInfo}
                    />
                    <Typography
                        gutterBottom
                        variant="body2"
                        component="h3"
                        sx={{
                            ...multiLineEllipsis(2),
                            textAlign: "center",
                            lineBreak: title ? "auto" : "anywhere", // Line break anywhere only if showing link
                        }}
                    >
                        {title}
                    </Typography>
                </Stack>
            </CardBox>
        </Tooltip>
    );
}

function LoadingCard() {
    return (
        <CardBox>
            <Stack
                direction="column"
                justifyContent="center"
                alignItems="center"
                sx={{
                    height: "100%",
                    overflow: "hidden",
                }}
            >
                <TextLoading size="subheader" lines={2} sx={{ width: "70%", opacity: "0.5" }} />
            </Stack>
        </CardBox>
    );
}

const contextActionsExcluded = [ObjectAction.Comment, ObjectAction.FindInPage] as const;

export function DirectoryListHorizontal({
    canUpdate = true,
    isEditing,
    list,
    loading,
    onAction,
    onDelete,
    openAddDialog,
}: DirectoryListHorizontalProps) {
    const { t } = useTranslation();
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const contextData = useObjectContextMenu();
    const actionData = useObjectActions({
        canNavigate: () => !isEditing,
        isListReorderable: isEditing,
        objectType: contextData.object?.__typename as ModelType | undefined,
        onAction,
        setLocation,
        ...contextData,
    });

    return (
        <>
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={contextData.anchorEl}
                exclude={contextActionsExcluded} // Find in page only relevant when viewing object - not in list. And shouldn't really comment without viewing full page
                object={contextData.object}
                onClose={contextData.closeContextMenu}
            />
            <Box minWidth={120} sx={{
                display: "flex",
                gap: 2,
                width: "100%",
                marginLeft: "auto",
                marginRight: "auto",
                paddingTop: 1,
                paddingBottom: 1,
                overflowX: "auto",
                // ...sxs?.list,
            }}>
                {/* Items */}
                {list.map((c: DirectoryItem, index) => (
                    <DirectoryCard
                        canUpdate={canUpdate}
                        key={`directory-item-card-${index}`}
                        data={c}
                        onContextMenu={contextData.handleContextMenu}
                        onDelete={onDelete}
                        aria-owns={contextData.object?.id}
                    />
                ))}
                {/* Dummy cards when loading */}
                {
                    loading && !Array.isArray(list) && Array.from(Array(DUMMY_LIST_LENGTH_SHORT).keys()).map((i) => (
                        <LoadingCard key={`directory-item-card-${i}`} />
                    ))
                }
                {/* Add button */}
                {(canUpdate ? <Tooltip placement="top" title={t("AddItem")}>
                    <CardBox
                        onClick={openAddDialog}
                        aria-label={t("AddItem")}
                        sx={{
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "center",
                        }}
                    >
                        <IconCommon name="Link" fill="secondary.contrastText" size={56} />
                    </CardBox>
                </Tooltip> : null)}
            </Box>
        </>
    );
}

export function DirectoryListVertical({
    canUpdate = true,
    handleToggleSelect,
    isEditing,
    isSelecting,
    list,
    loading,
    onAction,
    onClick,
    openAddDialog,
    selectedData,
}: DirectoryListVerticalProps) {
    const { t } = useTranslation();

    return (
        <>
            <ListContainer
                isEmpty={list.length === 0 && !loading}
            >
                <ObjectList
                    canNavigate={() => !isEditing}
                    dummyItems={new Array(DUMMY_LIST_LENGTH).fill("Project")}
                    handleToggleSelect={handleToggleSelect as (item: ListObject) => unknown}
                    hideUpdateButton={isEditing}
                    isSelecting={isSelecting}
                    items={list}
                    keyPrefix="directory-list-item"
                    loading={loading ?? false}
                    onAction={onAction}
                    onClick={onClick as (item: ListObject) => unknown}
                    selectedItems={selectedData}
                />
            </ListContainer>
            {/* Add button */}
            {canUpdate && <Box
                margin="auto"
                width="100%"
            >
                <Button
                    fullWidth onClick={openAddDialog}
                    startIcon={<IconCommon name="Add" />}
                    variant="outlined"
                >{t("AddItem")}</Button>
            </Box>}
        </>
    );
}

export function DirectoryList(props: DirectoryListProps) {
    const { canUpdate, directory, handleUpdate, mutate } = props;
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();

    const [viewMode, setViewMode] = useState<ViewMode>("list");
    const [sortBy, setSortBy] = useState<DirectoryListSortBy>(DirectoryListSortBy.NameAsc);
    const [timeFrame, setTimeFrame] = useState<TimeFrame | undefined>(undefined);

    const list = useMemo(() => initializeDirectoryList(
        directory,
        sortBy,
        timeFrame,
        session,
    ), [directory, session, sortBy, timeFrame]);

    const [isEditing, setIsEditing] = useState(false);
    useEffect(() => {
        if (!canUpdate) setIsEditing(false);
    }, [canUpdate]);

    // Add dialog
    const [isAddDialogOpen, setIsAddDialogOpen] = useState(false);
    const openAddDialog = useCallback(() => { setIsAddDialogOpen(true); }, []);
    const closeAddDialog = useCallback(() => { setIsAddDialogOpen(false); }, []);

    const onAdd = useCallback((item: DirectoryItem) => {
        closeAddDialog();
        if (!directory) return;
        // Dont add duplicates
        if (list.some(r => r.id === item.id)) {
            PubSub.get().publish("snack", { message: "Item already added", severity: "Warning" });
            return;
        }
        handleUpdate?.({
            ...directory,
            childApiVersions: (isOfType(item, "ApiVersion") ? [...directory.childApiVersions, item] : directory.childApiVersions) as ApiVersion[],
            childCodeVersions: (isOfType(item, "CodeVersion") ? [...directory.childCodeVersions, item] : directory.childCodeVersions) as CodeVersion[],
            childNoteVersions: (isOfType(item, "NoteVersion") ? [...directory.childNoteVersions, item] : directory.childNoteVersions) as NoteVersion[],
            childProjectVersions: (isOfType(item, "ProjectVersion") ? [...directory.childProjectVersions, item] : directory.childProjectVersions) as ProjectVersion[],
            childRoutineVersions: (isOfType(item, "RoutineVersion") ? [...directory.childRoutineVersions, item] : directory.childRoutineVersions) as RoutineVersion[],
            childStandardVersions: (isOfType(item, "StandardVersion") ? [...directory.childStandardVersions, item] : directory.childStandardVersions) as StandardVersion[],
            childTeams: (isOfType(item, "Team") ? [...directory.childTeams, item] : directory.childTeams) as Team[],
        });
    }, [closeAddDialog, directory, handleUpdate, list]);

    const [deleteMutation] = useLazyFetch<DeleteManyInput, Count>(endpointsActions.deleteMany);
    const onDelete = useCallback((item: DirectoryItem) => {
        if (!directory) return;
        if (mutate) {
            fetchLazyWrapper<DeleteManyInput, Count>({
                fetch: deleteMutation,
                inputs: { objects: [{ id: item.id, objectType: item.__typename as DeleteType }] },
                onSuccess: () => {
                    handleUpdate?.({
                        ...directory,
                        childApiVersions: isOfType(item, "ApiVersion") ? directory.childApiVersions.filter(i => i.id !== item.id) : directory.childApiVersions,
                        childCodeVersions: isOfType(item, "CodeVersion") ? directory.childCodeVersions.filter(i => i.id !== item.id) : directory.childCodeVersions,
                        childNoteVersions: isOfType(item, "NoteVersion") ? directory.childNoteVersions.filter(i => i.id !== item.id) : directory.childNoteVersions,
                        childProjectVersions: isOfType(item, "ProjectVersion") ? directory.childProjectVersions.filter(i => i.id !== item.id) : directory.childProjectVersions,
                        childRoutineVersions: isOfType(item, "RoutineVersion") ? directory.childRoutineVersions.filter(i => i.id !== item.id) : directory.childRoutineVersions,
                        childStandardVersions: isOfType(item, "StandardVersion") ? directory.childStandardVersions.filter(i => i.id !== item.id) : directory.childStandardVersions,
                        childTeams: isOfType(item, "Team") ? directory.childTeams.filter(i => i.id !== item.id) : directory.childTeams,
                    });
                },
            });
        }
        else {
            handleUpdate?.({
                ...directory,
                childApiVersions: isOfType(item, "ApiVersion") ? directory.childApiVersions.filter(i => i.id !== item.id) : directory.childApiVersions,
                childCodeVersions: isOfType(item, "CodeVersion") ? directory.childCodeVersions.filter(i => i.id !== item.id) : directory.childCodeVersions,
                childNoteVersions: isOfType(item, "NoteVersion") ? directory.childNoteVersions.filter(i => i.id !== item.id) : directory.childNoteVersions,
                childProjectVersions: isOfType(item, "ProjectVersion") ? directory.childProjectVersions.filter(i => i.id !== item.id) : directory.childProjectVersions,
                childRoutineVersions: isOfType(item, "RoutineVersion") ? directory.childRoutineVersions.filter(i => i.id !== item.id) : directory.childRoutineVersions,
                childStandardVersions: isOfType(item, "StandardVersion") ? directory.childStandardVersions.filter(i => i.id !== item.id) : directory.childStandardVersions,
                childTeams: isOfType(item, "Team") ? directory.childTeams.filter(i => i.id !== item.id) : directory.childTeams,
            });
        }
    }, [deleteMutation, directory, handleUpdate, mutate]);

    const {
        isSelecting,
        handleToggleSelecting,
        handleToggleSelect,
        selectedData,
        setIsSelecting,
        setSelectedData,
    } = useSelectableList<DirectoryItem>(list);
    const { onBulkActionStart, BulkDeleteDialogComponent } = useBulkObjectActions<DirectoryItem>({
        allData: list,
        selectedData,
        setAllData: (data) => {
            if (!directory) return;
            handleUpdate?.({
                ...directory,
                childApiVersions: data.filter(i => isOfType(i, "ApiVersion")),
                childNoteVersions: data.filter(i => isOfType(i, "NoteVersion")),
                childTeams: data.filter(i => isOfType(i, "Team")),
                childProjectVersions: data.filter(i => isOfType(i, "ProjectVersion")),
                childRoutineVersions: data.filter(i => isOfType(i, "RoutineVersion")),
                childCodeVersions: data.filter(i => isOfType(i, "CodeVersion")),
                childStandardVersions: data.filter(i => isOfType(i, "StandardVersion")),
            } as ProjectVersionDirectory);
        },
        setSelectedData: (data) => {
            setSelectedData(data);
            setIsSelecting(false);
        },
        setLocation,
    });
    const onAction = useCallback((action: keyof ObjectListActions<DirectoryItem>, ...data: unknown[]) => {
        switch (action) {
            case "Deleted": {
                if (!directory) return;
                const [id] = data as ArgsType<ObjectListActions<DirectoryItem>["Deleted"]>;
                const item = list.find(r => r.id === id);
                if (!item) {
                    PubSub.get().publish("snack", { message: "Item not found", severity: "Warning" });
                    return;
                }
                handleUpdate?.({
                    ...directory,
                    childApiVersions: isOfType(item, "ApiVersion") ? directory.childApiVersions.filter(i => i.id !== item.id) : directory.childApiVersions,
                    childCodeVersions: isOfType(item, "CodeVersion") ? directory.childCodeVersions.filter(i => i.id !== item.id) : directory.childCodeVersions,
                    childNoteVersions: isOfType(item, "NoteVersion") ? directory.childNoteVersions.filter(i => i.id !== item.id) : directory.childNoteVersions,
                    childProjectVersions: isOfType(item, "ProjectVersion") ? directory.childProjectVersions.filter(i => i.id !== item.id) : directory.childProjectVersions,
                    childRoutineVersions: isOfType(item, "RoutineVersion") ? directory.childRoutineVersions.filter(i => i.id !== item.id) : directory.childRoutineVersions,
                    childStandardVersions: isOfType(item, "StandardVersion") ? directory.childStandardVersions.filter(i => i.id !== item.id) : directory.childStandardVersions,
                    childTeams: isOfType(item, "Team") ? directory.childTeams.filter(i => i.id !== item.id) : directory.childTeams,
                } as ProjectVersionDirectory);
                break;
            }
            case "Updated": {
                if (!directory) return;
                const [updatedItem] = data as ArgsType<ObjectListActions<DirectoryItem>["Updated"]>;
                handleUpdate?.({
                    ...directory,
                    childApiVersions: isOfType(updatedItem, "ApiVersion") ? directory.childApiVersions.map(i => i.id === updatedItem.id ? updatedItem : i) : directory.childApiVersions,
                    childCodeVersions: isOfType(updatedItem, "CodeVersion") ? directory.childCodeVersions.map(i => i.id === updatedItem.id ? updatedItem : i) : directory.childCodeVersions,
                    childNoteVersions: isOfType(updatedItem, "NoteVersion") ? directory.childNoteVersions.map(i => i.id === updatedItem.id ? updatedItem : i) : directory.childNoteVersions,
                    childProjectVersions: isOfType(updatedItem, "ProjectVersion") ? directory.childProjectVersions.map(i => i.id === updatedItem.id ? updatedItem : i) : directory.childProjectVersions,
                    childRoutineVersions: isOfType(updatedItem, "RoutineVersion") ? directory.childRoutineVersions.map(i => i.id === updatedItem.id ? updatedItem : i) : directory.childRoutineVersions,
                    childStandardVersions: isOfType(updatedItem, "StandardVersion") ? directory.childStandardVersions.map(i => i.id === updatedItem.id ? updatedItem : i) : directory.childStandardVersions,
                    childTeams: isOfType(updatedItem, "Team") ? directory.childTeams.map(i => i.id === updatedItem.id ? updatedItem : i) : directory.childTeams,
                } as ProjectVersionDirectory);
                break;
            }
        }
    }, [directory, handleUpdate, list]);
    const onClick = useCallback((data: ListObject) => {
        //TODO
    }, []);

    const childProps: DirectoryListHorizontalProps & DirectoryListVerticalProps = {
        ...props,
        handleToggleSelect,
        isEditing,
        isSelecting,
        list,
        onAction,
        onClick,
        onDelete,
        openAddDialog,
        selectedData,
    };

    return (
        <>
            <FindObjectDialog
                find="List"
                isOpen={isAddDialogOpen}
                handleCancel={closeAddDialog}
                handleComplete={onAdd as (item: object) => unknown}
            />
            {BulkDeleteDialogComponent}
            <Box
                display="flex"
                flexDirection="column"
                justifyContent="center"
                alignItems="center"
                gap={1}
                paddingBottom="80px"
            >
                <DirectorySearchButtons
                    setSortBy={setSortBy}
                    setTimeFrame={setTimeFrame}
                    setViewMode={setViewMode}
                    sortBy={sortBy}
                    timeFrame={timeFrame}
                    viewMode={viewMode}
                />
                {
                    viewMode === "card" ?
                        <DirectoryListHorizontal {...childProps} /> :
                        <DirectoryListVertical {...childProps} />
                }
            </Box>
        </>
    );
}
