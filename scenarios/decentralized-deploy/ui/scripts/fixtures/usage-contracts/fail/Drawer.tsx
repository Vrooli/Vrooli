import { Dialog } from "@vrooli/react-component-library";

export function Drawer() {
  return (
    <Dialog
      footer={
        <footer>
          <button type="button">Änderungen verwerfen und zum vorherigen Bereich zurückkehren</button>
          <button type="button">Einstellungen für diesen Arbeitsbereich speichern</button>
          <button type="button">Änderungen später weiter bearbeiten</button>
          <button type="button">Diesen Vorgang vollständig abbrechen</button>
          <button type="button">Details und zusätzliche Optionen anzeigen</button>
        </footer>
      }
    />
  );
}
