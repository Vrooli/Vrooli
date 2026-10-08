package teams

import (
	"strings"
	"testing"
)

func TestCmdKnowledgeListSendsTopicPrefixQueryParam(t *testing.T) {
	fc := &fakeContext{t: t, response: KnowledgeListResponse{TeamID: "marketing-crew"}}
	err := cmdKnowledgeList(fc, []string{
		"marketing-crew",
		"--topic-prefix=research-inbox/",
	})
	if err != nil {
		t.Fatalf("cmdKnowledgeList error = %v", err)
	}
	fc.assertMethodPath(t, "GET", "/teams/marketing-crew/knowledge?last=20&topic_prefix=research-inbox%2F")
}

func TestCmdKnowledgeListSendsTopicQueryParam(t *testing.T) {
	fc := &fakeContext{t: t, response: KnowledgeListResponse{TeamID: "marketing-crew"}}
	err := cmdKnowledgeList(fc, []string{
		"marketing-crew",
		"--topic=audience-scan/foo",
	})
	if err != nil {
		t.Fatalf("cmdKnowledgeList error = %v", err)
	}
	fc.assertMethodPath(t, "GET", "/teams/marketing-crew/knowledge?last=20&topic=audience-scan%2Ffoo")
}

func TestCmdKnowledgeListRejectsTopicAndTopicPrefixTogether(t *testing.T) {
	fc := &fakeContext{t: t}
	err := cmdKnowledgeList(fc, []string{
		"marketing-crew",
		"--topic=research-inbox",
		"--topic-prefix=research-inbox/",
	})
	if err == nil {
		t.Fatal("expected error when both --topic and --topic-prefix supplied, got nil")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error = %v, want 'mutually exclusive'", err)
	}
	if fc.gotMethod != "" {
		t.Errorf("expected no API call when validation fails, got %s %s", fc.gotMethod, fc.gotPath)
	}
}

func TestCmdKnowledgeListNoFilterOmitsTopicParams(t *testing.T) {
	fc := &fakeContext{t: t, response: KnowledgeListResponse{TeamID: "marketing-crew"}}
	err := cmdKnowledgeList(fc, []string{"marketing-crew"})
	if err != nil {
		t.Fatalf("cmdKnowledgeList error = %v", err)
	}
	fc.assertMethodPath(t, "GET", "/teams/marketing-crew/knowledge?last=20")
}
