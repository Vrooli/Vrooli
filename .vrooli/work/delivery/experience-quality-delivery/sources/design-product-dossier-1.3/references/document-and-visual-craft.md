# Document and visual craft

## Reader experience

Create a purposeful cover with the product/service/system, document purpose, version/date, review status and authoring context. Use a linked table of contents, consistent heading hierarchy, running chapter cues, page numbers and direct references. Make the opening identify the problem/outcome, recommendation, current/approved/proposed boundaries, major blockers and smallest useful review route. For a long dossier, include a meaningful overview visual and linked contents/decision route. A systems context map or service blueprint can do the explanatory work of an app screen. Use the project's brand and domain-appropriate specimens; do not impose a universal palette, photographic cover, typography, page count or chapter list.

Use a restrained type and color system, generous but efficient spacing, legible figure labels, consistent captions and predictable decision callouts. Avoid dense walls of text, tiny tables, decorative filler, repeated boilerplate and empty chapter pages. Keep lengthy evidence tables and review logs in navigable appendices while surfacing their meaningful findings.

## Diagrams

Choose a diagram for the question: context/relationship, activity, sequence, state, domain/class, deployment only if relevant. Use UML notation correctly when labeling a diagram UML; label simpler conceptual diagrams honestly. Plan the final physical figure size before rendering. A wide graph with readable source labels may shrink to unreadable text on a portrait page; split it, use a focused table, or omit it rather than enlarging only the raster resolution. Every arrow, multiplicity, decision and transition should have a coherent meaning. Add readable captions explaining the takeaway, not merely naming the chart.

Check diagrams against the prose, source facts and other diagrams. Do not claim executable validation from rendering alone. Preserve editable source where useful.

## Mockups and alternatives

For each visual identify the screen, scenario, viewport, version, source and approval status. Distinguish historical approved artwork, observed implementation screenshots, and newly proposed design. A concept image is not an exact implementable UI contract; supplement it with precise interaction and content rules.

For comparisons hold the scenario and fidelity reasonably constant. Put alternatives side by side when legible; otherwise use facing/sequential pages with repeated context. State what actually differs and why the recommendation is preferable. Do not generate decorative variants that do not resolve a meaningful question.

For interface scope, show full flows and important states, not just attractive default screens. Include applicable mobile layouts and realistic content lengths. Explain missing, empty and failure states. Ensure controls, units and labels remain readable in the final document.

Before scaling a new visual system across a long dossier, render and critique a representative annotated visual spread, a dense technical/commercial page and an end-to-end flow. Settle hierarchy, readability, figure scale and navigation on that sample, then extend consistently.

## Separate visual atlas

For substantial visual product/UX scope, provide a separately browsable atlas alongside the complete dossier unless the user explicitly limits the deliverables or the adopted project method records an exception. Choose a suitable gallery, document or other supported surface within the existing project; a PDF is not compulsory. Make both entry modes obvious in the delivery note and project entry: **Browse visuals** and **Read full design/specification**. Link the atlas and specification to each other.

The atlas is a view of the current selected and candidate assets in the project's existing version records, not a second design authority or approval queue. Give it a clear index by journey/screen and state, stable page/state IDs, viewport variants when applicable, and the exact source-specification revision and visual version identities. On each page or item, identify the state, desktop/mobile or other viewport, selected/candidate status and actual approval scope; link to the corresponding detailed specification and durable original/editable asset. Keep alternatives comparable and distinguish them from the current recommendation. Preserve superseded versions and their historical status without presenting them as current.

Let the reader inspect full mockups independently of the prose. Preserve original-resolution images or vector sources, and offer full-size readable viewing plus focused details where a composed board would make controls too small. Pair applicable desktop/mobile states without shrinking both into illegible thumbnails; each must remain individually inspectable. An upscaled or downsampled whole-board thumbnail is not sufficient detail evidence. The main dossier must still expose readable consequential details and explanations.

Inspect the exact rendered atlas at its intended viewing scale, including dense labels, controls and representative viewport variants; inspect every delivered page for legibility, clipping and missing states. Verify the index, page/state identifiers, specification links and original-asset links against actual accessible destinations and exact revisions, including return navigation. Compare the atlas assets with the current dossier and version records after changes. Report the tested surface and any device/viewer limits; successful rendering or source hashes alone do not establish readability or live interaction. Apply the delivery-set reconciliation in the main method so the atlas, specification, originals and entry links advance together.

## Exact delivery inspection

Before and after text cleanup or normalization in an editable document, compare the identity and order of non-text structures in the document, headers and footers. In OOXML, preserve drawings, breaks, fields/instructions, hyperlinks and bookmarks. Do not assign a paragraph's or run's whole text merely to perform a no-op or broad replacement: setters can remove drawing and break nodes while leaving media files in the package. Modify only affected text-bearing nodes, preferably exact `w:t` nodes, and assert the protected structures remain unchanged unless their change was requested.

Count actual exported image placements, not merely embedded media or shared resource references. For example, PDF `get_image_info` describes placements; `get_images` can list shared resources on pages where the image was not drawn. Verify expected figure identity, order and placement against the exact export, then inspect the rendered pages. A retained image file, caption, relationship or resource count alone cannot prove that the figure is visible.

Render the exact export being delivered. Inspect every page for clipping, blank gaps, broken glyphs, orphaned headings, unreadable diagrams, image compression and table overflow. Verify contents links, external references, page order, bookmarks when available, and editable/export parity. Re-render after corrections that can alter pagination.

Use screenshots or a PDF as a review fallback when the primary surface fails, preserving a text-accessible version. Report the actual tested surface and untested environments. Do not label a document polished solely because a generator completed successfully.

Check real heading structure, meaningful links, text alternatives, table headers, logical reading order, selectable text and the accessibility of the actual export. Clean page images do not establish semantic accessibility. Report untested assistive technologies and devices rather than claiming conformance from automated or agent-only checks.

## Reviewable proposed design

When the purpose is to decide how a product, service or system should be designed, make the proposal and its critique visible within the document. Use consistent text-labeled treatments for sourced/current or owner-approved design, recommended design, decisions needed, and issues or evidence gaps. Keep assumptions and observed evidence distinguishable within those treatments. Include the recommendation's reason, alternatives and consequences where material. Reinforce labels with restrained color if requested; never rely on color alone. Place each treatment beside the relevant mockup or specification and link it to the project-owned review index. Preserve the distinction between owner-approved facts and author proposals.

Let mockups and precise interaction specifications lead a product-experience dossier. Use UML or other diagrams only when they explain a relationship, state or sequence better than the mockups. A reference document may establish the desired depth without dictating the new document's visual style or chapter balance.


## Reader-contract gate

Use the rendered sample and final artifact to answer: What is being proposed? What matters for this reader? What is sourced, assumed or still unverified? What needs deciding, and where is its evidence? What follows after review? Test navigation from the opening to a consequential choice, its detail and the application handoff. Fix subtle labels, unlinked choices and unreadable dense pages before scaling; do not treat attractive screenshots or a contents list alone as proof of review usability.
