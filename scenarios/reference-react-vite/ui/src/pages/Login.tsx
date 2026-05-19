import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { useState } from "react";
import { useAppContext } from "../contexts/AppContext";
import { ROUTES } from "../routes.generated";

export function Login() {
  const { setAuth } = useAppContext();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const [role, setRole] = useState<"viewer" | "editor" | "admin">("viewer");
  const { setRole: setCtxRole } = useAppContext();

  const onSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setCtxRole(role);
    setAuth("logged_in");
    const target = params.get("next") || ROUTES.dashboard;
    navigate(target, { replace: true });
  };

  return (
    <div
      data-testid="login-page"
      className="mx-auto flex max-w-sm flex-col gap-4 rounded-xl border border-white/10 bg-slate-900/50 p-6"
    >
      <h2 className="text-xl font-semibold">Sign in</h2>
      <p className="text-xs text-slate-400">
        Illustrative — no real credentials. Pick a role to demo the gating contexts.
      </p>
      <form onSubmit={onSubmit} className="flex flex-col gap-3">
        <label className="flex flex-col gap-1 text-xs text-slate-400">
          Role
          <select
            data-testid="login-role"
            value={role}
            onChange={(e) => setRole(e.target.value as typeof role)}
            className="rounded border border-white/10 bg-slate-950 px-2 py-1 text-sm text-slate-100"
          >
            <option value="viewer">viewer</option>
            <option value="editor">editor</option>
            <option value="admin">admin</option>
          </select>
        </label>
        <button
          type="submit"
          data-testid="login-submit"
          className="rounded bg-blue-600 px-3 py-2 text-sm text-white hover:bg-blue-500"
        >
          Sign in
        </button>
      </form>
      <Link
        to={ROUTES.forgotPassword}
        data-testid="forgot-password-link"
        className="text-xs text-slate-400 hover:text-slate-200"
      >
        Forgot password?
      </Link>
    </div>
  );
}
