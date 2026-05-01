/**
 * ActionEditorPanel - Dense contract inspector/editor for Actions.
 *
 * Execution is delegated to the governed Action API. The UI only collects
 * input JSON and renders the response envelope.
 */

import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { Archive, Bolt, CheckCircle2, Clipboard, Menu, Play, Plus, RotateCcw, Save, ShieldCheck, Trash2, X, XCircle } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useActionsData } from '@/hooks/useActionsData'
import { copyToClipboard } from '@/lib/clipboard'
import { toast } from '@/hooks/use-toast'
import type { Action, ActionRunResponse, ActionValidationResponse, UpdateActionRequest } from '@/types'
import type { ActionExample, ActionInput, ActionInputType, ActionOutput, ActionOutputType, ActionPermissions, ActionStatus } from '@/lib/schemas'

interface ActionEditorPanelProps {
  actionId: string
  onClose: () => void
  onOpenSidebar?: () => void
  className?: string
}

type ActionContractDraft = Omit<Action, 'createdAt' | 'updatedAt' | 'revision'>

const ACTION_STATUSES: ActionStatus[] = ['active', 'draft', 'archived']
const OWNER_TYPES = ['project', 'scenario', 'resource', 'team'] as const
const INPUT_TYPES: ActionInputType[] = ['string', 'number', 'integer', 'boolean', 'file', 'path', 'scenario', 'team', 'action']
const OUTPUT_TYPES: ActionOutputType[] = ['string', 'number', 'integer', 'boolean', 'file', 'path', 'json']
const PERMISSION_KEYS: Array<keyof ActionPermissions> = [
  'filesystemRead',
  'filesystemWrite',
  'localhostNetwork',
  'externalNetwork',
  'apiRead',
  'apiWrite',
  'processStart',
  'processStop',
  'hostConfigure',
  'secretRead',
  'secretWrite',
  'destructive',
]

