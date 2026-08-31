/** @vrooliComponentSource hooks.use-direction */

export function useDirection() {
  return (
    typeof document !== "undefined" ? document.documentElement.dir : "ltr"
  ) as "ltr" | "rtl";
}
