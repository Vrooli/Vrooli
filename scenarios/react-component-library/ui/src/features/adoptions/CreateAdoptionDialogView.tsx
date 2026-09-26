import { Button } from "@vrooli/react-component-library/Button/2";
import { Dialog } from "@vrooli/react-component-library/Dialog/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { ResolveSource } from "../../api/adoptions";
import { StyleFitVerdictKind } from "../../api/components";
import { VerdictKind } from "../../api/deps";
import { errorMessage } from "../../lib/errorMessage";
import {
  PathSourceBadge,
  StyleFitBlock,
  VerdictBlock,
  WarnAcknowledgement,
} from "./CreateAdoptionDialogBlocks";

type AdoptionDialogModel = Record<string, any>;

export function renderCreateAdoptionDialog({ t, ...model }: AdoptionDialogModel) {
  const {
    open,
    onClose,
    componentId,
    setComponentId,
    scenario,
    setScenario,
    adoptedPath,
    setAdoptedPath,
    setPathUserEdited,
    setPathSource,
    pathResolving,
    pathSource,
    pathWarnings,
    adoptedVersion,
    setAdoptedVersion,
    replaceExisting,
    setReplaceExisting,
    validating,
    verdict,
    kind,
    verdictKindString,
    ack,
    setAck,
    styleValidating,
    styleVerdict,
    styleKind,
    createMutation,
    proceedDisabled,
    overwriteRequired,
    componentIdInputRef,
  } = model;
  return (
    <Dialog
      open={open}
      title={t(strings.adoptions.create.title)}
      description={t(strings.adoptions.create.subtitle)}
      onClose={onClose}
      closeLabel={t(strings.adoptions.create.cancelAction)}
      initialFocusRef={componentIdInputRef}
      className="max-w-md"
      footer={
        <div className="flex items-center justify-end gap-space-2xs">
          <Button
            variant="secondary"
            data-testid={selectors.adoptions.createCancel}
            onClick={onClose}
            disabled={createMutation.isPending}
          >
            {t(strings.adoptions.create.cancelAction)}
          </Button>
          <Button
            data-testid={selectors.adoptions.createConfirm}
            onClick={() => createMutation.mutate()}
            disabled={proceedDisabled}
          >
            {createMutation.isPending
              ? t(strings.adoptions.creating)
              : overwriteRequired
                ? t(strings.adoptions.create.confirmOverwriteAction)
                : t(strings.adoptions.create.confirmAction)}
          </Button>
        </div>
      }
    >
      <div data-testid={selectors.adoptions.createDialog}>
        <div className="mt-space-xs space-y-space-2xs">
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.adoptions.create.componentIdLabel)}
            <Input
              ref={componentIdInputRef}
              data-testid={selectors.adoptions.createComponentId}
              value={componentId}
              onChange={(e) => setComponentId(e.target.value)}
              placeholder={t(strings.adoptions.create.componentIdPlaceholder)}
              className="mt-space-3xs"
            />
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.adoptions.create.scenarioLabel)}
            <Input
              data-testid={selectors.adoptions.createScenario}
              value={scenario}
              onChange={(e) => setScenario(e.target.value)}
              placeholder={t(strings.adoptions.create.scenarioPlaceholder)}
              className="mt-space-3xs"
            />
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.adoptions.create.adoptedPathLabel)}
            <Input
              data-testid={selectors.adoptions.createAdoptedPath}
              value={adoptedPath}
              onChange={(e) => {
                setAdoptedPath(e.target.value);
                setPathUserEdited(true);
                setPathSource(ResolveSource.EXPLICIT);
              }}
              placeholder={t(strings.adoptions.create.adoptedPathPlaceholder)}
              className="mt-space-3xs"
            />
            <PathSourceBadge
              resolving={pathResolving}
              source={pathSource}
              warnings={pathWarnings}
            />
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.adoptions.create.adoptedVersionLabel)}
            <Input
              data-testid={selectors.adoptions.createAdoptedVersion}
              value={adoptedVersion}
              onChange={(e) => setAdoptedVersion(e.target.value)}
              placeholder={t(strings.adoptions.create.adoptedVersionPlaceholder)}
              className="mt-space-3xs"
            />
          </label>
          <label className="flex items-start gap-space-2xs rounded-md border border-app-warning/30 bg-app-warning/10 p-space-2xs text-xs text-app-foreground">
            <input
              type="checkbox"
              aria-label={t(strings.adoptions.create.replaceExistingLabel)}
              checked={replaceExisting}
              onChange={(event) => setReplaceExisting(event.target.checked)}
            />
            <span>
              <span className="block font-medium">
                {t(strings.adoptions.create.replaceExistingLabel)}
              </span>
              <span className="text-app-muted-foreground">
                {t(strings.adoptions.create.replaceExistingHelp)}
              </span>
            </span>
          </label>
        </div>

        <VerdictBlock
          validating={validating}
          verdict={verdict}
          kind={kind}
          verdictKindString={verdictKindString}
          ack={ack}
          setAck={setAck}
        />
        <StyleFitBlock validating={styleValidating} verdict={styleVerdict} />
        {kind !== VerdictKind.WARN && styleKind === StyleFitVerdictKind.WARN && (
          <WarnAcknowledgement ack={ack} setAck={setAck} />
        )}

        {createMutation.error && (
          <p
            data-testid={selectors.adoptions.createError}
            className="mt-space-xs text-xs text-app-danger"
          >
            {overwriteRequired
              ? t(strings.adoptions.create.overwriteRequired)
              : errorMessage(createMutation.error, t)}
          </p>
        )}
      </div>
    </Dialog>
  );
}
