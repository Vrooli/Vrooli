// Displays a list of resources. If the user can modify the list, 
// it will display options for adding, removing, and sorting
import { CommonKey, Count, DeleteManyInput, DUMMY_ID, endpointPostDeleteMany, Resource, ResourceListFor, ResourceUsedFor } from "@local/shared";
import { Box, IconButton, Stack, styled, Tooltip, Typography, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { TextLoading } from "components/lists/TextLoading/TextLoading";
import { SessionContext } from "contexts/SessionContext";
import { useDebounce } from "hooks/useDebounce";
import { useLazyFetch } from "hooks/useLazyFetch";
import usePress from "hooks/usePress";
import { DeleteIcon, EditIcon, LinkIcon } from "icons";
import { forwardRef, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { DragDropContext, Draggable, Droppable, DropResult } from "react-beautiful-dnd";
import { useTranslation } from "react-i18next";
import { openLink, useLocation } from "route";
import { multiLineEllipsis, noSelect } from "styles";
import { getResourceIcon } from "utils/display/getResourceIcon";
import { getDisplay } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { getUserLanguages } from "utils/display/translationTools";
import { getResourceType, getResourceUrl } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { updateArray } from "utils/shape/general";
import { NewResourceShape, resourceInitialValues, ResourceUpsert } from "views/objects/resource";
import { ResourceListItemContextMenu } from "../ResourceListItemContextMenu/ResourceListItemContextMenu";
import { ResourceCardProps, ResourceListHorizontalProps } from "../types";

const ResourceBox = styled(Box)(({ theme }) => ({
    ...noSelect,
    display: "block",
    boxShadow: theme.shadows[4],
    background: theme.palette.primary.light,
    color: theme.palette.secondary.contrastText,
    borderRadius: "16px",
    margin: "auto",
    padding: theme.spacing(1),
    cursor: "pointer",
    width: "194px",
    minWidth: "194px",
    minHeight: "120px",
    height: "120px",
    position: "relative",
    "&:hover": {
        filter: "brightness(120%)",
        transition: "filter 0.2s",
    },
})) as any;// TODO: Fix any - https://github.com/mui/material-ui/issues/38274

const ResourceCard = forwardRef<any, ResourceCardProps>(({
    data,
    dragProps,
    dragHandleProps,
    index,
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
            title: title ? title : t((data.usedFor ?? "Context") as CommonKey),
            subtitle,
        };
    }, [data, session, t]);

    const Icon = useMemo(() => {
        return getResourceIcon(data.usedFor ?? ResourceUsedFor.Related, data.link);
    }, [data.link, data.usedFor]);

    const href = useMemo(() => getResourceUrl(data.link), [data]);
    const handleClick = useCallback((target: EventTarget) => {
        // Check if edit or delete button was clicked
        const targetId: string | undefined = target.id;
        if (targetId && targetId.startsWith("edit-")) {
            onEdit?.(index);
        }
        else if (targetId && targetId.startsWith("delete-")) {
            onDelete?.(index);
        }
        else {
            // If no resource type or link, show error
            const resourceType = getResourceType(data.link);
            if (!resourceType || !href) {
                PubSub.get().publishSnack({ messageKey: "CannotOpenLink", severity: "Error" });
                return;
            }
            // Open link
            else openLink(setLocation, href);
        }
    }, [data.link, href, index, onDelete, onEdit, setLocation]);
    const [handleClickDebounce] = useDebounce(handleClick, 100);
    const handleContextMenu = useCallback((target: EventTarget) => {
        onContextMenu(target, index);
    }, [onContextMenu, index]);

    const pressEvents = usePress({
        onLongPress: handleContextMenu,
        onClick: handleClickDebounce,
        onRightClick: handleContextMenu,
        pressDelay: isEditing ? 1500 : 300,
    });

    return (
        <Tooltip placement="top" title={`${subtitle ? subtitle + " - " : ""}${data.link}`}>
            <ResourceBox
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
                            ...multiLineEllipsis(3),
                            textAlign: "center",
                            lineBreak: title ? "auto" : "anywhere", // Line break anywhere only if showing link
                        }}
                    >
                        {firstString(title, data.link)}
                    </Typography>
                </Stack>
            </ResourceBox>
        </Tooltip>
    );
});

