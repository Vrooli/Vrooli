import { CaptureGrid } from "./CaptureGrid";
export default function CaptureGridStory() {
  return (
    <CaptureGrid
      cells={[
        {
          id: "mobile-light",
          viewport: "mobile",
          theme: "light",
          status: "pass",
        },
        {
          id: "desktop-dark",
          viewport: "desktop",
          theme: "dark",
          status: "missing",
        },
      ]}
    />
  );
}
