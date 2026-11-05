import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { getRampJson, targetsFromPayload, type RampTarget } from "../api/ramp";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

export function TargetMatrixPage() {
  const { t } = useTranslation();
  const [targets, setTargets] = useState<RampTarget[]>([]);
  const [error, setError] = useState("");
  useEffect(() => { void getRampJson<unknown>("/android/targets").then((value) => setTargets(targetsFromPayload(value))).catch((reason: unknown) => setError(reason instanceof Error ? reason.message : t(strings.errors.unknown))); }, [t]);
  return <section className="flex flex-col gap-4" aria-labelledby="target-matrix-heading"><div><h2 id="target-matrix-heading" className="text-2xl font-semibold">{t(strings.pages.targets.title)}</h2><p className="text-app-muted-foreground">{t(strings.pages.targets.description)}</p></div>{error && <p role="alert" className="text-red-600">{error}</p>}<div className="grid gap-4 lg:grid-cols-2">{targets.map((target) => <Card key={target.id}><CardHeader><CardTitle className="flex items-center justify-between gap-3"><span>{target.label || target.id || t(strings.pages.targets.title)}</span><StatusBadge tone={target.available ? "success" : "danger"}>{target.available ? t(strings.pages.targets.ready) : t(strings.pages.targets.unavailable)}</StatusBadge></CardTitle></CardHeader><CardContent className="space-y-2 text-sm"><p><span className="text-app-muted-foreground">{t(strings.pages.targets.kind)}</span> {target.device_kind || t(strings.pages.readiness.unknown)} · {target.transport?.kind || t(strings.pages.readiness.unknown)}</p><p>{target.available ? t(strings.pages.targets.readyForSelection) : target.reason || t(strings.pages.targets.unavailable)}</p><p className="text-app-muted-foreground">{t(strings.pages.targets.next)} {target.next_action || t(strings.pages.targets.description)}</p></CardContent></Card>)}</div>{!error && targets.length === 0 && <p className="text-app-muted-foreground">{t(strings.pages.targets.empty)}</p>}</section>;
}
