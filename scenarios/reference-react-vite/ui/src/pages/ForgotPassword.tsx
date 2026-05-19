import { useNavigate } from "react-router-dom";
import { ROUTES } from "../routes.generated";

export function ForgotPassword() {
  const navigate = useNavigate();
  return (
    <div
      data-testid="forgot-password-page"
      className="mx-auto flex max-w-sm flex-col gap-3 rounded-xl border border-white/10 bg-slate-900/50 p-6"
    >
      <h2 className="text-xl font-semibold">Forgot password</h2>
      <p className="text-xs text-slate-400">
        Illustrative page only — no recovery flow is wired up.
      </p>
      <button
        type="button"
        data-testid="forgot-password-back"
        onClick={() => navigate(ROUTES.login)}
        className="self-start rounded border border-white/10 px-3 py-2 text-sm text-slate-200 hover:bg-white/5"
      >
        Back to sign in
      </button>
    </div>
  );
}
