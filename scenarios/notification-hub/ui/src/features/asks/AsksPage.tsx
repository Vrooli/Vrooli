import { useDeskLayout } from "../../hooks/useDeskLayout";
import { DeskPage } from "./DeskPage";
import { RequestList } from "./HistoryPage";

/**
 * The queue: a decision room on a wide screen, the single-column queue below
 * the desk width. History always keeps the list composition.
 */
export function AsksPage() {
  const desk = useDeskLayout();
  return desk ? <DeskPage /> : <RequestList queue />;
}
