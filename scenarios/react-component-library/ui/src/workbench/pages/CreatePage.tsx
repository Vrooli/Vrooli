import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { CircleAlert, FileCode2, Layers, Plus, Sparkles } from "lucide-react";

import { componentsClient } from "../../api/components";
import { adoptionsClient } from "../../api/adoptions";
import { useTranslation } from "../../i18n";
import { appRoutes } from "../../routes";
import { useCatalog, useDesignScenarios, wbKeys } from "../data";
import {
  Banner,
  Chip,
  Field,
  Panel,
  PanelBody,
  PanelHead,
  Select,
  TextArea,
  TextInput,
  WbButton,
} from "../primitives";

type Route = "create" | "extract" | "adopt";

const DOC_PATH = "docs/concepts/ASSET-DERIVATION.md";

export function CreatePage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const catalog = useCatalog();
  const scenarios = useDesignScenarios();
  const [route, setRoute] = useState<Route>("create");

  const [slug, setSlug] = useState("");
  const [libraryId, setLibraryId] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [description, setDescription] = useState("");
  const [tags, setTags] = useState("");
  const [initialVersion, setInitialVersion] = useState("0.1.0");

  const [sourceScenario, setSourceScenario] = useState("");
  const [sourceFile, setSourceFile] = useState("");
  const [slot, setSlot] = useState("ui-primitive");

  const [adoptComponent, setAdoptComponent] = useState("");
  const [adoptScenario, setAdoptScenario] = useState("");
  const [adoptVersion, setAdoptVersion] = useState("");
  const [preflight, setPreflight] = useState<{
    blocking: boolean;
    styleFit: string;
    dependencyVerdict: string;
    warnings: string[];
    unsatisfiedTokens: string[];
  } | null>(null);

  const create = useMutation({
    mutationFn: () =>
      componentsClient.initializeComponent({
        libraryId,
        slug,
        displayName,
        description,
        tags: tags
          .split(",")
          .map((tag) => tag.trim())
          .filter(Boolean),
        initialVersion,
        fileName: `${displayName.replace(/[^A-Za-z0-9]/g, "") || "Component"}.tsx`,
        initialSource: [
          "/**",
          ` * @libraryId ${libraryId || `react-component-library:${slug}`}`,
          ` * @displayName ${displayName || slug}`,
          ` * @description ${description || "A new shared component."}`,
          ` * @version ${initialVersion}`,
          " * @tags []",
          " */",
          'import type { ReactNode } from "react";',
          "",
          `export interface ${displayName.replace(/[^A-Za-z0-9]/g, "") || "Component"}Props {`,
          "  children?: ReactNode;",
          "}",
          "",
          `export function ${displayName.replace(/[^A-Za-z0-9]/g, "") || "Component"}({ children }: ${displayName.replace(/[^A-Za-z0-9]/g, "") || "Component"}Props) {`,
          "  return <div>{children}</div>;",
          "}",
          "",
        ].join("\n"),
      }),
    onSuccess: (response) => {
      void queryClient.invalidateQueries({ queryKey: wbKeys.catalog });
      if (response.component?.id) void navigate(`/assets/${encodeURIComponent(response.component.id)}`);
    },
  });

  const extract = useMutation({
    mutationFn: () =>
      componentsClient.ingestComponent({
        scenario: sourceScenario,
        sourceFile,
        slug,
        displayName,
        description,
        tags: tags
          .split(",")
          .map((tag) => tag.trim())
          .filter(Boolean),
        slot,
        version: initialVersion,
        sourceFiles: [],
        acceptBehaviorLoss: false,
        experienceContractPath: "",
      }),
    onSuccess: (response) => {
      void queryClient.invalidateQueries({ queryKey: wbKeys.catalog });
      const id = (response as { component?: { id?: string } }).component?.id;
      if (id) void navigate(`/assets/${encodeURIComponent(id)}`);
    },
  });

  const check = useMutation({
    mutationFn: () =>
      adoptionsClient.preflightAdoption({
        componentId: adoptComponent,
        scenario: adoptScenario,
        version: adoptVersion,
      }),
    onSuccess: (response) => {
      setPreflight({
        blocking: response.blocking,
        styleFit: response.styleFit,
        dependencyVerdict: response.dependencyVerdict,
        warnings: response.warnings,
        unsatisfiedTokens: response.unsatisfiedTokens,
      });
    },
  });

  const link = useMutation({
    mutationFn: () =>
      adoptionsClient.linkAdoption({
        componentId: adoptComponent,
        scenario: adoptScenario,
        version: adoptVersion,
        importSubpath: "",
        confirmExisting: false,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: wbKeys.catalog });
      void navigate(appRoutes.components);
    },
  });

  const assets = catalog.data ?? [];
  const selectedAsset = assets.find((asset) => asset.id === adoptComponent);
  const versions = useMemo(() => {
    if (!selectedAsset) return [];
    const values = [selectedAsset.latestVersion, selectedAsset.version, selectedAsset.draftVersion].filter(
      (value): value is string => Boolean(value),
    );
    return [...new Set(values)];
  }, [selectedAsset]);

  const routes: Array<{ id: Route; index: string; title: string; body: string }> = [
    {
      id: "create",
      index: "01 · CREATE",
      title: t("workbench.create.createTitle", { defaultValue: "Create a component" }),
      body: t("workbench.create.createBody", {
        defaultValue: "Start from catalog intent and a new versioned source draft.",
      }),
    },
    {
      id: "extract",
      index: "02 · EXTRACT",
      title: t("workbench.create.extractTitle", { defaultValue: "Extract from a scenario" }),
      body: t("workbench.create.extractBody", {
        defaultValue: "Harvest a real component and keep its source provenance.",
      }),
    },
    {
      id: "adopt",
      index: "03 · ADOPT",
      title: t("workbench.create.adoptTitle", { defaultValue: "Adopt into a scenario" }),
      body: t("workbench.create.adoptBody", {
        defaultValue: "Pin a released asset and synchronize its tokens and styles.",
      }),
    },
  ];

  const active = routes.find((entry) => entry.id === route) ?? routes[0];
  if (!active) return null;

  return (
    <div className="wb-page" data-testid="create-page">
      <div className="wb-pagehead">
        <div className="wb-pagehead-copy">
          <p className="wb-kicker">{t("workbench.create.kicker", { defaultValue: "Library · New asset" })}</p>
          <h1 className="wb-h1">{t("workbench.create.title", { defaultValue: "Add to the library" })}</h1>
          <p className="wb-lede">
            {t("workbench.create.lede", {
              defaultValue:
                "Three governed routes into the same lifecycle. Every route ends in a draft, focused checks and an immutable successor.",
            })}
          </p>
        </div>
        <div className="wb-actions">
          <WbButton variant="ghost" size="md" onClick={() => void navigate(appRoutes.components)}>
            {t("workbench.create.cancel", { defaultValue: "Cancel" })}
          </WbButton>
        </div>
      </div>

      <div className="wb-create-tabs">
        {routes.map((entry) => (
          <button
            key={entry.id}
            type="button"
            className="wb-create-tab"
            aria-pressed={route === entry.id}
            onClick={() => setRoute(entry.id)}
          >
            <small>{entry.index}</small>
            <b>{entry.title}</b>
            <span>{entry.body}</span>
          </button>
        ))}
      </div>

      <div className="wb-create-layout">
        <Panel>
          <PanelHead
            title={active.title}
            meta={t("workbench.create.draftOnly", { defaultValue: "Draft only · no release yet" })}
          />
          <PanelBody flush>
            {route === "create" ? (
              <div className="wb-form-grid">
                <Field label={t("workbench.create.slug", { defaultValue: "Slug" })} htmlFor="create-slug">
                  <TextInput id="create-slug" value={slug} onChange={(event) => setSlug(event.target.value)} placeholder={t("workbench.create.slugPlaceholder", { defaultValue: "AgentCard" })} />
                </Field>
                <Field label={t("workbench.create.libraryId", { defaultValue: "Library id" })} htmlFor="create-library">
                  <TextInput
                    id="create-library"
                    value={libraryId}
                    onChange={(event) => setLibraryId(event.target.value)}
                    placeholder={t("workbench.create.libraryPlaceholder", { defaultValue: "react-component-library:AgentCard" })}
                  />
                </Field>
                <Field label={t("workbench.create.displayName", { defaultValue: "Display name" })} htmlFor="create-name">
                  <TextInput id="create-name" value={displayName} onChange={(event) => setDisplayName(event.target.value)} />
                </Field>
                <Field label={t("workbench.create.version", { defaultValue: "Initial version" })} htmlFor="create-version">
                  <TextInput id="create-version" value={initialVersion} onChange={(event) => setInitialVersion(event.target.value)} />
                </Field>
                <div className="is-full">
                  <Field label={t("workbench.create.description", { defaultValue: "Description" })} htmlFor="create-desc">
                    <TextArea id="create-desc" value={description} onChange={(event) => setDescription(event.target.value)} />
                  </Field>
                </div>
                <div className="is-full">
                  <Field label={t("workbench.create.tags", { defaultValue: "Tags" })} htmlFor="create-tags">
                    <TextInput id="create-tags" value={tags} onChange={(event) => setTags(event.target.value)} placeholder={t("workbench.create.tagsPlaceholder", { defaultValue: "data-display, card" })} />
                  </Field>
                </div>
                {create.isError ? (
                  <div className="is-full">
                    <Banner tone="red" icon={<CircleAlert aria-hidden size={15} />}>
                      {create.error instanceof Error ? create.error.message : String(create.error)}
                    </Banner>
                  </div>
                ) : null}
                <div className="is-full">
                  <p className="wb-muted-2 wb-small" style={{ marginBottom: 8 }}>
                    {t("workbench.create.requiredHint", {
                      defaultValue: "Slug and display name are required before a draft can be created.",
                    })}
                  </p>
                </div>
                <div className="is-full wb-row">
                  <WbButton
                    variant="primary"
                    size="md"
                    onClick={() => create.mutate()}
                    disabled={create.isPending || !slug || !displayName}
                  >
                    <Plus aria-hidden size={14} />
                    {create.isPending
                      ? t("workbench.create.creating", { defaultValue: "Creating…" })
                      : t("workbench.create.createAction", { defaultValue: "Create draft" })}
                  </WbButton>
                </div>
              </div>
            ) : null}

            {route === "extract" ? (
              <div className="wb-form-grid">
                <Field label={t("workbench.create.sourceScenario", { defaultValue: "Source scenario" })} htmlFor="extract-scenario">
                  <Select
                    id="extract-scenario"
                    value={sourceScenario}
                    onChange={(event) => setSourceScenario(event.target.value)}
                  >
                    <option value="">{t("workbench.create.chooseScenario", { defaultValue: "Choose a scenario" })}</option>
                    {(scenarios.data ?? []).map((entry) => (
                      <option key={entry.scenario} value={entry.scenario}>
                        {entry.scenario}
                      </option>
                    ))}
                  </Select>
                </Field>
                <Field label={t("workbench.create.sourcePath", { defaultValue: "Source path" })} htmlFor="extract-path">
                  <TextInput
                    id="extract-path"
                    value={sourceFile}
                    onChange={(event) => setSourceFile(event.target.value)}
                    placeholder={t("workbench.create.sourcePlaceholder", { defaultValue: "ui/src/components/AgentCard.tsx" })}
                  />
                </Field>
                <Field label={t("workbench.create.assetName", { defaultValue: "Asset name" })} htmlFor="extract-name">
                  <TextInput id="extract-name" value={displayName} onChange={(event) => setDisplayName(event.target.value)} />
                </Field>
                <Field label={t("workbench.create.slot", { defaultValue: "Slot" })} htmlFor="extract-slot">
                  <TextInput id="extract-slot" value={slot} onChange={(event) => setSlot(event.target.value)} />
                </Field>
                <Field label={t("workbench.create.version", { defaultValue: "Initial version" })} htmlFor="extract-version">
                  <TextInput id="extract-version" value={initialVersion} onChange={(event) => setInitialVersion(event.target.value)} />
                </Field>
                <Field label={t("workbench.create.slug", { defaultValue: "Slug" })} htmlFor="extract-slug">
                  <TextInput id="extract-slug" value={slug} onChange={(event) => setSlug(event.target.value)} />
                </Field>
                <div className="is-full">
                  <Field label={t("workbench.create.description", { defaultValue: "Description" })} htmlFor="extract-desc">
                    <TextArea id="extract-desc" value={description} onChange={(event) => setDescription(event.target.value)} />
                  </Field>
                </div>
                <div className="is-full">
                  <Field label={t("workbench.create.provenance", { defaultValue: "Why this belongs in the library" })} htmlFor="extract-tags">
                    <TextInput id="extract-tags" value={tags} onChange={(event) => setTags(event.target.value)} placeholder={t("workbench.create.tagsPlaceholder", { defaultValue: "data-display, card" })} />
                  </Field>
                </div>
                {extract.isError ? (
                  <div className="is-full">
                    <Banner tone="red" icon={<CircleAlert aria-hidden size={15} />}>
                      {extract.error instanceof Error ? extract.error.message : String(extract.error)}
                    </Banner>
                  </div>
                ) : null}
                <div className="is-full wb-row">
                  <WbButton
                    variant="primary"
                    size="md"
                    onClick={() => extract.mutate()}
                    disabled={extract.isPending || !sourceScenario || !sourceFile || !displayName}
                  >
                    <FileCode2 aria-hidden size={14} />
                    {extract.isPending
                      ? t("workbench.create.extracting", { defaultValue: "Extracting…" })
                      : t("workbench.create.extractAction", { defaultValue: "Extract draft" })}
                  </WbButton>
                </div>
              </div>
            ) : null}

            {route === "adopt" ? (
              <div className="wb-form-grid">
                <Field label={t("workbench.create.asset", { defaultValue: "Asset" })} htmlFor="adopt-asset">
                  <Select
                    id="adopt-asset"
                    value={adoptComponent}
                    onChange={(event) => {
                      setAdoptComponent(event.target.value);
                      setPreflight(null);
                      const asset = assets.find((entry) => entry.id === event.target.value);
                      setAdoptVersion(asset?.latestVersion || asset?.version || "");
                    }}
                  >
                    <option value="">{t("workbench.create.chooseAsset", { defaultValue: "Choose an asset" })}</option>
                    {assets.slice(0, 200).map((asset) => (
                      <option key={asset.id} value={asset.id}>
                        {asset.displayName} · {asset.latestVersion || asset.version || "draft"}
                      </option>
                    ))}
                  </Select>
                </Field>
                <Field label={t("workbench.create.targetScenario", { defaultValue: "Target scenario" })} htmlFor="adopt-scenario">
                  <Select
                    id="adopt-scenario"
                    value={adoptScenario}
                    onChange={(event) => {
                      setAdoptScenario(event.target.value);
                      setPreflight(null);
                    }}
                  >
                    <option value="">{t("workbench.create.chooseScenario", { defaultValue: "Choose a scenario" })}</option>
                    {(scenarios.data ?? []).map((entry) => (
                      <option key={entry.scenario} value={entry.scenario}>
                        {entry.scenario}
                      </option>
                    ))}
                  </Select>
                </Field>
                <Field label={t("workbench.create.version", { defaultValue: "Version" })} htmlFor="adopt-version">
                  <Select id="adopt-version" value={adoptVersion} onChange={(event) => setAdoptVersion(event.target.value)}>
                    {versions.map((value) => (
                      <option key={value} value={value}>
                        {value}
                      </option>
                    ))}
                  </Select>
                </Field>
                <div className="is-full wb-row">
                  <WbButton
                    variant="ghost"
                    size="md"
                    onClick={() => check.mutate()}
                    disabled={check.isPending || !adoptComponent || !adoptScenario}
                  >
                    <Sparkles aria-hidden size={14} />
                    {check.isPending
                      ? t("workbench.create.checking", { defaultValue: "Checking…" })
                      : t("workbench.create.preflight", { defaultValue: "Preflight adoption" })}
                  </WbButton>
                  <WbButton
                    variant="primary"
                    size="md"
                    onClick={() => link.mutate()}
                    disabled={link.isPending || !preflight || preflight.blocking}
                  >
                    <Layers aria-hidden size={14} />
                    {link.isPending
                      ? t("workbench.create.linking", { defaultValue: "Linking…" })
                      : t("workbench.create.link", { defaultValue: "Link adoption" })}
                  </WbButton>
                </div>
                {check.isError || link.isError ? (
                  <div className="is-full">
                    <Banner tone="red" icon={<CircleAlert aria-hidden size={15} />}>
                      {(check.error ?? link.error) instanceof Error
                        ? ((check.error ?? link.error) as Error).message
                        : String(check.error ?? link.error)}
                    </Banner>
                  </div>
                ) : null}
                {preflight ? (
                  <div className="is-full">
                    <div className="wb-row" style={{ flexWrap: "wrap" }}>
                      <Chip tone={preflight.blocking ? "red" : "green"}>
                        {preflight.blocking
                          ? t("workbench.create.blocked", { defaultValue: "Blocked" })
                          : t("workbench.create.ready", { defaultValue: "Ready" })}
                      </Chip>
                      <Chip tone="ice">{preflight.styleFit}</Chip>
                      <Chip>{preflight.dependencyVerdict}</Chip>
                    </div>
                    {preflight.unsatisfiedTokens.length > 0 ? (
                      <Banner tone="amber" icon={<CircleAlert aria-hidden size={15} />}>
                        {t("workbench.create.tokensMissing", {
                          defaultValue: "Unsatisfied tokens: {{tokens}}",
                          tokens: preflight.unsatisfiedTokens.join(", "),
                        })}
                      </Banner>
                    ) : null}
                    {preflight.warnings.map((warning) => (
                      <p key={warning} className="wb-small wb-muted">
                        {warning}
                      </p>
                    ))}
                  </div>
                ) : null}
              </div>
            ) : null}
          </PanelBody>
        </Panel>

        <aside className="wb-stack">
          <Panel>
            <PanelHead title={t("workbench.create.next", { defaultValue: "What happens next" })} meta={t("workbench.create.steps", { defaultValue: "4 steps" })} />
            <PanelBody flush>
              <div className="wb-steps">
                {[
                  [t("workbench.create.step1", { defaultValue: "Draft created" }), t("workbench.create.step1Body", { defaultValue: "A mutable version folder and manifest pointer." })],
                  [t("workbench.create.step2", { defaultValue: "Derived artifacts" }), t("workbench.create.step2Body", { defaultValue: "Story contract, dependency lock, exports." })],
                  [t("workbench.create.step3", { defaultValue: "Focused checks" }), t("workbench.create.step3Body", { defaultValue: "Types, stories and catalog gates." })],
                  [t("workbench.create.step4", { defaultValue: "Publish successor" }), t("workbench.create.step4Body", { defaultValue: "Immutable release with provenance." })],
                ].map(([title, body], index) => (
                  <div key={title} className="wb-step">
                    <b>{index + 1}</b>
                    <span>
                      {title}
                      <small>{body}</small>
                    </span>
                  </div>
                ))}
              </div>
            </PanelBody>
          </Panel>

          <Panel>
            <PanelHead title={t("workbench.create.summary", { defaultValue: "Summary" })} meta={t("workbench.create.beforeAnalysis", { defaultValue: "Before analysis" })} />
            <PanelBody>
              <div className="wb-summary">
                <div className="wb-row-between">
                  <span>{t("workbench.create.owner", { defaultValue: "Owner" })}</span>
                  <b>{t("workbench.create.ownerValue", { defaultValue: "Local operator" })}</b>
                </div>
                <div className="wb-row-between">
                  <span>{t("workbench.create.policy", { defaultValue: "Policy" })}</span>
                  <b>{t("workbench.create.policyValue", { defaultValue: "Governed" })}</b>
                </div>
                <div className="wb-row-between">
                  <span>{t("workbench.create.draft", { defaultValue: "Draft" })}</span>
                  <b>{initialVersion}</b>
                </div>
                <div className="wb-row-between">
                  <span>{t("workbench.create.target", { defaultValue: "Target" })}</span>
                  <b>{route === "adopt" ? adoptScenario || "—" : route === "extract" ? slot : "catalog"}</b>
                </div>
              </div>
              <p className="wb-muted-2 wb-small" style={{ marginTop: 10 }}>
                {t("workbench.create.summaryNote", {
                  defaultValue:
                    "No route publishes directly. The draft is inspectable, and the release is a separate governed step.",
                })}
              </p>
            </PanelBody>
          </Panel>

          <Panel>
            <PanelHead title={t("workbench.create.docs", { defaultValue: "Reference" })} />
            <PanelBody>
              <p className="wb-mono wb-small wb-muted">{DOC_PATH}</p>
              <p className="wb-small wb-muted" style={{ marginTop: 6 }}>
                {t("workbench.create.docsBody", {
                  defaultValue:
                    "The authored source loop — draft, validate, publish, compile, inspect — is unchanged by the workbench redesign.",
                })}
              </p>
            </PanelBody>
          </Panel>
        </aside>
      </div>
    </div>
  );
}
