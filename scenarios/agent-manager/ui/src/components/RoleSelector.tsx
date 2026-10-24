import type { RolePolicyCatalog } from "@vrooli/proto-types/agent-manager/v1/api/service_pb";
import { Label } from "./ui/label";

interface RoleSelectorProps {
  catalog?: RolePolicyCatalog;
  value: string;
  onChange: (roleRef: string) => void;
  label?: string;
  id?: string;
}

// Role selection is the only mutable coding-agent intent exposed by the UI.
// Resource policy resolution chooses concrete runners and models at run creation.
export function RoleSelector({
  catalog,
  value,
  onChange,
  label = "Role",
  id = "roleRef",
}: RoleSelectorProps) {
  const roles = catalog?.roles ?? [];
  const effectiveValue = value || catalog?.defaultRole || "";

  return (
    <div className="space-y-2">
      <Label htmlFor={id}>{label} *</Label>
      <select
        id={id}
        value={effectiveValue}
        onChange={(event) => onChange(event.target.value)}
        className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
        disabled={roles.length === 0}
      >
        {roles.length === 0 ? (
          <option value="">No active roles available</option>
        ) : (
          roles.map((role) => (
            <option key={role.roleRef} value={role.roleRef}>
              {role.roleRef} — {role.description || role.intent}
            </option>
          ))
        )}
      </select>
      {roles.length === 0 ? (
        <p className="text-xs text-destructive">Load a valid role policy catalog before creating a profile or custom run.</p>
      ) : (
        <p className="text-xs text-muted-foreground">
          The selected role resolves its runner and model candidates through resource-owned policy at run creation.
        </p>
      )}
    </div>
  );
}
