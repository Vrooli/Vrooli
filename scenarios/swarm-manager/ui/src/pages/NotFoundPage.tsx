/**
 * Not Found Page (404)
 *
 * Displayed when the user navigates to a route that doesn't exist.
 *
 * Recovery path: Navigate to home (Ideas page)
 *
 * Key principles:
 * - Clear, friendly message (not technical "404" jargon)
 * - Obvious action to get back on track
 * - Consistent styling with the rest of the app
 */

import { useNavigate } from "react-router-dom";
import { MapPin, Home } from "lucide-react";
import { Button } from "../components/ui/button";
import { selectors } from "../consts/selectors";

export function NotFoundPage() {
  const navigate = useNavigate();

  const handleGoHome = () => {
    navigate("/ideas", { replace: true });
  };

  return (
    <div
      className="flex min-h-[60vh] items-center justify-center p-4"
      data-testid={selectors.notFound.page}
    >
      <div className="max-w-md text-center">
        <MapPin className="mx-auto h-16 w-16 text-slate-500" />
        <h1
          className="mt-6 text-2xl font-semibold text-slate-100"
          data-testid={selectors.notFound.title}
        >
          Page not found
        </h1>
        <p
          className="mt-3 text-slate-400"
          data-testid={selectors.notFound.message}
        >
          The page you're looking for doesn't exist or may have been moved.
        </p>
        <Button
          className="mt-6"
          onClick={handleGoHome}
          data-testid={selectors.notFound.homeButton}
        >
          <Home className="mr-2 h-4 w-4" />
          Go to Ideas
        </Button>
      </div>
    </div>
  );
}
