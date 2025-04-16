import { DragDropContext, Draggable, DraggableProvidedDragHandleProps, DraggableProvidedDraggableProps, DropResult, Droppable } from "@hello-pangea/dnd";
import { Count, DUMMY_ID, DeleteManyInput, DeleteType, ListObject, Resource, ResourceListFor, ResourceList as ResourceListType, ResourceUsedFor, TranslationKeyCommon, endpointsActions, updateArray } from "@local/shared";
import { Box, Button, IconButton, Tooltip, Typography, styled } from "@mui/material";
import { useField } from "formik";
import { SyntheticEvent, forwardRef, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../../api/fetchWrapper.js";
import { SessionContext } from "../../../contexts/session.js";
import { useBulkObjectActions } from "../../../hooks/objectActions.js";
import { useLazyFetch } from "../../../hooks/useLazyFetch.js";
import { useSelectableList } from "../../../hooks/useSelectableList.js";
import { IconCommon } from "../../../icons/Icons.js";
import { openLink } from "../../../route/openLink.js";
import { useLocation } from "../../../route/router.js";
import { multiLineEllipsis, noSelect } from "../../../styles.js";
import { ArgsType } from "../../../types.js";
import { DUMMY_LIST_LENGTH, DUMMY_LIST_LENGTH_SHORT, ELEMENT_IDS } from "../../../utils/consts.js";
import { getResourceIcon } from "../../../utils/display/getResourceIcon.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { firstString } from "../../../utils/display/stringTools.js";
import { getUserLanguages } from "../../../utils/display/translationTools.js";
import { getResourceUrl } from "../../../utils/navigation/openObject.js";
import { PubSub } from "../../../utils/pubsub.js";
import { ResourceUpsert, resourceInitialValues } from "../../../views/objects/resource/ResourceUpsert.js";
import { ResourceListInputProps } from "../../inputs/types.js";
import { ObjectList } from "../../lists/ObjectList/ObjectList.js";
import { TextLoading } from "../../lists/TextLoading/TextLoading.js";
import { ObjectListActions } from "../../lists/types.js";

const CONTENT_SPACING = 1;
const ICON_SIZE = 20;

export interface ResourceCardProps {
    data: Resource;
    dragProps: DraggableProvidedDraggableProps;
    dragHandleProps: DraggableProvidedDragHandleProps | null | undefined;
    /** 
     * Hides edit and delete icons when in edit mode, 
     * making only drag'n'drop and the context menu available.
     **/
    isEditing: boolean;
    onEdit: (data: Resource) => unknown;
    onDelete: (data: Resource) => unknown;
}

export type ResourceListProps = {
    canUpdate?: boolean;
    handleUpdate?: (updatedList: ResourceListType) => unknown;
    horizontal?: boolean;
    id?: string;
    list: ResourceListType | null | undefined;
    loading?: boolean;
    mutate?: boolean;
    parent: { __typename: ResourceListFor | `${ResourceListFor}`, id: string };
}

export type ResourceListHorizontalProps = ResourceListProps & {
    handleToggleSelect: (data: Resource) => unknown;
    isEditing: boolean;
    isSelecting: boolean;
    onAction: (action: keyof ObjectListActions<Resource>, ...data: unknown[]) => unknown;
    onClick: (data: Resource) => unknown;
    onDelete: (data: Resource) => unknown;
    openAddDialog: () => unknown;
    openUpdateDialog: (data: Resource) => unknown;
    selectedData: Resource[];
    toggleEditing: () => unknown;
}

export type ResourceListVerticalProps = ResourceListHorizontalProps

const CardsBox = styled(Box)(({ theme }) => ({
    alignItems: "flex-start",
    display: "flex",
    flexWrap: "wrap",
    gap: theme.spacing(1),
    paddingBottom: 1,
    justifyContent: "center",
    width: "100%",
}));
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
const PlaceholderIcon = styled(Box)(({ theme }) => ({
    width: ICON_SIZE,
    height: ICON_SIZE,
    borderRadius: "16px",
    backgroundColor: theme.palette.text.primary,
    opacity: 0.2,
}));

const titleStyle = {
    ...multiLineEllipsis(1),
    flexGrow: 1,
} as const;

const loadingCardStyle = {
    display: "flex",
    alignItems: "center",
    gap: CONTENT_SPACING,
    width: "100%",
} as const;

const loadingTextStyle = {
    width: "60%",
    flexGrow: 1,
    opacity: 0.3,
} as const;

const ResourceCard = forwardRef<unknown, ResourceCardProps>(({
    data,
    dragProps,
    dragHandleProps,
    isEditing,
    onEdit,
    onDelete,
}, ref) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const { title, subtitle } = useMemo(() => {
        const { title, subtitle } = getDisplay(data, getUserLanguages(session));
        return {
            title: title ? title : t((data.usedFor ?? "Context") as TranslationKeyCommon),
            subtitle,
        };
    }, [data, session, t]);

    const Icon = useMemo(() => {
        return getResourceIcon({
            fill: "background.textSecondary",
            link: data.link,
            usedFor: data.usedFor ?? ResourceUsedFor.Related,
        });
    }, [data.link, data.usedFor]);

    const href = useMemo(() => getResourceUrl(data.link), [data]);
    const handleClick = useCallback((event: SyntheticEvent) => {
        event.preventDefault();
        const target = event.target as HTMLElement | SVGElement | null;
        if (!target) return;
        const classList = target.classList;
        const hasDeleteClass = Array.from(classList).some(c => c.startsWith("delete-"));
        if (hasDeleteClass) {
            onDelete?.(data);
        }
        else if (isEditing) {
            onEdit?.(data);
        } else if (href) {
            openLink(setLocation, href);
        } else {
            console.error("[ResourceCard] Unhandled click event", event.target);
        }
    }, [data, href, isEditing, onDelete, onEdit, setLocation]);

    return (
        <Tooltip placement="top" title={`${subtitle ? subtitle + " - " : ""}${data.link}`}>
            <CardBox
                ref={ref}
                {...dragProps}
                {...dragHandleProps}
                component="a"
                href={href}
                onClick={handleClick}
            >
                {Icon}
                <Typography
                    variant="caption"
                    sx={titleStyle}
                >
                    {firstString(title, data.link)}
                </Typography>
                {isEditing && <IconButton className="delete-icon-button">
                    <IconCommon
                        className="delete-icon"
                        fill="background.textSecondary"
                        name="Delete"
                    />
                </IconButton>}
            </CardBox>
        </Tooltip>
    );
});
ResourceCard.displayName = "ResourceCard";

