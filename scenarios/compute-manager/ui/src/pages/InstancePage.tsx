import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";

import { fetchInstance } from "../api/compute";
import { strings } from "../consts/strings";
import { selectors } from "../consts/selectors";
import { useTranslation } from "../i18n";

export function InstancePage() {
  const { t } = useTranslation();
  const { id = "" } = useParams();
  const [instance, setInstance] = useState<Awaited<ReturnType<typeof fetchInstance>>["instance"]>();
  const [error, setError] = useState(false);
  useEffect(() => { fetchInstance(id).then((response) => setInstance(response.instance)).catch(() => setError(true)); }, [id]);
  return <section data-testid={selectors.pages.instance} className="flex flex-col gap-space-md" aria-labelledby="instance-heading"><PageHeader headingId="instance-heading" title={t(strings.pages.instance.title)} description={t(strings.pages.instance.description)} />{error ? <p role="alert">{t(strings.pages.instance.error)}</p> : instance ? <dl className="grid gap-space-sm rounded-panel border border-border bg-surface p-space-md"><div><dt className="font-semibold">{t(strings.pages.instance.state)}</dt><dd>{instance.state}</dd></div><div><dt className="font-semibold">{t(strings.pages.instance.address)}</dt><dd>{instance.address || "—"}</dd></div><div><dt className="font-semibold">{t(strings.pages.instance.provider)}</dt><dd>{instance.provider}</dd></div></dl> : <p role="status">{t(strings.pages.instance.loading)}</p>}</section>;
}
