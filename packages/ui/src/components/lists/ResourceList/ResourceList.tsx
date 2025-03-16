// Displays a list of resources. If the user can modify the list, 
// it will display options for adding, removing, and sorting
import { DragDropContext, Draggable, DropResult, Droppable } from "@hello-pangea/dnd";
import { Count, DUMMY_ID, DeleteManyInput, DeleteType, ListObject, Resource, ResourceList as ResourceListType, ResourceUsedFor, TranslationKeyCommon, endpointsActions, updateArray } from "@local/shared";
import { Box, Button, IconButton, ListItem, ListItemText, Stack, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { useField } from "formik";
import { forwardRef, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { SessionContext } from "../../../contexts.js";
import { UsePressEvent, usePress } from "../../../hooks/gestures.js";
import { useBulkObjectActions, useObjectActions } from "../../../hooks/objectActions.js";
import { useDebounce } from "../../../hooks/useDebounce.js";
import { useLazyFetch } from "../../../hooks/useLazyFetch.js";
import { useObjectContextMenu } from "../../../hooks/useObjectContextMenu.js";
import { useSelectableList } from "../../../hooks/useSelectableList.js";
import { AddIcon, CloseIcon, DeleteIcon, EditIcon, LinkIcon, OpenInNewIcon } from "../../../icons/common.js";
import { openLink } from "../../../route/openLink.js";
import { useLocation } from "../../../route/router.js";
import { CardBox, multiLineEllipsis } from "../../../styles.js";
import { ArgsType } from "../../../types.js";
import { ObjectAction } from "../../../utils/actions/objectActions.js";
import { DUMMY_LIST_LENGTH, DUMMY_LIST_LENGTH_SHORT, ELEMENT_IDS } from "../../../utils/consts.js";
import { getResourceIcon } from "../../../utils/display/getResourceIcon.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { firstString } from "../../../utils/display/stringTools.js";
import { getUserLanguages } from "../../../utils/display/translationTools.js";
import { getResourceType, getResourceUrl } from "../../../utils/navigation/openObject.js";
import { PubSub } from "../../../utils/pubsub.js";
import { ResourceUpsert, resourceInitialValues } from "../../../views/objects/resource/ResourceUpsert.js";
import { ObjectActionMenu } from "../../dialogs/ObjectActionMenu/ObjectActionMenu.js";
import { ResourceListInputProps } from "../../inputs/types.js";
import { ObjectList } from "../../lists/ObjectList/ObjectList.js";
import { TextLoading } from "../../lists/TextLoading/TextLoading.js";
import { ObjectListActions } from "../../lists/types.js";
import { ResourceCardProps, ResourceListHorizontalProps, ResourceListItemProps, ResourceListProps, ResourceListVerticalProps } from "../types.js";

const StyledListItem = styled(ListItem)(({ theme }) => ({
    display: "flex",
    background: theme.palette.background.paper,
    color: theme.palette.background.textPrimary,
    borderBottom: `1px solid ${theme.palette.divider}`,
    padding: theme.spacing(1),
    cursor: "pointer",
}));

export function ResourceListItem({
    canUpdate,
    data,
    handleContextMenu,
    handleDelete,
    handleEdit,
    index,
    loading,
}: ResourceListItemProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { title, subtitle } = useMemo(() => getDisplay(data, getUserLanguages(session)), [data, session]);

    const Icon = useMemo(() => getResourceIcon(data.usedFor ?? ResourceUsedFor.Related, data.link), [data]);

    const href = useMemo(() => getResourceUrl(data.link), [data]);
    const handleClick = useCallback(({ target }: UsePressEvent) => {
        // Ignore if clicked edit or delete button
        if (target.id && ["delete-icon-button", "edit-icon-button"].includes(target.id)) return;
        // If no resource type or link, show error
        const resourceType = getResourceType(data.link);
        if (!resourceType || !href) {
            PubSub.get().publish("snack", { messageKey: "CannotOpenLink", severity: "Error" });
            return;
        }
        // Open link
        else openLink(setLocation, href);
    }, [data.link, href, setLocation]);

    const onEdit = useCallback((e: any) => {
        handleEdit(index);
    }, [handleEdit, index]);

    const onDelete = useCallback(() => {
        handleDelete(index);
    }, [handleDelete, index]);

    const pressEvents = usePress({
        onLongPress: (target) => { handleContextMenu(target, index); },
        onClick: handleClick,
        onRightClick: (target) => { handleContextMenu(target, index); },
    });

    return (
        <Tooltip placement="top" title="Open in new tab">
            <StyledListItem
                disablePadding
                {...pressEvents}
                onClick={(e) => { e.preventDefault(); }}
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore
                component="a"
                href={href}
            >
                <IconButton sx={{
                    width: "48px",
                    height: "48px",
                }}>
                    {typeof Icon === "function" ? <Icon fill={palette.background.textPrimary} width="80%" height="80%" /> : Icon}
                </IconButton>
                <Stack direction="column" spacing={1} pl={2} sx={{ width: "-webkit-fill-available" }}>
                    {/* Name/Title */}
                    {loading ? <TextLoading /> : <ListItemText
                        primary={firstString(title, data.link)}
                        sx={{ ...multiLineEllipsis(1) }}
                    />}
                    {/* Bio/Description */}
                    {loading ? <TextLoading /> : <ListItemText
                        primary={subtitle}
                        sx={{ ...multiLineEllipsis(2), color: palette.text.secondary }}
                    />}
                </Stack>
                {
                    canUpdate && <IconButton id='delete-icon-button' onClick={onDelete}>
                        <DeleteIcon fill={palette.background.textPrimary} />
                    </IconButton>
                }
                {
                    canUpdate && <IconButton id='edit-icon-button' onClick={onEdit}>
                        <EditIcon fill={palette.background.textPrimary} />
                    </IconButton>
                }
                <IconButton>
                    <OpenInNewIcon fill={palette.background.textPrimary} />
                </IconButton>
            </StyledListItem>
        </Tooltip>
    );
}

const ResourceCard = forwardRef<unknown, ResourceCardProps>(({
    data,
    dragProps,
    dragHandleProps,
    isEditing,
    onContextMenu,
    onEdit,
    onDelete,
}, ref) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();

    const { title, subtitle } = useMemo(() => {
        const { title, subtitle } = getDisplay(data, getUserLanguages(session));
        return {
            title: title ? title : t((data.usedFor ?? "Context") as TranslationKeyCommon),
            subtitle,
        };
    }, [data, session, t]);

    const Icon = useMemo(() => {
        return getResourceIcon(data.usedFor ?? ResourceUsedFor.Related, data.link);
    }, [data.link, data.usedFor]);

    const href = useMemo(() => getResourceUrl(data.link), [data]);
    const handleClick = useCallback(({ target }: UsePressEvent) => {
        // Check if edit or delete button was clicked
        const targetId: string | undefined = target.id;
        if (targetId && targetId.startsWith("edit-")) {
            onEdit?.(data);
        }
        else if (targetId && targetId.startsWith("delete-")) {
            onDelete?.(data);
        }
        else {
            // If no resource type or link, show error
            const resourceType = getResourceType(data.link);
            if (!resourceType || !href) {
                PubSub.get().publish("snack", { messageKey: "CannotOpenLink", severity: "Error" });
                return;
            }
            // Open link
            else openLink(setLocation, href);
        }
    }, [data, href, onDelete, onEdit, setLocation]);
    const [handleClickDebounce] = useDebounce(handleClick, 100);
    const handleContextMenu = useCallback(({ target }: UsePressEvent) => {
        onContextMenu(target, data);
    }, [data, onContextMenu]);

    const pressEvents = usePress({
        onLongPress: handleContextMenu,
        onClick: handleClickDebounce,
        onRightClick: handleContextMenu,
        pressDelay: isEditing ? 1500 : 300,
    });

    return (
        <Tooltip placement="top" title={`${subtitle ? subtitle + " - " : ""}${data.link}`}>
            <CardBox
                ref={ref}
                {...dragProps}
                {...dragHandleProps}
                {...pressEvents}
                component="a"
                href={href}
                onClick={(e) => e.preventDefault()}
            >
                {/* Edit and delete icons, only visible on hover */}
                {isEditing && (
                    <>
                        <Tooltip title={t("Edit")}>
                            <IconButton
                                id='edit-icon-button'
                                sx={{ background: "#c5ab17", position: "absolute", top: 4, left: 4 }}
                            >
                                <EditIcon id='edit-icon' fill={palette.secondary.contrastText} />
                            </IconButton>
                        </Tooltip>
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
                    {typeof Icon === "function" ? <Icon fill={palette.primary.contrastText} /> : Icon}
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
                        {firstString(title, data.link)}
                    </Typography>
                </Stack>
            </CardBox>
        </Tooltip>
    );
});
ResourceCard.displayName = "ResourceCard";

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

export function ResourceListHorizontal({
    canUpdate = true,
    handleUpdate,
    id,
    isEditing,
    list,
    loading,
    onAction,
    onDelete,
    openAddDialog,
    openUpdateDialog,
    title,
    sxs,
}: ResourceListHorizontalProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const onDragEnd = useCallback((result: DropResult) => {
        const { source, destination } = result;
        if (!isEditing || !destination || source.index === destination.index) return;
        // Handle the reordering of the resources in the list
        if (handleUpdate && list) {
            handleUpdate({
                ...list,
                resources: updateArray(list.resources, source.index, list.resources[destination.index]),
            });
        }
    }, [isEditing, handleUpdate, list]);

    const contextData = useObjectContextMenu();
    const actionData = useObjectActions({
        canNavigate: () => !isEditing,
        isListReorderable: isEditing,
        objectType: "Resource",
        onAction,
        setLocation,
        ...contextData,
    });

    const menu = useMemo(() => (
        <ObjectActionMenu
            actionData={actionData}
            anchorEl={contextData.anchorEl}
            exclude={[ObjectAction.FindInPage]}
            object={contextData.object}
            onClose={contextData.closeContextMenu}
        />
    ), [actionData, contextData]);

    const isEmpty = !list?.resources?.length;
    const boxSx = {
        display: "flex",
        gap: 2,
        width: "100%",
        marginLeft: "auto",
        marginRight: "auto",
        paddingTop: title ? 0 : 1,
        paddingBottom: 1,
        overflowX: "auto",
        ...sxs?.list,
    } as const;

    // Don't need a drag/drop list if there aren't any items
    if (isEmpty && !loading) {
        // If you can add an item, show a button instead of a card. 
        // This is more visually appealing
        if (canUpdate) {
            return (
                <>
                    {menu}
                    <Button
                        variant="text"
                        color="primary"
                        onClick={openAddDialog}
                        startIcon={<LinkIcon />}
                        sx={{ paddingBottom: 1 }}
                    >
                        {t("AddResource")}
                    </Button>
                </>
            );
        }
        return null;
    }
    // If empty but loading, show loading cards
    if (isEmpty && loading) {
        return (
            <>
                <Box
                    id={id ?? ELEMENT_IDS.ResourceCards}
                    justifyContent="center"
                    alignItems="center"
                    sx={boxSx}
                >
                    {Array.from(Array(DUMMY_LIST_LENGTH_SHORT).keys()).map((i) => (
                        <LoadingCard key={`resource-card-${i}`} />
                    ))}
                </Box>
            </>
        );
    }
    // Otherwise, show drag/drop list with item cards, loading cards (if relavant), and add card (if relevant)
    return (
        <>
            {menu}
            <DragDropContext onDragEnd={onDragEnd}>
                <Droppable droppableId="resource-list" direction="horizontal">
                    {(providedDrop) => (
                        <Box
                            ref={providedDrop.innerRef}
                            id={id}
                            {...providedDrop.droppableProps}
                            justifyContent="center"
                            alignItems="center"
                            sx={boxSx}>
                            {/* Resources */}
                            {list?.resources?.map((c: Resource, index) => (
                                <Draggable
                                    key={`resource-card-${index}`}
                                    draggableId={`resource-card-${index}`}
                                    index={index}
                                    isDragDisabled={!isEditing}
                                >
                                    {(providedDrag) => (
                                        <ResourceCard
                                            ref={providedDrag.innerRef}
                                            dragProps={providedDrag.draggableProps}
                                            dragHandleProps={providedDrag.dragHandleProps}
                                            key={`resource-card-${index}`}
                                            isEditing={isEditing}
                                            data={{ ...c, list }}
                                            onContextMenu={contextData.handleContextMenu}
                                            onEdit={openUpdateDialog}
                                            onDelete={onDelete}
                                        />
                                    )}
                                </Draggable>
                            ))}
                            {/* Dummy cards when loading */}
                            {
                                loading && !Array.isArray(list?.resources) && Array.from(Array(DUMMY_LIST_LENGTH_SHORT).keys()).map((i) => (
                                    <LoadingCard key={`resource-card-${i}`} />
                                ))
                            }
                            {/* Add button */}
                            {canUpdate ? <Tooltip placement="top" title={t("AddResource")}>
                                <CardBox
                                    onClick={openAddDialog}
                                    aria-label={t("AddResource")}
                                    sx={{
                                        display: "flex",
                                        alignItems: "center",
                                        justifyContent: "center",
                                    }}
                                >
                                    <LinkIcon fill={palette.secondary.contrastText} width='56px' height='56px' />
                                </CardBox>
                            </Tooltip> : null}
                            {providedDrop.placeholder}
                        </Box>
                    )}
                </Droppable>
            </DragDropContext>
        </>
    );
}

const editingIconButtonStyle = { width: "24px", height: "24px" } as const;

export function ResourceListVertical({
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
}: ResourceListVerticalProps) {
    const { t } = useTranslation();

    return (
        <>
            <ObjectList
                canNavigate={() => !isEditing}
                dummyItems={new Array(DUMMY_LIST_LENGTH).fill("Resource")}
                handleToggleSelect={handleToggleSelect as (item: ListObject) => unknown}
                hideUpdateButton={isEditing}
                isSelecting={isSelecting}
                items={list?.resources ?? []}
                keyPrefix="resource-list-item"
                loading={loading ?? false}
                onAction={onAction}
                onClick={onClick as (item: ListObject) => unknown}
                selectedItems={selectedData}
            />
            {/* Add button */}
            {canUpdate && <Box sx={{
                maxWidth: "400px",
                margin: "auto",
                paddingTop: 5,
            }}>
                <Button
                    fullWidth onClick={openAddDialog}
                    startIcon={<AddIcon />}
                    variant="outlined"
                >{t("AddResource")}</Button>
            </Box>}
        </>
    );
}

export function ResourceList(props: ResourceListProps) {
    const { canUpdate, handleUpdate, horizontal, list, mutate, parent, title } = props;
    const { t } = useTranslation();
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const [isEditing, setIsEditing] = useState(false);
    useEffect(() => {
        if (!canUpdate) setIsEditing(false);
    }, [canUpdate]);

    // Upsert dialog
    const [editingIndex, setEditingIndex] = useState<number>(-1);
    const [isAddDialogOpen, setIsAddDialogOpen] = useState(false);
    const openAddDialog = useCallback(() => { setIsAddDialogOpen(true); }, []);
    const closeAddDialog = useCallback(() => { setIsAddDialogOpen(false); setEditingIndex(-1); }, []);
    const openUpdateDialog = useCallback((resource: Resource) => {
        const index = list?.resources?.findIndex(r => r.id === resource.id) ?? -1;
        setEditingIndex(index);
        setIsAddDialogOpen(true);
    }, [list?.resources]);
    const onCompleted = useCallback((resource: Resource) => {
        closeAddDialog();
        if (!list || !handleUpdate) return;
        const index = resource.index;
        if (index && index >= 0) {
            handleUpdate({
                ...list,
                resources: updateArray(list.resources, index, resource),
            });
        }
        else {
            handleUpdate({
                ...list,
                resources: [...(list?.resources ?? []), resource],
            });
        }
    }, [closeAddDialog, handleUpdate, list]);
    const onDeleted = useCallback((resource: Resource) => {
        closeAddDialog();
        if (!list || !handleUpdate) return;
        handleUpdate({
            ...list,
            resources: list.resources.filter(r => r.id !== resource.id),
        });
    }, [closeAddDialog, handleUpdate, list]);

    const [deleteMutation] = useLazyFetch<DeleteManyInput, Count>(endpointsActions.deleteMany);
    const onDelete = useCallback((item: Resource) => {
        if (!list) return;
        if (mutate && item.id) {
            fetchLazyWrapper<DeleteManyInput, Count>({
                fetch: deleteMutation,
                inputs: { objects: [{ id: item.id, objectType: DeleteType.Resource }] },
                onSuccess: () => {
                    handleUpdate?.({
                        ...list,
                        resources: list.resources.filter(r => r.id !== item.id),
                    });
                },
            });
        }
        else if (handleUpdate) {
            handleUpdate({
                ...list,
                resources: list.resources.filter(r => r.id !== item.id),
            });
        }
    }, [deleteMutation, handleUpdate, list, mutate]);

    const upsertDialog = useMemo(() => (
        isAddDialogOpen ? <ResourceUpsert
            display="dialog"
            isCreate={editingIndex < 0}
            isOpen={isAddDialogOpen}
            isMutate={mutate ?? false}
            onCancel={closeAddDialog}
            onClose={closeAddDialog}
            onCompleted={onCompleted}
            onDeleted={onDeleted}
            overrideObject={(editingIndex >= 0 && list?.resources ?
                { ...list.resources[editingIndex as number], index: editingIndex } :
                resourceInitialValues(undefined, {
                    index: 0,
                    list: {
                        __connect: true,
                        ...(list?.id && list.id !== DUMMY_ID ? list : { listFor: parent }),
                        id: list?.id ?? DUMMY_ID,
                        __typename: "ResourceList",
                    },
                }) as Resource)}
        /> : null
    ), [closeAddDialog, editingIndex, isAddDialogOpen, list, mutate, onCompleted, onDeleted, parent]);

    const {
        isSelecting,
        handleToggleSelecting,
        handleToggleSelect,
        selectedData,
        setIsSelecting,
        setSelectedData,
    } = useSelectableList<Resource>(list?.resources ?? []);
    const { onBulkActionStart, BulkDeleteDialogComponent } = useBulkObjectActions<Resource>({
        allData: list?.resources ?? [],
        selectedData,
        setAllData: (data) => {
            //TODO
        },
        setSelectedData: (data) => {
            setSelectedData(data);
            setIsSelecting(false);
        },
        setLocation,
    });
    const onAction = useCallback((action: keyof ObjectListActions<Resource>, ...data: unknown[]) => {
        switch (action) {
            case "Deleted": {
                if (!list) return;
                const [id] = data as ArgsType<ObjectListActions<Resource>["Deleted"]>;
                const item = list.resources?.find(r => r.id === id);
                if (!item) {
                    PubSub.get().publish("snack", { message: "Item not found", severity: "Warning" });
                    return;
                }
                onDeleted(item);
                break;
            }
            case "Updated": {
                if (!list) return;
                const [updatedItem] = data as ArgsType<ObjectListActions<Resource>["Updated"]>;
                handleUpdate?.({
                    ...list,
                    resources: list.resources.map(r => r.id === updatedItem.id ? updatedItem : r),
                });
                break;
            }
        }
    }, [handleUpdate, list, onDeleted]);
    const onClick = useCallback((data: ListObject) => {
        //TODO
    }, []);

    const childProps: ResourceListHorizontalProps & ResourceListVerticalProps = {
        ...props,
        handleToggleSelect,
        isEditing,
        isSelecting,
        onAction,
        onClick,
        onDelete,
        openAddDialog,
        openUpdateDialog,
        selectedData,
    };

    return (
        <>
            {upsertDialog}
            {BulkDeleteDialogComponent}
            {title && <Box display="flex" flexDirection="row" alignItems="center">
                <Typography component="h2" variant="h6" textAlign="left">{title}</Typography>
                {true && <Tooltip title={t("Edit")}>
                    <IconButton onClick={() => { setIsEditing(e => !e); }}>
                        {isEditing ?
                            <CloseIcon fill={palette.secondary.main} style={editingIconButtonStyle} /> :
                            <EditIcon fill={palette.secondary.main} style={editingIconButtonStyle} />
                        }
                    </IconButton>
                </Tooltip>}
            </Box>}
            {
                horizontal ?
                    <ResourceListHorizontal {...childProps} /> :
                    <ResourceListVertical {...childProps} />
            }
        </>
    );
}

export function ResourceListInput({
    disabled = false,
    horizontal,
    isCreate,
    isLoading = false,
    parent,
    ...props
}: ResourceListInputProps) {
    const [field, , helpers] = useField("resourceList");

    const handleUpdate = useCallback((newList: ResourceListType) => {
        helpers.setValue(newList);
    }, [helpers]);

    return (
        <ResourceList
            horizontal={horizontal}
            list={field.value}
            canUpdate={!disabled}
            handleUpdate={handleUpdate}
            loading={isLoading}
            mutate={!isCreate}
            parent={parent}
            {...props}
        />
    );
}
