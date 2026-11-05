import { CaptureCard } from "../features/capture/CaptureCard";
import { CategoriesCard } from "../features/categories/CategoriesCard";
import { TriageQueue } from "../features/triage/TriageQueue";
import { SourcesCard } from "../features/sources/SourcesCard";
import { SearchCard } from "../features/retrieval/SearchCard";
import { selectors } from "../consts/selectors";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";

export function SignalsPage() {
  return <ExperienceSurface surfaceId="signals" state="static" data-testid={selectors.pages.signals} aria-labelledby="signals-heading" className="flex flex-col gap-4"><h2 id="signals-heading" className="text-2xl font-semibold">Signal inbox</h2><CaptureCard /><SourcesCard /><SearchCard /><TriageQueue /><CategoriesCard /></ExperienceSurface>;
}
