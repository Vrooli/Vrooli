import { useTranslation } from "../../i18n";
import { Field, Select, TextInput, Toggle } from "../primitives";
import type { TemplateNode } from "../templates/spec";

const TEXT_VARIANTS = [
  "display",
  "title",
  "subtitle",
  "body",
  "label",
  "caption",
  "eyebrow",
  "mono",
  "price",
];

/** Property editor for one selected layer. Shared by the inspector only. */
export function NodeInspector({
  node: selectedNode,
  onChange,
}: {
  node: TemplateNode;
  onChange: (key: string, value: string | number | boolean) => void;
}) {
  const { t } = useTranslation();
  const props = selectedNode.props;
  const stringValue = (key: string): string => (typeof props[key] === "string" ? props[key] : "");
  const numberValue = (key: string, fallback = 0): number =>
    typeof props[key] === "number" ? props[key] : fallback;
  const has = (key: string): boolean => key in props;
  const isButton = selectedNode.type === "button";

  return (
    <div className="wb-stack tight">
      <span className="wb-caption">{selectedNode.type}</span>

      {has("text") ? (
        <Field label={t("workbench.studio.text", { defaultValue: "Text" })}>
          <TextInput
            value={stringValue("text")}
            onChange={(event) => onChange("text", event.target.value)}
          />
        </Field>
      ) : null}

      {has("placeholder") ? (
        <Field label={t("workbench.studio.placeholder", { defaultValue: "Placeholder" })}>
          <TextInput
            value={stringValue("placeholder")}
            onChange={(event) => onChange("placeholder", event.target.value)}
          />
        </Field>
      ) : null}

      {has("name") ? (
        <Field label={t("workbench.studio.name", { defaultValue: "Name" })}>
          <TextInput
            value={stringValue("name")}
            onChange={(event) => onChange("name", event.target.value)}
          />
        </Field>
      ) : null}

      {has("label") ? (
        <Field label={t("workbench.studio.label", { defaultValue: "Label" })}>
          <TextInput
            value={stringValue("label")}
            onChange={(event) => onChange("label", event.target.value)}
          />
        </Field>
      ) : null}

      {has("hint") ? (
        <Field label={t("workbench.studio.hint", { defaultValue: "Slot hint" })}>
          <TextInput
            value={stringValue("hint")}
            onChange={(event) => onChange("hint", event.target.value)}
          />
        </Field>
      ) : null}

      {selectedNode.type === "image" ? (
        <>
          <Field label={t("workbench.studio.imageAlt", { defaultValue: "Alt text" })}>
            <TextInput
              value={stringValue("alt")}
              onChange={(event) => onChange("alt", event.target.value)}
            />
          </Field>
          <Field label={t("workbench.studio.imageFit", { defaultValue: "Fit" })}>
            <Select
              value={stringValue("fit") || "cover"}
              onChange={(event) => onChange("fit", event.target.value)}
            >
              <option value="cover">{t("workbench.studio.fitCover", { defaultValue: "Cover" })}</option>
              <option value="contain">
                {t("workbench.studio.fitContain", { defaultValue: "Contain" })}
              </option>
            </Select>
          </Field>
          <Field label={t("workbench.studio.imageHeight", { defaultValue: "Height" })}>
            <TextInput
              type="number"
              min={40}
              value={numberValue("h", 220)}
              onChange={(event) =>
                onChange("h", Math.max(40, Number(event.target.value)))
              }
            />
          </Field>
        </>
      ) : null}

      {selectedNode.type === "stat" ? (
        <>
          <Field label={t("workbench.studio.value", { defaultValue: "Value" })}>
            <TextInput
              value={stringValue("value")}
              onChange={(event) => onChange("value", event.target.value)}
            />
          </Field>
          <Field label={t("workbench.studio.detail", { defaultValue: "Detail" })}>
            <TextInput
              value={stringValue("detail")}
              onChange={(event) => onChange("detail", event.target.value)}
            />
          </Field>
        </>
      ) : null}

      {selectedNode.type === "meter" ? (
        <Field label={t("workbench.studio.value", { defaultValue: "Value" })}>
          <TextInput
            type="number"
            min={0}
            max={100}
            value={numberValue("value")}
            onChange={(event) =>
              onChange("value", Math.max(0, Math.min(100, Number(event.target.value))))
            }
          />
        </Field>
      ) : null}

      {has("variant") ? (
        <Field label={t("workbench.studio.variant", { defaultValue: "Variant" })}>
          <Select
            value={stringValue("variant")}
            onChange={(event) => onChange("variant", event.target.value)}
          >
            {(isButton ? ["primary", "secondary", "ghost"] : TEXT_VARIANTS).map((value) => (
              <option key={value} value={value}>
                {value}
              </option>
            ))}
          </Select>
        </Field>
      ) : null}

      {has("tone") ? (
        <Field label={t("workbench.studio.tone", { defaultValue: "Tone" })}>
          <Select
            value={stringValue("tone")}
            onChange={(event) => onChange("tone", event.target.value)}
          >
            {["ink", "muted", "accent", "secondary", "green", "amber", "red", "ice"].map((value) => (
              <option key={value} value={value}>
                {value}
              </option>
            ))}
          </Select>
        </Field>
      ) : null}

      {has("shape") ? (
        <Field label={t("workbench.studio.shape", { defaultValue: "Art shape" })}>
          <Select
            value={stringValue("shape")}
            onChange={(event) => onChange("shape", event.target.value)}
          >
            {["none", "orb", "vase", "arch", "rings", "wave", "grid", "portrait"].map((value) => (
              <option key={value} value={value}>
                {value}
              </option>
            ))}
          </Select>
        </Field>
      ) : null}

      {has("align") ? (
        <Field label={t("workbench.studio.align", { defaultValue: "Align" })}>
          <Select
            value={stringValue("align")}
            onChange={(event) => onChange("align", event.target.value)}
          >
            {["left", "center", "right", "start", "end"].map((value) => (
              <option key={value} value={value}>
                {value}
              </option>
            ))}
          </Select>
        </Field>
      ) : null}

      {has("gap") ? (
        <Field label={t("workbench.studio.gap", { defaultValue: "Gap" })}>
          <TextInput
            type="number"
            value={numberValue("gap")}
            onChange={(event) => onChange("gap", Number(event.target.value))}
          />
        </Field>
      ) : null}

      {has("cols") ? (
        <Field label={t("workbench.studio.cols", { defaultValue: "Columns" })}>
          <TextInput
            type="number"
            min={1}
            max={6}
            value={numberValue("cols", 1)}
            onChange={(event) =>
              onChange("cols", Math.max(1, Math.min(6, Number(event.target.value))))
            }
          />
        </Field>
      ) : null}

      {has("pad") ? (
        <Field label={t("workbench.studio.pad", { defaultValue: "Padding" })}>
          <TextInput
            type="number"
            value={numberValue("pad")}
            onChange={(event) => onChange("pad", Number(event.target.value))}
          />
        </Field>
      ) : null}

      {has("h") ? (
        <Field label={t("workbench.studio.height", { defaultValue: "Height" })}>
          <TextInput
            type="number"
            value={numberValue("h")}
            onChange={(event) => onChange("h", Number(event.target.value))}
          />
        </Field>
      ) : null}

      {has("elevated") ? (
        <div className="wb-row-between">
          <span className="wb-label">
            {t("workbench.studio.elevated", { defaultValue: "Elevated" })}
          </span>
          <Toggle
            checked={Boolean(props.elevated)}
            onChange={(next) => onChange("elevated", next)}
            label={t("workbench.studio.elevated", { defaultValue: "Elevated" })}
          />
        </div>
      ) : null}

      <p className="wb-muted-2 wb-small">
        {t("workbench.studio.editNote", {
          defaultValue:
            "Edits update the local composition immediately. Accept creates a revision; it does not publish a library release.",
        })}
      </p>
    </div>
  );
}
