/**
 * Displays all search options for an organization
 */
import { Bookmark, BookmarkCreateInput, BookmarkFor, BookmarkList, BookmarkSearchInput, BookmarkSearchResult, Count, DeleteManyInput, DeleteType, endpointGetBookmarks, endpointPostBookmark, endpointPostDeleteMany, lowercaseFirstLetter, uuid } from "@local/shared";
import { Checkbox, DialogTitle, FormControlLabel, IconButton, List, ListItem, useTheme } from "@mui/material";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { SessionContext } from "contexts/SessionContext";
import { useLazyFetch } from "hooks/useLazyFetch";
import { AddIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { shapeBookmark } from "utils/shape/models/bookmark";
import { BookmarkListUpsert } from "views/objects/bookmarkList";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { SelectBookmarkListDialogProps } from "../types";

const titleId = "select-bookmark-dialog-title";

export const SelectBookmarkListDialog = ({
    objectId,
    objectType,
    onClose,
    isCreate,
    isOpen,
}: SelectBookmarkListDialogProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [lists, setLists] = useState<BookmarkList[]>(getCurrentUser(session).bookmarkLists ?? []);
    const [selectedLists, setSelectedLists] = useState<BookmarkList[]>([]);

    useEffect(() => {
        setLists(getCurrentUser(session).bookmarkLists ?? []);
    }, [session]);

    // Fetch all bookmarks for object
    const [refetch, { data, loading: isFindLoading }] = useLazyFetch<BookmarkSearchInput, BookmarkSearchResult>({
        ...endpointGetBookmarks,
        inputs: { [`${lowercaseFirstLetter(objectType)}Id`]: objectId! },
    });
    useEffect(() => {
        if (!isCreate && isOpen) {
            refetch();
        } else {
            setSelectedLists([]);
        }
    }, [refetch, isCreate, isOpen, objectId]);
    useEffect(() => {
        if (data) {
            setSelectedLists(data.edges.map(e => e.node.list));
        }
    }, [data]);

    const [create, { loading: isCreateLoading }] = useLazyFetch<BookmarkCreateInput, Bookmark>(endpointPostBookmark);
    const [deleteMutation, { loading: isDeleteLoading }] = useLazyFetch<DeleteManyInput, Count>(endpointPostDeleteMany);
    const handleSubmit = useCallback(async () => {
        // Iterate over selected lists
        for (const list of selectedLists) {
            // If the list was not already selected, add the bookmark
            if (isCreate || !data?.edges.some(e => e.node.id === list.id)) {
                await create(shapeBookmark.create({
                    __typename: "Bookmark",
                    id: uuid(),
                    to: {
                        __typename: objectType as BookmarkFor,
                        id: objectId!,
                    },
                    list: {
                        __typename: "BookmarkList",
                        id: list.id,
                    },
                }));
            }
        }
        // Iterate over non-selected lists
        const deletedBookmarks = data?.edges.filter(e => !selectedLists.some(sl => sl.id === e.node.list.id));
        if (deletedBookmarks) {
            await deleteMutation({
                ids: deletedBookmarks.map(e => e.node.id),
                objectType: DeleteType.Bookmark,
            });
        }
        onClose(selectedLists.length > 0);
    }, [selectedLists, create, data?.edges, deleteMutation, isCreate, objectId, objectType, onClose]);

    const onCancel = useCallback(() => {
        setSelectedLists([]);
        onClose(selectedLists.length > 0);
    }, [onClose, selectedLists.length, setSelectedLists]);

    const [isCreateOpen, setIsCreateOpen] = useState(false);
    const openCreate = useCallback(() => { setIsCreateOpen(true); }, []);
    const closeCreate = useCallback(() => { setIsCreateOpen(false); }, []);
    const onCreated = useCallback((bookmarkList: BookmarkList) => {
        setLists([...lists, bookmarkList]);
        setSelectedLists([bookmarkList]);
    }, [lists]);
    const onDeleted = useCallback((bookmarkList: BookmarkList) => {
        setLists(lists.filter(l => l.id !== bookmarkList.id));
        setSelectedLists([]);
    }, [lists]);

    const listItems = useMemo(() => lists.sort((a, b) => a.label.localeCompare(b.label)).map(list => (
        <ListItem key={list.id} onClick={() => {
            if (selectedLists.some(l => l.id === list.id)) {
                setSelectedLists(selectedLists.filter(l => l.id !== list.id));
            } else {
                setSelectedLists([...selectedLists, list]);
            }
        }}>
            <FormControlLabel
                control={
                    <Checkbox
                        checked={selectedLists.some(l => l.id === list.id)}
                    />
                }
                label={`${list.label} (${list.bookmarksCount})`}
            />
        </ListItem>
    )), [lists, selectedLists]);

    return (
        <>
            <BookmarkListUpsert
                display="dialog"
                isCreate={true}
                isOpen={isCreateOpen && isOpen}
                onCancel={closeCreate}
                onClose={closeCreate}
                onCompleted={onCreated}
                onDeleted={onDeleted}
                overrideObject={{ __typename: "BookmarkList" }}
            />
            {/* Main dialog */}
            <LargeDialog
                id="select-bookmark-list-dialog"
                isOpen={isOpen}
                onClose={() => onClose(selectedLists.length > 0)}
                titleId={titleId}
            >
                {/* Top bar with title and add list button */}
                <DialogTitle id={titleId} sx={{ display: "flex" }}>
                    {t("SelectLists")}
                    <IconButton
                        edge="end"
                        onClick={openCreate}
                        sx={{
                            marginLeft: "auto",
                        }}
                    >
                        <AddIcon fill={palette.secondary.main} />
                    </IconButton>
                </DialogTitle>
                {/* Checkmarked list (sorted alphabetically)  */}
                <List sx={{
                    // Make sure the list is display above both the BottomNav and any 
                    // snack messages
                    paddingBottom: "calc(128px + env(safe-area-inset-bottom))",
                }}>
                    {listItems}
                </List>
                {/* Search/Cancel buttons */}
                <BottomActionsButtons
                    display={"dialog"}
                    isCreate={false}
                    loading={isFindLoading || isCreateLoading || isDeleteLoading}
                    onCancel={onCancel}
                    onSubmit={handleSubmit}
                />
            </LargeDialog>
        </>
    );
};
