import type { ReactNode } from "react";
import { ExportDialogContext, type ExportDialogContextValue } from "./ExportDialogContext";

interface ExportDialogProviderProps {
  children: ReactNode;
  value: ExportDialogContextValue;
}

export function ExportDialogProvider({
  children,
  value,
}: ExportDialogProviderProps): JSX.Element {
  return (
    <ExportDialogContext.Provider value={value}>
      {children}
    </ExportDialogContext.Provider>
  );
}
