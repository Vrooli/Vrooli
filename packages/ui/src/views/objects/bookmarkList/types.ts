import { type BookmarkList, type BookmarkListShape } from "@local/shared";
import { type CrudPropsDialog, type CrudPropsPage, type CrudPropsPartial, type FormProps, type ObjectViewProps } from "../../../types.js";

type BookmarkListUpsertPropsPage = CrudPropsPage;
type BookmarkListUpsertPropsDialog = CrudPropsDialog<BookmarkList>;
type BookmarkListUpsertPropsPartial = CrudPropsPartial<BookmarkList>;
export type BookmarkListUpsertProps = BookmarkListUpsertPropsPage | BookmarkListUpsertPropsDialog | BookmarkListUpsertPropsPartial;
export type BookmarkListFormProps = FormProps<BookmarkList, BookmarkListShape>
export type BookmarkListViewProps = ObjectViewProps<BookmarkList>
