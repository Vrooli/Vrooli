// Displays a list of resources. If the user can modify the list, 
// it will display options for adding, removing, and sorting
import { Count, DeleteManyInput, endpointPostDeleteMany } from "@local/shared";
import { Box, CircularProgress, Tooltip, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog";
import { CardGrid } from "components/lists/CardGrid/CardGrid";
import { cardRoot } from "components/lists/styles";
import { SessionContext } from "contexts/SessionContext";
import { useLazyFetch } from "hooks/useLazyFetch";
import { LinkIcon } from "icons";
import { useCallback, useContext, useMemo, useState } from "react";
import { getDisplay } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { DirectoryCard } from "../DirectoryCard/DirectoryCard";
import { DirectoryListItemContextMenu } from "../DirectoryListItemContextMenu/DirectoryListItemContextMenu";
import { DirectoryItem, DirectoryListHorizontalProps } from "../types";

export const DirectoryListHorizontal = ({
    canUpdate = true,
    directory,
    handleUpdate,
    loading = false,
    mutate = true,
    zIndex,
}: DirectoryListHorizontalProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();

    // Organize items into a single list by name. 
    // TODO add options to sort by created_at, updated_at, maybe other things
    const list = useMemo(() => {
        if (!directory) return [];
        const items = [
            ...directory.childApiVersions,
            ...directory.childNoteVersions,
            ...directory.childOrganizations,
            ...directory.childProjectVersions,
            ...directory.childRoutineVersions,
            ...directory.childSmartContractVersions,
            ...directory.childStandardVersions,
        ];
        return items.sort((a, b) => {
            const { title: aTitle } = getDisplay(a as any, getUserLanguages(session));
            const { title: bTitle } = getDisplay(b as any, getUserLanguages(session));
            return aTitle.localeCompare(bTitle);
        });
    }, [directory, session]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const [selectedItem, setSelectedItem] = useState<DirectoryItem | null>(null);
    const selectedIndex = useMemo(() => selectedItem ? list.findIndex(r => r.id === selectedItem.id) : -1, [list, selectedItem]);
    const contextId = useMemo(() => `directory-context-menu-${selectedItem?.id}`, [selectedItem]);
    const openContext = useCallback((target: EventTarget, index: number) => {
        setContextAnchor(target);
        const resource = list[index];
        setSelectedItem(resource as any);
    }, [list]);
    const closeContext = useCallback(() => {
        setContextAnchor(null);
        setSelectedItem(null);
    }, []);

    // Add item dialog
    const [isDialogOpen, setIsDialogOpen] = useState(false);
    const openDialog = useCallback(() => { setIsDialogOpen(true); }, []);
    const closeDialog = useCallback(() => { setIsDialogOpen(false); }, []);

    const onAdd = useCallback((item: DirectoryItem) => {
        console.log("directoryyy onAdd", item);
        if (!directory) return;
        // Dont add duplicates
        if (list.some(r => r.id === item.id)) {
            PubSub.get().publishSnack({ message: "Item already added", severity: "Warning" });
            return;
        }
        if (handleUpdate) {
            console.log("directoryyy handleUpdate", {
                ...directory,
                childApiVersions: item.__typename === "ApiVersion" ? [...directory.childApiVersions, item] : directory.childApiVersions as any[],
                childNoteVersions: item.__typename === "NoteVersion" ? [...directory.childNoteVersions, item] : directory.childNoteVersions as any[],
                childOrganizations: item.__typename === "Organization" ? [...directory.childOrganizations, item] : directory.childOrganizations as any[],
                childProjectVersions: item.__typename === "ProjectVersion" ? [...directory.childProjectVersions, item] : directory.childProjectVersions as any[],
                childRoutineVersions: item.__typename === "RoutineVersion" ? [...directory.childRoutineVersions, item] : directory.childRoutineVersions as any[],
                childSmartContractVersions: item.__typename === "SmartContractVersion" ? [...directory.childSmartContractVersions, item] : directory.childSmartContractVersions as any[],
                childStandardVersions: item.__typename === "StandardVersion" ? [...directory.childStandardVersions, item] : directory.childStandardVersions as any[],
            });
            handleUpdate({
                ...directory,
                childApiVersions: item.__typename === "ApiVersion" ? [...directory.childApiVersions, item] : directory.childApiVersions as any[],
                childNoteVersions: item.__typename === "NoteVersion" ? [...directory.childNoteVersions, item] : directory.childNoteVersions as any[],
                childOrganizations: item.__typename === "Organization" ? [...directory.childOrganizations, item] : directory.childOrganizations as any[],
                childProjectVersions: item.__typename === "ProjectVersion" ? [...directory.childProjectVersions, item] : directory.childProjectVersions as any[],
                childRoutineVersions: item.__typename === "RoutineVersion" ? [...directory.childRoutineVersions, item] : directory.childRoutineVersions as any[],
                childSmartContractVersions: item.__typename === "SmartContractVersion" ? [...directory.childSmartContractVersions, item] : directory.childSmartContractVersions as any[],
                childStandardVersions: item.__typename === "StandardVersion" ? [...directory.childStandardVersions, item] : directory.childStandardVersions as any[],
            });
        }
        closeDialog();
    }, [closeDialog, directory, handleUpdate, list]);

    const [deleteMutation] = useLazyFetch<DeleteManyInput, Count>(endpointPostDeleteMany);
    const onDelete = useCallback((index: number) => {
        if (!directory) return;
        const item = list[index];
        if (mutate) {
            fetchLazyWrapper<DeleteManyInput, Count>({
                fetch: deleteMutation,
                inputs: { ids: [item.id], objectType: item.__typename as any },
                onSuccess: () => {
                    if (handleUpdate) {
                        handleUpdate({
                            ...directory,
                            childApiVersions: item.__typename === "ApiVersion" ? directory.childApiVersions.filter(i => i.id !== item.id) : directory.childApiVersions as any[],
                            childNoteVersions: item.__typename === "NoteVersion" ? directory.childNoteVersions.filter(i => i.id !== item.id) : directory.childNoteVersions as any[],
                            childOrganizations: item.__typename === "Organization" ? directory.childOrganizations.filter(i => i.id !== item.id) : directory.childOrganizations as any[],
                            childProjectVersions: item.__typename === "ProjectVersion" ? directory.childProjectVersions.filter(i => i.id !== item.id) : directory.childProjectVersions as any[],
                            childRoutineVersions: item.__typename === "RoutineVersion" ? directory.childRoutineVersions.filter(i => i.id !== item.id) : directory.childRoutineVersions as any[],
                            childSmartContractVersions: item.__typename === "SmartContractVersion" ? directory.childSmartContractVersions.filter(i => i.id !== item.id) : directory.childSmartContractVersions as any[],
                            childStandardVersions: item.__typename === "StandardVersion" ? directory.childStandardVersions.filter(i => i.id !== item.id) : directory.childStandardVersions as any[],
                        });
                    }
                },
            });
        }
        else if (handleUpdate) {
            handleUpdate({
                ...directory,
                childApiVersions: item.__typename === "ApiVersion" ? directory.childApiVersions.filter(i => i.id !== item.id) : directory.childApiVersions as any[],
                childNoteVersions: item.__typename === "NoteVersion" ? directory.childNoteVersions.filter(i => i.id !== item.id) : directory.childNoteVersions as any[],
                childOrganizations: item.__typename === "Organization" ? directory.childOrganizations.filter(i => i.id !== item.id) : directory.childOrganizations as any[],
                childProjectVersions: item.__typename === "ProjectVersion" ? directory.childProjectVersions.filter(i => i.id !== item.id) : directory.childProjectVersions as any[],
                childRoutineVersions: item.__typename === "RoutineVersion" ? directory.childRoutineVersions.filter(i => i.id !== item.id) : directory.childRoutineVersions as any[],
                childSmartContractVersions: item.__typename === "SmartContractVersion" ? directory.childSmartContractVersions.filter(i => i.id !== item.id) : directory.childSmartContractVersions as any[],
                childStandardVersions: item.__typename === "StandardVersion" ? directory.childStandardVersions.filter(i => i.id !== item.id) : directory.childStandardVersions as any[],
            });
        }
    }, [deleteMutation, directory, handleUpdate, list, mutate]);

    return (
        <>
            {/* Add item dialog */}
            <FindObjectDialog
                find="List"
                isOpen={isDialogOpen}
                handleCancel={closeDialog}
                handleComplete={onAdd as any}
                zIndex={zIndex + 1}
            />
            {/* Right-click context menu */}
            <DirectoryListItemContextMenu
                canUpdate={canUpdate}
                id={contextId}
                anchorEl={contextAnchor}
                index={selectedIndex ?? -1}
                onClose={closeContext}
                onDelete={onDelete}
                data={selectedItem}
                zIndex={zIndex + 1}
            />
            {/* {title && <Typography component="h2" variant="h5" textAlign="left">{title}</Typography>} */}
            <CardGrid minWidth={120} sx={{

            }}>
                {/* Directory items */}
                {list.map((c: DirectoryItem, index) => (
                    <DirectoryCard
                        canUpdate={canUpdate}
                        key={`directory-item-card-${index}`}
                        index={index}
                        data={c}
                        onContextMenu={openContext}
                        onDelete={onDelete}
                        aria-owns={selectedIndex ? contextId : undefined}
                    />
                )) as any}
                {
                    loading && (
                        <CircularProgress sx={{
                            position: "absolute",
                            top: "50%",
                            left: "50%",
                            transform: "translate(-50%, -50%)",
                            color: palette.mode === "light" ? palette.secondary.light : "white",
                        }} />
                    )
                }
                {/* Add item button */}
                {(canUpdate ? <Tooltip placement="top" title="Add item">
                    <Box
                        onClick={openDialog}
                        aria-label="Add item"
                        sx={{
                            ...cardRoot,
                            background: palette.primary.light,
                            width: "120px",
                            minWidth: "120px",
                            height: "120px",
                            minHeight: "120px",
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "center",
                        }}
                    >
                        <LinkIcon fill={palette.secondary.contrastText} width='56px' height='56px' />
                    </Box>
                </Tooltip> : null) as any}
            </CardGrid>
        </>
    );
};