function LoadingCard() {
    return (
        <CardBox>
            <Box sx={loadingCardStyle}>
                <PlaceholderIcon />
                <TextLoading size="caption" lines={1} sx={loadingTextStyle} />
            </Box>
        </CardBox>
    );
}

function ResourceListHorizontal({
    canUpdate = true,
    handleUpdate,
    id,
    isEditing,
    list,
    loading,
    onDelete,
    openAddDialog,
    openUpdateDialog,
    toggleEditing,
}: ResourceListHorizontalProps) {
    const { t } = useTranslation();

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

    const isEmpty = !list?.resources?.length;

    // If empty, not loading, and can't add an item, show "No resources"
    if (isEmpty && !loading && !canUpdate) {
        return <Typography variant="body1" color="text.secondary">No resources</Typography>;
    }
    // If empty but loading, show loading cards
    if (isEmpty && loading) {
        return (
            <>
                <CardsBox
                    id={id ?? ELEMENT_IDS.ResourceCards}
                >
                    {Array.from(Array(DUMMY_LIST_LENGTH_SHORT).keys()).map((i) => (
                        <LoadingCard key={`resource-card-${i}`} />
                    ))}
                </CardsBox>
            </>
        );
    }
    // Otherwise, show drag/drop list with item cards, loading cards (if relavant), and add card (if relevant)
    return (
        <>
            <DragDropContext onDragEnd={onDragEnd}>
                <Droppable droppableId="resource-list" direction="horizontal">
                    {(providedDrop) => (
                        <CardsBox
                            ref={providedDrop.innerRef}
                            id={id}
                            {...providedDrop.droppableProps}
                        >
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
                            {canUpdate && <Tooltip placement="top" title={t("AddResource")}>
                                <CardBox onClick={openAddDialog}>
                                    <IconCommon name="Add" size={ICON_SIZE} />
                                    <Typography
                                        variant="caption"
                                        component="span"
                                        sx={titleStyle}
                                    >
                                        {t("AddResource")}
                                    </Typography>
                                </CardBox>
                            </Tooltip>}
                            {providedDrop.placeholder}
                            {/* Edit button */}
                            {canUpdate && <Tooltip title={t("Edit")}>
                                <IconButton onClick={toggleEditing}>
                                    <IconCommon name={isEditing ? "Close" : "Edit"} fill="secondary.main" size={24} />
                                </IconButton>
                            </Tooltip>}
                        </CardsBox>
                    )}
                </Droppable>
            </DragDropContext>
        </>
    );
}

