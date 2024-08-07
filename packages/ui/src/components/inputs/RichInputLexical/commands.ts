import { type LexicalNode } from "./nodes/LexicalNode";
import { BaseSelection, ElementFormatType, InsertTableCommandPayload, LexicalCommand as LC, LinkAttributes, PasteCommandType, TextFormatType } from "./types";

export function createCommand<T>(type?: string): LC<T> {
    return { type };
}

// All available commands (e.g. key commands, text formatting, etc.). 
// Commands can be registered, which allows you to listen for them and execute custom logic when they are triggered.
export const SELECTION_CHANGE_COMMAND: LC<void> = createCommand("SELECTION_CHANGE_COMMAND");
export const SELECTION_INSERT_CLIPBOARD_NODES_COMMAND: LC<{ nodes: LexicalNode[]; selection: BaseSelection }> = createCommand("SELECTION_INSERT_CLIPBOARD_NODES_COMMAND");
export const CLICK_COMMAND: LC<MouseEvent> = createCommand("CLICK_COMMAND");
export const DELETE_CHARACTER_COMMAND: LC<boolean> = createCommand("DELETE_CHARACTER_COMMAND");
export const INSERT_LINE_BREAK_COMMAND: LC<boolean> = createCommand("INSERT_LINE_BREAK_COMMAND");
export const INSERT_PARAGRAPH_COMMAND: LC<void> = createCommand("INSERT_PARAGRAPH_COMMAND");
export const CONTROLLED_TEXT_INSERTION_COMMAND: LC<InputEvent | string> = createCommand("CONTROLLED_TEXT_INSERTION_COMMAND");
export const PASTE_COMMAND: LC<PasteCommandType> = createCommand("PASTE_COMMAND");
export const REMOVE_TEXT_COMMAND: LC<InputEvent | null> = createCommand("REMOVE_TEXT_COMMAND");
export const DELETE_WORD_COMMAND: LC<boolean> = createCommand("DELETE_WORD_COMMAND");
export const DELETE_LINE_COMMAND: LC<boolean> = createCommand("DELETE_LINE_COMMAND");
export const FORMAT_TEXT_COMMAND: LC<TextFormatType> = createCommand("FORMAT_TEXT_COMMAND");
export const UNDO_COMMAND: LC<void> = createCommand("UNDO_COMMAND");
export const REDO_COMMAND: LC<void> = createCommand("REDO_COMMAND");
export const KEY_ARROW_RIGHT_COMMAND: LC<KeyboardEvent> = createCommand("KEY_ARROW_RIGHT_COMMAND");
export const MOVE_TO_END: LC<KeyboardEvent> = createCommand("MOVE_TO_END");
export const KEY_ARROW_LEFT_COMMAND: LC<KeyboardEvent> = createCommand("KEY_ARROW_LEFT_COMMAND");
export const MOVE_TO_START: LC<KeyboardEvent> = createCommand("MOVE_TO_START");
export const KEY_ARROW_UP_COMMAND: LC<KeyboardEvent> = createCommand("KEY_ARROW_UP_COMMAND");
export const KEY_ARROW_DOWN_COMMAND: LC<KeyboardEvent> = createCommand("KEY_ARROW_DOWN_COMMAND");
export const KEY_ENTER_COMMAND: LC<KeyboardEvent | null> = createCommand("KEY_ENTER_COMMAND");
export const KEY_SPACE_COMMAND: LC<KeyboardEvent> = createCommand("KEY_SPACE_COMMAND");
export const KEY_BACKSPACE_COMMAND: LC<KeyboardEvent> = createCommand("KEY_BACKSPACE_COMMAND");
export const KEY_ESCAPE_COMMAND: LC<KeyboardEvent> = createCommand("KEY_ESCAPE_COMMAND");
export const KEY_DELETE_COMMAND: LC<KeyboardEvent> = createCommand("KEY_DELETE_COMMAND");
export const KEY_TAB_COMMAND: LC<KeyboardEvent> = createCommand("KEY_TAB_COMMAND");
export const INSERT_TAB_COMMAND: LC<void> = createCommand("INSERT_TAB_COMMAND");
export const INDENT_CONTENT_COMMAND: LC<void> = createCommand("INDENT_CONTENT_COMMAND");
export const OUTDENT_CONTENT_COMMAND: LC<void> = createCommand("OUTDENT_CONTENT_COMMAND");
export const DROP_COMMAND: LC<DragEvent> = createCommand("DROP_COMMAND");
export const FORMAT_ELEMENT_COMMAND: LC<ElementFormatType> = createCommand("FORMAT_ELEMENT_COMMAND");
export const DRAGSTART_COMMAND: LC<DragEvent> = createCommand("DRAGSTART_COMMAND");
export const DRAGOVER_COMMAND: LC<DragEvent> = createCommand("DRAGOVER_COMMAND");
export const DRAGEND_COMMAND: LC<DragEvent> = createCommand("DRAGEND_COMMAND");
export const COPY_COMMAND: LC<ClipboardEvent | KeyboardEvent | null> = createCommand("COPY_COMMAND");
export const CUT_COMMAND: LC<ClipboardEvent | KeyboardEvent | null> = createCommand("CUT_COMMAND");
export const SELECT_ALL_COMMAND: LC<KeyboardEvent> = createCommand("SELECT_ALL_COMMAND");
export const CLEAR_EDITOR_COMMAND: LC<void> = createCommand("CLEAR_EDITOR_COMMAND");
export const CLEAR_HISTORY_COMMAND: LC<void> = createCommand("CLEAR_HISTORY_COMMAND");
export const CAN_REDO_COMMAND: LC<boolean> = createCommand("CAN_REDO_COMMAND");
export const CAN_UNDO_COMMAND: LC<boolean> = createCommand("CAN_UNDO_COMMAND");
export const FOCUS_COMMAND: LC<FocusEvent> = createCommand("FOCUS_COMMAND");
export const BLUR_COMMAND: LC<FocusEvent> = createCommand("BLUR_COMMAND");
export const KEY_MODIFIER_COMMAND: LC<KeyboardEvent> = createCommand("KEY_MODIFIER_COMMAND");
export const CODE_BLOCK_COMMAND = createCommand("CODE_BLOCK_COMMAND");
export const TOGGLE_LINK_COMMAND: LC<string | LinkAttributes | null> = createCommand("TOGGLE_LINK_COMMAND");
export const INSERT_UNORDERED_LIST_COMMAND: LC<void> = createCommand("INSERT_UNORDERED_LIST_COMMAND");
export const INSERT_CHECK_LIST_COMMAND: LC<void> = createCommand("INSERT_CHECK_LIST_COMMAND");
export const INSERT_ORDERED_LIST_COMMAND: LC<void> = createCommand("INSERT_ORDERED_LIST_COMMAND");
export const INSERT_HORIZONTAL_RULE_COMMAND: LC<void> = createCommand("INSERT_HORIZONTAL_RULE_COMMAND");
export const REMOVE_LIST_COMMAND: LC<void> = createCommand("REMOVE_LIST_COMMAND");
export const INSERT_TABLE_COMMAND: LC<InsertTableCommandPayload> = createCommand("INSERT_TABLE_COMMAND");
export const DRAG_DROP_PASTE: LC<File[]> = createCommand("DRAG_DROP_PASTE_FILE");
