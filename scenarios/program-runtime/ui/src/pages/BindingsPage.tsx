import { BindingRegistry } from "../features/bindings/BindingRegistry";
import { selectors } from "../consts/selectors";

export function BindingsPage() {
  return <section data-testid={selectors.pages.bindings} className="flex flex-col gap-4"><h2 className="text-2xl font-semibold">Bindings</h2><BindingRegistry /></section>;
}
