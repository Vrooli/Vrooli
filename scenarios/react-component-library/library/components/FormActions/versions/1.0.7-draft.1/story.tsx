import { FormActions } from "./FormActions";

export function Default() {
  return <FormActions />;
}

export function GermanLongActions() {
  return (
    <main lang="de" style={{ width: "100%", maxWidth: 320, padding: 12, boxSizing: "border-box" }}>
      <FormActions
        cancelLabel="Abbrechen"
        resetLabel="Eingaben zurücksetzen"
        submitLabel="Änderungen speichern"
      />
    </main>
  );
}
