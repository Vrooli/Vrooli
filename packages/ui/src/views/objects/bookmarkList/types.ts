import { BookmarkList, BookmarkListShape } from "@local/shared";
import { CrudPropsDialog, CrudPropsPage, CrudPropsPartial, FormProps, ObjectViewProps } from "../../../types.js";

type BookmarkListUpsertPropsPage = CrudPropsPage;
type BookmarkListUpsertPropsDialog = CrudPropsDialog<BookmarkList>;
type BookmarkListUpsertPropsPartial = CrudPropsPartial<BookmarkList>;
export type BookmarkListUpsertProps = BookmarkListUpsertPropsPage | BookmarkListUpsertPropsDialog | BookmarkListUpsertPropsPartial;
export type BookmarkListFormProps = FormProps<BookmarkList, BookmarkListShape>
export type BookmarkListViewProps = ObjectViewProps<BookmarkList>
