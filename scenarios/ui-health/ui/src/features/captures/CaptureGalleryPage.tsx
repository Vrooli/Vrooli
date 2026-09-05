import { CaptureGallery } from "../../components/CaptureGallery";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

export function CaptureGalleryPage() {
  const { t } = useTranslation();
  return (
    <section
      data-testid={selectors.pages.captures}
      aria-labelledby="captures-heading"
      className="flex flex-col gap-6"
    >
      <header className="flex flex-col gap-1">
        <h2 id="captures-heading" className="text-2xl font-semibold tracking-tight">
          {t(strings.pages.captures.title)}
        </h2>
        <p className="text-sm text-app-muted-foreground">
          {t(strings.pages.captures.description)}
        </p>
      </header>

      <CaptureGallery />
    </section>
  );
}
