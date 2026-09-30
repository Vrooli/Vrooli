export interface DictationScript {
  id: string;
  title: string;
  purpose: string;
  text: string;
  tags: string[];
  difficulty: "short" | "medium" | "long";
}

export const CUSTOM_SCRIPT_ID = "custom";

export const DICTATION_SCRIPTS: DictationScript[] = [
  {
    id: "numbers-currency",
    title: "Numbers and currency",
    purpose: "Exercises digits, decimals, and money formatting.",
    text: "Invoice 3087 totals $1,249.65, due on July 14 at 9:30 AM.",
    tags: ["script:numbers", "script:currency", "condition:clean"],
    difficulty: "short",
  },
  {
    id: "punctuation-heavy",
    title: "Punctuation-heavy sentence",
    purpose: "Checks commas, quotes, dashes, and question marks.",
    text: "She asked, \"Are we ready?\" I said yes, but only after reviewing the final checklist.",
    tags: ["script:punctuation", "script:quotes", "condition:clean"],
    difficulty: "medium",
  },
  {
    id: "proper-nouns",
    title: "Proper nouns",
    purpose: "Stresses names, places, and organizations.",
    text: "Maya Chen met Rafael Ortiz near Lake Merritt before presenting to Northstar Robotics.",
    tags: ["script:proper-nouns", "script:names", "condition:clean"],
    difficulty: "medium",
  },
  {
    id: "code-terms",
    title: "Code and product terms",
    purpose: "Covers technical words that often get normalized incorrectly.",
    text: "Set API version v1, enable PCM s16le streaming, and restart the websocket client.",
    tags: ["script:technical", "script:code", "condition:clean"],
    difficulty: "medium",
  },
  {
    id: "homophones",
    title: "Homophones",
    purpose: "Separates words that sound alike from context.",
    text: "Their team will meet there to review whether the weather delayed the release.",
    tags: ["script:homophones", "condition:clean"],
    difficulty: "medium",
  },
  {
    id: "very-short",
    title: "Very short command",
    purpose: "Validates tiny utterances and fast finalization.",
    text: "Start timer.",
    tags: ["script:short", "script:command"],
    difficulty: "short",
  },
  {
    id: "long-utterance",
    title: "Long uninterrupted utterance",
    purpose: "Tests sustained speech and final transcript stability.",
    text: "During the quarterly review, the operations team compared customer feedback, support latency, and model quality before deciding which workflow should be improved first.",
    tags: ["script:long", "script:business", "condition:clean"],
    difficulty: "long",
  },
  {
    id: "filler-words",
    title: "Natural filler words",
    purpose: "Keeps hesitations and restarts visible for correction.",
    text: "Um, I think we should, you know, wait five minutes and then try the recording again.",
    tags: ["script:filler", "script:natural-speech"],
    difficulty: "medium",
  },
  {
    id: "multilingual-names",
    title: "Multilingual names",
    purpose: "Exercises accents, transliterations, and non-English names.",
    text: "Saoirse, Jose, Aiko, and Francois discussed the Sao Paulo launch schedule.",
    tags: ["script:multilingual", "script:proper-nouns"],
    difficulty: "medium",
  },
  {
    id: "noisy-room-phrasing",
    title: "Noisy-room phrasing",
    purpose: "Captures repetition and clarification that happen in loud rooms.",
    text: "Please confirm the address again: 42 Market Street, suite 700, not 17.",
    tags: ["script:noisy-room", "script:numbers"],
    difficulty: "medium",
  },
  {
    id: "command-like",
    title: "Command-like phrase",
    purpose: "Differentiates dictation from app-control language.",
    text: "Open the dashboard, filter by failed runs, and export the summary.",
    tags: ["script:command", "script:workflow"],
    difficulty: "medium",
  },
  {
    id: "acronyms",
    title: "Acronyms and initials",
    purpose: "Checks capitalization-sensitive technical tokens.",
    text: "The QA team compared CPU, GPU, and API logs before filing the RCA.",
    tags: ["script:acronyms", "script:technical"],
    difficulty: "short",
  },
];

export function findDictationScript(id: string): DictationScript | null {
  return DICTATION_SCRIPTS.find((script) => script.id === id) ?? null;
}
