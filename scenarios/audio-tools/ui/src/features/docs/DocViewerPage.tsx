import { useEffect, useState } from "react";
import { ArrowLeft } from "lucide-react";
import { Link, useParams } from "react-router-dom";
import { Button } from "../../components/ui/button";
import { Panel } from "../../components/ui/panel";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { getDocContent } from "./docsContent";
import { MarkdownRenderer } from "./MarkdownRenderer";

export function DocViewerPage() {
  const { t } = useTranslation();
  const params = useParams<{ "*": string }>();
  const path = params["*"] ?? "";
  const [document, setDocument] = useState<{
    path: string;
    state: "loading" | "loaded" | "missing" | "failed";
    content?: string;
  }>({ path: "", state: "missing" });
  const title = deriveTitle(path);

  useEffect(() => {
    let cancelled = false;
    if (!path) {
      return () => { cancelled = true; };
    }

    setDocument({ path, state: "loading" });
    void getDocContent(path).then(
      (content) => {
        if (cancelled) return;
        setDocument(content === null ? { path, state: "missing" } : { path, state: "loaded", content });
      },
      () => {
        if (!cancelled) setDocument({ path, state: "failed" });
      },
    );
    return () => { cancelled = true; };
  }, [path]);

  const currentDocument = document.path === path ? document : { path, state: path ? "loading" as const : "missing" as const };

  return (
    <div className="flex max-w-4xl flex-col gap-4 md:gap-6">
      <div>
        <Button variant="outline" size="sm" asChild>
          <Link to="/docs">
            <ArrowLeft className="me-1 h-4 w-4" aria-hidden="true" />
            {t(strings.docs.backToList)}
          </Link>
        </Button>
      </div>
      <PageHeader title={title} description={path} />
      <Panel title={path || t(strings.docs.title)}>
        {currentDocument.state === "loading" ? (
          <p role="status" className="text-sm text-app-muted-foreground">{t(strings.docs.loading)}</p>
        ) : currentDocument.state === "failed" ? (
          <p role="alert" className="text-sm text-app-muted-foreground">{t(strings.docs.loadError)}</p>
        ) : currentDocument.state === "missing" ? (
          <div className="flex flex-col gap-1">
            <p className="font-medium text-app-foreground">{t(strings.docs.notFoundTitle)}</p>
            <p className="text-sm text-app-muted-foreground">
              {t(strings.docs.notFoundDescription)}
            </p>
          </div>
        ) : (
          <MarkdownRenderer content={currentDocument.content ?? ""} />
        )}
      </Panel>
    </div>
  );
}

function deriveTitle(path: string): string {
  if (!path) return "Docs";
  const file = path.split("/").pop() ?? path;
  return file.replace(/\.md$/i, "");
}
