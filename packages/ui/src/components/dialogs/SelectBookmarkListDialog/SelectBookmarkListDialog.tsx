/**
 * Displays all search options for an organization
 */
import { AddIcon, Bookmark, BookmarkCreateInput, BookmarkFor, BookmarkList, BookmarkSearchInput, BookmarkSearchResult, Count, DeleteManyInput, lowercaseFirstLetter, uuid } from "@local/shared";
import { Checkbox, DialogTitle, FormControlLabel, IconButton, List, ListItem, useTheme } from "@mui/material";
import { useCustomLazyQuery, useCustomMutation } from "api";
import { bookmarkCreate } from "api/generated/endpoints/bookmark_create";
import { bookmarkFindMany } from "api/generated/endpoints/bookmark_findMany";
import { deleteOneOrManyDeleteMany } from "api/generated/endpoints/deleteOneOrMany_deleteMany";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { useDisplayApolloError } from "utils/hooks/useDisplayApolloError";
import { SessionContext } from "utils/SessionContext";
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
    zIndex,
}: SelectBookmarkListDialogProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [lists, setLists] = useState<BookmarkList[]>(getCurrentUser(session).bookmarkLists ?? []);
    const [selectedLists, setSelectedLists] = useState<BookmarkList[]>([]);
    console.log("listssssssss", lists);
    console.log("selected listssssssss", selectedLists);

    useEffect(() => {
        setLists(getCurrentUser(session).bookmarkLists ?? []);
    }, [session]);

    // Fetch all bookmarks for object
    const [refetch, { data, loading: isFindLoading, error }] = useCustomLazyQuery<BookmarkSearchResult, BookmarkSearchInput>(bookmarkFindMany, {
        variables: {
            [`${lowercaseFirstLetter(objectType)}Id`]: objectId!,
        },
        errorPolicy: "all",
    });
    useEffect(() => {
        if (!isCreate && isOpen) {
            refetch();
        } else {
            setSelectedLists([]);
        }
    }, [refetch, isCreate, objectId]);
    useDisplayApolloError(error);
    useEffect(() => {
        if (data) {
            setSelectedLists(data.edges.map(e => e.node.list));
        }
    }, [data]);

    const [create, { loading: isCreateLoading }] = useCustomMutation<Bookmark, BookmarkCreateInput>(bookmarkCreate);
    const [deleteMutation, { loading: isDeleteLoading }] = useCustomMutation<Count, DeleteManyInput>(deleteOneOrManyDeleteMany);
    const handleSubmit = useCallback(async () => {
        // Iterate over selected lists
        for (const list of selectedLists) {
            // If the list was not already selected, add the bookmark
            if (isCreate || !data?.edges.some(e => e.node.id === list.id)) {
                await create({
                    variables: {
                        input: shapeBookmark.create({
                            id: uuid(),
                            bookmarkFor: {
                                __typename: objectType as BookmarkFor,
                                id: objectId!,
                            },
                            list: {
                                __typename: "BookmarkList",
                                id: list.id,
                            },
                        }),
                    },
                });
            }
        }
        // Iterate over non-selected lists
        const deletedBookmarks = data?.edges.filter(e => !selectedLists.some(sl => sl.id === e.node.list.id));
        if (deletedBookmarks) {
            await deleteMutation({
                variables: {
                    input: {
                        ids: deletedBookmarks.map(e => e.node.id),
                        objectType: "Bookmark",
                    },
                },
            });
        }
        onClose(selectedLists.length > 0);
    }, [selectedLists, create, deleteMutation, lists, objectId, onClose]);

    const onCancel = useCallback(() => {
        setSelectedLists([]);
        onClose(selectedLists.length > 0);
    }, [setSelectedLists]);

    const [isCreateOpen, setIsCreateOpen] = useState(false);
    const openCreate = useCallback(() => { setIsCreateOpen(true); }, []);
    const closeCreate = useCallback(() => { setIsCreateOpen(false); }, []);
    const onCreated = useCallback((bookmarkList: BookmarkList) => {
        setLists([...lists, bookmarkList]);
        setSelectedLists([bookmarkList]);
    }, []);

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
            {/* Dialog for creating a new bookmark list */}
            <LargeDialog
                id="add-bookmark-list-dialog"
                onClose={closeCreate}
                isOpen={isCreateOpen && isOpen}
                zIndex={zIndex + 1}
            >
                <BookmarkListUpsert
                    display="dialog"
                    isCreate={true}
                    onCancel={closeCreate}
                    onCompleted={onCreated}
                    zIndex={zIndex + 1}
                />
            </LargeDialog>
            {/* Main dialog */}
            <LargeDialog
                id="select-bookmark-list-dialog"
                isOpen={isOpen}
                onClose={() => onClose(selectedLists.length > 0)}
                titleId={titleId}
                zIndex={zIndex}
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
                <GridSubmitButtons
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