const LoadingCard = () => {
    return (
        <ResourceBox>
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
        </ResourceBox>
    );
};

export const ResourceListHorizontal = ({
    canUpdate = true,
    handleUpdate,
    id,
    list,
    loading = false,
    mutate = true,
    parent,
    title,
}: ResourceListHorizontalProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [isEditing, setIsEditing] = useState(false);
    useEffect(() => {
        if (!canUpdate) setIsEditing(false);
    }, [canUpdate]);

    const onDragEnd = useCallback((result: DropResult) => {
        const { source, destination } = result;
        if (!isEditing || !destination || source.index === destination.index) return;
        // Handle the reordering of the resources in the list
        if (handleUpdate && list) {
            handleUpdate({
                ...list,
                resources: updateArray(list.resources, source.index, list.resources[destination.index]) as any[],
            });
        }
    }, [isEditing, handleUpdate, list]);

    const [deleteMutation] = useLazyFetch<DeleteManyInput, Count>(endpointPostDeleteMany);
    const onDelete = useCallback((index: number) => {
        if (!list) return;
        const resource = list.resources[index];
        if (mutate && resource.id) {
            fetchLazyWrapper<DeleteManyInput, Count>({
                fetch: deleteMutation,
                inputs: { ids: [resource.id], objectType: "Resource" as any },
                onSuccess: () => {
                    if (handleUpdate) {
                        handleUpdate({
                            ...list,
                            resources: list.resources.filter(r => r.id !== resource.id) as any,
                        });
                    }
                },
            });
        }
        else if (handleUpdate) {
            handleUpdate({
                ...list,
                resources: list.resources.filter(r => r.id !== resource.id) as any,
            });
        }
    }, [deleteMutation, handleUpdate, list, mutate]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const [selectedResource, setSelectedResource] = useState<Resource | null>(null);
    const selectedIndex = useMemo(() => selectedResource ? list?.resources?.findIndex(r => r.id === selectedResource.id) : -1, [list, selectedResource]);
    const contextId = useMemo(() => `resource-context-menu-${selectedResource?.id}`, [selectedResource]);
    const openContext = useCallback((target: EventTarget, index: number) => {
        setContextAnchor(target);
        const resource = list?.resources[index];
        setSelectedResource(resource ?? null);
    }, [list?.resources]);
    const closeContext = useCallback(() => {
        setContextAnchor(null);
        setSelectedResource(null);
    }, []);

    // Add/update resource dialog
    const [editingIndex, setEditingIndex] = useState<number>(-1);
    const [isDialogOpen, setIsDialogOpen] = useState(false);
    const openDialog = useCallback(() => { setIsDialogOpen(true); }, []);
    const closeDialog = useCallback(() => { setIsDialogOpen(false); setEditingIndex(-1); }, []);
    const openUpdateDialog = useCallback((index: number) => {
        setEditingIndex(index);
        setIsDialogOpen(true);
    }, []);
    const onCompleted = useCallback((resource: Resource) => {
        closeDialog();
        if (!list || !handleUpdate) return;
        const index = resource.index;
        if (index && index >= 0) {
            handleUpdate({
                ...list,
                resources: updateArray(list.resources, index, resource) as Resource[],
            });
        }
        else {
            handleUpdate({
                ...list,
                resources: [...(list?.resources ?? []), resource],
            });
        }
    }, [closeDialog, handleUpdate, list]);

    const dialog = useMemo(() => (
        list ? <ResourceUpsert
            isCreate={editingIndex < 0}
            isOpen={isDialogOpen}
            isMutate={mutate}
            onCancel={closeDialog}
            onCompleted={onCompleted}
            overrideObject={editingIndex >= 0 && list?.resources ?
                { ...list.resources[editingIndex as number], index: editingIndex } as NewResourceShape :
                resourceInitialValues(undefined, {
                    index: 0,
                    list: list?.id && list.id !== DUMMY_ID ? { id: list.id } : { listForType: parent.__typename as ResourceListFor, listForId: parent.id },
                }) as NewResourceShape}
        /> : null
    ), [closeDialog, editingIndex, isDialogOpen, list, mutate, onCompleted, parent.__typename, parent.id]);

    return (
        <>
            {/* Add resource dialog */}
            {dialog}
            {/* Right-click context menu */}
            <ResourceListItemContextMenu
                canUpdate={canUpdate}
                id={contextId}
                anchorEl={contextAnchor}
                index={selectedIndex ?? -1}
                onClose={closeContext}
                onAddBefore={() => {
                    setEditingIndex(selectedIndex ?? 0);
                    openDialog();
                }}
                onAddAfter={() => {
                    setEditingIndex(selectedIndex ? selectedIndex + 1 : 0);
                    openDialog();
                }}
                onDelete={onDelete}
                onEdit={() => openUpdateDialog(selectedIndex ?? 0)}
                onMove={(index: number) => {
                    if (handleUpdate && list) {
                        handleUpdate({
                            ...list,
                            resources: updateArray(list.resources, selectedIndex ?? 0, list.resources[index]) as any[],
                        });
                    }
                }}
                resource={selectedResource}
            />
            {title && <Box display="flex" flexDirection="row" alignItems="center">
                <Typography component="h2" variant="h6" textAlign="left">{title}</Typography>
                {true && <Tooltip title={t("Edit")}>
                    <IconButton onClick={() => { setIsEditing(e => !e); }}>
                        <EditIcon fill={palette.secondary.main} style={{ width: "24px", height: "24px" }} />
                    </IconButton>
                </Tooltip>}
            </Box>}
            <DragDropContext onDragEnd={onDragEnd}>
                <Droppable droppableId="resource-list" direction="horizontal">
                    {(provided) => (
                        <Box
                            ref={provided.innerRef}
                            id={id}
                            {...provided.droppableProps}
                            justifyContent="flex-start"
                            alignItems="center"
                            sx={{
                                display: "flex",
                                gap: 2,
                                width: "100%",
                                marginLeft: "auto",
                                marginRight: "auto",
                                paddingTop: title ? 0 : 1,
                                paddingBottom: 1,
                                overflowX: "auto",
                            }}>
                            {/* Resources */}
                            {list?.resources?.map((c: Resource, index) => (
                                <Draggable
                                    key={`resource-card-${index}`}
                                    draggableId={`resource-card-${index}`}
                                    index={index}
                                    isDragDisabled={!isEditing}
                                >
                                    {(provided) => (
                                        <ResourceCard
                                            ref={provided.innerRef}
                                            dragProps={provided.draggableProps}
                                            dragHandleProps={provided.dragHandleProps}
                                            key={`resource-card-${index}`}
                                            index={index}
                                            isEditing={isEditing}
                                            data={c}
                                            onContextMenu={openContext}
                                            onEdit={openUpdateDialog}
                                            onDelete={onDelete}
                                            aria-owns={selectedIndex ? contextId : undefined}
                                        />
                                    )}
                                </Draggable>
                            ))}
                            {/* Dummy resources when loading */}
                            {
                                loading && !Array.isArray(list?.resources) && Array.from(Array(3).keys()).map((i) => (
                                    <LoadingCard key={`resource-card-${i}`} />
                                ))
                            }
                            {/* Add resource button */}
                            {canUpdate ? <Tooltip placement="top" title={t("CreateResource")}>
                                <ResourceBox
                                    onClick={openDialog}
                                    aria-label={t("CreateResource")}
                                    sx={{
                                        display: "flex",
                                        alignItems: "center",
                                        justifyContent: "center",
                                    }}
                                >
                                    <LinkIcon fill={palette.secondary.contrastText} width='56px' height='56px' />
                                </ResourceBox>
                            </Tooltip> : null}
                            {provided.placeholder}
                        </Box>
                    )}
                </Droppable>
            </DragDropContext>
        </>
    );
};
