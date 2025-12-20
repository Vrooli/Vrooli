import { FC, memo } from 'react';
import type { NodeProps } from 'reactflow';
import { Cookie } from 'lucide-react';
import { useActionParams } from '@hooks/useActionParams';
import {
  useSyncedString,
  useSyncedNumber,
  useSyncedBoolean,
  useSyncedSelect,
} from '@hooks/useSyncedField';
import BaseNode from './BaseNode';
import { NodeTextField, NodeTextArea, NodeNumberField, NodeSelectField, NodeCheckbox, FieldRow } from './fields';

// SetCookieParams interface for V2 native action params
interface SetCookieParams {
  name?: string;
  value?: string;
  url?: string;
  domain?: string;
  path?: string;
  sameSite?: string;
  secure?: boolean;
  httpOnly?: boolean;
  ttlSeconds?: number;
  expiresAt?: string;
  timeoutMs?: number;
  waitForMs?: number;
}

const SAME_SITE_OPTIONS = [
  { value: '', label: 'Browser default' },
  { value: 'lax', label: 'Lax' },
  { value: 'strict', label: 'Strict' },
  { value: 'none', label: 'None' },
];

const MIN_TIMEOUT = 100;

const SetCookieNode: FC<NodeProps> = ({ selected, id }) => {
  const { params, updateParams } = useActionParams<SetCookieParams>(id);

  // String fields
  const name = useSyncedString(params?.name ?? '', {
    onCommit: (v) => updateParams({ name: v || undefined }),
  });
  const value = useSyncedString(params?.value ?? '', {
    trim: false,
    onCommit: (v) => updateParams({ value: v }),
  });
  const url = useSyncedString(params?.url ?? '', {
    onCommit: (v) => updateParams({ url: v || undefined }),
  });
  const domain = useSyncedString(params?.domain ?? '', {
    onCommit: (v) => updateParams({ domain: v || undefined }),
  });
  const path = useSyncedString(params?.path ?? '', {
    onCommit: (v) => updateParams({ path: v || undefined }),
  });
  const expiresAt = useSyncedString(params?.expiresAt ?? '', {
    onCommit: (v) => updateParams({ expiresAt: v || undefined }),
  });

  // Select fields
  const sameSite = useSyncedSelect(params?.sameSite ?? '', {
    onCommit: (v) => updateParams({ sameSite: v || undefined }),
  });

  // Boolean fields
  const secure = useSyncedBoolean(params?.secure ?? false, {
    onCommit: (v) => updateParams({ secure: v || undefined }),
  });
  const httpOnly = useSyncedBoolean(params?.httpOnly ?? false, {
    onCommit: (v) => updateParams({ httpOnly: v || undefined }),
  });

  // Number fields
  const ttlSeconds = useSyncedNumber(params?.ttlSeconds ?? 0, {
    min: 0,
    onCommit: (v) => updateParams({ ttlSeconds: v || undefined }),
  });
  const timeoutMs = useSyncedNumber(params?.timeoutMs ?? 30000, {
    min: MIN_TIMEOUT,
    fallback: 30000,
    onCommit: (v) => updateParams({ timeoutMs: v }),
  });
  const waitForMs = useSyncedNumber(params?.waitForMs ?? 0, {
    min: 0,
    onCommit: (v) => updateParams({ waitForMs: v || undefined }),
  });

  return (
    <BaseNode selected={selected} icon={Cookie} iconClassName="text-amber-300" title="Set Cookie">
      <div className="space-y-3 text-xs">
        <FieldRow>
          <NodeTextField field={name} label="Cookie name" placeholder="sessionId" />
          <NodeTextField field={path} label="Path" placeholder="/" />
        </FieldRow>

        <NodeTextArea field={value} label="Value" placeholder="Cookie payload" rows={2} />

        <FieldRow>
          <NodeTextField
            field={url}
            label="Target URL"
            placeholder="https://example.com/app"
            description="Provide URL or domain to scope the cookie."
          />
          <NodeTextField field={domain} label="Domain" placeholder=".example.com" />
        </FieldRow>

        <FieldRow>
          <NodeSelectField field={sameSite} label="SameSite" options={SAME_SITE_OPTIONS} />
          <div className="flex items-center gap-4 pt-5">
            <NodeCheckbox field={secure} label="Secure" />
            <NodeCheckbox field={httpOnly} label="HTTP only" />
          </div>
        </FieldRow>

        <FieldRow>
          <NodeNumberField field={ttlSeconds} label="TTL (seconds)" min={0} />
          <NodeTextField
            field={expiresAt}
            label="Expires at (RFC3339)"
            placeholder="2025-12-31T23:59:59Z"
          />
        </FieldRow>

        <FieldRow>
          <NodeNumberField field={timeoutMs} label="Timeout (ms)" min={MIN_TIMEOUT} />
          <NodeNumberField field={waitForMs} label="Post-set wait (ms)" min={0} />
        </FieldRow>
      </div>
    </BaseNode>
  );
};

export default memo(SetCookieNode);