export function ActionEditorPanel({
  actionId,
  onClose,
  onOpenSidebar,
  className,
}: ActionEditorPanelProps) {
  const {
    actions,
    isLoading,
    updateAction,
    deleteAction,
    validateAction,
    runAction,
    isUpdating,
    isDeleting,
    isValidating,
    isRunning,
  } = useActionsData()

  const action = useMemo(
    () => actions.find((candidate) => candidate.id === actionId),
    [actions, actionId]
  )
  const [jsonDraft, setJsonDraft] = useState('')
  const [isDirty, setIsDirty] = useState(false)
  const [parseError, setParseError] = useState<string | null>(null)
  const [validation, setValidation] = useState<ActionValidationResponse | null>(null)
  const [runInputJson, setRunInputJson] = useState('{}')
  const [runInputError, setRunInputError] = useState<string | null>(null)
  const [runResult, setRunResult] = useState<ActionRunResponse | null>(null)
  const [confirmHardDelete, setConfirmHardDelete] = useState(false)
  const isMobileSidebarToggle = Boolean(onOpenSidebar)
  const parsedDraft = useMemo(() => parseDraft(jsonDraft), [jsonDraft])

  useEffect(() => {
    if (!action) return
    setJsonDraft(JSON.stringify(stripRuntimeFields(action), null, 2))
    setIsDirty(false)
    setParseError(null)
    setValidation(null)
    setRunInputJson(formatRunInput(action))
    setRunInputError(null)
    setRunResult(null)
    setConfirmHardDelete(false)
  }, [action])

  const handleSave = async () => {
    if (!action) return
    setParseError(null)
    let parsed: UpdateActionRequest
    try {
      parsed = JSON.parse(jsonDraft) as UpdateActionRequest
    } catch (error) {
      setParseError(error instanceof Error ? error.message : 'Invalid JSON')
      return
    }

    const response = await updateAction(action.id, parsed)
    setValidation(response.validation)
    setJsonDraft(JSON.stringify(stripRuntimeFields(response.action), null, 2))
    setIsDirty(false)
  }

  const handleValidate = async () => {
    const result = await validateAction(actionId)
    setValidation(result)
  }

  const handleDiscard = () => {
    if (!action) return
    setJsonDraft(JSON.stringify(stripRuntimeFields(action), null, 2))
    setParseError(null)
    setIsDirty(false)
  }

  const patchDraft = (mutate: (draft: ActionContractDraft) => ActionContractDraft) => {
    const current = parseDraft(jsonDraft)
    if (!current.ok) {
      setParseError(current.error)
      return
    }
    const next = mutate(current.value)
    setJsonDraft(JSON.stringify(next, null, 2))
    setParseError(null)
    setIsDirty(true)
  }

  const handleDelete = async (hard = false) => {
    if (hard && !confirmHardDelete) {
      setConfirmHardDelete(true)
      return
    }
    await deleteAction(actionId, hard)
    onClose()
  }

  const handleRun = async (dryRun: boolean) => {
    setRunInputError(null)
    let input: Record<string, unknown>
    try {
      input = parseRunInput(runInputJson)
    } catch (error) {
      setRunInputError(error instanceof Error ? error.message : 'Invalid input JSON')
      return
    }
    const result = await runAction(actionId, { input, dryRun })
    setRunResult(result)
    setValidation(result.validation)
  }

  if (isLoading) {
    return (
      <div className={cn('flex items-center justify-center py-16', className)}>
        <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  if (!action) {
    return (
      <div className={cn('flex flex-col items-center justify-center py-16', className)}>
        <Bolt className="h-12 w-12 text-muted-foreground mb-4" />
        <p className="text-sm text-muted-foreground">Action not found</p>
        <button type="button" onClick={onClose} className="mt-4 text-sm text-primary hover:underline">
          Go back
        </button>
      </div>
    )
  }

  const activeValidation = validation
  const command = action.command.argv.join(' ')

  return (
    <div className={cn('flex flex-col h-full', className)}>
      <div className="flex items-center gap-2 px-4 py-3 border-b border-border">
        <button
          type="button"
          onClick={onOpenSidebar ?? onClose}
          className="h-9 w-9 flex items-center justify-center rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition-colors flex-shrink-0"
          aria-label={isMobileSidebarToggle ? 'Open sidebar' : 'Close editor'}
          title={isMobileSidebarToggle ? 'Open sidebar' : 'Close'}
        >
          {isMobileSidebarToggle ? <Menu className="h-5 w-5" /> : <X className="h-5 w-5" />}
        </button>
        <div className="flex-shrink-0 w-8 h-8 rounded-md bg-muted flex items-center justify-center">
          <Bolt className="h-4 w-4 text-muted-foreground" />
        </div>
        <div className="flex-1 min-w-0">
          <h2 className="text-sm font-semibold text-foreground truncate">{action.name}</h2>
          <p className="text-xs text-muted-foreground truncate">{action.id}</p>
        </div>
        {isDirty && (
          <span className="hidden min-[390px]:inline-flex px-2.5 py-1 bg-amber-500/20 text-amber-300 rounded-md text-xs font-medium">
            Unsaved
          </span>
        )}
        <HeaderButton label="Validate" onClick={() => void handleValidate()} disabled={isValidating}>
          <ShieldCheck className="h-4 w-4" />
        </HeaderButton>
        <HeaderButton label="Save" onClick={() => void handleSave()} disabled={!isDirty || isUpdating}>
          <Save className="h-4 w-4" />
        </HeaderButton>
      </div>

      <div className="flex-1 overflow-y-auto px-4 py-4">
        <div className="grid grid-cols-1 xl:grid-cols-[minmax(0,1fr)_360px] gap-4">
          <section className="min-w-0 space-y-4">
            <TypedContractEditor
              draft={parsedDraft.ok ? parsedDraft.value : null}
              parseError={parsedDraft.ok ? null : parsedDraft.error}
              onPatch={patchDraft}
            />
            <div className="border border-border rounded-md overflow-hidden">
              <div className="flex items-center justify-between gap-2 px-3 py-2 border-b border-border bg-muted/30">
                <div className="min-w-0">
                  <h3 className="text-xs font-semibold text-foreground">Contract JSON</h3>
                  <p className="text-[11px] text-muted-foreground truncate">Validated by the API before persistence.</p>
                </div>
                <div className="flex items-center gap-1">
                  <IconButton label="Copy JSON" onClick={() => void copyContract(jsonDraft)}>
                    <Clipboard className="h-4 w-4" />
                  </IconButton>
                  <IconButton label="Discard changes" onClick={handleDiscard} disabled={!isDirty}>
                    <RotateCcw className="h-4 w-4" />
                  </IconButton>
                </div>
              </div>
              <textarea
                value={jsonDraft}
                onChange={(event) => {
                  setJsonDraft(event.target.value)
                  setIsDirty(true)
                  setParseError(null)
                }}
                spellCheck={false}
                aria-invalid={Boolean(parseError || !parsedDraft.ok)}
                className={cn(
                  'h-[520px] w-full resize-y border-0 bg-background px-3 py-3',
                  'font-mono text-xs leading-5 text-foreground outline-none',
                  'focus:ring-2 focus:ring-primary/40'
                )}
                aria-label="Action contract JSON"
              />
            </div>
            {parseError && (
              <div className="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-xs text-destructive">
                {parseError}
              </div>
            )}
          </section>

          <aside className="min-w-0 space-y-4">
            <SummaryPanel action={action} command={command} />
            <ValidationPanel validation={activeValidation} />
            <RunPanel
              action={action}
              inputJson={runInputJson}
              inputError={runInputError}
              result={runResult}
              disabled={isDirty || isRunning}
              isRunning={isRunning}
              onInputChange={(value) => {
                setRunInputJson(value)
                setRunInputError(null)
              }}
              onLoadExample={() => {
                setRunInputJson(formatRunInput(action))
                setRunInputError(null)
              }}
              onRun={handleRun}
            />
            <section className="border border-border rounded-md">
              <div className="px-3 py-2 border-b border-border">
                <h3 className="text-xs font-semibold text-foreground">Lifecycle</h3>
              </div>
              <div className="px-3 py-3 grid grid-cols-1 gap-2">
                <button
                  type="button"
                  onClick={() => void handleDelete(false)}
                  disabled={isDeleting}
                  className="flex items-center justify-center gap-2 px-3 py-2 text-sm rounded-md border border-border hover:bg-muted text-foreground"
                >
                  <Archive className="h-4 w-4" />
                  Archive
                </button>
                <button
                  type="button"
                  onClick={() => void handleDelete(true)}
                  disabled={isDeleting}
                  className={cn(
                    'flex items-center justify-center gap-2 px-3 py-2 text-sm rounded-md border border-destructive/40 text-destructive hover:bg-destructive/10',
                    confirmHardDelete && 'bg-destructive/10'
                  )}
                  title={confirmHardDelete ? 'Confirm permanent Action deletion' : 'Request permanent Action deletion'}
                >
                  <Trash2 className="h-4 w-4" />
                  {confirmHardDelete ? 'Confirm hard delete' : 'Hard delete'}
                </button>
              </div>
            </section>
          </aside>
        </div>
      </div>
    </div>
  )
}

function TypedContractEditor({
  draft,
  parseError,
  onPatch,
}: {
  draft: ActionContractDraft | null
  parseError: string | null
  onPatch: (mutate: (draft: ActionContractDraft) => ActionContractDraft) => void
}) {
  if (!draft) {
    return (
      <section className="border border-border rounded-md">
        <div className="px-3 py-2 border-b border-border">
          <h3 className="text-xs font-semibold text-foreground">Contract Fields</h3>
        </div>
        <div className="px-3 py-4 text-xs text-muted-foreground">
          Fix the JSON draft before using typed contract fields.
          {parseError && <p className="mt-2 text-destructive break-words">{parseError}</p>}
        </div>
      </section>
    )
  }

  return (
    <section className="border border-border rounded-md overflow-hidden">
      <div className="px-3 py-2 border-b border-border bg-muted/30">
        <h3 className="text-xs font-semibold text-foreground">Contract Fields</h3>
        <p className="text-[11px] text-muted-foreground">Edits update the JSON draft below.</p>
      </div>
      <div className="divide-y divide-border">
        <IdentityFields draft={draft} onPatch={onPatch} />
        <CommandFields draft={draft} onPatch={onPatch} />
        <SchemaFields draft={draft} onPatch={onPatch} />
        <PermissionFields draft={draft} onPatch={onPatch} />
        <ExampleFields draft={draft} onPatch={onPatch} />
      </div>
    </section>
  )
}

function IdentityFields({
  draft,
  onPatch,
}: {
  draft: ActionContractDraft
  onPatch: (mutate: (draft: ActionContractDraft) => ActionContractDraft) => void
}) {
  return (
    <div className="px-3 py-3 space-y-3">
      <SectionTitle title="Identity" />
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        <TextField label="ID" value={draft.id} onChange={(id) => onPatch((current) => ({ ...current, id }))} />
        <TextField label="Name" value={draft.name} onChange={(name) => onPatch((current) => ({ ...current, name }))} />
        <SelectField
          label="Status"
          value={draft.status}
          options={ACTION_STATUSES}
          onChange={(status) => onPatch((current) => ({ ...current, status: status as ActionStatus }))}
        />
        <TextField
          label="Tags"
          value={draft.tags.join(', ')}
          onChange={(tags) => onPatch((current) => ({
            ...current,
            tags: tags.split(',').map((tag) => tag.trim()).filter(Boolean),
          }))}
        />
        <SelectField
          label="Owner type"
          value={draft.owner.type}
          options={[...OWNER_TYPES]}
          onChange={(type) => onPatch((current) => ({
            ...current,
            owner: { ...current.owner, type: type as ActionContractDraft['owner']['type'] },
          }))}
        />
        <TextField
          label="Owner ID"
          value={draft.owner.id}
          onChange={(id) => onPatch((current) => ({ ...current, owner: { ...current.owner, id } }))}
        />
      </div>
      <TextAreaField
        label="Description"
        rows={3}
        value={draft.description}
        onChange={(description) => onPatch((current) => ({ ...current, description }))}
      />
    </div>
  )
}

function CommandFields({
  draft,
  onPatch,
}: {
  draft: ActionContractDraft
  onPatch: (mutate: (draft: ActionContractDraft) => ActionContractDraft) => void
}) {
  return (
    <div className="px-3 py-3 space-y-3">
      <SectionTitle title="Command" />
      <TextAreaField
        label="Argv tokens"
        rows={Math.max(3, Math.min(8, draft.command.argv.length + 1))}
        value={draft.command.argv.join('\n')}
        mono
        hint="One argv token per line. Placeholders must match declared inputs."
        onChange={(value) => onPatch((current) => ({
          ...current,
          command: { argv: value.split('\n').map((token) => token.trim()).filter(Boolean) },
        }))}
      />
    </div>
  )
}

function SchemaFields({
  draft,
  onPatch,
}: {
  draft: ActionContractDraft
  onPatch: (mutate: (draft: ActionContractDraft) => ActionContractDraft) => void
}) {
  const inputs = Object.entries(draft.inputs)
  const outputs = Object.entries(draft.outputs)

  return (
    <div className="px-3 py-3 space-y-4">
      <div className="flex items-center justify-between gap-2">
        <SectionTitle title="Inputs" />
        <SmallButton
          label="Add input"
          onClick={() => onPatch((current) => ({
            ...current,
            inputs: {
              ...current.inputs,
              [nextObjectKey(current.inputs, 'input')]: defaultInput(),
            },
          }))}
        >
          <Plus className="h-3.5 w-3.5" />
        </SmallButton>
      </div>
      {inputs.length === 0 ? (
        <p className="text-xs text-muted-foreground">No declared inputs.</p>
      ) : (
        <div className="space-y-2">
          {inputs.map(([name, input]) => (
            <InputRow
              key={name}
              name={name}
              input={input}
              onRename={(nextName) => onPatch((current) => ({ ...current, inputs: renameKey(current.inputs, name, nextName) }))}
              onChange={(nextInput) => onPatch((current) => ({ ...current, inputs: { ...current.inputs, [name]: nextInput } }))}
              onRemove={() => onPatch((current) => ({ ...current, inputs: omitKey(current.inputs, name) }))}
            />
          ))}
        </div>
      )}

      <div className="flex items-center justify-between gap-2 pt-2">
        <SectionTitle title="Outputs" />
        <SmallButton
          label="Add output"
          onClick={() => onPatch((current) => ({
            ...current,
            outputs: {
              ...current.outputs,
              [nextObjectKey(current.outputs, 'output')]: { type: 'string', description: '' },
            },
          }))}
        >
          <Plus className="h-3.5 w-3.5" />
        </SmallButton>
      </div>
      {outputs.length === 0 ? (
        <p className="text-xs text-muted-foreground">No declared outputs.</p>
      ) : (
        <div className="space-y-2">
          {outputs.map(([name, output]) => (
            <OutputRow
              key={name}
              name={name}
              output={output}
              onRename={(nextName) => onPatch((current) => ({ ...current, outputs: renameKey(current.outputs, name, nextName) }))}
              onChange={(nextOutput) => onPatch((current) => ({ ...current, outputs: { ...current.outputs, [name]: nextOutput } }))}
              onRemove={() => onPatch((current) => ({ ...current, outputs: omitKey(current.outputs, name) }))}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function InputRow({
  name,
  input,
  onRename,
  onChange,
  onRemove,
}: {
  name: string
  input: ActionInput
  onRename: (name: string) => void
  onChange: (input: ActionInput) => void
  onRemove: () => void
}) {
  return (
    <div className="rounded-md border border-border px-2.5 py-2 space-y-2">
      <div className="grid grid-cols-1 md:grid-cols-[minmax(0,1fr)_130px_90px_32px] gap-2">
        <TextField label="Name" value={name} onChange={onRename} />
        <SelectField
          label="Type"
          value={input.type}
          options={INPUT_TYPES}
          onChange={(type) => onChange({ ...input, type: type as ActionInputType })}
        />
        <label className="flex items-end gap-2 pb-2 text-xs text-muted-foreground">
          <input
            type="checkbox"
            checked={input.required}
            onChange={(event) => onChange({ ...input, required: event.target.checked })}
            className="h-4 w-4 accent-primary"
          />
          Required
        </label>
        <IconButton label="Remove input" onClick={onRemove}>
          <Trash2 className="h-4 w-4" />
        </IconButton>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
        <TextField
          label="Default"
          value={formatDefaultValue(input.default)}
          onChange={(value) => onChange({ ...input, default: parseDefaultValue(value, input.type) })}
        />
        <TextField
          label="Description"
          value={input.description}
          onChange={(description) => onChange({ ...input, description })}
        />
      </div>
    </div>
  )
}

function OutputRow({
  name,
  output,
  onRename,
  onChange,
  onRemove,
}: {
  name: string
  output: ActionOutput
  onRename: (name: string) => void
  onChange: (output: ActionOutput) => void
  onRemove: () => void
}) {
  return (
    <div className="rounded-md border border-border px-2.5 py-2 space-y-2">
      <div className="grid grid-cols-1 md:grid-cols-[minmax(0,1fr)_130px_32px] gap-2">
        <TextField label="Name" value={name} onChange={onRename} />
        <SelectField
          label="Type"
          value={output.type}
          options={OUTPUT_TYPES}
          onChange={(type) => onChange({ ...output, type: type as ActionOutputType })}
        />
        <IconButton label="Remove output" onClick={onRemove}>
          <Trash2 className="h-4 w-4" />
        </IconButton>
      </div>
      <TextField
        label="Description"
        value={output.description}
        onChange={(description) => onChange({ ...output, description })}
      />
    </div>
  )
}

function PermissionFields({
  draft,
  onPatch,
}: {
  draft: ActionContractDraft
  onPatch: (mutate: (draft: ActionContractDraft) => ActionContractDraft) => void
}) {
  return (
    <div className="px-3 py-3 space-y-3">
      <SectionTitle title="Permissions" />
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
        {PERMISSION_KEYS.map((permission) => (
          <label
            key={permission}
            className="flex items-center gap-2 rounded-md border border-border px-2.5 py-2 text-xs text-foreground"
          >
            <input
              type="checkbox"
              checked={Boolean(draft.permissions[permission])}
              onChange={(event) => onPatch((current) => ({
                ...current,
                permissions: { ...current.permissions, [permission]: event.target.checked },
              }))}
              className="h-4 w-4 accent-primary"
            />
            <span className="break-all">{permission}</span>
          </label>
        ))}
      </div>
    </div>
  )
}

function ExampleFields({
  draft,
  onPatch,
}: {
  draft: ActionContractDraft
  onPatch: (mutate: (draft: ActionContractDraft) => ActionContractDraft) => void
}) {
  return (
    <div className="px-3 py-3 space-y-3">
      <div className="flex items-center justify-between gap-2">
        <SectionTitle title="Examples" />
        <SmallButton
          label="Add example"
          onClick={() => onPatch((current) => ({
            ...current,
            examples: [...current.examples, { description: '', input: {} }],
          }))}
        >
          <Plus className="h-3.5 w-3.5" />
        </SmallButton>
      </div>
      {draft.examples.length === 0 ? (
        <p className="text-xs text-muted-foreground">No examples yet.</p>
      ) : (
        <div className="space-y-2">
          {draft.examples.map((example, index) => (
            <ExampleRow
              key={index}
              example={example}
              index={index}
              onChange={(nextExample) => onPatch((current) => ({
                ...current,
                examples: current.examples.map((candidate, candidateIndex) => (
                  candidateIndex === index ? nextExample : candidate
                )),
              }))}
              onRemove={() => onPatch((current) => ({
                ...current,
                examples: current.examples.filter((_candidate, candidateIndex) => candidateIndex !== index),
              }))}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function ExampleRow({
  example,
  index,
  onChange,
  onRemove,
}: {
  example: ActionExample
  index: number
  onChange: (example: ActionExample) => void
  onRemove: () => void
}) {
  const [inputError, setInputError] = useState<string | null>(null)

  return (
    <div className="rounded-md border border-border px-2.5 py-2 space-y-2">
      <div className="grid grid-cols-[minmax(0,1fr)_32px] gap-2">
        <TextField
          label={`Example ${index + 1}`}
          value={example.description}
          onChange={(description) => onChange({ ...example, description })}
        />
        <IconButton label="Remove example" onClick={onRemove}>
          <Trash2 className="h-4 w-4" />
        </IconButton>
      </div>
      <TextAreaField
        label="Input JSON"
        rows={4}
        mono
        value={JSON.stringify(example.input, null, 2)}
        onChange={(value) => {
          try {
            const parsed = JSON.parse(value) as Record<string, unknown>
            setInputError(null)
            onChange({ ...example, input: parsed })
          } catch (error) {
            setInputError(error instanceof Error ? error.message : 'Invalid JSON')
          }
        }}
      />
      {inputError && <p className="text-xs text-destructive break-words">{inputError}</p>}
    </div>
  )
}

function SummaryPanel({ action, command }: { action: Action; command: string }) {
  return (
    <section className="border border-border rounded-md">
      <div className="px-3 py-2 border-b border-border">
        <h3 className="text-xs font-semibold text-foreground">Summary</h3>
      </div>
      <dl className="px-3 py-3 space-y-3 text-xs">
        <SummaryItem label="Status" value={action.status} />
        <SummaryItem label="Owner" value={`${action.owner.type}:${action.owner.id}`} />
        <SummaryItem label="Inputs" value={String(Object.keys(action.inputs).length)} />
        <SummaryItem label="Outputs" value={String(Object.keys(action.outputs).length)} />
        <SummaryItem label="Command" value={command || 'No command'} mono />
        {action.tags.length > 0 && <SummaryItem label="Tags" value={action.tags.join(', ')} />}
      </dl>
    </section>
  )
}

function SummaryItem({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className={cn('mt-1 break-words text-foreground', mono && 'font-mono text-[11px]')}>{value}</dd>
    </div>
  )
}

function ValidationPanel({ validation }: { validation: ActionValidationResponse | null }) {
  if (!validation) {
    return (
      <section className="border border-border rounded-md">
        <div className="px-3 py-2 border-b border-border">
          <h3 className="text-xs font-semibold text-foreground">Validation</h3>
        </div>
        <div className="px-3 py-4 text-xs text-muted-foreground">
          Validate this Action to inspect command ownership, permissions, and contract checks.
        </div>
      </section>
    )
  }

  const Icon = validation.valid ? CheckCircle2 : XCircle
  return (
    <section className="border border-border rounded-md" aria-live="polite">
      <div className="px-3 py-2 border-b border-border flex items-center gap-2">
        <Icon className={cn('h-4 w-4', validation.valid ? 'text-emerald-400' : 'text-destructive')} />
        <h3 className="text-xs font-semibold text-foreground">
          {validation.valid ? 'Valid' : 'Invalid'}
        </h3>
        <span className="ml-auto text-[11px] text-muted-foreground">
          {validation.runnable ? 'runnable' : 'not runnable'}
        </span>
      </div>
      <div className="divide-y divide-border">
        {validation.checks.map((check) => (
          <div key={`${check.code}-${check.path}-${check.message}`} className="px-3 py-2">
            <div className="flex items-center gap-2">
              <span className={cn(
                'h-2 w-2 rounded-full flex-shrink-0',
                check.status === 'passed' ? 'bg-emerald-400' : check.status === 'warning' ? 'bg-amber-400' : 'bg-destructive'
              )} />
              <span className="text-xs font-medium text-foreground truncate">{check.code}</span>
              <span className="ml-auto text-[10px] text-muted-foreground">{check.status}</span>
            </div>
            <p className="mt-1 text-xs text-muted-foreground break-words">{check.message}</p>
            {check.path && <p className="mt-1 font-mono text-[10px] text-muted-foreground">{check.path}</p>}
          </div>
        ))}
      </div>
    </section>
  )
}

function RunPanel({
  action,
  inputJson,
  inputError,
  result,
  disabled,
  isRunning,
  onInputChange,
  onLoadExample,
  onRun,
}: {
  action: Action
  inputJson: string
  inputError: string | null
  result: ActionRunResponse | null
  disabled: boolean
  isRunning: boolean
  onInputChange: (value: string) => void
  onLoadExample: () => void
  onRun: (dryRun: boolean) => void
}) {
  const hasExample = action.examples.length > 0
  const runDisabledReason = disabled
    ? isRunning
      ? 'Action run is in progress.'
      : 'Save or discard contract changes before running the persisted Action.'
    : ''

  return (
    <section className="border border-border rounded-md" aria-live="polite">
      <div className="px-3 py-2 border-b border-border flex items-center justify-between gap-2">
        <h3 className="text-xs font-semibold text-foreground">Run</h3>
        {hasExample && (
          <button
            type="button"
            onClick={onLoadExample}
            className="text-[11px] text-primary hover:underline"
          >
            Load example
          </button>
        )}
      </div>
      <div className="px-3 py-3 space-y-3">
        <TextAreaField
          label="Run input JSON"
          rows={5}
          mono
          value={inputJson}
          onChange={onInputChange}
        />
        {inputError && (
          <p className="text-xs text-destructive break-words">{inputError}</p>
        )}
        <div className="grid grid-cols-2 gap-2">
          <button
            type="button"
            onClick={() => onRun(true)}
            disabled={disabled}
            className={cn(
              'flex items-center justify-center gap-2 rounded-md border border-border px-3 py-2 text-sm text-foreground hover:bg-muted',
              disabled && 'cursor-not-allowed opacity-50'
            )}
          >
            <ShieldCheck className="h-4 w-4" />
            Dry run
          </button>
          <button
            type="button"
            onClick={() => onRun(false)}
            disabled={disabled}
            className={cn(
              'flex items-center justify-center gap-2 rounded-md border border-primary/50 bg-primary/10 px-3 py-2 text-sm text-foreground hover:bg-primary/20',
              disabled && 'cursor-not-allowed opacity-50'
            )}
          >
            <Play className="h-4 w-4" />
            Run
          </button>
        </div>
        {runDisabledReason && (
          <p className="text-xs text-muted-foreground">{runDisabledReason}</p>
        )}
        {action.permissions.destructive && (
          <p className="rounded-md border border-destructive/40 bg-destructive/10 px-2.5 py-2 text-xs text-destructive">
            This Action declares destructive permissions. The API still enforces run eligibility and command governance.
          </p>
        )}
        {result && <RunResult result={result} />}
      </div>
    </section>
  )
}

function RunResult({ result }: { result: ActionRunResponse }) {
  const success = result.status === 'completed' || result.status === 'dry-run'
  return (
    <div className="rounded-md border border-border bg-muted/20">
      <div className="flex items-center gap-2 border-b border-border px-2.5 py-2">
        <span className={cn('h-2 w-2 rounded-full', success ? 'bg-emerald-400' : 'bg-destructive')} />
        <span className="text-xs font-semibold text-foreground">{result.status}</span>
        <span className="ml-auto text-[11px] text-muted-foreground">{result.durationMs}ms</span>
      </div>
      <div className="space-y-2 px-2.5 py-2 text-xs">
        {result.argv.length > 0 && (
          <RunBlock label="Argv" value={result.argv.join('\n')} />
        )}
        {result.error && <p className="text-destructive break-words">{result.error}</p>}
        {result.stdout && (
          <RunBlock
            label={result.stdoutTruncated ? 'Stdout truncated' : 'Stdout'}
            value={result.stdout}
          />
        )}
        {result.stderr && (
          <RunBlock
            label={result.stderrTruncated ? 'Stderr truncated' : 'Stderr'}
            value={result.stderr}
          />
        )}
        {result.output && (
          <RunBlock label="Output" value={JSON.stringify(result.output, null, 2)} />
        )}
      </div>
    </div>
  )
}

function RunBlock({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="mb-1 text-[11px] font-medium text-muted-foreground">{label}</p>
      <pre className="max-h-44 overflow-auto whitespace-pre-wrap rounded-md bg-background px-2 py-2 font-mono text-[11px] leading-5 text-foreground">
        {value}
      </pre>
    </div>
  )
}

function HeaderButton({
  label,
  disabled,
  onClick,
  children,
}: {
  label: string
  disabled?: boolean
  onClick: () => void
  children: ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={cn(
        'h-9 min-w-9 px-2.5 rounded-lg flex items-center justify-center gap-1.5',
        'border border-border text-sm text-foreground hover:bg-muted transition-colors',
        disabled && 'opacity-50 cursor-not-allowed'
      )}
      title={label}
      aria-label={label}
    >
      {children}
      <span className="hidden sm:inline">{label}</span>
    </button>
  )
}

function IconButton({
  label,
  disabled,
  onClick,
  children,
}: {
  label: string
  disabled?: boolean
  onClick: () => void
  children: ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={cn(
        'h-8 w-8 rounded-md flex items-center justify-center text-muted-foreground hover:text-foreground hover:bg-muted',
        disabled && 'opacity-50 cursor-not-allowed'
      )}
      title={label}
      aria-label={label}
    >
      {children}
    </button>
  )
}

function SmallButton({
  label,
  onClick,
  children,
}: {
  label: string
  onClick: () => void
  children: ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="inline-flex items-center gap-1.5 rounded-md border border-border px-2 py-1 text-xs text-foreground hover:bg-muted"
      aria-label={label}
      title={label}
    >
      {children}
      <span>{label}</span>
    </button>
  )
}

function SectionTitle({ title }: { title: string }) {
  return <h4 className="text-xs font-semibold text-foreground">{title}</h4>
}

function TextField({
  label,
  value,
  onChange,
}: {
  label: string
  value: string
  onChange: (value: string) => void
}) {
  return (
    <label className="block min-w-0">
      <span className="block text-[11px] font-medium text-muted-foreground mb-1">{label}</span>
      <input
        type="text"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className={cn(
          'w-full rounded-md border border-border bg-background px-2.5 py-2',
          'text-xs text-foreground outline-none focus:ring-2 focus:ring-primary/40'
        )}
      />
    </label>
  )
}

function SelectField({
  label,
  value,
  options,
  onChange,
}: {
  label: string
  value: string
  options: readonly string[]
  onChange: (value: string) => void
}) {
  return (
    <label className="block min-w-0">
      <span className="block text-[11px] font-medium text-muted-foreground mb-1">{label}</span>
      <select
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className={cn(
          'w-full rounded-md border border-border bg-background px-2.5 py-2',
          'text-xs text-foreground outline-none focus:ring-2 focus:ring-primary/40'
        )}
      >
        {options.map((option) => (
          <option key={option} value={option}>{option}</option>
        ))}
      </select>
    </label>
  )
}

function TextAreaField({
  label,
  value,
  rows,
  hint,
  mono,
  onChange,
}: {
  label: string
  value: string
  rows: number
  hint?: string
  mono?: boolean
  onChange: (value: string) => void
}) {
  return (
    <label className="block min-w-0">
      <span className="block text-[11px] font-medium text-muted-foreground mb-1">{label}</span>
      <textarea
        value={value}
        rows={rows}
        onChange={(event) => onChange(event.target.value)}
        spellCheck={false}
        className={cn(
          'w-full resize-y rounded-md border border-border bg-background px-2.5 py-2',
          'text-xs text-foreground outline-none focus:ring-2 focus:ring-primary/40',
          mono && 'font-mono leading-5'
        )}
      />
      {hint && <span className="mt-1 block text-[11px] text-muted-foreground">{hint}</span>}
    </label>
  )
}

function stripRuntimeFields(action: Action): Omit<Action, 'createdAt' | 'updatedAt' | 'revision'> {
  const { createdAt: _createdAt, updatedAt: _updatedAt, revision: _revision, ...contract } = action
  return contract
}

function parseDraft(json: string): { ok: true; value: ActionContractDraft } | { ok: false; error: string } {
  try {
    return { ok: true, value: JSON.parse(json) as ActionContractDraft }
  } catch (error) {
    return { ok: false, error: error instanceof Error ? error.message : 'Invalid JSON' }
  }
}

function renameKey<T>(record: Record<string, T>, oldKey: string, newKey: string): Record<string, T> {
  const trimmed = newKey.trim()
  if (!trimmed || trimmed === oldKey) return record
  const next: Record<string, T> = {}
  for (const [key, value] of Object.entries(record)) {
    next[key === oldKey ? trimmed : key] = value
  }
  return next
}

function omitKey<T>(record: Record<string, T>, keyToRemove: string): Record<string, T> {
  const next: Record<string, T> = {}
  for (const [key, value] of Object.entries(record)) {
    if (key !== keyToRemove) next[key] = value
  }
  return next
}

function nextObjectKey(record: Record<string, unknown>, prefix: string): string {
  if (!record[prefix]) return prefix
  for (let index = 1; index < 1000; index += 1) {
    const candidate = `${prefix}${index}`
    if (!record[candidate]) return candidate
  }
  return `${prefix}${Date.now()}`
}

function formatDefaultValue(value: unknown): string {
  if (value === undefined) return ''
  if (typeof value === 'string') return value
  return JSON.stringify(value)
}

function parseDefaultValue(value: string, type: ActionInputType): unknown {
  if (value === '') return undefined
  if (type === 'boolean') return value === 'true'
  if (type === 'number' || type === 'integer') {
    const parsed = Number(value)
    return Number.isNaN(parsed) ? value : parsed
  }
  try {
    return JSON.parse(value)
  } catch {
    return value
  }
}

function formatRunInput(action: Action): string {
  const exampleInput = action.examples[0]?.input
  if (exampleInput) return JSON.stringify(exampleInput, null, 2)
  const input: Record<string, unknown> = {}
  for (const [name, spec] of Object.entries(action.inputs)) {
    if (spec.default !== undefined) {
      input[name] = spec.default
    }
  }
  return JSON.stringify(input, null, 2)
}

function parseRunInput(json: string): Record<string, unknown> {
  const parsed = JSON.parse(json) as unknown
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('Run input must be a JSON object.')
  }
  return parsed as Record<string, unknown>
}

function defaultInput(): ActionInput {
  return {
    type: 'string',
    description: '',
    required: true,
    enum: [],
    pattern: '',
    allowMultiline: false,
  }
}

async function copyContract(json: string) {
  try {
    await copyToClipboard(json)
    toast({ title: 'Copied', description: 'Action contract JSON copied.' })
  } catch (error) {
    toast({
      title: 'Copy failed',
      description: error instanceof Error ? error.message : 'Unable to copy contract JSON.',
      variant: 'destructive',
    })
  }
}