function ResourceListVertical({
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
    toggleEditing,
}: ResourceListVerticalProps) {
    const { t } = useTranslation();
    const canNavigate = useCallback(() => !isEditing, [isEditing]);
    const items = useMemo(() => list?.resources ?? [], [list]);

    return (
        <>
            <ObjectList
                canNavigate={canNavigate}
                dummyItems={new Array(DUMMY_LIST_LENGTH).fill("Resource")}
                handleToggleSelect={handleToggleSelect as (item: ListObject) => unknown}
                hideUpdateButton={isEditing}
                isSelecting={isSelecting}
                items={items}
                keyPrefix="resource-list-item"
                loading={loading ?? false}
                onAction={onAction}
                onClick={onClick as (item: ListObject) => unknown}
                selectedItems={selectedData}
            />
            {/* Add button */}
            {canUpdate && <Box
                maxWidth="400px"
                margin="auto"
                paddingTop={5}
            >
                <Button
                    fullWidth onClick={openAddDialog}
                    startIcon={<IconCommon name="Add" />}
                    variant="outlined"
                >{t("AddResource")}</Button>
            </Box>}
            {/* Edit button */}
            {canUpdate && <Box
                maxWidth="400px"
                margin="auto"
                paddingTop={5}
            >
                <Button
                    fullWidth onClick={toggleEditing}
                    startIcon={<IconCommon name={isEditing ? "Close" : "Edit"} />}
                    variant="outlined"
                >{t("Edit")}</Button>
            </Box>}
        </>
    );
}

export function ResourceList(props: ResourceListProps) {
    const { canUpdate, handleUpdate, horizontal, list, mutate, parent, title } = props;
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const [isEditing, setIsEditing] = useState(false);
    useEffect(() => {
        if (!canUpdate) setIsEditing(false);
    }, [canUpdate]);

    function toggleEditing() {
        setIsEditing(e => !e);
    }

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

    const upsertDialog = useMemo(() => {
        if (!isAddDialogOpen) return null;

        const overrideObject = editingIndex >= 0 && list?.resources
            ? { ...list.resources[editingIndex as number], index: editingIndex }
            : resourceInitialValues(undefined, {
                index: 0,
                list: {
                    __connect: true,
                    ...(list?.id && list.id !== DUMMY_ID ? list : { listFor: parent }),
                    id: list?.id ?? DUMMY_ID,
                    __typename: "ResourceList",
                },
            }) as Resource;

        return <ResourceUpsert
            display="dialog"
            isCreate={editingIndex < 0}
            isOpen={isAddDialogOpen}
            isMutate={mutate ?? false}
            onCancel={closeAddDialog}
            onClose={closeAddDialog}
            onCompleted={onCompleted}
            onDeleted={onDeleted}
            overrideObject={overrideObject}
        />;
    }, [closeAddDialog, editingIndex, isAddDialogOpen, list, mutate, onCompleted, onDeleted, parent]);

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
        toggleEditing,
    };

    return (
        <Box>
            {upsertDialog}
            {BulkDeleteDialogComponent}
            {
                horizontal ?
                    <ResourceListHorizontal {...childProps} /> :
                    <ResourceListVertical {...childProps} />
            }
        </Box>
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
