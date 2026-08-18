import { Surface } from "./Surface";

type SurfaceStoryProps = {
  args: {
    elevation?: "flat" | "raised" | "floating" | "overlay";
    variant?: "solid" | "translucent";
  };
};

/**
 * A surface is judged in the context it is meant to occupy: content behind it
 * makes the translucent treatment observable instead of letting a flat card
 * pass as a complete surface specimen.
 */
export function Specimen({ args }: SurfaceStoryProps) {
  const translucent = args.variant === "translucent";
  return (
    <div
      role="region"
      aria-label="Component specimen"
      className="relative grid min-h-stage place-items-center overflow-hidden rounded-panel p-space-lg"
      style={{
        background:
          "linear-gradient(135deg, var(--color-primary), var(--color-accent) 48%, var(--color-primary-strong))",
      }}
    >
      <div
        aria-hidden="true"
        className="absolute inset-0 opacity-40"
        style={{
          backgroundImage:
            "radial-gradient(circle at 20% 20%, white 0 8%, transparent 9%), radial-gradient(circle at 75% 65%, white 0 10%, transparent 11%)",
          backgroundSize: "9rem 9rem, 12rem 12rem",
        }}
      />
      <Surface
        elevation={args.elevation}
        aria-label={translucent ? "Translucent surface" : "Surface"}
        style={
          translucent
            ? {
                background:
                  "color-mix(in srgb, var(--color-surface) 72%, transparent)",
                backdropFilter: "blur(14px)",
              }
            : undefined
        }
        className="relative z-10 grid gap-space-2xs p-space-lg"
      >
        <strong>{translucent ? "Translucent surface" : "Surface"}</strong>
        <span>Content remains legible over the background.</span>
      </Surface>
    </div>
  );
}
