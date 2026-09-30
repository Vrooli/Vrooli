import { Card, CardDescription, CardTitle } from "../../../components/ui/card";
import { useTranslation } from "../../../i18n";
import { strings } from "../../../consts/strings";

export function UsageStat({ title, value, tone = "neutral" }: { title: string; value: string | number; tone?: "neutral" | "danger" }) {
  const { t } = useTranslation();
  return (
    <Card padding="md">
      <CardTitle>{title}</CardTitle>
      <p className={"mt-2 text-2xl font-semibold " + (tone === "danger" ? "text-app-danger" : "text-app-foreground")}>
        {value}
      </p>
      <CardDescription className="mt-1">{t(strings.common.last24h)}</CardDescription>
    </Card>
  );
}
