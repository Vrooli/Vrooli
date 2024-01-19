import { ApiVersion, Count, DeleteManyInput, DeleteType, GqlModelType, NoteVersion, Organization, ProjectVersion, ProjectVersionDirectory, RoutineVersion, SmartContractVersion, StandardVersion, endpointPostDeleteMany, isOfType } from "@local/shared";
import { Box, Button, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { ObjectActionMenu } from "components/dialogs/ObjectActionMenu/ObjectActionMenu";
import { SelectOrCreateObject } from "components/dialogs/types";
import { ObjectList } from "components/lists/ObjectList/ObjectList";
import { TextLoading } from "components/lists/TextLoading/TextLoading";
import { ObjectListActions } from "components/lists/types";
import { SessionContext } from "contexts/SessionContext";
import { useBulkObjectActions } from "hooks/useBulkObjectActions";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useObjectActions } from "hooks/useObjectActions";
import { useObjectContextMenu } from "hooks/useObjectContextMenu";
import usePress from "hooks/usePress";
import { useSelectableList } from "hooks/useSelectableList";
import { AddIcon, ApiIcon, CloseIcon, DeleteIcon, EditIcon, HelpIcon, LinkIcon, NoteIcon, OrganizationIcon, ProjectIcon, RoutineIcon, SmartContractIcon, StandardIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { CardBox, multiLineEllipsis } from "styles";
import { ArgsType } from "types";
import { ObjectAction } from "utils/actions/objectActions";
import { ListObject, getDisplay } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { getObjectUrl } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { initializeDirectoryList } from "../common";
import { DirectoryCardProps, DirectoryItem, DirectoryListHorizontalProps, DirectoryListProps, DirectoryListVerticalProps } from "../types";

/**
 * Unlike ResourceCard, these aren't draggable. This is because the objects 
 * are not stored in an order - they are stored by object type
 */
export const DirectoryCard = ({
    canUpdate,
    data,
    onContextMenu,
    onDelete,
}: DirectoryCardProps) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [showIcons, setShowIcons] = useState(false);

    const { title, subtitle } = useMemo(() => getDisplay(data, getUserLanguages(session)), [data, session]);

    const Icon = useMemo(() => {
        if (!data || !data.__typename) return HelpIcon;
        if (isOfType(data, "ApiVersion")) return ApiIcon;
        if (isOfType(data, "NoteVersion")) return NoteIcon;
        if (isOfType(data, "Organization")) return OrganizationIcon;
        if (isOfType(data, "ProjectVersion")) return ProjectIcon;
        if (isOfType(data, "RoutineVersion")) return RoutineIcon;
        if (isOfType(data, "SmartContractVersion")) return SmartContractIcon;
        if (isOfType(data, "StandardVersion")) return StandardIcon;
        return HelpIcon;
    }, [data]);

    const href = useMemo(() => data ? getObjectUrl(data) : "#", [data]);
    const handleClick = useCallback((target: EventTarget) => {
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
    const handleContextMenu = useCallback((target: EventTarget) => {
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
                                <DeleteIcon id='delete-icon' fill={palette.secondary.contrastText} />
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
                    <Icon fill="white" />
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
};

const LoadingCard = () => {
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
};

export const DirectoryListHorizontal = ({
    canUpdate = true,
    isEditing,
    list,
    loading,
    onAction,
    onDelete,
    openAddDialog,
    title,
}: DirectoryListHorizontalProps) => {
    const { t } = useTranslation();
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const contextData = useObjectContextMenu();
    const actionData = useObjectActions({
        canNavigate: () => !isEditing,
        isListReorderable: isEditing,
        objectType: contextData.object?.__typename as GqlModelType | undefined,
        onAction,
        setLocation,
        ...contextData,
    });

    return (
        <>
            <ObjectActionMenu
                actionData={actionData}
                anchorEl={contextData.anchorEl}
                exclude={[ObjectAction.Comment, ObjectAction.FindInPage]} // Find in page only relevant when viewing object - not in list. And shouldn't really comment without viewing full page
                object={contextData.object}
                onClose={contextData.closeContextMenu}
            />
            <Box minWidth={120} sx={{
                display: "flex",
                gap: 2,
                width: "100%",
                marginLeft: "auto",
                marginRight: "auto",
                paddingTop: title ? 0 : 1,
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
                    loading && !Array.isArray(list) && Array.from(Array(3).keys()).map((i) => (
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
                        <LinkIcon fill={palette.secondary.contrastText} width='56px' height='56px' />
                    </CardBox>
                </Tooltip> : null)}
            </Box>
        </>
    );
};

export const DirectoryListVertical = ({
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
}: DirectoryListVerticalProps) => {
    const { t } = useTranslation();

    return (
        <>
            <ObjectList
                canNavigate={() => !isEditing}
                dummyItems={new Array(5).fill("Project")}
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
            {/* Add button */}
            {canUpdate && <Box sx={{
                maxWidth: "400px",
                margin: "auto",
            }}>
                <Button
                    fullWidth onClick={openAddDialog}
                    startIcon={<AddIcon />}
                    variant="outlined"
                >{t("AddItem")}</Button>
            </Box>}
        </>
    );
};

type ViewMode = "card" | "list";

const DirectoryViewModeToggle = ({
    viewMode,
    onViewModeChange,
}: {
    viewMode: ViewMode;
    onViewModeChange: (mode: ViewMode) => unknown;
}) => (
    <Box sx={{
        display: "flex",
        gap: 1,
        width: "100%",
        marginRight: "auto",
        marginBottom: "10px",
    }}>
        <Button variant={viewMode === "card" ? "contained" : "outlined"} color="primary" onClick={() => onViewModeChange("card")}>
            Card View
        </Button>
        <Button variant={viewMode === "list" ? "contained" : "outlined"} color="primary" onClick={() => onViewModeChange("list")}>
            List View
        </Button>
    </Box>
);

export const DirectoryList = (props: DirectoryListProps) => {
    const { canUpdate, directory, handleUpdate, mutate, sortBy, title } = props;
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const list = useMemo(() => initializeDirectoryList(directory, sortBy, session), [directory, session, sortBy]);
    const [viewMode, setViewMode] = useState<ViewMode>("list");
    const [isEditing, setIsEditing] = useState(false);
    useEffect(() => {
        if (!canUpdate) setIsEditing(false);
    }, [canUpdate]);

    const handleViewModeChange = useCallback((mode: ViewMode) => {
        setViewMode(mode);
    }, []);

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
            childNoteVersions: (isOfType(item, "NoteVersion") ? [...directory.childNoteVersions, item] : directory.childNoteVersions) as NoteVersion[],
            childOrganizations: (isOfType(item, "Organization") ? [...directory.childOrganizations, item] : directory.childOrganizations) as Organization[],
            childProjectVersions: (isOfType(item, "ProjectVersion") ? [...directory.childProjectVersions, item] : directory.childProjectVersions) as ProjectVersion[],
            childRoutineVersions: (isOfType(item, "RoutineVersion") ? [...directory.childRoutineVersions, item] : directory.childRoutineVersions) as RoutineVersion[],
            childSmartContractVersions: (isOfType(item, "SmartContractVersion") ? [...directory.childSmartContractVersions, item] : directory.childSmartContractVersions) as SmartContractVersion[],
            childStandardVersions: (isOfType(item, "StandardVersion") ? [...directory.childStandardVersions, item] : directory.childStandardVersions) as StandardVersion[],
        });
    }, [closeAddDialog, directory, handleUpdate, list]);

    const [deleteMutation] = useLazyFetch<DeleteManyInput, Count>(endpointPostDeleteMany);
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
                        childNoteVersions: isOfType(item, "NoteVersion") ? directory.childNoteVersions.filter(i => i.id !== item.id) : directory.childNoteVersions,
                        childOrganizations: isOfType(item, "Organization") ? directory.childOrganizations.filter(i => i.id !== item.id) : directory.childOrganizations,
                        childProjectVersions: isOfType(item, "ProjectVersion") ? directory.childProjectVersions.filter(i => i.id !== item.id) : directory.childProjectVersions,
                        childRoutineVersions: isOfType(item, "RoutineVersion") ? directory.childRoutineVersions.filter(i => i.id !== item.id) : directory.childRoutineVersions,
                        childSmartContractVersions: isOfType(item, "SmartContractVersion") ? directory.childSmartContractVersions.filter(i => i.id !== item.id) : directory.childSmartContractVersions,
                        childStandardVersions: isOfType(item, "StandardVersion") ? directory.childStandardVersions.filter(i => i.id !== item.id) : directory.childStandardVersions,
                    });
                },
            });
        }
        else {
            handleUpdate?.({
                ...directory,
                childApiVersions: isOfType(item, "ApiVersion") ? directory.childApiVersions.filter(i => i.id !== item.id) : directory.childApiVersions,
                childNoteVersions: isOfType(item, "NoteVersion") ? directory.childNoteVersions.filter(i => i.id !== item.id) : directory.childNoteVersions,
                childOrganizations: isOfType(item, "Organization") ? directory.childOrganizations.filter(i => i.id !== item.id) : directory.childOrganizations,
                childProjectVersions: isOfType(item, "ProjectVersion") ? directory.childProjectVersions.filter(i => i.id !== item.id) : directory.childProjectVersions,
                childRoutineVersions: isOfType(item, "RoutineVersion") ? directory.childRoutineVersions.filter(i => i.id !== item.id) : directory.childRoutineVersions,
                childSmartContractVersions: isOfType(item, "SmartContractVersion") ? directory.childSmartContractVersions.filter(i => i.id !== item.id) : directory.childSmartContractVersions,
                childStandardVersions: isOfType(item, "StandardVersion") ? directory.childStandardVersions.filter(i => i.id !== item.id) : directory.childStandardVersions,
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
    } = useSelectableList<DirectoryItem>();
    const { onBulkActionStart, BulkDeleteDialogComponent } = useBulkObjectActions<DirectoryItem>({
        allData: list,
        selectedData,
        setAllData: (data) => {
            if (!directory) return;
            handleUpdate?.({
                ...directory,
                childApiVersions: data.filter(i => isOfType(i, "ApiVersion")),
                childNoteVersions: data.filter(i => isOfType(i, "NoteVersion")),
                childOrganizations: data.filter(i => isOfType(i, "Organization")),
                childProjectVersions: data.filter(i => isOfType(i, "ProjectVersion")),
                childRoutineVersions: data.filter(i => isOfType(i, "RoutineVersion")),
                childSmartContractVersions: data.filter(i => isOfType(i, "SmartContractVersion")),
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
                    childNoteVersions: isOfType(item, "NoteVersion") ? directory.childNoteVersions.filter(i => i.id !== item.id) : directory.childNoteVersions,
                    childOrganizations: isOfType(item, "Organization") ? directory.childOrganizations.filter(i => i.id !== item.id) : directory.childOrganizations,
                    childProjectVersions: isOfType(item, "ProjectVersion") ? directory.childProjectVersions.filter(i => i.id !== item.id) : directory.childProjectVersions,
                    childRoutineVersions: isOfType(item, "RoutineVersion") ? directory.childRoutineVersions.filter(i => i.id !== item.id) : directory.childRoutineVersions,
                    childSmartContractVersions: isOfType(item, "SmartContractVersion") ? directory.childSmartContractVersions.filter(i => i.id !== item.id) : directory.childSmartContractVersions,
                    childStandardVersions: isOfType(item, "StandardVersion") ? directory.childStandardVersions.filter(i => i.id !== item.id) : directory.childStandardVersions,
                } as ProjectVersionDirectory);
                break;
            }
            case "Updated": {
                if (!directory) return;
                const [updatedItem] = data as ArgsType<ObjectListActions<DirectoryItem>["Updated"]>;
                handleUpdate?.({
                    ...directory,
                    childApiVersions: isOfType(updatedItem, "ApiVersion") ? directory.childApiVersions.map(i => i.id === updatedItem.id ? updatedItem : i) : directory.childApiVersions,
                    childNoteVersions: isOfType(updatedItem, "NoteVersion") ? directory.childNoteVersions.map(i => i.id === updatedItem.id ? updatedItem : i) : directory.childNoteVersions,
                    childOrganizations: isOfType(updatedItem, "Organization") ? directory.childOrganizations.map(i => i.id === updatedItem.id ? updatedItem : i) : directory.childOrganizations,
                    childProjectVersions: isOfType(updatedItem, "ProjectVersion") ? directory.childProjectVersions.map(i => i.id === updatedItem.id ? updatedItem : i) : directory.childProjectVersions,
                    childRoutineVersions: isOfType(updatedItem, "RoutineVersion") ? directory.childRoutineVersions.map(i => i.id === updatedItem.id ? updatedItem : i) : directory.childRoutineVersions,
                    childSmartContractVersions: isOfType(updatedItem, "SmartContractVersion") ? directory.childSmartContractVersions.map(i => i.id === updatedItem.id ? updatedItem : i) : directory.childSmartContractVersions,
                    childStandardVersions: isOfType(updatedItem, "StandardVersion") ? directory.childStandardVersions.map(i => i.id === updatedItem.id ? updatedItem : i) : directory.childStandardVersions,
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
                handleComplete={onAdd as (item: SelectOrCreateObject) => unknown}
            />
            {BulkDeleteDialogComponent}
            <Box sx={{
                display: "flex",
                flexDirection: "column",
                justifyContent: "center",
                alignItems: "center",
                gap: 1,
            }}>
                <DirectoryViewModeToggle viewMode={viewMode} onViewModeChange={handleViewModeChange} />
                {title && <Box display="flex" flexDirection="row" alignItems="center">
                    <Typography component="h2" variant="h6" textAlign="left">{title}</Typography>
                    {true && <Tooltip title={t("Edit")}>
                        <IconButton onClick={() => { setIsEditing(e => !e); }}>
                            {isEditing ?
                                <CloseIcon fill={palette.secondary.main} style={{ width: "24px", height: "24px" }} /> :
                                <EditIcon fill={palette.secondary.main} style={{ width: "24px", height: "24px" }} />
                            }
                        </IconButton>
                    </Tooltip>}
                </Box>}
                {
                    viewMode === "card" ?
                        <DirectoryListHorizontal {...childProps} /> :
                        <DirectoryListVertical {...childProps} />
                }
            </Box>
        </>
    );
};
