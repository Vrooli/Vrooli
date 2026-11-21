import { Alert } from "@vrooli/react-component-library/Alert/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { useParams } from "react-router-dom";

import { homeFileUrl } from "../api/client";
import { useHomeText } from "../api/desk";
import { HtmlOpen } from "../components/HtmlOpen";
import { Markdown } from "../components/Markdown";
import { PathText } from "../components/PathText";
import { QueryView } from "../components/QueryView";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useResearchHome } from "../hooks/useResearchHome";
import { useTranslation } from "../i18n";
import { basename, dirname } from "../lib/paths";

const IMAGE = /\.(png|jpe?g|gif|webp|svg)$/i;

/**
 * Any research-home file, read through the desk's read-only file route:
 * Markdown rendered with its links kept inside the desk, HTML sandboxed,
 * images shown, anything else as text.
 */
export function FilePage() {
  const { t } = useTranslation();
  const rel = useParams()["*"] ?? "";
  const home = useResearchHome();
  const path = home ? `${home}/${rel.split("/").map(decodeURIComponent).join("/")}` : "";
  const url = home ? homeFileUrl(path, home) : null;
  const isHtml = /\.html?$/i.test(path);
  const isImage = IMAGE.test(path);
  const text = useHomeText(url && !isHtml && !isImage ? url : null);
  return (
    <div data-testid={selectors.pages.file} className="rd-page">
      <PageHeader headingId="file-heading" title={basename(path) || t(strings.files.title)} description={path ? <PathText path={path} /> : null} />
      {!home ? (
        <p className="text-body-sm">{t(strings.common.loading)}</p>
      ) : !url ? (
        <Alert tone="warning" title={t(strings.files.outside)} />
      ) : isHtml ? (
        <HtmlOpen src={url} name={basename(path)} />
      ) : isImage ? (
        <img src={url} alt={basename(path)} className="rd-image" />
      ) : (
        <QueryView query={text}>
          {(content) => (/\.md$/i.test(path) ? <Markdown content={content} baseDir={dirname(path)} /> : <pre className="rd-exact">{content}</pre>)}
        </QueryView>
      )}
    </div>
  );
}
