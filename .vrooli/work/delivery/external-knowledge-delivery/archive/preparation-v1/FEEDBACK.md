# Operator Feedback

## Open

### FB-001 — 2026-10-14 — Selected Drive folders first

> Google Drive is the most important so I would've once selected Google Drive folders to be what we focus on initially but we should certainly be thinking about and setting up the architecture so that when we extend this to Google calendar and Gmail and everything like that, we don't have to reinvent the wheel

Disposition: incorporated in the architecture and candidate queue. Activation remains deferred.

### FB-002 — 2026-10-14 — Approved architecture and additional research

> Thanks great findings and recommendations. This is exactly how I would architect it so great job here. I agree that this would for document manager be an extension of its contract and I also agree that we can't store. Literally nothing for each file they get synced for the Google Drive one and also I guess one thing you didn't mention is to make the AI search more effective. What we do for scenarios is sort of define what sorts of questions we would like or we would like anticipate an agent to ask and we in bed like a different variations of the document and chunks of the document with those questions. I don't know if I don't know if I explain that right but it basically we sort of anticipate how we would ask about these documents and that improves the results also for documents specifically they're a bit different than some things I get embedded since they do need to be chunked like I mentioned I believe we do have chunking strategies somewhere. I don't know if it's in the shared AI search packages or in knowledge observatory or somewhere else but we would need to decide how we're going to chunk the things that get parted and if there be different strategies, depending on the type of document, and you know what we actually could embed and what not to embed and like how it differs because I don't think we have currently embedded everything that would be needed like spreadsheets and images and PDF files and stuff like that. Like we're gonna need to decide how to do those things and where those capabilities live.

> Also for integration hub that is in fact the right location to do that the Google oh off connection but it's also true that I don't know how developed integration hub is. I don't think we've actually used it yet so it may require a lot more development to get it in a working state not exactly sure.

Disposition: shared chunker/composer and format registry located. Anticipated-question views require an explicit measured recipe; do not claim eval questions are already indexed representations. Integration Hub pilot needs qualification and extension.

### FB-003 — 2026-10-14 — Disabled finite team and email UX handoff

> And for the Gmail stuff and like the new email scenario that would need to be created, we should probably think about what we want there and cause I think it would make sense for all of this that we were discussing now we should establish a finite delivery team and not enable it yet, but you know store all of this information you found and you know like the information you keep finding all the research and stuff. We should organize this into a finite delivery team so that when we are ready to implement this, we can enable the team and have it set up everything we're gonna probably need to do the O off stuff before we enable that team and you're gonna have to walk me through that but yeah, I think I'm for the majority of this work it would make sense that it all gets done at one finite delivery team since it is a lot of work, but it is important that the agents implementing it understand the overall architecture, and you know, like the context around all of this.

> So go ahead and look into how the proper way to set up a fun delivery team is and start doing that and also give me a detailed prompt about the email scenario and what capabilities we need for it and I'm going to pass that along to my ChatGPT. to have it like I'm gonna work with that agent to design the UX for the email scenario and you know more specifications beyond this Gmail stuff we already know about. Just because the ChatGPT dot is better at doing the UX design. It just makes sense for me to design it there and then I'll have it update the finite delivery team so also include the name of the finite delivery team in the prompt you give me.

Disposition: commission `external-knowledge-delivery` disabled; preserve architecture/research and an email-design prompt. Pending: future UX decisions, guided Google setup and later explicit activation. Never wake or enable this team merely because new research or UX arrives.
