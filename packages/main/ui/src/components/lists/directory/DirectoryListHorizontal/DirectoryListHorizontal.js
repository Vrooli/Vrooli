import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { LinkIcon } from "@local/icons";
import { Box, CircularProgress, Tooltip, useTheme } from "@mui/material";
import { useCallback, useContext, useMemo, useState } from "react";
import { deleteOneOrManyDeleteMany } from "../../../../api/generated/endpoints/deleteOneOrMany_deleteMany";
import { useCustomMutation } from "../../../../api/hooks";
import { mutationWrapper } from "../../../../api/utils";
import { getDisplay } from "../../../../utils/display/listTools";
import { getUserLanguages } from "../../../../utils/display/translationTools";
import { PubSub } from "../../../../utils/pubsub";
import { SessionContext } from "../../../../utils/SessionContext";
import { FindObjectDialog } from "../../../dialogs/FindObjectDialog/FindObjectDialog";
import { CardGrid } from "../../CardGrid/CardGrid";
import { cardRoot } from "../../styles";
import { DirectoryCard } from "../DirectoryCard/DirectoryCard";
import { DirectoryListItemContextMenu } from "../DirectoryListItemContextMenu/DirectoryListItemContextMenu";
export const DirectoryListHorizontal = ({ canUpdate = true, directory, handleUpdate, loading = false, mutate = true, zIndex, }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const list = useMemo(() => {
        if (!directory)
            return [];
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
            const { title: aTitle } = getDisplay(a, getUserLanguages(session));
            const { title: bTitle } = getDisplay(b, getUserLanguages(session));
            return aTitle.localeCompare(bTitle);
        });
    }, [directory, session]);
    const [contextAnchor, setContextAnchor] = useState(null);
    const [selectedItem, setSelectedItem] = useState(null);
    const selectedIndex = useMemo(() => selectedItem ? list.findIndex(r => r.id === selectedItem.id) : -1, [list, selectedItem]);
    const contextId = useMemo(() => `directory-context-menu-${selectedItem?.id}`, [selectedItem]);
    const openContext = useCallback((target, index) => {
        setContextAnchor(target);
        const resource = list[index];
        setSelectedItem(resource);
    }, [list]);
    const closeContext = useCallback(() => {
        setContextAnchor(null);
        setSelectedItem(null);
    }, []);
    const [isDialogOpen, setIsDialogOpen] = useState(false);
    const openDialog = useCallback(() => { setIsDialogOpen(true); }, []);
    const closeDialog = useCallback(() => { setIsDialogOpen(false); }, []);
    const onAdd = useCallback((item) => {
        console.log("directoryyy onAdd", item);
        if (!directory)
            return;
        if (list.some(r => r.id === item.id)) {
            PubSub.get().publishSnack({ message: "Item already added", severity: "Warning" });
            return;
        }
        if (handleUpdate) {
            console.log("directoryyy handleUpdate", {
                ...directory,
                childApiVersions: item.__typename === "ApiVersion" ? [...directory.childApiVersions, item] : directory.childApiVersions,
                childNoteVersions: item.__typename === "NoteVersion" ? [...directory.childNoteVersions, item] : directory.childNoteVersions,
                childOrganizations: item.__typename === "Organization" ? [...directory.childOrganizations, item] : directory.childOrganizations,
                childProjectVersions: item.__typename === "ProjectVersion" ? [...directory.childProjectVersions, item] : directory.childProjectVersions,
                childRoutineVersions: item.__typename === "RoutineVersion" ? [...directory.childRoutineVersions, item] : directory.childRoutineVersions,
                childSmartContractVersions: item.__typename === "SmartContractVersion" ? [...directory.childSmartContractVersions, item] : directory.childSmartContractVersions,
                childStandardVersions: item.__typename === "StandardVersion" ? [...directory.childStandardVersions, item] : directory.childStandardVersions,
            });
            handleUpdate({
                ...directory,
                childApiVersions: item.__typename === "ApiVersion" ? [...directory.childApiVersions, item] : directory.childApiVersions,
                childNoteVersions: item.__typename === "NoteVersion" ? [...directory.childNoteVersions, item] : directory.childNoteVersions,
                childOrganizations: item.__typename === "Organization" ? [...directory.childOrganizations, item] : directory.childOrganizations,
                childProjectVersions: item.__typename === "ProjectVersion" ? [...directory.childProjectVersions, item] : directory.childProjectVersions,
                childRoutineVersions: item.__typename === "RoutineVersion" ? [...directory.childRoutineVersions, item] : directory.childRoutineVersions,
                childSmartContractVersions: item.__typename === "SmartContractVersion" ? [...directory.childSmartContractVersions, item] : directory.childSmartContractVersions,
                childStandardVersions: item.__typename === "StandardVersion" ? [...directory.childStandardVersions, item] : directory.childStandardVersions,
            });
        }
        closeDialog();
    }, [closeDialog, directory, handleUpdate, list]);
    const [deleteMutation] = useCustomMutation(deleteOneOrManyDeleteMany);
    const onDelete = useCallback((index) => {
        if (!directory)
            return;
        const item = list[index];
        if (mutate) {
            mutationWrapper({
                mutation: deleteMutation,
                input: { ids: [item.id], objectType: item.__typename },
                onSuccess: () => {
                    if (handleUpdate) {
                        handleUpdate({
                            ...directory,
                            childApiVersions: item.__typename === "ApiVersion" ? directory.childApiVersions.filter(i => i.id !== item.id) : directory.childApiVersions,
                            childNoteVersions: item.__typename === "NoteVersion" ? directory.childNoteVersions.filter(i => i.id !== item.id) : directory.childNoteVersions,
                            childOrganizations: item.__typename === "Organization" ? directory.childOrganizations.filter(i => i.id !== item.id) : directory.childOrganizations,
                            childProjectVersions: item.__typename === "ProjectVersion" ? directory.childProjectVersions.filter(i => i.id !== item.id) : directory.childProjectVersions,
                            childRoutineVersions: item.__typename === "RoutineVersion" ? directory.childRoutineVersions.filter(i => i.id !== item.id) : directory.childRoutineVersions,
                            childSmartContractVersions: item.__typename === "SmartContractVersion" ? directory.childSmartContractVersions.filter(i => i.id !== item.id) : directory.childSmartContractVersions,
                            childStandardVersions: item.__typename === "StandardVersion" ? directory.childStandardVersions.filter(i => i.id !== item.id) : directory.childStandardVersions,
                        });
                    }
                },
            });
        }
        else if (handleUpdate) {
            handleUpdate({
                ...directory,
                childApiVersions: item.__typename === "ApiVersion" ? directory.childApiVersions.filter(i => i.id !== item.id) : directory.childApiVersions,
                childNoteVersions: item.__typename === "NoteVersion" ? directory.childNoteVersions.filter(i => i.id !== item.id) : directory.childNoteVersions,
                childOrganizations: item.__typename === "Organization" ? directory.childOrganizations.filter(i => i.id !== item.id) : directory.childOrganizations,
                childProjectVersions: item.__typename === "ProjectVersion" ? directory.childProjectVersions.filter(i => i.id !== item.id) : directory.childProjectVersions,
                childRoutineVersions: item.__typename === "RoutineVersion" ? directory.childRoutineVersions.filter(i => i.id !== item.id) : directory.childRoutineVersions,
                childSmartContractVersions: item.__typename === "SmartContractVersion" ? directory.childSmartContractVersions.filter(i => i.id !== item.id) : directory.childSmartContractVersions,
                childStandardVersions: item.__typename === "StandardVersion" ? directory.childStandardVersions.filter(i => i.id !== item.id) : directory.childStandardVersions,
            });
        }
    }, [deleteMutation, directory, handleUpdate, list, mutate]);
    return (_jsxs(_Fragment, { children: [_jsx(FindObjectDialog, { find: "List", isOpen: isDialogOpen, handleCancel: closeDialog, handleComplete: onAdd, zIndex: zIndex + 1 }), _jsx(DirectoryListItemContextMenu, { canUpdate: canUpdate, id: contextId, anchorEl: contextAnchor, index: selectedIndex ?? -1, onClose: closeContext, onDelete: onDelete, data: selectedItem, zIndex: zIndex + 1 }), _jsxs(CardGrid, { minWidth: 120, sx: {}, children: [list.map((c, index) => (_jsx(DirectoryCard, { canUpdate: canUpdate, index: index, data: c, onContextMenu: openContext, onDelete: onDelete, "aria-owns": Boolean(selectedIndex) ? contextId : undefined }, `directory-item-card-${index}`))), loading && (_jsx(CircularProgress, { sx: {
                            position: "absolute",
                            top: "50%",
                            left: "50%",
                            transform: "translate(-50%, -50%)",
                            color: palette.mode === "light" ? palette.secondary.light : "white",
                        } })), (canUpdate ? _jsx(Tooltip, { placement: "top", title: "Add item", children: _jsx(Box, { onClick: openDialog, "aria-label": "Add item", sx: {
                                ...cardRoot,
                                background: palette.primary.light,
                                width: "120px",
                                minWidth: "120px",
                                height: "120px",
                                minHeight: "120px",
                                display: "flex",
                                alignItems: "center",
                                justifyContent: "center",
                            }, children: _jsx(LinkIcon, { fill: palette.secondary.contrastText, width: '56px', height: '56px' }) }) }) : null)] })] }));
};
//# sourceMappingURL=DirectoryListHorizontal.js.map