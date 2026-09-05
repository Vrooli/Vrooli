import { DeclarationsPage } from "./DeclarationsPage";
import { DocumentPage } from "./DocumentPage";
import { StylesPage } from "./StylesPage";
import { VariationPage, VariationBoard } from "./VariationPage";

export { VariationBoard };

type Surface = "variation" | "styles" | "document" | "declarations";

export function ProseSurfacePage({ surface }: { surface: Surface }) {
  if (surface === "variation") return <VariationPage />;
  if (surface === "styles") return <StylesPage />;
  if (surface === "document") return <DocumentPage />;
  return <DeclarationsPage />;
}
