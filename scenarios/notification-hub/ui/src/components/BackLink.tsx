import { ChevronLeft } from "lucide-react";
import { Link, type LinkProps } from "react-router-dom";

/** The way back from a drill-in page; the arrow turns around in right-to-left languages. */
export function BackLink({ children, ...props }: Omit<LinkProps, "className">) {
  return (
    <Link
      className="-ms-2 inline-flex min-h-11 w-fit items-center gap-1 rounded-control pe-3 ps-1 text-sm font-medium text-app-primary hover:underline focus-visible:outline focus-visible:outline-2"
      {...props}
    >
      <ChevronLeft size={18} aria-hidden="true" className="rtl:rotate-180" />
      {children}
    </Link>
  );
}
