import { createClient } from "@connectrpc/connect";
import { ReviewService } from "@vrooli/proto-types/content-desk/v1/review/review_pb";
import { transport } from "./client";

export const reviewClient = createClient(ReviewService, transport);
