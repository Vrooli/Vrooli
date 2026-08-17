import { useQuery } from "@tanstack/react-query";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";

import { componentsClient } from "../api/components";
import { ComponentEditor } from "../features/components/ComponentEditor";

/** Full-viewport preview surface used by pop-out and deep-link workflows. */
export function PreviewPopoutPage() {
  const { id = "" } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const view = searchParams.get("view") === "canvas" ? false : true;
  const componentQuery = useQuery({
    queryKey: ["components", "preview-popout", id],
    queryFn: () => componentsClient.getComponent({ id }),
    enabled: Boolean(id),
  });
  const libraryId = componentQuery.data?.component?.libraryId || id;

  return (
    <main
      data-testid="preview-popout"
      aria-label={`Preview ${libraryId}`}
      data-preview-story={searchParams.get("story") || ""}
      data-preview-view={view ? "focus" : "canvas"}
      className="flex h-screen min-h-0 w-screen flex-col overflow-hidden bg-app-background"
    >
      <ComponentEditor
        id={id}
        libraryId={libraryId}
        renderable
        stageMode={view}
        chromeless
        activePane="preview"
        selectedStory={searchParams.get("story") || undefined}
        onSelectedStoryChange={(story) => {
          const next = new URLSearchParams(searchParams);
          next.set("story", story);
          void navigate({ search: `?${next.toString()}` }, { replace: true });
        }}
        onActivePaneChange={() => undefined}
        onClose={() => void navigate("/")}
      />
    </main>
  );
}
