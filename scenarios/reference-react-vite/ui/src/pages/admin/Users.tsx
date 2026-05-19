export function AdminUsers() {
  return (
    <div data-testid="admin-users-page" className="flex flex-col gap-2">
      <h2 className="text-2xl font-semibold">Admin · Users</h2>
      <p className="text-sm text-slate-400">Visible only when role=admin.</p>
    </div>
  );
}
