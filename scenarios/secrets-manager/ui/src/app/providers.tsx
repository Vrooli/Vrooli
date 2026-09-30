import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { ThemeProvider } from "../theme/ThemeProvider";

const queryClient = new QueryClient();

export function Providers({ children }: { children: ReactNode }) {
  return <ThemeProvider><QueryClientProvider client={queryClient}>{children}</QueryClientProvider></ThemeProvider>;
}
