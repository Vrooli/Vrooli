#!/bin/bash

# Fix BookmarkListDbFactory.ts
sed -i 's/Prisma\.BookmarkList,/any, \/\/ Using any for model type/' BookmarkListDbFactory.ts
sed -i 's/Prisma\.BookmarkListInclude,/any, \/\/ Using any for include type/' BookmarkListDbFactory.ts
sed -i 's/): Promise<Prisma\.BookmarkList>/): Promise<any>/g' BookmarkListDbFactory.ts
sed -i 's/: Prisma\.BookmarkListInclude/: any/g' BookmarkListDbFactory.ts
sed -i 's/record: Prisma\.BookmarkList/record: any/g' BookmarkListDbFactory.ts
sed -i 's/Prisma\.BookmarkList\[\]/any[]/g' BookmarkListDbFactory.ts

# Fix CommentDbFactory.ts  
sed -i 's/Prisma\.comment,/any, \/\/ Using any for model type/' CommentDbFactory.ts
sed -i 's/Prisma\.commentInclude,/any, \/\/ Using any for include type/' CommentDbFactory.ts
sed -i 's/): Promise<Prisma\.comment>/): Promise<any>/g' CommentDbFactory.ts
sed -i 's/: Prisma\.commentInclude/: any/g' CommentDbFactory.ts
sed -i 's/record: Prisma\.comment/record: any/g' CommentDbFactory.ts
sed -i 's/Prisma\.comment\[\]/any[]/g' CommentDbFactory.ts
sed -i 's/generatePK()/generatePK().toString()/g' CommentDbFactory.ts
sed -i 's/ownedByUser: { connect: { id: generatePK() } }/ownedByUser: { connect: { id: generatePK().toString() } }/g' CommentDbFactory.ts
sed -i 's/ownedByTeam: { connect: { id: generatePK() } }/ownedByTeam: { connect: { id: generatePK().toString() } }/g' CommentDbFactory.ts

# Fix BookmarkDbFactory.ts
sed -i 's/Prisma\.Bookmark,/any, \/\/ Using any for model type/' BookmarkDbFactory.ts  
sed -i 's/Prisma\.BookmarkInclude,/any, \/\/ Using any for include type/' BookmarkDbFactory.ts
sed -i 's/): Promise<Prisma\.Bookmark>/): Promise<any>/g' BookmarkDbFactory.ts
sed -i 's/: Prisma\.BookmarkInclude/: any/g' BookmarkDbFactory.ts
sed -i 's/record: Prisma\.Bookmark/record: any/g' BookmarkDbFactory.ts
sed -i 's/Prisma\.Bookmark\[\]/any[]/g' BookmarkDbFactory.ts
sed -i 's/generatePK()/generatePK().toString()/g' BookmarkDbFactory.ts
sed -i 's/by: { connect: { id: generatePK() } }/by: { connect: { id: generatePK().toString() } }/g' BookmarkDbFactory.ts

# Fix ViewDbFactory.ts
sed -i 's/Prisma\.view,/any, \/\/ Using any for model type/' ViewDbFactory.ts
sed -i 's/Prisma\.viewInclude,/any, \/\/ Using any for include type/' ViewDbFactory.ts  
sed -i 's/): Promise<Prisma\.view>/): Promise<any>/g' ViewDbFactory.ts
sed -i 's/: Prisma\.viewInclude/: any/g' ViewDbFactory.ts
sed -i 's/record: Prisma\.view/record: any/g' ViewDbFactory.ts
sed -i 's/Prisma\.view\[\]/any[]/g' ViewDbFactory.ts
sed -i 's/generatePK()/generatePK().toString()/g' ViewDbFactory.ts
sed -i 's/by: { connect: { id: generatePK() } }/by: { connect: { id: generatePK().toString() } }/g' ViewDbFactory.ts

# Fix ReactionDbFactory.ts
sed -i 's/Prisma\.reaction,/any, \/\/ Using any for model type/' ReactionDbFactory.ts
sed -i 's/Prisma\.reactionInclude,/any, \/\/ Using any for include type/' ReactionDbFactory.ts
sed -i 's/): Promise<Prisma\.reaction>/): Promise<any>/g' ReactionDbFactory.ts
sed -i 's/: Prisma\.reactionInclude/: any/g' ReactionDbFactory.ts
sed -i 's/record: Prisma\.reaction/record: any/g' ReactionDbFactory.ts
sed -i 's/Prisma\.reaction\[\]/any[]/g' ReactionDbFactory.ts
sed -i 's/generatePK()/generatePK().toString()/g' ReactionDbFactory.ts
sed -i 's/by: { connect: { id: generatePK() } }/by: { connect: { id: generatePK().toString() } }/g' ReactionDbFactory.ts

# Fix ReactionSummaryDbFactory.ts
sed -i 's/Prisma\.reaction_summary,/any, \/\/ Using any for model type/' ReactionSummaryDbFactory.ts
sed -i 's/Prisma\.reaction_summaryInclude,/any, \/\/ Using any for include type/' ReactionSummaryDbFactory.ts
sed -i 's/): Promise<Prisma\.reaction_summary>/): Promise<any>/g' ReactionSummaryDbFactory.ts
sed -i 's/: Prisma\.reaction_summaryInclude/: any/g' ReactionSummaryDbFactory.ts
sed -i 's/record: Prisma\.reaction_summary/record: any/g' ReactionSummaryDbFactory.ts
sed -i 's/Prisma\.reaction_summary\[\]/any[]/g' ReactionSummaryDbFactory.ts
sed -i 's/generatePK()/generatePK().toString()/g' ReactionSummaryDbFactory.ts

echo "Type fixes applied!"