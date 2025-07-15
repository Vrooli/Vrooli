import Box from "@mui/material/Box";
import Skeleton from "@mui/material/Skeleton";
import { styled, keyframes } from "@mui/material/styles";

const shimmer = keyframes`
    0% {
        opacity: 0.7;
    }
    50% {
        opacity: 1;
    }
    100% {
        opacity: 0.7;
    }
`;

const LoadingContainer = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    justifyContent: "center",
    flex: 1,
    padding: theme.spacing(3),
}));

const MessageSkeleton = styled(Box)(({ theme }) => ({
    display: "flex",
    gap: theme.spacing(2),
    width: "100%",
    maxWidth: 600,
    marginBottom: theme.spacing(3),
    animation: `${shimmer} 2s ease-in-out infinite`,
}));

const BubbleSkeleton = styled(Skeleton)(({ theme }) => ({
    borderRadius: theme.spacing(2),
    flexGrow: 1,
}));

interface EmptyStateLoadingProps {
    showMessages?: boolean;
    showBotInfo?: boolean;
}

export function EmptyStateLoading({ showMessages = false, showBotInfo = false }: EmptyStateLoadingProps) {
    if (showMessages) {
        // Show skeleton messages as if there's a conversation loading
        return (
            <LoadingContainer sx={{ justifyContent: "flex-start", paddingTop: 4 }}>
                {/* User message skeleton */}
                <MessageSkeleton sx={{ justifyContent: "flex-end" }}>
                    <BubbleSkeleton 
                        variant="rectangular" 
                        width="70%" 
                        height={60}
                        sx={{ bgcolor: "primary.main", opacity: 0.1 }}
                    />
                </MessageSkeleton>
                
                {/* Bot response skeleton */}
                <MessageSkeleton>
                    <Skeleton variant="circular" width={40} height={40} />
                    <Box sx={{ flexGrow: 1, maxWidth: "70%" }}>
                        <BubbleSkeleton variant="rectangular" height={80} />
                        <Box sx={{ display: "flex", gap: 1, mt: 1 }}>
                            <Skeleton variant="rectangular" width={60} height={24} sx={{ borderRadius: 1 }} />
                            <Skeleton variant="rectangular" width={80} height={24} sx={{ borderRadius: 1 }} />
                        </Box>
                    </Box>
                </MessageSkeleton>

                {/* Another user message */}
                <MessageSkeleton sx={{ justifyContent: "flex-end" }}>
                    <BubbleSkeleton 
                        variant="rectangular" 
                        width="60%" 
                        height={50}
                        sx={{ bgcolor: "primary.main", opacity: 0.1 }}
                    />
                </MessageSkeleton>

                {/* Bot typing indicator */}
                <MessageSkeleton>
                    <Skeleton variant="circular" width={40} height={40} />
                    <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                        <Skeleton variant="circular" width={8} height={8} />
                        <Skeleton variant="circular" width={8} height={8} sx={{ animationDelay: "0.2s" }} />
                        <Skeleton variant="circular" width={8} height={8} sx={{ animationDelay: "0.4s" }} />
                    </Box>
                </MessageSkeleton>
            </LoadingContainer>
        );
    }

    // Initial loading state - show centered loading skeleton
    return (
        <LoadingContainer>
            {/* Bot header skeleton - only show if showBotInfo is true */}
            {showBotInfo && (
                <Box sx={{ display: "flex", alignItems: "center", gap: 2, mb: 2 }}>
                    <Skeleton 
                        variant="circular" 
                        width={48} 
                        height={48}
                    />
                    <Skeleton 
                        variant="text" 
                        width={150} 
                        height={32}
                    />
                </Box>
            )}
            
            {/* Description skeleton */}
            <Skeleton 
                variant="text" 
                width={400} 
                height={24}
                sx={{ mb: 4 }}
            />
            
            {/* Feature cards skeleton */}
            <Box 
                sx={{ 
                    display: "grid", 
                    gridTemplateColumns: "repeat(4, 1fr)", 
                    gap: 1.5,
                    width: "100%",
                    maxWidth: 500,
                    mb: 3,
                }}
            >
                {[1, 2, 3, 4].map((i) => (
                    <Box key={i} sx={{ animation: `${shimmer} 2s ease-in-out infinite`, animationDelay: `${i * 0.1}s` }}>
                        <Skeleton variant="rectangular" height={80} sx={{ borderRadius: 1 }} />
                    </Box>
                ))}
            </Box>
            
            {/* Suggestions skeleton */}
            <Box sx={{ display: "flex", gap: 1, flexWrap: "wrap", justifyContent: "center" }}>
                <Skeleton variant="rectangular" width={140} height={32} sx={{ borderRadius: 3 }} />
                <Skeleton variant="rectangular" width={120} height={32} sx={{ borderRadius: 3 }} />
                <Skeleton variant="rectangular" width={100} height={32} sx={{ borderRadius: 3 }} />
            </Box>
        </LoadingContainer>
    );
}
