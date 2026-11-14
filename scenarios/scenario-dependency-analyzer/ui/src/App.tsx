import { Providers } from "./app/providers";
import { AppRouter } from "./app/routes";

export default function App() {
  return (
    <Providers>
      <AppRouter />
    </Providers>
  );
}
