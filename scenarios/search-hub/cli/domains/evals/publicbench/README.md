# Public benchmark inputs for Search Hub evals

This package imports the **official SciFact dev split** as source-linked Search Hub evaluation cases and registers a Search Hub-owned lexical provider over the full staged corpus. It does not download the dataset or invoke a model. The API provider is a bounded BM25 implementation exposed through Search Hub's ordinary provider endpoint and scored by the shared Search Hub EvalService grader.

## Source and rights

- Dataset: [SciFact](https://github.com/allenai/scifact), described in the authors' [data schema](https://github.com/allenai/scifact/blob/master/doc/data.md) and [README](https://github.com/allenai/scifact/blob/master/README.md).
- Split: `claims_dev.jsonl` (`dev`). The authors state that test labels are not public, so this importer intentionally refuses a test-split fallback.
- Expected retrieval documents and their per-document labels come from each dev claim's `evidence[doc_id][].label`. Every expected ID must exist in the full `corpus.jsonl`; absent evidence is never silently dropped.
- Claims and evidence annotations: CC BY 4.0. Abstract corpus: ODC-By 1.0. Preserve the [official license notice](https://github.com/allenai/scifact/blob/master/LICENSE.md) and cite the [SciFact paper](https://arxiv.org/abs/2004.14974).
- The exact `claims_dev.jsonl` and `corpus.jsonl` bytes are bound by `corpus_sha256` in the suite description and every case note. The suite ID includes the selected case count and a digest prefix, so importing another subset or source artifact creates a distinct suite instead of replacing its denominator. Each case includes `scifact/dev/<claim id>`, source split, the source annotation path and label values, and both applicable licenses.

## Bounded subset

`--max-cases` defaults to 20 and is capped at the 300-claim dev split size. The importer sorts claim IDs numerically and selects the first N records with evidence annotations; it counts and skips claims without evidence for this retrieval task. It keeps the complete upstream corpus so the evaluation retains benchmark distractors, not only gold abstracts.

```text
search-hub evals import-publicbench \
  --dataset scifact-dev \
  --data-dir <staged-sciFact-data-directory> \
  --provider-id publicbench.scifact.dev \
  --max-cases 20
```

The importer validates evidence IDs against all documents in `corpus.jsonl`, registers the provider descriptor through RegistryService, and registers the suite through EvalService. The provider searches every record in the same full corpus, returning document IDs, titles, snippets, and BM25 scores. Both provider descriptor and each provider request pin the exact `corpus.jsonl` SHA-256; Search Hub rejects requests with HTTP 409 if the running API loaded different corpus bytes than the suite importer staged.

## Runtime, provider, and data staging contract

1. Stage the authors' official release and extract `claims_dev.jsonl` and `corpus.jsonl` into a persistent local directory. The owner-provided archive is [data.tar.gz](https://scifact.s3-us-west-2.amazonaws.com/release/latest/data.tar.gz); this command does not acquire it.
2. Configure the Search Hub API process with `SEARCH_HUB_SCIFACT_DATA_DIR=<that exact directory>`. The API loads and indexes `corpus.jsonl` at startup. It never loads claims into the provider index.
3. Restart Search Hub through its scenario owner after the environment and data are staged. Without a configured/readable corpus, the provider endpoint responds `503`; provider registration's normal reachability probe will then fail closed.
4. Run the import command with `--data-dir` set to the same directory. It deterministically builds source-linked cases, registers provider ID from `--provider-id` at `/api/v1/publicbench/scifact/search`, then registers the suite bound to that provider. Provider search is lexical BM25 over the complete upstream abstract corpus, without embeddings or inference.
5. After host capacity permits, run `search-hub evals validate <suite-id>` and inspect its corpus evidence. Review and freeze the intended denominator, then run `search-hub evals run <suite-id> --tag publicbench-baseline` to record shared-grader scores.

The endpoint uses the same JSON provider contract as other Search Hub leaves: POST `{"query":"...","limit":10}` returns `results[]` with `id`, `title`, raw BM25 `score`, `snippet`, and `path`. The provider ID is caller-selected and should be unique per staged source version; the suite ID itself contains the selected case count and source-file digest. Registration runs Search Hub's ordinary endpoint reachability probe, which sends one fixed probe query; no benchmark query or inference is part of import.
