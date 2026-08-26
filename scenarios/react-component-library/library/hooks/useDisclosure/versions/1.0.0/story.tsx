import { useDisclosure } from "./useDisclosure";
export function Default() {
  const disclosure = useDisclosure();
  return (
    <button data-testid="hooks.use-disclosure"
      type="button"
      aria-expanded={disclosure.open}
      onClick={disclosure.onToggle}
    >
      {disclosure.open ? "open" : "closed"}
    </button>
  );
}
