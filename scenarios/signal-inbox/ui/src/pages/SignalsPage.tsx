import { CaptureCard } from "../features/capture/CaptureCard";
import { CategoriesCard } from "../features/categories/CategoriesCard";
import { selectors } from "../consts/selectors";

export function SignalsPage() {
  return <section data-testid={selectors.pages.signals} aria-labelledby="signals-heading" className="flex flex-col gap-4"><h2 id="signals-heading" className="text-2xl font-semibold">Signal inbox</h2><CaptureCard /><CategoriesCard /></section>;
}
