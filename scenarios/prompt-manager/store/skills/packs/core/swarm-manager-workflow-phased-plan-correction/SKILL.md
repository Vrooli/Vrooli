# Phased Plan Slice Correction

Correct your latest slice result in this same conversation. Resolve every defect the review note names, then re-verify. You may update progress only for the bound Plan Manager execution after validation passes. Do not mutate plan content or backlog records. Return one complete replacement result under the same outcome decision table as the original slice — not a delta, and not a new slice. A correction turn never sets `correctionRequired` again: if defects remain that you cannot fix here, return `blocked` and name them. Do not rely on this conversation for anything beyond this correction.

<review_note>{{.review_note}}</review_note>
