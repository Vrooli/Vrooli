import { BookmarkList, BookmarkListShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, FormProps, ObjectViewProps } from "../../../types";

type BookmarkListUpsertPropsPage = CrudPropsPage;
type BookmarkListUpsertPropsDialog = CrudPropsDialog<BookmarkList>;
export type BookmarkListUpsertProps = BookmarkListUpsertPropsPage | BookmarkListUpsertPropsDialog;
export type BookmarkListFormProps = FormProps<BookmarkList, BookmarkListShape>
export type BookmarkListViewProps = ObjectViewProps<BookmarkList>
