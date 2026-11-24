import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { initializeComponent, ingestComponent } from "../../api/components";
import { adoptionsClient } from "../../api/adoptions";
import { useTranslation } from "../../i18n";
import { appRoutes } from "../../routes";
import { useCatalog, useDesignScenarios, wbKeys } from "../data";
import { Panel, PanelBody, PanelHead } from "../primitives";
import { CreateGuidance } from "./CreateGuidance";
import { AdoptComponentForm, CreateComponentForm, ExtractComponentForm } from "./CreateRouteForms";

type Route = "create" | "extract" | "adopt";

export function CreateWorkflowLayout({ route, title }: { route: Route; title: string }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const catalog = useCatalog();
  const scenarios = useDesignScenarios();
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
      initializeComponent({
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
      if (response.component?.id)
        void navigate(`/assets/${encodeURIComponent(response.component.id)}`);
    },
  });

  const extract = useMutation({
    mutationFn: () =>
      ingestComponent({
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
    const values = [
      selectedAsset.latestVersion,
      selectedAsset.version,
      selectedAsset.draftVersion,
    ].filter((value): value is string => Boolean(value));
    return [...new Set(values)];
  }, [selectedAsset]);

  return (
    <div className="wb-create-layout">
      <Panel>
        <PanelHead
          title={title}
          meta={t("workbench.create.draftOnly", { defaultValue: "Draft only · no release yet" })}
        />
        <PanelBody flush>
          {route === "create" ? (
            <CreateComponentForm
              slug={slug}
              setSlug={setSlug}
              libraryId={libraryId}
              setLibraryId={setLibraryId}
              displayName={displayName}
              setDisplayName={setDisplayName}
              description={description}
              setDescription={setDescription}
              tags={tags}
              setTags={setTags}
              initialVersion={initialVersion}
              setInitialVersion={setInitialVersion}
              create={create}
            />
          ) : null}
          {route === "extract" ? (
            <ExtractComponentForm
              sourceScenario={sourceScenario}
              setSourceScenario={setSourceScenario}
              scenarios={scenarios.data ?? []}
              sourceFile={sourceFile}
              setSourceFile={setSourceFile}
              displayName={displayName}
              setDisplayName={setDisplayName}
              slot={slot}
              setSlot={setSlot}
              initialVersion={initialVersion}
              setInitialVersion={setInitialVersion}
              slug={slug}
              setSlug={setSlug}
              description={description}
              setDescription={setDescription}
              tags={tags}
              setTags={setTags}
              extract={extract}
            />
          ) : null}
          {route === "adopt" ? (
            <AdoptComponentForm
              adoptComponent={adoptComponent}
              setAdoptComponent={setAdoptComponent}
              setPreflight={setPreflight}
              assets={assets}
              adoptScenario={adoptScenario}
              setAdoptScenario={setAdoptScenario}
              scenarios={scenarios.data ?? []}
              adoptVersion={adoptVersion}
              setAdoptVersion={setAdoptVersion}
              versions={versions}
              check={check}
              link={link}
              preflight={preflight}
            />
          ) : null}
        </PanelBody>
      </Panel>

      <CreateGuidance
        initialVersion={initialVersion}
        route={route}
        adoptScenario={adoptScenario}
        slot={slot}
      />
    </div>
  );
}
