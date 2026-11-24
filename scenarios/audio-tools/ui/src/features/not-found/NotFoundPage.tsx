import { Link } from "react-router-dom";
import { Button } from "../../components/ui/button";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";

export function NotFoundPage() {
  const { t } = useTranslation();
  return (
    <div className="flex max-w-2xl flex-col gap-4">
      <PageHeader title={t(strings.notFound.title)} description={t(strings.notFound.description)} />
      <Button variant="outline" asChild>
        <Link to="/">{t(strings.notFound.backToOverview)}</Link>
      </Button>
    </div>
  );
}
